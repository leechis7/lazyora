package oracle

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPendingChangesTrackReadsWritesAndResolution(t *testing.T) {
	for _, commit := range []bool{true, false} {
		s, mock := mockSession(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
		if _, err := s.Execute(context.Background(), "SELECT 1 FROM dual"); err != nil || s.PendingChanges() {
			t.Fatalf("SELECT marked pending: %v", err)
		}
		mock.ExpectExec("UPDATE items").WillReturnResult(sqlmock.NewResult(0, 1))
		if _, err := s.Execute(context.Background(), "UPDATE items SET n = 2"); err != nil || !s.PendingChanges() {
			t.Fatalf("UPDATE not pending: %v", err)
		}
		mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
		if _, err := s.Execute(context.Background(), "SELECT 1 FROM dual"); err != nil || !s.PendingChanges() {
			t.Fatalf("SELECT cleared pending DML: %v", err)
		}
		if commit {
			mock.ExpectCommit()
		} else {
			mock.ExpectRollback()
		}
		if err := s.Finish(commit); err != nil || s.PendingChanges() {
			t.Fatalf("resolution did not clear pending: %v", err)
		}
	}
}

func TestFailedStatementStillWarnsAndFailedFinishCannotBeRetriedAsSuccess(t *testing.T) {
	for _, commit := range []bool{true, false} {
		s, mock := mockSession(t)
		mock.ExpectBegin()
		mock.ExpectExec("BEGIN").WillReturnError(errors.New("cancelled during block"))
		if _, err := s.Execute(context.Background(), "BEGIN do_work; END;"); err == nil || !s.PendingChanges() {
			t.Fatal("failed block did not preserve change warning")
		}
		if commit {
			mock.ExpectCommit().WillReturnError(errors.New("connection lost"))
		} else {
			mock.ExpectRollback().WillReturnError(errors.New("connection lost"))
		}
		if err := s.Finish(commit); err == nil || !s.PendingChanges() {
			t.Fatal("failed transaction resolution did not keep warning")
		}
		if err := s.Finish(commit); err == nil {
			t.Fatal("failed resolution was silently accepted on retry")
		}
		if _, err := s.Execute(context.Background(), "SELECT 1 FROM dual"); err == nil {
			t.Fatal("uncertain session accepted more SQL")
		}
	}
}
