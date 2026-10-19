package oracle_test

import (
	"context"
	"os"
	"testing"
	"time"

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
