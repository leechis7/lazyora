package oracle

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPendingJournalTracksSQLAndErrorsUntilSuccessfulResolution(t *testing.T) {
	for _, commit := range []bool{false, true} {
		t.Run(fmt.Sprint(commit), func(t *testing.T) {
			s, mock := mockSession(t)
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE items").WillReturnResult(sqlmock.NewResult(0, 3))
			if _, err := s.Execute(context.Background(), "UPDATE items SET n=2;"); err != nil {
				t.Fatal(err)
			}
			mock.ExpectExec("BEGIN").WillReturnError(errors.New("cancelled during PL/SQL"))
			if _, err := s.Execute(context.Background(), "BEGIN do_work; END;"); err == nil {
				t.Fatal("expected block failure")
			}
			mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
			if _, err := s.Execute(context.Background(), "SELECT 1 FROM dual"); err != nil {
				t.Fatal(err)
			}
			journal := s.PendingWork()
			if len(journal.Statements) != 2 || journal.Statements[0].SQL != "UPDATE items SET n=2" || journal.Statements[0].Affected != 3 || journal.Statements[0].At.IsZero() || journal.Statements[1].Error == "" {
				t.Fatalf("wrong journal: %+v", journal)
			}
			journal.Statements[0].SQL = "mutated snapshot"
			if s.PendingWork().Statements[0].SQL == "mutated snapshot" {
				t.Fatal("snapshot changed journal")
			}
			if commit {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}
			if err := s.Finish(commit); err != nil {
				t.Fatal(err)
			}
			if len(s.PendingWork().Statements) != 0 || s.PendingChanges() {
				t.Fatal("successful resolution retained journal")
			}
		})
	}
}

func TestPendingJournalSurvivesFailedCommitAndBoundsMemory(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE items").WillReturnResult(sqlmock.NewResult(0, 1))
	_, err := s.Execute(context.Background(), "UPDATE items SET n=1")
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectCommit().WillReturnError(errors.New("connection lost"))
	if s.Finish(true) == nil || len(s.PendingWork().Statements) != 1 {
		t.Fatal("failed commit lost journal")
	}
	if s.Finish(true) == nil || len(s.PendingWork().Statements) != 1 {
		t.Fatal("repeat commit silently cleared journal")
	}
	// Unicode truncation and omission counts must not silently lose the total.
	fresh := &Session{}
	for i := 0; i < MaxPendingStatements+3; i++ {
		fresh.recordStatement(time.Now(), fmt.Sprint(i), 0, nil)
	}
	work := fresh.PendingWork()
	if len(work.Statements) != MaxPendingStatements || work.Omitted != 3 || work.Statements[0].SQL != "3" {
		t.Fatal("journal bound lost oldest count or newest work")
	}
	fresh.recordStatement(time.Now(), strings.Repeat("한글", MaxPendingSQLBytes), -1, nil)
	last := fresh.PendingWork().Statements[MaxPendingStatements-1]
	if !last.Truncated || len(last.SQL) > MaxPendingSQLBytes || !utf8.ValidString(last.SQL) {
		t.Fatal("SQL limit broke UTF-8 or omitted truncation marker")
	}
}
