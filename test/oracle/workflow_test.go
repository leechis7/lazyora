package oracle_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"lazyora/internal/exportdata"
	"lazyora/internal/oracle"
)

func workflowSession(t *testing.T) *oracle.Session {
	t.Helper()
	if os.Getenv("LAZYORA_INTEGRATION") != "1" {
		t.Skip("run make -C test/oracle test")
	}
	portText := os.Getenv("ORACLE_PORT")
	if portText == "" {
		portText = "1521"
	}
	port, err := oracle.ParsePort(portText)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	s, err := oracle.Open(ctx, oracle.Config{Host: "127.0.0.1", Port: port, User: "lazyora", Password: os.Getenv("APP_USER_PASSWORD"), Mode: "service", Name: "FREEPDB1"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

func TestOracleWorkflowIntegration(t *testing.T) {
	s := workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	values := map[string]oracle.BindValue{
		"N":    {Kind: "number", Text: "123456789012345678901234567890.12345678"},
		"TEXT": {Kind: "text", Text: "한글 ' ; : secret"},
		"D":    {Kind: "date", Text: "2026-10-19"},
		"TS":   {Kind: "timestamp", Text: "2026-10-19 12:34:56.123456"},
		"1":    {Null: true},
	}
	bound, bindErr := s.Execute(ctx, "SELECT :n, :text, :TEXT, TO_CHAR(:d,'YYYY-MM-DD'), TO_CHAR(:ts,'YYYY-MM-DD HH24:MI:SS.FF6'), :1 FROM dual", values)
	if bindErr != nil {
		t.Fatal(bindErr)
	}
	if bound.Rows[0][0] != values["N"].Text || bound.Rows[0][1] != values["TEXT"].Text || bound.Rows[0][2] != values["TEXT"].Text || bound.Rows[0][3] != "2026-10-19" || bound.Rows[0][4] != "2026-10-19 12:34:56.123456" || bound.Rows[0][5] != "NULL" {
		t.Fatalf("bind roundtrip: %+v", bound.Rows)
	}
	if _, bindErr = s.Script(ctx, "SELECT :text FROM dual; BEGIN IF :n > 0 THEN NULL; END IF; END;\n/", values); bindErr != nil {
		t.Fatal(bindErr)
	}
	if bindErr = s.Finish(false); bindErr != nil {
		t.Fatal(bindErr)
	}
	report, err := s.Script(ctx, "SELECT 1 AS n FROM dual;\nBEGIN NULL; END;\n/\nUPDATE sample_rows SET amount=amount WHERE id=1;\nSELECT 2 AS n FROM dual;")
	if err != nil || len(report.Steps) != 4 || report.Steps[3].Result.Rows[0][0] != "2" || !s.PendingChanges() {
		t.Fatalf("script: %+v %v", report, err)
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
	report, err = s.Script(ctx, "UPDATE sample_rows SET amount=amount WHERE id=1; ROLLBACK; SELECT 3 AS n FROM dual;")
	if err != nil || s.PendingChanges() || len(report.Steps) != 3 {
		t.Fatalf("rollback script: %+v %v", report, err)
	}
	report, err = s.Script(ctx, "SELECT 4 AS n FROM dual; SELECT missing_workflow_column FROM dual; DELETE FROM sample_rows;")
	if err == nil || len(report.Steps) != 2 || s.PendingChanges() {
		t.Fatalf("stop on error: %+v %v", report, err)
	}
}

func TestOraclePlanIntegration(t *testing.T) {
	s := workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	before, err := s.Execute(ctx, "SELECT amount FROM sample_rows WHERE id=1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Execute(ctx, "UPDATE sample_rows SET amount=amount+100000 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	steps, err := s.Explain(ctx, "SELECT * FROM sample_rows WHERE id=:id", map[string]oracle.BindValue{"ID": {Kind: "number", Text: "1"}})
	if err != nil || len(steps) == 0 || !s.PendingChanges() || len(s.PendingWork().Statements) != 1 {
		t.Fatalf("bound plan: %+v %v", steps, err)
	}
	if _, err = s.Explain(ctx, "DELETE FROM sample_rows WHERE id=1", nil); err != nil {
		t.Fatal(err)
	}
	count, err := s.Execute(ctx, "SELECT COUNT(*) FROM sample_rows WHERE id=1")
	if err != nil || count.Rows[0][0] != "1" {
		t.Fatal("explained DELETE was executed")
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
	after, err := s.Execute(ctx, "SELECT amount FROM sample_rows WHERE id=1")
	if err != nil || after.Rows[0][0] != before.Rows[0][0] {
		t.Fatal("explain committed pending DML")
	}
}

func TestOracleExportIntegration(t *testing.T) {
	s := workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	table := fmt.Sprintf("LORA_EXP_%d", time.Now().UnixNano()%1000000000000)
	if _, err := s.Execute(ctx, "CREATE TABLE "+table+" AS SELECT * FROM sample_rows WHERE 1=0"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := s.Execute(context.Background(), "DROP TABLE "+table+" PURGE"); err != nil {
			t.Error(err)
		}
	}()
	result, err := s.Execute(ctx, "SELECT * FROM sample_rows ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = exportdata.Write(ctx, &out, exportdata.Dataset{Columns: result.Columns, Rows: result.Values}, exportdata.Options{Format: "sql", Table: table, Columns: result.Columns}); err != nil {
		t.Fatal(err)
	}
	report, err := s.Script(ctx, out.String())
	if err != nil || len(report.Steps) != len(result.Rows) {
		t.Fatalf("INSERT export import: %v", err)
	}
	restored, err := s.Execute(ctx, "SELECT * FROM "+table+" ORDER BY id")
	if err != nil || !reflect.DeepEqual(restored.Rows, result.Rows) {
		t.Fatalf("roundtrip mismatch:\n%v\n%v\n%v", result.Rows, restored.Rows, err)
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
}
