package oracle_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

func TestOracleSessionBrowserIntegration(t *testing.T) {
	admin := workflowSession(t)
	blocker := workflowSession(t)
	waiter := workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	execute := func(s *oracle.Session, sql string) {
		t.Helper()
		if _, err := s.Execute(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	own := func(s *oracle.Session) oracle.SessionEntry {
		t.Helper()
		list, err := s.ListSessions(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range list.Entries {
			if entry.Self {
				if entry.CanTerminate {
					t.Fatal("own session must not be eligible")
				}
				return entry
			}
		}
		t.Fatal("own session missing")
		return oracle.SessionEntry{}
	}
	adminEntry, blockerEntry, waiterEntry := own(admin), own(blocker), own(waiter)
	if _, err := admin.TerminateBlockingSession(ctx, adminEntry); !errors.Is(err, oracle.ErrSessionProtected) {
		t.Fatalf("self termination was not rejected: %v", err)
	}
	if _, err := admin.TerminateBlockingSession(ctx, blockerEntry); !errors.Is(err, oracle.ErrSessionChanged) {
		t.Fatalf("non-blocking session was not rejected: %v", err)
	}

	execute(admin, "CREATE TABLE lazyora_session_probe (id NUMBER PRIMARY KEY, n NUMBER)")
	t.Cleanup(func() {
		_ = admin.Finish(false)
		if _, err := admin.Execute(context.Background(), "DROP TABLE lazyora_session_probe PURGE"); err != nil {
			t.Error(err)
		}
	})
	execute(admin, "INSERT INTO lazyora_session_probe VALUES (1, 10)")
	execute(admin, "INSERT INTO lazyora_session_probe VALUES (2, 20)")
	if err := admin.Finish(true); err != nil {
		t.Fatal(err)
	}
	execute(admin, "UPDATE lazyora_session_probe SET n = 21 WHERE id = 2")
	execute(blocker, "UPDATE lazyora_session_probe SET n = 999 WHERE id = 1")
	waitCtx, stopWait := context.WithCancel(ctx)
	waitDone := make(chan error, 1)
	go func() {
		_, err := waiter.Execute(waitCtx, "UPDATE lazyora_session_probe SET n = n + 1 WHERE id = 1")
		waitDone <- err
		close(waitDone)
	}()
	defer func() {
		stopWait()
		_ = blocker.Finish(false)
		<-waitDone
		_ = waiter.Finish(false)
	}()

	var target oracle.SessionEntry
	for target.ID.SID == 0 {
		list, err := admin.ListSessions(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var relationship bool
		for _, entry := range list.Entries {
			if entry.ID == waiterEntry.ID {
				relationship = entry.BlockedBy == blockerEntry.ID
			}
		}
		if relationship {
			for _, entry := range list.Entries {
				if entry.ID == blockerEntry.ID && entry.Waiters > 0 && entry.CanTerminate {
					target = entry
				}
			}
		}
		if target.ID.SID == 0 {
			select {
			case <-ctx.Done():
				t.Fatal("Oracle did not expose an eligible blocker and its waiter")
			case <-time.After(250 * time.Millisecond):
			}
		}
	}
	stale := target
	stale.Logon = "20000101000000"
	if _, err := admin.TerminateBlockingSession(ctx, stale); !errors.Is(err, oracle.ErrSessionChanged) {
		t.Fatalf("stale login identity was not rejected: %v", err)
	}
	if _, err := admin.TerminateBlockingSession(ctx, target); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-waitDone:
		if err != nil {
			t.Fatalf("waiter did not resume: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("waiter still blocked")
	}
	if err := waiter.Finish(true); err != nil {
		t.Fatal(err)
	}
	if !admin.PendingChanges() {
		t.Fatal("browser changed the editor transaction state")
	}
	if err := admin.Finish(false); err != nil {
		t.Fatal(err)
	}
	got, err := admin.Execute(ctx, "SELECT n FROM lazyora_session_probe ORDER BY id")
	if err != nil || len(got.Rows) != 2 || got.Rows[0][0] != "11" || got.Rows[1][0] != "20" {
		t.Fatalf("blocker rollback or editor transaction isolation: rows=%v err=%v", got.Rows, err)
	}
}
