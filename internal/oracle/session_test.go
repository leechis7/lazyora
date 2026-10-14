package oracle

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func mockSession(t *testing.T) (*Session, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	s := &Session{db: db, conn: conn}
	t.Cleanup(func() {
		s.Close()
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
	return s, mock
}

func TestDMLWaitsForExplicitCommit(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE items SET n = 1").WillReturnResult(sqlmock.NewResult(0, 2))
	ctx, cancel := context.WithCancel(context.Background())
	result, err := s.Execute(ctx, "UPDATE items SET n = 1;")
	if err != nil || result.Affected != 2 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	// Cancelling one operation must not implicitly roll back its session.
	cancel()
	mock.ExpectExec("UPDATE items SET n = 2").WillReturnResult(sqlmock.NewResult(0, 2))
	if _, err = s.Execute(context.Background(), "UPDATE items SET n = 2;"); err != nil {
		t.Fatal(err)
	}
	mock.ExpectCommit()
	if err = s.Finish(true); err != nil {
		t.Fatal(err)
	}
}

func TestCloseRollsBackUncommittedChanges(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM items").WillReturnResult(sqlmock.NewResult(0, 1))
	if _, err := s.Execute(context.Background(), "DELETE FROM items"); err != nil {
		t.Fatal(err)
	}
	mock.ExpectRollback()
	// The cleanup closes the session and must consume the rollback expectation.
}

func TestResultLimitAndNull(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"N", "OPTIONAL"})
	for i := 0; i <= MaxRows; i++ {
		rows.AddRow(i, nil)
	}
	mock.ExpectQuery("SELECT n, optional FROM items").WillReturnRows(rows).RowsWillBeClosed()
	result, err := s.Execute(context.Background(), "SELECT n, optional FROM items")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Truncated || len(result.Rows) != MaxRows || result.Rows[0][1] != "NULL" {
		t.Fatalf("invalid bounded result: %+v", result)
	}
	mock.ExpectRollback()
}

func TestSQLFailureDoesNotAutoCommit(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE missing SET n = 1").WillReturnError(errors.New("ORA-00942"))
	if _, err := s.Execute(context.Background(), "UPDATE missing SET n = 1"); err == nil {
		t.Fatal("expected SQL error")
	}
	mock.ExpectRollback()
	if err := s.Finish(false); err != nil {
		t.Fatal(err)
	}
}

func TestFormatValues(t *testing.T) {
	stamp := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	if got := formatValue(stamp); got != "2026-10-14 12:00:00Z" {
		t.Fatal(got)
	}
	if got := formatValue([]byte{0, 255}); got != "00ff" {
		t.Fatal(got)
	}
}
