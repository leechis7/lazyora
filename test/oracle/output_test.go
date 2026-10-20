package oracle_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

func TestOracleOutputIntegration(t *testing.T) {
	s := workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := s.SetOutput(ctx, true); err != nil {
		t.Fatal(err)
	}
	if s.PendingChanges() {
		t.Fatal("enabling output marked DML pending")
	}
	result, err := s.Execute(ctx, `BEGIN
DBMS_OUTPUT.PUT_LINE('한글 [red]');
DBMS_OUTPUT.PUT_LINE('');
DBMS_OUTPUT.PUT_LINE(REPLACE(RPAD('x', 10000, 'x'), 'x', '한'));
END;`)
	if err != nil || result.Output.Error != "" || len(result.Output.Lines) != 3 {
		t.Fatalf("output lines=%d error=%s SQL=%v", len(result.Output.Lines), result.Output.Error, err)
	}
	if result.Output.Lines[0] != "한글 [red]" || result.Output.Lines[1] != "" || result.Output.Lines[2] != strings.Repeat("한", 10000) {
		t.Fatalf("Unicode, empty line or long output was changed: first=%q blank=%q long bytes=%d", result.Output.Lines[0], result.Output.Lines[1], len(result.Output.Lines[2]))
	}
	if !s.PendingChanges() {
		t.Fatal("collection cleared the conservative PL/SQL pending state")
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
	result, err = s.Execute(ctx, `BEGIN FOR i IN 1..2001 LOOP DBMS_OUTPUT.PUT_LINE(''); END LOOP; END;`)
	if err != nil || result.Output.Error != "" || len(result.Output.Lines) != oracle.MaxOutputLines || !result.Output.Truncated {
		t.Fatalf("line cap: lines=%d truncated=%v output=%s SQL=%v", len(result.Output.Lines), result.Output.Truncated, result.Output.Error, err)
	}
	result, err = s.Execute(ctx, `BEGIN DBMS_OUTPUT.PUT_LINE('after'); END;`)
	if err != nil || result.Output.Error != "" || len(result.Output.Lines) != 1 || result.Output.Lines[0] != "after" {
		t.Fatalf("output carried across executions: %+v SQL=%v", result.Output, err)
	}
	result, err = s.Execute(ctx, `BEGIN DBMS_OUTPUT.PUT_LINE('before error'); RAISE_APPLICATION_ERROR(-20001,'expected'); END;`)
	if err == nil || result.Output.Error != "" || len(result.Output.Lines) != 1 || result.Output.Lines[0] != "before error" {
		t.Fatalf("output lost on SQL failure: %+v SQL=%v", result.Output, err)
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
	if err = s.SetOutput(ctx, false); err != nil {
		t.Fatal(err)
	}
	if s.OutputEnabled() || s.PendingChanges() {
		t.Fatal("disabling output changed user transaction state")
	}
	baseline, err := s.Execute(ctx, "SELECT title FROM sample_rows WHERE id=1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Execute(ctx, "UPDATE sample_rows SET title='output transaction check' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if err = s.SetOutput(ctx, true); err != nil {
		t.Fatal(err)
	}
	other := workflowSession(t)
	visible, err := other.Execute(ctx, "SELECT title FROM sample_rows WHERE id=1")
	if err != nil || visible.Rows[0][0] != baseline.Rows[0][0] {
		t.Fatalf("enabling output committed DML: %+v %v", visible, err)
	}
	report, err := s.Script(ctx, "BEGIN DBMS_OUTPUT.PUT_LINE('one'); END;\n/\nBEGIN DBMS_OUTPUT.PUT_LINE('two'); END;\n/")
	if err != nil || len(report.Steps) != 2 || len(report.Steps[0].Result.Output.Lines) != 1 || len(report.Steps[1].Result.Output.Lines) != 1 || report.Steps[0].Result.Output.Lines[0] != "one" || report.Steps[1].Result.Output.Lines[0] != "two" {
		t.Fatalf("script outputs not attributed to their statements: %+v %v", report, err)
	}
	if err = s.SetOutput(ctx, false); err != nil {
		t.Fatal(err)
	}
	current, err := s.Execute(ctx, "SELECT title FROM sample_rows WHERE id=1")
	if err != nil || current.Rows[0][0] != "output transaction check" {
		t.Fatalf("output control lost pending DML: %+v %v", current, err)
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
	current, err = s.Execute(ctx, "SELECT title FROM sample_rows WHERE id=1")
	if err != nil || current.Rows[0][0] != baseline.Rows[0][0] {
		t.Fatalf("DML did not remain rollbackable: %+v %v", current, err)
	}
}
