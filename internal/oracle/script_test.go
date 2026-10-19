package oracle

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestScriptStopsAtFirstErrorAndKeepsPendingDML(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE items SET n=1").WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectQuery("SELECT missing FROM items").WillReturnError(errors.New("missing column"))
	mock.ExpectRollback()
	report, err := s.Script(context.Background(), "UPDATE items SET n=1;\nSELECT missing FROM items;\nDELETE FROM items;")
	if err == nil || len(report.Steps) != 2 || report.Total != 3 || !s.PendingChanges() || len(s.PendingWork().Statements) != 1 {
		t.Fatalf("%+v %v", report, err)
	}
}

func TestScriptMultipleResultsAndExplicitCommit(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 1 FROM dual").WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	mock.ExpectExec("UPDATE items SET n=2").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 2 FROM dual").WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(2))
	mock.ExpectRollback()
	report, err := s.Script(context.Background(), "SELECT 1 FROM dual; UPDATE items SET n=2; COMMIT; SELECT 2 FROM dual;")
	if err != nil || len(report.Steps) != 4 || report.Steps[0].Result.Rows[0][0] != "1" || report.Steps[3].Result.Rows[0][0] != "2" || s.PendingChanges() {
		t.Fatalf("%+v %v", report, err)
	}
}

func TestScriptPreflightAndCancellationSubmitNothing(t *testing.T) {
	s, _ := mockSession(t)
	for _, text := range []string{"SELECT 1 FROM dual; SET SERVEROUTPUT ON", "PROMPT hi", "ROLLBACK TO SAVEPOINT x;", "SELECT &name FROM dual;", strings.Repeat("SELECT 1 FROM dual;", MaxScriptStatements+1), "COMMIT WORK WORK;"} {
		if _, err := s.Script(context.Background(), text); err == nil {
			t.Fatalf("unsupported script accepted: %s", text)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report, err := s.Script(ctx, "DELETE FROM items; SELECT 1 FROM dual;")
	if !errors.Is(err, context.Canceled) || len(report.Steps) != 1 || s.PendingChanges() {
		t.Fatalf("cancel not honored: %+v %v", report, err)
	}
}
