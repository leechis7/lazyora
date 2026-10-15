package oracle_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

// Opt-in: uses the local Compose test schema and rolls back its DML.
func TestOracleFreeIntegration(t *testing.T) {
	if os.Getenv("LAZYORA_INTEGRATION") != "1" {
		t.Skip("test/oracle/에서 make up 후 make test로 실행")
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
	defer s.Close()
	result, err := s.Execute(ctx, "SELECT title, note, amount, created_at, changed_at FROM sample_rows ORDER BY id;")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 3 {
		t.Fatalf("expected 3 sample rows, got %d", len(result.Rows))
	}
	if result.Rows[0][0] != "첫 번째 테스트" || result.Rows[0][1] != "한글 표시 확인" {
		t.Fatal("Korean text did not round-trip")
	}
	if result.Rows[1][1] != "NULL" || result.Rows[1][2] != "NULL" || result.Rows[1][4] != "NULL" {
		t.Fatal("NULL values were not preserved")
	}
	if !strings.HasPrefix(result.Rows[0][3], "2026-10-15") {
		t.Fatal("unexpected DATE value")
	}
	if result.Rows[0][2] != "12345.6789" || result.Rows[2][2] != "-0.0001" {
		t.Fatal("unexpected NUMBER values")
	}
	if !strings.HasPrefix(result.Rows[0][4], "2026-10-15 09:30:00.123456") {
		t.Fatal("unexpected TIMESTAMP value")
	}
	updated, err := s.Execute(ctx, "UPDATE sample_rows SET title = '롤백 확인' WHERE id = 1;")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Affected != 1 {
		t.Fatalf("expected one updated row, got %d", updated.Affected)
	}
	result, err = s.Execute(ctx, "SELECT title FROM sample_rows WHERE id = 1")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || result.Rows[0][0] != "롤백 확인" {
		t.Fatal("update was not visible inside the transaction")
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
	result, err = s.Execute(ctx, "SELECT title FROM sample_rows WHERE id = 1")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || result.Rows[0][0] != "첫 번째 테스트" {
		t.Fatal("rollback did not restore the sample row")
	}
}
