package oracle

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	go_ora "github.com/sijms/go-ora/v3"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDefinitionUsesBoundNamesAndPreservesTransaction(t *testing.T) {
	s, m := definitionSession(t, "CREATE PACKAGE ...", 18)
	ctx := context.Background()
	m.ExpectBegin()
	m.ExpectExec("UPDATE items SET n=1").WillReturnResult(sqlmock.NewResult(0, 1))
	if _, err := s.Execute(ctx, "UPDATE items SET n=1"); err != nil {
		t.Fatal(err)
	}
	tx := s.tx
	object := Object{Owner: `Mixed'Owner`, Name: `명칭"[red]`, Kind: "PACKAGE"}
	args := []driver.Value{"PACKAGE_SPEC", object.Name, object.Owner, sqlmock.AnyArg()}
	for i := 0; i < (ddlCharacters+3999)/4000; i++ {
		args = append(args, sqlmock.AnyArg())
	}
	m.ExpectExec("DECLARE v_ddl").WithArgs(args...).WillReturnResult(sqlmock.NewResult(0, 0))
	got, err := s.Definition(ctx, object, "ddl")
	if err != nil || got.DDL != "CREATE PACKAGE ..." || s.tx != tx || !s.PendingChanges() || len(s.PendingWork().Statements) != 1 {
		t.Fatalf("definition changed transaction: %+v %v", got, err)
	}
	m.ExpectRollback()
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
}
func TestDefinitionDDLBoundErrorsAndUnsupportedType(t *testing.T) {
	s, m := definitionSession(t, "", ddlCharacters+1)
	o := Object{Owner: "APP", Name: "T", Kind: "TABLE"}
	m.ExpectExec("DECLARE v_ddl").WillReturnResult(sqlmock.NewResult(0, 0))
	if _, err := s.Definition(context.Background(), o, "ddl"); err == nil {
		t.Fatal("oversized DDL accepted")
	}
	m.ExpectExec("DECLARE v_ddl").WillReturnError(errors.New("ORA-31603: not found or inaccessible"))
	if _, err := s.Definition(context.Background(), o, "ddl"); err == nil || s.tx != nil || s.PendingChanges() {
		t.Fatal("permission error hidden or transaction changed")
	}
	o.Kind = "USER"
	if _, err := s.Definition(context.Background(), o, "ddl"); err == nil {
		t.Fatal("unsupported metadata type accepted")
	}
}
func TestConstraintDefinitionIncludesOrderedReferences(t *testing.T) {
	s, m := definitionSession(t, `"値" > 0`, 0)
	o := Object{Owner: "APP", Name: "T", Kind: "TABLE"}
	m.ExpectQuery("SELECT constraint_name, constraint_type").WithArgs("APP", "T").WillReturnRows(sqlmock.NewRows([]string{"NAME", "TYPE", "STATUS", "VALIDATED", "DEFERRABLE", "DEFERRED", "R_OWNER", "R_NAME", "DELETE"}).AddRow("FK", "R", "ENABLED", "VALIDATED", "DEFERRABLE", "IMMEDIATE", "Other", "PK", "CASCADE").AddRow("CK", "C", "ENABLED", "NOT VALIDATED", "NOT DEFERRABLE", "IMMEDIATE", nil, nil, nil)).RowsWillBeClosed()
	m.ExpectQuery("SELECT constraint_name, column_name").WillReturnRows(sqlmock.NewRows([]string{"NAME", "COLUMN"}).AddRow("CK", "値").AddRow("FK", "A").AddRow("FK", "B")).RowsWillBeClosed()
	m.ExpectExec("DECLARE v_text").WillReturnResult(sqlmock.NewResult(0, 0))
	m.ExpectQuery("SELECT constraint_name, reference_table").WillReturnRows(sqlmock.NewRows([]string{"NAME", "TABLE", "COLUMN"}).AddRow("FK", "Parent", "X").AddRow("FK", "Parent", "Y")).RowsWillBeClosed()
	got, err := s.Definition(context.Background(), o, "constraints")
	if err != nil || len(got.Constraints) != 2 || got.Constraints[0].ReferenceTable != "Parent" || got.Constraints[0].ReferenceColumns[1] != "Y" || got.Constraints[1].Condition != `"値" > 0` {
		t.Fatalf("constraints: %+v %v", got, err)
	}
}
func TestIndexDefinitionIncludesExpressionsAndDirection(t *testing.T) {
	s, m := definitionSession(t, `UPPER("Title")`, 0)
	o := Object{Owner: "APP", Name: "IDX", Kind: "INDEX"}
	m.ExpectQuery("SELECT owner, index_name").WithArgs("APP", "IDX").WillReturnRows(sqlmock.NewRows([]string{"OWNER", "NAME", "TYPE", "UNIQUE", "STATUS", "TABLE_OWNER", "TABLE_NAME"}).AddRow("APP", "IDX", "FUNCTION-BASED NORMAL", "NONUNIQUE", "VALID", "APP", "T")).RowsWillBeClosed()
	m.ExpectQuery("SELECT index_owner, index_name, column_position, column_name").WillReturnRows(sqlmock.NewRows([]string{"OWNER", "NAME", "POS", "COLUMN", "DIRECTION"}).AddRow("APP", "IDX", 1, "SYS_NC1$", "DESC")).RowsWillBeClosed()
	m.ExpectExec("DECLARE v_text").WillReturnResult(sqlmock.NewResult(0, 0))
	got, err := s.Definition(context.Background(), o, "indexes")
	if err != nil || len(got.Indexes) != 1 || got.Indexes[0].Columns[0].Expression != `UPPER("Title")` || got.Indexes[0].Columns[0].Direction != "DESC" {
		t.Fatalf("indexes: %+v %v", got, err)
	}
}
func TestDefinitionRowLimits(t *testing.T) {
	s, m := mockSession(t)
	rows := sqlmock.NewRows([]string{"OWNER", "NAME", "TYPE", "UNIQUE", "STATUS", "TABLE_OWNER", "TABLE_NAME"})
	for i := 0; i < definitionRows+1; i++ {
		rows.AddRow("APP", "I", "NORMAL", "UNIQUE", "VALID", "APP", "T")
	}
	m.ExpectQuery("SELECT owner, index_name").WillReturnRows(rows).RowsWillBeClosed()
	m.ExpectQuery("SELECT index_owner.*column_name").WillReturnRows(sqlmock.NewRows([]string{"O", "N", "P", "C", "D"}))
	got, err := s.Definition(context.Background(), Object{Owner: "APP", Name: "T", Kind: "TABLE"}, "indexes")
	if err != nil || !got.Limited || len(got.Indexes) != definitionRows {
		t.Fatalf("limit: %d %v %v", len(got.Indexes), got.Limited, err)
	}
}

type definitionConverter struct {
	ddl    string
	length int
}

func (c *definitionConverter) ConvertValue(value any) (driver.Value, error) {
	if out, ok := value.(go_ora.Out); ok {
		switch dest := out.Dest.(type) {
		case *int:
			*dest = c.length
		case *sql.NullString:
			*dest = sql.NullString{String: c.ddl, Valid: c.ddl != ""}
			c.ddl = ""
		}
		return int64(0), nil
	}
	return driver.DefaultParameterConverter.ConvertValue(value)
}
func definitionSession(t *testing.T, ddl string, length int) (*Session, sqlmock.Sqlmock) {
	t.Helper()
	db, m, err := sqlmock.New(sqlmock.ValueConverterOption(&definitionConverter{ddl: ddl, length: length}))
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
		if err := m.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
	return s, m
}
