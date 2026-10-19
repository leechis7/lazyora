package oracle

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestExplainPreservesPendingTransaction(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE items SET n=1").WillReturnResult(sqlmock.NewResult(0, 1))
	if _, err := s.Execute(context.Background(), "UPDATE items SET n=1"); err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("EXPLAIN PLAN SET STATEMENT_ID = 'LAZYORA_[0-9a-f]{20}' FOR SELECT n FROM items").WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectQuery("SELECT id,parent_id,operation").WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"ID", "PARENT_ID", "OPERATION", "OPTIONS", "OWNER", "NAME", "COST", "ROWS", "ACCESS", "FILTER"}).AddRow(0, nil, "SELECT STATEMENT", nil, nil, nil, "2", "3", nil, nil).AddRow(1, 0, "TABLE ACCESS", "FULL", "APP", "ITEMS", "2", "3", nil, "N=1"))
	mock.ExpectExec("DELETE FROM plan_table WHERE statement_id = :1").WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectRollback()
	steps, err := s.Explain(context.Background(), "SELECT n FROM items;", nil)
	if err != nil || len(steps) != 2 || steps[1].Parent != 0 || steps[1].Filter != "N=1" || !s.PendingChanges() || len(s.PendingWork().Statements) != 1 {
		t.Fatalf("%+v %v", steps, err)
	}
}

func TestExplainMissingPlanTableAndInvalidSQL(t *testing.T) {
	s, mock := mockSession(t)
	for _, text := range []string{"DROP TABLE items;", "BEGIN NULL; END;", "SELECT 1 FROM dual; DELETE FROM items;"} {
		if _, err := s.Explain(context.Background(), text, nil); err == nil {
			t.Fatal("invalid explain accepted")
		}
	}
	mock.ExpectBegin()
	mock.ExpectExec("EXPLAIN PLAN SET").WillReturnError(errors.New("ORA-00942: table or view does not exist"))
	mock.ExpectExec("DELETE FROM plan_table WHERE statement_id = :1").WillReturnError(errors.New("ORA-00942"))
	mock.ExpectRollback()
	if _, err := s.Explain(context.Background(), "SELECT 1 FROM dual", nil); err == nil || s.PendingChanges() {
		t.Fatal("metadata failure changed DB work tracking")
	}
}
