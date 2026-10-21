package oracle_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

func TestOracleFormatIntegration(t *testing.T) {
	s := workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for _, query := range []string{
		`SELECT N'한글' AS a,q'[literal; / -- :bind]' AS b,1.e+2 AS c,1.25e-3 AS d,.5f AS e FROM dual`,
		`SELECT /*+ NO_MERGE */ EXTRACT(YEAR FROM DATE '2026-10-21') AS yr,TRIM(BOTH 'x' FROM 'x한글x') AS txt,SUM(1) OVER(PARTITION BY 1 ORDER BY 1) AS total FROM dual`,
		"SELECT 'first\nsecond' AS text,CASE WHEN 1=1 THEN 'yes' ELSE 'no' END AS flag FROM dual -- comment  ",
	} {
		formatted, err := oracle.FormatSQL(query)
		if err != nil {
			t.Fatal(err)
		}
		before, err := s.Execute(ctx, query)
		if err != nil {
			t.Fatal(err)
		}
		after, err := s.Execute(ctx, formatted)
		if err != nil {
			t.Fatalf("formatted SELECT: %v\n%s", err, formatted)
		}
		if !reflect.DeepEqual(before.Rows, after.Rows) || !reflect.DeepEqual(before.Columns, after.Columns) {
			t.Fatalf("query meaning changed: before=%+v after=%+v", before, after)
		}
	}
	if err := s.SetOutput(ctx, true); err != nil {
		t.Fatal(err)
	}
	block := `DECLARE total NUMBER:=0; BEGIN FOR i IN 1..10 LOOP IF MOD(i,2)=0 THEN total:=total+i; ELSE total:=total; END IF; END LOOP; DBMS_OUTPUT.PUT_LINE('합계 '||total); EXCEPTION WHEN OTHERS THEN RAISE; END;`
	formatted, err := oracle.FormatSQL(block)
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.Execute(ctx, block)
	if err != nil {
		t.Fatal(err)
	}
	after, err := s.Execute(ctx, formatted)
	if err != nil {
		t.Fatalf("formatted PL/SQL: %v\n%s", err, formatted)
	}
	if !reflect.DeepEqual(before.Output.Lines, after.Output.Lines) || len(after.Output.Lines) != 1 || after.Output.Lines[0] != "합계 30" {
		t.Fatalf("PLSQL meaning changed: %+v", after)
	}
	script := "BEGIN DBMS_OUTPUT.PUT_LINE(q'[one; /]'); END;\n/\nSELECT 1 AS n FROM dual;\nBEGIN DBMS_OUTPUT.PUT_LINE('two'); END;\n/\n"
	formatted, err = oracle.FormatSQL(script)
	if err != nil {
		t.Fatal(err)
	}
	report, err := s.Script(ctx, formatted)
	if err != nil || len(report.Steps) != 3 || report.Steps[0].Result.Output.Lines[0] != "one; /" || report.Steps[2].Result.Output.Lines[0] != "two" {
		t.Fatalf("script boundaries: %+v %v", report, err)
	}
	object := oracle.Object{Owner: "LAZYORA", Name: "BROWSER_FUNCTION", Kind: "FUNCTION"}
	source, err := s.Source(ctx, object)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := s.Compile(context.Background(), object, source); err != nil {
			t.Error(err)
		}
	})
	formatted, err = oracle.FormatSQL(source)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := s.Compile(ctx, object, formatted)
	if err != nil || compiled.Object.Status != "VALID" || len(compiled.Diagnostics) != 0 {
		t.Fatalf("source compile: %+v %v\n%s", compiled, err, formatted)
	}
	result, err := s.Execute(ctx, "SELECT browser_function(42) AS n FROM dual")
	if err != nil || result.Rows[0][0] != "42" {
		t.Fatalf("formatted function: %+v %v", result, err)
	}
	if !strings.Contains(formatted, "BEGIN\n") {
		t.Fatal("source was not formatted")
	}
	if err := s.Finish(false); err != nil {
		t.Fatal(err)
	}
}
