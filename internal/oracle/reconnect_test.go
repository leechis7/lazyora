package oracle

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func pendingSession(t *testing.T) (*Session, sqlmock.Sqlmock) {
	t.Helper()
	s, mock := mockSession(t)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE items").WillReturnResult(sqlmock.NewResult(0, 1))
	if _, err := s.Execute(context.Background(), "UPDATE items SET n=2"); err != nil {
		t.Fatal(err)
	}
	return s, mock
}

func TestReconnectRequiresExplicitResolution(t *testing.T) {
	for _, choice := range []ReconnectResolution{"", "invalid", ReconnectDiscardUncertain} {
		t.Run(string(choice), func(t *testing.T) {
			old, mock := pendingSession(t)
			got, err := reconnect(context.Background(), old, choice, func() (*Session, error) {
				t.Fatal("target opened without consent")
				return nil, nil
			})
			if err == nil || got != old || !old.PendingChanges() {
				t.Fatal("unconfirmed reconnect changed the old session")
			}
			mock.ExpectRollback()
		})
	}
}

func TestReconnectUncertainRecoveryRequiresSeparateConsent(t *testing.T) {
	old, mock := pendingSession(t)
	mock.ExpectCommit().WillReturnError(errors.New("commit reply lost"))
	if err := old.Finish(true); err == nil || !old.TransactionUncertain() {
		t.Fatal("failed commit did not mark uncertain outcome")
	}
	// Ordinary retry must not silently discard the unknown transaction.
	unused, unusedMock := mockSession(t)
	unusedMock.ExpectClose()
	got, err := reconnect(context.Background(), old, ReconnectCommit, func() (*Session, error) { return unused, nil })
	if err == nil || got != old || !old.TransactionUncertain() {
		t.Fatal("retry silently cleared uncertainty")
	}
	// Even explicit abandonment must preserve the session if the target fails.
	got, err = reconnect(context.Background(), old, ReconnectDiscardUncertain, func() (*Session, error) {
		return nil, errors.New("target unavailable")
	})
	if err == nil || got != old || !old.TransactionUncertain() {
		t.Fatal("target failure lost uncertain session")
	}
	next, _ := mockSession(t)
	got, err = reconnect(context.Background(), old, ReconnectDiscardUncertain, func() (*Session, error) { return next, nil })
	if err != nil || got != next || next.TransactionUncertain() {
		t.Fatalf("explicit uncertain recovery failed: %v", err)
	}
}

func TestReconnectTargetFailurePreservesTransaction(t *testing.T) {
	for _, choice := range []ReconnectResolution{ReconnectCommit, ReconnectRollback} {
		old, mock := pendingSession(t)
		want := errors.New("target unavailable")
		got, err := reconnect(context.Background(), old, choice, func() (*Session, error) { return nil, want })
		if !errors.Is(err, want) || got != old || !old.PendingChanges() || len(old.PendingWork().Statements) != 1 {
			t.Fatalf("lost existing transaction: %v", err)
		}
		// Existing session must still accept SQL and explicit rollback.
		mock.ExpectExec("UPDATE items").WillReturnResult(sqlmock.NewResult(0, 1))
		if _, err := old.Execute(context.Background(), "UPDATE items SET n=3"); err != nil {
			t.Fatal(err)
		}
		mock.ExpectRollback()
	}
}

func TestReconnectResolvesOnlyAfterOpeningTarget(t *testing.T) {
	for _, choice := range []ReconnectResolution{ReconnectCommit, ReconnectRollback} {
		t.Run(string(choice), func(t *testing.T) {
			old, mock := pendingSession(t)
			next, _ := mockSession(t)
			got, err := reconnect(context.Background(), old, choice, func() (*Session, error) {
				if !old.PendingChanges() {
					t.Fatal("transaction resolved before target opened")
				}
				if choice == ReconnectCommit {
					mock.ExpectCommit()
				} else {
					mock.ExpectRollback()
				}
				return next, nil
			})
			if err != nil || got != next || old.PendingChanges() {
				t.Fatalf("reconnect failed: %v", err)
			}
			if err := old.conn.Raw(func(any) error { return nil }); !errors.Is(err, sql.ErrConnDone) {
				t.Fatal("old connection was not closed")
			}
		})
	}
}

func TestReconnectFinishFailureKeepsOldAndClosesTarget(t *testing.T) {
	for _, choice := range []ReconnectResolution{ReconnectCommit, ReconnectRollback} {
		t.Run(string(choice), func(t *testing.T) {
			old, mock := pendingSession(t)
			next, nextMock := mockSession(t)
			want := errors.New("connection lost during finish")
			if choice == ReconnectCommit {
				mock.ExpectCommit().WillReturnError(want)
			} else {
				mock.ExpectRollback().WillReturnError(want)
			}
			nextMock.ExpectClose()
			got, err := reconnect(context.Background(), old, choice, func() (*Session, error) { return next, nil })
			if !errors.Is(err, want) || got != old || !old.PendingChanges() || len(old.PendingWork().Statements) != 1 {
				t.Fatalf("uncertain transaction discarded: %v", err)
			}
			if err := next.conn.Raw(func(any) error { return nil }); !errors.Is(err, sql.ErrConnDone) {
				t.Fatal("unused target connection leaked")
			}
		})
	}
}

func TestReconnectCancellationPreservesTransaction(t *testing.T) {
	for _, duringOpen := range []bool{false, true} {
		old, mock := pendingSession(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if !duringOpen {
			cancel()
		}
		got, err := reconnect(ctx, old, ReconnectCommit, func() (*Session, error) {
			if !duringOpen {
				t.Fatal("opened target despite cancellation")
			}
			next, nextMock := mockSession(t)
			nextMock.ExpectClose()
			cancel()
			return next, nil
		})
		if !errors.Is(err, context.Canceled) || got != old || !old.PendingChanges() {
			t.Fatalf("cancellation resolved old transaction: %v", err)
		}
		mock.ExpectRollback()
	}
}

func TestReconnectFirstAndReadOnlyConnections(t *testing.T) {
	for _, existing := range []bool{false, true} {
		var old *Session
		if existing {
			var mock sqlmock.Sqlmock
			old, mock = mockSession(t)
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
			if _, err := old.Execute(context.Background(), "SELECT 1 FROM dual"); err != nil {
				t.Fatal(err)
			}
			mock.ExpectRollback() // release a read-only transaction on close
		}
		next, _ := mockSession(t)
		got, err := reconnect(context.Background(), old, "", func() (*Session, error) { return next, nil })
		if err != nil || got != next {
			t.Fatalf("clean connection failed: %v", err)
		}
	}
}
