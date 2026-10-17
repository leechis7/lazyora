package oracle

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCatalogWithoutStartingTransaction(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectQuery("SELECT SYS_CONTEXT").WillReturnRows(sqlmock.NewRows([]string{"SCHEMA"}).AddRow("Mixed Case"))
	mock.ExpectQuery("SELECT username FROM all_users").WillReturnRows(sqlmock.NewRows([]string{"USERNAME"}).AddRow("SYS").AddRow("Mixed Case")).RowsWillBeClosed()
	catalog, err := s.Catalog(context.Background())
	if err != nil || catalog.Current != "Mixed Case" || len(catalog.Schemas) != 2 {
		t.Fatalf("catalog=%+v err=%v", catalog, err)
	}
	if s.tx != nil {
		t.Fatal("catalog started a transaction")
	}
}

func TestBrowsingPreservesTransactionAndBindsNames(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE items SET n = 1").WillReturnResult(sqlmock.NewResult(0, 1))
	if _, err := s.Execute(context.Background(), "UPDATE items SET n = 1"); err != nil {
		t.Fatal(err)
	}
	tx := s.tx
	owner, name := `Mixed'Owner`, `Table"[red]`
	mock.ExpectQuery("SELECT object_name, object_type, status FROM all_objects").WithArgs(owner).
		WillReturnRows(sqlmock.NewRows([]string{"OBJECT_NAME", "OBJECT_TYPE", "STATUS"}).AddRow(name, "TABLE", "VALID").AddRow("View", "VIEW", "VALID")).RowsWillBeClosed()
	objects, err := s.Objects(context.Background(), owner)
	if err != nil || len(objects) != 2 || objects[0].Owner != owner || objects[1].Kind != "VIEW" {
		t.Fatalf("objects=%+v err=%v", objects, err)
	}
	mock.ExpectQuery("SELECT c.column_name, c.data_type").WithArgs(owner, name).
		WillReturnRows(sqlmock.NewRows([]string{"NAME", "TYPE", "LENGTH", "PRECISION", "SCALE", "CHARS", "SEMANTICS", "NULLABLE", "COMMENTS"}).
			AddRow("ID", "NUMBER", 22, 10, 0, 0, nil, "N", "식별자 [red]").
			AddRow("한글", "VARCHAR2", 400, nil, nil, 100, "C", "Y", nil)).RowsWillBeClosed()
	columns, err := s.Columns(context.Background(), objects[0])
	if err != nil || len(columns) != 2 || columns[0].DataType != "NUMBER(10,0)" || columns[0].Nullable || columns[1].DataType != "VARCHAR2(100 CHAR)" || !columns[1].Nullable || columns[0].Comment != "식별자 [red]" || columns[1].Comment != "" {
		t.Fatalf("columns=%+v err=%v", columns, err)
	}
	mock.ExpectQuery("SELECT object_name").WithArgs("denied").WillReturnError(errors.New("ORA-00942"))
	if _, err := s.Objects(context.Background(), "denied"); err == nil {
		t.Fatal("dictionary failure was hidden")
	}
	if s.tx != tx {
		t.Fatal("browsing changed the transaction")
	}
	mock.ExpectRollback()
	if err := s.Finish(false); err != nil {
		t.Fatal(err)
	}
}

func TestDictionaryEmptyAndReadFailure(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectQuery("SELECT object_name").WithArgs("EMPTY").WillReturnRows(sqlmock.NewRows([]string{"NAME", "TYPE", "STATUS"}))
	objects, err := s.Objects(context.Background(), "EMPTY")
	if err != nil || len(objects) != 0 {
		t.Fatalf("objects=%+v err=%v", objects, err)
	}
	want := errors.New("network interrupted")
	mock.ExpectQuery("SELECT c.column_name").WithArgs("X", "Y").WillReturnRows(sqlmock.NewRows([]string{"NAME"}).AddRow("x").RowError(0, want))
	if _, err := s.Columns(context.Background(), Object{Owner: "X", Name: "Y"}); !errors.Is(err, want) {
		t.Fatalf("expected read error, got %v", err)
	}
}

func TestObjectKindsAndMaterializedViewStorageDeduplication(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectQuery("SELECT object_name").WithArgs("APP").WillReturnRows(
		sqlmock.NewRows([]string{"NAME", "TYPE", "STATUS"}).
			AddRow("P", "PACKAGE", "VALID").AddRow("P", "PACKAGE BODY", "INVALID").
			AddRow("M", "MATERIALIZED VIEW", "VALID").AddRow("M", "TABLE", "VALID").AddRow("T", "TABLE", "VALID"))
	objects, err := s.Objects(context.Background(), "APP")
	if err != nil || len(objects) != 4 {
		t.Fatalf("objects=%+v err=%v", objects, err)
	}
	if objects[1].Kind != "PACKAGE BODY" || objects[1].Status != "INVALID" || !objects[2].HasColumns() || objects[1].HasColumns() {
		t.Fatal("body identity, status or column routing lost")
	}
	mock.ExpectQuery("SELECT status FROM all_objects").WithArgs("APP", "P", "PACKAGE BODY").
		WillReturnRows(sqlmock.NewRows([]string{"STATUS"}).AddRow("VALID")).RowsWillBeClosed()
	info, err := s.ObjectInfo(context.Background(), objects[1])
	if err != nil || info.Kind != "PACKAGE BODY" || info.Status != "VALID" {
		t.Fatalf("info=%+v err=%v", info, err)
	}
	mock.ExpectQuery("SELECT status FROM all_objects").WithArgs("APP", "P", "PACKAGE BODY").
		WillReturnRows(sqlmock.NewRows([]string{"STATUS"}))
	if _, err := s.ObjectInfo(context.Background(), objects[1]); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("removed object did not report missing metadata: %v", err)
	}
}

func TestColumnTypeFormatting(t *testing.T) {
	number := func(n int64) sql.NullInt64 { return sql.NullInt64{Int64: n, Valid: true} }
	unset := sql.NullInt64{}
	for _, tc := range []struct {
		kind, expected                  string
		length, precision, scale, chars sql.NullInt64
		semantics                       string
	}{
		{"NUMBER", "NUMBER", number(22), unset, unset, unset, ""},
		{"NUMBER", "NUMBER(*,0)", number(22), unset, number(0), unset, ""},
		{"NUMBER", "NUMBER(8,-2)", number(22), number(8), number(-2), unset, ""},
		{"VARCHAR2", "VARCHAR2(30 BYTE)", number(30), unset, unset, number(30), "B"},
		{"NVARCHAR2", "NVARCHAR2(100)", number(200), unset, unset, number(100), "C"},
		{"FLOAT", "FLOAT(126)", number(22), number(126), unset, unset, ""},
		{"RAW", "RAW(16)", number(16), unset, unset, unset, ""},
		{"TIMESTAMP(6)", "TIMESTAMP(6)", number(11), unset, number(6), unset, ""},
	} {
		if got := columnType(tc.kind, tc.length, tc.precision, tc.scale, tc.chars, sql.NullString{String: tc.semantics, Valid: tc.semantics != ""}); got != tc.expected {
			t.Errorf("%s: got %s want %s", tc.kind, got, tc.expected)
		}
	}
}
