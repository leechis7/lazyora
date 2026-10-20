package oracle_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

func TestOracleRoutineIntegration(t *testing.T) {
	s := workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	object := oracle.Object{Owner: "LAZYORA", Name: "ROUTINE_TEST", Kind: "PACKAGE"}
	for _, ddl := range []string{
		`CREATE OR REPLACE PACKAGE routine_test AS
 FUNCTION scalar(p_n IN OUT NUMBER, p_text VARCHAR2 DEFAULT '기본') RETURN VARCHAR2;
 PROCEDURE typed(p_d IN OUT DATE, p_t IN OUT TIMESTAMP_UNCONSTRAINED, p_b IN OUT BOOLEAN, p_r IN OUT RAW);
 PROCEDURE cursors(p_n NUMBER, p_c OUT SYS_REFCURSOR, p_null OUT SYS_REFCURSOR, p_io IN OUT SYS_REFCURSOR);
 TYPE row_cursor IS REF CURSOR RETURN sample_rows%ROWTYPE;
 PROCEDURE strong_cursor(p OUT row_cursor,p_null OUT row_cursor);
 FUNCTION cursor_return RETURN SYS_REFCURSOR;
 FUNCTION overloaded(p NUMBER) RETURN VARCHAR2;
 FUNCTION overloaded(p VARCHAR2) RETURN VARCHAR2;
 PROCEDURE no_args;
 PROCEDURE over_noargs;
 PROCEDURE over_noargs(p NUMBER);
 PROCEDURE null_scalar(p OUT VARCHAR2);
 PROCEDURE fail_cursor(p OUT SYS_REFCURSOR);
 PROCEDURE modify_row(p VARCHAR2);
 END;`,
		`CREATE OR REPLACE PACKAGE BODY routine_test AS
 FUNCTION scalar(p_n IN OUT NUMBER, p_text VARCHAR2 DEFAULT '기본') RETURN VARCHAR2 IS
 BEGIN p_n:=p_n+0.00000001; RETURN p_text; END;
 PROCEDURE typed(p_d IN OUT DATE, p_t IN OUT TIMESTAMP_UNCONSTRAINED, p_b IN OUT BOOLEAN, p_r IN OUT RAW) IS
 BEGIN p_d:=p_d+1; p_t:=p_t+INTERVAL '1' SECOND; p_b:=NOT p_b; p_r:=HEXTORAW('ABCD'); END;
 PROCEDURE cursors(p_n NUMBER, p_c OUT SYS_REFCURSOR, p_null OUT SYS_REFCURSOR, p_io IN OUT SYS_REFCURSOR) IS
 BEGIN OPEN p_c FOR SELECT LEVEL AS n, CASE WHEN LEVEL=1 THEN '한글 [red]' END AS txt FROM dual CONNECT BY LEVEL<=p_n;
 OPEN p_io FOR SELECT 99 AS n FROM dual; END;
 PROCEDURE strong_cursor(p OUT row_cursor,p_null OUT row_cursor) IS BEGIN OPEN p FOR SELECT * FROM sample_rows WHERE id=1; END;
 FUNCTION cursor_return RETURN SYS_REFCURSOR IS c SYS_REFCURSOR;
 BEGIN OPEN c FOR SELECT 'returned' AS txt FROM dual; RETURN c; END;
 FUNCTION overloaded(p NUMBER) RETURN VARCHAR2 IS BEGIN RETURN 'number'; END;
 FUNCTION overloaded(p VARCHAR2) RETURN VARCHAR2 IS BEGIN RETURN 'text'; END;
 PROCEDURE no_args IS BEGIN DBMS_OUTPUT.PUT_LINE('no args'); END;
 PROCEDURE over_noargs IS BEGIN NULL; END;
 PROCEDURE over_noargs(p NUMBER) IS BEGIN NULL; END;
 PROCEDURE null_scalar(p OUT VARCHAR2) IS BEGIN p:=NULL; END;
 PROCEDURE fail_cursor(p OUT SYS_REFCURSOR) IS BEGIN OPEN p FOR SELECT 1 FROM dual; RAISE_APPLICATION_ERROR(-20043,'expected routine error'); END;
 PROCEDURE modify_row(p VARCHAR2) IS BEGIN UPDATE sample_rows SET title=p WHERE id=1; END;
 END;`,
	} {
		if _, err := s.Execute(ctx, ddl); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _, _ = s.Execute(context.Background(), "DROP PACKAGE routine_test") })
	routines, err := s.Routines(ctx, object)
	if err != nil {
		t.Fatal(err)
	}
	find := func(name string) oracle.Routine {
		t.Helper()
		for _, r := range routines {
			if r.Member == name {
				return r
			}
		}
		t.Fatalf("missing %s: %+v", name, routines)
		return oracle.Routine{}
	}
	noArgOverloads := 0
	for _, r := range routines {
		if r.Member == "OVER_NOARGS" {
			noArgOverloads++
			if len(r.Arguments) == 0 {
				if _, err = s.RunRoutine(ctx, r, nil); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	if noArgOverloads != 2 {
		t.Fatalf("no-argument overload missing: %+v", routines)
	}
	value := func(text string) oracle.RoutineInput { return oracle.RoutineInput{Value: oracle.BindValue{Text: text}} }
	null := oracle.RoutineInput{Value: oracle.BindValue{Null: true}}
	scalar := find("SCALAR")
	result, err := s.RunRoutine(ctx, scalar, map[int]oracle.RoutineInput{1: value("123456789012345678901234567890.12345678"), 2: {Default: true}})
	if err != nil || len(result.Scalars.Rows) != 2 || result.Scalars.Rows[0][3] != "기본" || result.Scalars.Rows[1][3] != "123456789012345678901234567890.12345679" {
		t.Fatalf("scalars: %+v %v", result, err)
	}
	result, err = s.RunRoutine(ctx, find("TYPED"), map[int]oracle.RoutineInput{1: value("2026-10-21"), 2: value("2026-10-21 12:34:56.123456789"), 3: value("true"), 4: value("0011")})
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"2026-10-22 00:00:00", "2026-10-21 12:34:57.123456789", "false", "ABCD"} {
		if result.Scalars.Rows[i][3] != want {
			t.Fatalf("typed %d: %+v", i, result.Scalars)
		}
	}
	for _, r := range routines {
		if r.Member != "OVERLOADED" {
			continue
		}
		in := value("abc")
		want := "text"
		if r.Arguments[1].DataType == "NUMBER" {
			in = value("7")
			want = "number"
		}
		result, err = s.RunRoutine(ctx, r, map[int]oracle.RoutineInput{1: in})
		if err != nil || result.Scalars.Rows[0][3] != want {
			t.Fatalf("overload %s: %+v %v", r.Overload, result, err)
		}
	}
	result, err = s.RunRoutine(ctx, find("NULL_SCALAR"), nil)
	if err != nil || !result.Scalars.Values[0][3].Null {
		t.Fatalf("NULL scalar: %+v %v", result, err)
	}
	if err = s.SetOutput(ctx, true); err != nil {
		t.Fatal(err)
	}
	result, err = s.RunRoutine(ctx, find("NO_ARGS"), nil)
	if err != nil || len(result.Output.Lines) != 1 || result.Output.Lines[0] != "no args" {
		t.Fatalf("no args/output: %+v %v", result, err)
	}
	if err = s.SetOutput(ctx, false); err != nil {
		t.Fatal(err)
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
	// Cursor input starts NULL, OUT NULL stays distinguishable from an empty set,
	// and large results are bounded without replaying the procedure.
	cursorInputs := map[int]oracle.RoutineInput{1: value("1005"), 4: null}
	result, err = s.RunRoutine(ctx, find("CURSORS"), cursorInputs)
	if err != nil {
		t.Fatalf("cursors: %v", err)
	}
	if len(result.Cursors) != 3 || len(result.Cursors[0].Result.Rows) != 1000 || !result.Cursors[0].Result.Truncated || result.Cursors[0].Result.Query != nil || result.Cursors[0].Result.Rows[0][1] != "한글 [red]" || !result.Cursors[1].Null || result.Cursors[2].Result.Rows[0][0] != "99" {
		t.Fatalf("cursors: count=%d details=%+v", len(result.Cursors), result.Cursors)
	}
	result, err = s.RunRoutine(ctx, find("STRONG_CURSOR"), nil)
	if err != nil || len(result.Cursors) != 2 || len(result.Cursors[0].Result.Rows) != 1 || !result.Cursors[1].Null {
		t.Fatalf("strong cursor: %+v %v", result, err)
	}
	result, err = s.RunRoutine(ctx, find("CURSOR_RETURN"), nil)
	if err != nil || result.Cursors[0].Name != "RETURN" || result.Cursors[0].Result.Rows[0][0] != "returned" {
		t.Fatalf("cursor return: %+v %v", result, err)
	}
	// Exercise success, NULL cursor and failure repeatedly so leaked server
	// cursors fail with ORA-01000 instead of passing only a one-shot smoke test.
	openedCount := func() (int, bool) {
		result, err := s.Execute(ctx, "SELECT value FROM v$mystat m JOIN v$statname n ON m.statistic#=n.statistic# WHERE n.name='opened cursors current'")
		if err != nil {
			return 0, false
		}
		var count int
		_, err = fmt.Sscan(result.Rows[0][0], &count)
		return count, err == nil
	}
	before, measured := openedCount()
	for i := 0; i < 120; i++ {
		cursorInputs[1] = value("2")
		if _, err = s.RunRoutine(ctx, find("CURSORS"), cursorInputs); err != nil {
			t.Fatalf("cursor repeat %d: %v", i, err)
		}
		if _, err = s.RunRoutine(ctx, find("FAIL_CURSOR"), nil); err == nil || !strings.Contains(err.Error(), "ORA-20043") {
			t.Fatalf("expected error: %v", err)
		}
	}
	if after, ok := openedCount(); measured && ok && after > before+5 {
		t.Fatalf("server cursors leaked: before=%d after=%d", before, after)
	}
	if measured {
		t.Log("current-session open cursor count remained bounded across repeated success/error calls")
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
	baseline, err := s.Execute(ctx, "SELECT title FROM sample_rows WHERE id=1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.RunRoutine(ctx, find("MODIFY_ROW"), map[int]oracle.RoutineInput{1: value("routine pending")})
	if err != nil {
		t.Fatal(err)
	}
	observer := workflowSession(t)
	other, err := observer.Execute(ctx, "SELECT title FROM sample_rows WHERE id=1")
	if err != nil || other.Rows[0][0] != baseline.Rows[0][0] || !s.PendingChanges() {
		t.Fatalf("routine auto committed: %v", err)
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
	restored, err := s.Execute(ctx, "SELECT title FROM sample_rows WHERE id=1")
	if err != nil || restored.Rows[0][0] != baseline.Rows[0][0] {
		t.Fatal(fmt.Sprint("rollback failed ", err))
	}
}
