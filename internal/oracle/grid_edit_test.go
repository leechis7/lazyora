package oracle

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
)

func TestEditTypeValidationAndNoCapability(t *testing.T) {
	for _, kind := range []string{"CLOB", "BLOB", "LONG", "XMLTYPE", "TIMESTAMP WITH TIME ZONE", "NVARCHAR2"} {
		if editableType(kind) {
			t.Fatalf("unsupported type editable: %s", kind)
		}
	}
	for _, tc := range []struct{ kind, text string }{{"NUMBER", "1;DELETE"}, {"DATE", "2026-02-30 00:00:00"}, {"TIMESTAMP", "wrong"}, {"RAW", "0G"}, {"RAW", "0"}} {
		if _, err := editValue(tc.kind, tc.text, false); err == nil {
			t.Fatalf("invalid %s accepted", tc.kind)
		}
	}
	s := &Session{}
	if err := s.ApplyEdit(context.Background(), EditChange{ID: 1}); err == nil {
		t.Fatal("arbitrary result granted edit permission")
	}
	if _, err := s.EditableTable(context.Background(), Object{Owner: "A", Name: "B", Kind: "VIEW"}); err == nil {
		t.Fatal("view editable")
	}
}

func TestGridEditRollbackFailureBlocksFurtherWork(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectBegin()
	var err error
	s.tx, err = s.conn.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	s.edit = &editSnapshot{id: 1, object: Object{Owner: "A", Name: "B", Kind: "TABLE"}, columns: []Column{{Name: "VALUE", DataType: "VARCHAR2", Nullable: true}}, rows: [][]Cell{{{Type: "VARCHAR2", Text: "old"}}}, rowIDs: []string{"rowid"}}
	mock.ExpectExec("SAVEPOINT LAZYORA_").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT ROWIDTOCHAR").WillReturnRows(sqlmock.NewRows([]string{"ROWID"}).AddRow("rowid"))
	mock.ExpectExec("UPDATE").WillReturnError(errors.New("connection dropped"))
	mock.ExpectExec("ROLLBACK TO SAVEPOINT").WillReturnError(errors.New("rollback not acknowledged"))
	if err = s.ApplyEdit(context.Background(), EditChange{ID: 1, Row: 0, Column: 0, Text: "new"}); err == nil || !s.TransactionUncertain() || !s.PendingChanges() {
		t.Fatal("uncertain result was cleared")
	}
	if _, err = s.Execute(context.Background(), "SELECT 1 FROM dual"); err == nil {
		t.Fatal("executed after uncertain update")
	}
	mock.ExpectRollback()
}
