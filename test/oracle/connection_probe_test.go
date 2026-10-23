package oracle_test

import (
	"context"
	"lazyora/internal/oracle"
	"os"
	"testing"
	"time"
)

func TestOracleConnectionProbeIntegration(t *testing.T) {
	s := workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := s.Execute(ctx, "UPDATE sample_rows SET amount=amount WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	port := 1521
	if value := os.Getenv("ORACLE_PORT"); value != "" {
		var err error
		port, err = oracle.ParsePort(value)
		if err != nil {
			t.Fatal(err)
		}
	}
	config := oracle.Config{Host: "127.0.0.1", Port: port, User: "lazyora", Password: os.Getenv("APP_USER_PASSWORD"), Name: "FREEPDB1", Mode: "service"}
	if err := oracle.ProbeConnection(ctx, config); err != nil {
		t.Fatal(err)
	}
	if !s.PendingChanges() {
		t.Fatal("probe ended working transaction")
	}
	if _, err := s.Execute(ctx, "SELECT 1 FROM dual"); err != nil {
		t.Fatal("working session replaced", err)
	}
	if err := s.Finish(false); err != nil {
		t.Fatal(err)
	}
}
