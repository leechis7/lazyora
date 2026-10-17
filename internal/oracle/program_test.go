package oracle

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"regexp"
	"strings"
	"testing"
)

func TestSourceDDLTargetsAndTerminators(t *testing.T) {
	object := Object{Owner: `Mixed"Owner`, Name: "F", Kind: "FUNCTION"}
	ddl, err := SourceDDL(object, "FUNCTION f RETURN VARCHAR2 IS BEGIN RETURN 'a;b'; END;\r\n/\r\n")
	if err != nil || ddl != "CREATE OR REPLACE FUNCTION \"Mixed\"\"Owner\".\"F\" RETURN VARCHAR2 IS BEGIN RETURN 'a;b'; END;" {
		t.Fatalf("ddl=%q err=%v", ddl, err)
	}
	for _, source := range []string{
		"FUNCTION other RETURN NUMBER IS BEGIN RETURN 1; END;",
		"FUNCTION other_owner.f RETURN NUMBER IS BEGIN RETURN 1; END;",
		"PROCEDURE f IS BEGIN NULL; END;", "DROP FUNCTION f", "",
	} {
		if _, err := SourceDDL(object, source); err == nil {
			t.Fatalf("accepted wrong target: %s", source)
		}
	}
	for _, kind := range []string{"PACKAGE", "PACKAGE BODY", "TYPE", "TYPE BODY", "TRIGGER", "PROCEDURE"} {
		object := Object{Owner: "APP", Name: "P", Kind: kind}
		if _, err := SourceDDL(object, "CREATE OR REPLACE "+kind+" app.p AS x;"); err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		ddl, err := compileDDL(object)
		if err != nil || !strings.Contains(ddl, `"APP"."P" COMPILE`) {
			t.Fatalf("%s: %s %v", kind, ddl, err)
		}
	}
}

func TestSourceAndArgumentReadsBindIdentity(t *testing.T) {
	s, mock := mockSession(t)
	object := Object{Owner: "APP", Name: "P", Kind: "PACKAGE BODY"}
	mock.ExpectQuery("SELECT text FROM all_source").WithArgs("APP", "P", "PACKAGE BODY").WillReturnRows(
		sqlmock.NewRows([]string{"TEXT"}).AddRow("PACKAGE BODY p AS\n").AddRow("END;\n")).RowsWillBeClosed()
	source, err := s.Source(context.Background(), object)
	if err != nil || !strings.Contains(source, `PACKAGE BODY "APP"."P"`) {
		t.Fatalf("%s %v", source, err)
	}
	mock.ExpectQuery("FROM all_arguments WHERE owner = :1 AND package_name = :2").WithArgs("APP", "P").WillReturnRows(
		sqlmock.NewRows([]string{"MEMBER", "OVERLOAD", "NAME", "POS", "DEPTH", "TYPE", "MODE", "OWNER", "TYPE_NAME", "SUBNAME"}).
			AddRow("F", "2", "-", 0, 0, "NUMBER", "OUT", nil, nil, nil).
			AddRow("F", "2", "P_REC", 1, 0, "PL/SQL RECORD", "IN", "APP", "P", "R"))
	arguments, err := s.Arguments(context.Background(), object)
	if err != nil || len(arguments) != 2 || arguments[0].Position != 0 || arguments[1].DataType != "APP.P.R" {
		t.Fatalf("%+v %v", arguments, err)
	}
	if s.PendingChanges() || s.tx != nil {
		t.Fatal("source reads changed transaction state")
	}
}

func TestCompilePreservesSourceAndReportsErrors(t *testing.T) {
	s, mock := mockSession(t)
	object := Object{Owner: "APP", Name: "F", Kind: "FUNCTION"}
	source := `CREATE OR REPLACE FUNCTION "APP"."F" RETURN NUMBER IS BEGIN RETURN missing_name; END;`
	mock.ExpectExec(regexp.QuoteMeta(source)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT status FROM all_objects").WithArgs("APP", "F", "FUNCTION").WillReturnRows(sqlmock.NewRows([]string{"STATUS"}).AddRow("INVALID"))
	mock.ExpectQuery("FROM all_errors").WithArgs("APP", "F", "FUNCTION").WillReturnRows(
		sqlmock.NewRows([]string{"LINE", "POSITION", "ATTRIBUTE", "TEXT"}).AddRow(1, 48, "ERROR", "identifier missing_name must be declared"))
	result, err := s.Compile(context.Background(), object, source)
	if err != nil || !result.Applied || result.Object.Status != "INVALID" || len(result.Diagnostics) != 1 || result.Diagnostics[0].Position != 48 {
		t.Fatalf("%+v %v", result, err)
	}
	if !s.PendingChanges() {
		t.Fatal("DDL should retain conservative change warning")
	}
	if work := s.PendingWork(); len(work.Statements) != 1 || work.Statements[0].SQL != source {
		t.Fatal("compile DDL missing from journal")
	}
	if err := s.Finish(false); err != nil || len(s.PendingWork().Statements) != 0 {
		t.Fatal("DDL-only journal was not cleared by resolution")
	}
}

func TestCompilePermissionFailureIsNotSuccess(t *testing.T) {
	s, mock := mockSession(t)
	object := Object{Owner: "APP", Name: "P", Kind: "PACKAGE BODY"}
	mock.ExpectExec(regexp.QuoteMeta(`ALTER PACKAGE "APP"."P" COMPILE BODY`)).WillReturnError(errors.New("ORA-01031"))
	mock.ExpectQuery("SELECT status FROM all_objects").WithArgs("APP", "P", "PACKAGE BODY").WillReturnRows(sqlmock.NewRows([]string{"STATUS"}).AddRow("VALID"))
	mock.ExpectQuery("FROM all_errors").WithArgs("APP", "P", "PACKAGE BODY").WillReturnRows(sqlmock.NewRows([]string{"LINE", "POSITION", "ATTRIBUTE", "TEXT"}))
	result, err := s.Compile(context.Background(), object, "")
	if err == nil || result.Applied {
		t.Fatalf("permission error treated as saved: %+v %v", result, err)
	}
}
