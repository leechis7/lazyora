package oracle

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func routineArg(name, typ, direction string, pos int) RoutineArgument {
	return RoutineArgument{Argument: Argument{Name: name, DataType: typ, Direction: direction, Position: pos}}
}
func TestRoutineCallBindsValuesAndPinsNames(t *testing.T) {
	r := Routine{Object: Object{Owner: `Mixed"Owner`, Name: "P", Kind: "PACKAGE"}, Member: `f"n`, Overload: "1", Arguments: []RoutineArgument{routineArg("-", "NUMBER", "OUT", 0), routineArg("A", "NUMBER", "IN/OUT", 1), routineArg("Text", "VARCHAR2", "IN", 2)}}
	secret := "secret '; END; DROP TABLE x; --"
	call, err := buildRoutineCall(r, map[int]RoutineInput{1: {Value: BindValue{Text: "123456789012345678901234567890.12345678"}}, 2: {Value: BindValue{Text: secret}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"Mixed""Owner"."P"."f""n"("A" => v1, "Text" => v2)`, "TO_CHAR(v0, 'TM9'", "v1 NUMBER;", "v1 := :LAZYORA_I1"} {
		if !strings.Contains(call.text, expected) {
			t.Fatalf("missing %s in %s", expected, call.text)
		}
	}
	if strings.Contains(call.text, secret) || len(call.outputs) != 2 {
		t.Fatalf("unsafe call: %+v", call)
	}
}
func TestRoutineDefaultsUnsupportedTypesAndValidation(t *testing.T) {
	r := Routine{Object: Object{Owner: "APP", Name: "P", Kind: "PROCEDURE"}, Member: "P", Overload: "-", Arguments: []RoutineArgument{routineArg("A", "NUMBER", "IN", 1)}}
	r.Arguments[0].Defaulted = true
	call, err := buildRoutineCall(r, map[int]RoutineInput{1: {Default: true}})
	if err != nil || strings.Contains(call.text, "v0") || !strings.Contains(call.text, `"APP"."P"()`) {
		t.Fatalf("%s %v", call.text, err)
	}
	r.Overload = "2"
	if ValidateRoutineInputs(r, map[int]RoutineInput{1: {Default: true}}) == nil {
		t.Fatal("ambiguous overload allowed")
	}
	r.Overload = "-"
	for _, typ := range []string{"PL/SQL RECORD", "TABLE", "CLOB", "BLOB", "TIMESTAMP WITH TIME ZONE", "NUMBER); END;"} {
		r.Arguments[0].DataType = typ
		if ValidateRoutineInputs(r, map[int]RoutineInput{1: {Value: BindValue{Null: true}}}) == nil {
			t.Fatalf("accepted %s", typ)
		}
	}
	for _, test := range []struct{ typ, value string }{{"NUMBER", "12; DROP"}, {"DATE", "yesterday"}, {"TIMESTAMP", "2026"}, {"RAW", "0xz"}, {"RAW", "f"}, {"BOOLEAN", "perhaps"}, {"REF CURSOR", "cursor"}} {
		r.Arguments[0].DataType = test.typ
		if ValidateRoutineInputs(r, map[int]RoutineInput{1: {Value: BindValue{Text: test.value}}}) == nil {
			t.Fatalf("accepted %+v", test)
		}
	}
	r.Arguments[0] = routineArg("A", "NUMBER", "IN", 1)
	r.Arguments[0].Depth = 1
	if ValidateRoutineInputs(r, map[int]RoutineInput{1: {Value: BindValue{Null: true}}}) == nil {
		t.Fatal("nested metadata accepted")
	}
}
func TestRoutineCursorNullAndErrorCleanupSQL(t *testing.T) {
	r := Routine{Object: Object{Owner: "APP", Name: "P", Kind: "PROCEDURE"}, Member: "P", Arguments: []RoutineArgument{routineArg("C", "REF CURSOR", "IN/OUT", 1)}}
	call, err := buildRoutineCall(r, map[int]RoutineInput{1: {Value: BindValue{Null: true}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"v0 SYS_REFCURSOR", "IF v0%ISOPEN", "WHERE 1=0", "EXCEPTION WHEN OTHERS", "CLOSE v0;", ":LAZYORA_O0 := c0"} {
		if !strings.Contains(call.text, fragment) {
			t.Fatal(fragment)
		}
	}
}
func routineColumns() []string {
	return []string{"MEMBER", "OVERLOAD", "NAME", "POS", "DEPTH", "TYPE", "MODE", "PLS", "OWNER", "TYPE_NAME", "SUBNAME", "DEFAULTED"}
}
func TestRoutines10gFallbackAndNoArgumentMembers(t *testing.T) {
	s, mock := mockSession(t)
	object := Object{Owner: "APP", Name: "P", Kind: "PACKAGE"}
	mock.ExpectQuery("defaulted").WithArgs("APP", "P").WillReturnError(errors.New(`ORA-00904: "DEFAULTED": invalid identifier`))
	mock.ExpectQuery("'N'").WithArgs("APP", "P").WillReturnRows(sqlmock.NewRows(routineColumns()).AddRow("F", "1", "-", 0, 0, "NUMBER", "OUT", "NUMBER", nil, nil, nil, "N").AddRow("F", "1", "A", 1, 0, "NUMBER", "IN", "PLS_INTEGER", nil, nil, nil, "N")).RowsWillBeClosed()
	mock.ExpectQuery("SELECT procedure_name, NVL").WithArgs("APP", "P").WillReturnError(errors.New("ORA-00904: OVERLOAD"))
	mock.ExpectQuery("SELECT DISTINCT procedure_name, '-' FROM all_procedures").WithArgs("APP", "P").WillReturnRows(sqlmock.NewRows([]string{"NAME", "OVERLOAD"}).AddRow("F", "-").AddRow("NO_ARGS", "-")).RowsWillBeClosed()
	routines, err := s.Routines(context.Background(), object)
	if err != nil || len(routines) != 2 || len(routines[0].Arguments) != 2 || routines[0].Arguments[1].Defaulted || routines[1].Member != "NO_ARGS" {
		t.Fatalf("%+v %v", routines, err)
	}
	if s.pending || s.tx != nil {
		t.Fatal("metadata created transaction")
	}
}
func TestRoutinesDoNotHidePermissionErrors(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectQuery("FROM all_arguments").WillReturnError(errors.New("ORA-01031: insufficient privileges"))
	if _, err := s.Routines(context.Background(), Object{Owner: "APP", Name: "P", Kind: "PACKAGE"}); err == nil {
		t.Fatal("error hidden")
	}
}
func TestRoutineFailureKeepsTransactionAndNeverRecordsInput(t *testing.T) {
	s, mock := mockSession(t)
	r := Routine{Object: Object{Owner: "APP", Name: "P", Kind: "PROCEDURE"}, Member: "P", Arguments: []RoutineArgument{routineArg("A", "VARCHAR2", "IN", 1)}}
	mock.ExpectBegin()
	mock.ExpectExec("DECLARE").WillReturnError(errors.New("ORA-20000: failure"))
	_, err := s.RunRoutine(context.Background(), r, map[int]RoutineInput{1: {Value: BindValue{Text: "secret"}}})
	if err == nil || !s.PendingChanges() || s.tx == nil {
		t.Fatalf("%v", err)
	}
	work := s.PendingWork()
	if len(work.Statements) != 1 || strings.Contains(work.Statements[0].SQL, "secret") {
		t.Fatalf("%+v", work)
	}
	mock.ExpectRollback()
}
