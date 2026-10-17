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
	catalog, err := s.Catalog(ctx)
	if err != nil || catalog.Current != "LAZYORA" {
		t.Fatalf("catalog=%+v err=%v", catalog, err)
	}
	objects, err := s.Objects(ctx, catalog.Current)
	if err != nil {
		t.Fatal(err)
	}
	var sampleTable, sampleView oracle.Object
	for _, object := range objects {
		if object.Name == "SAMPLE_ROWS" && object.Kind == "TABLE" {
			sampleTable = object
		}
		if object.Name == "SAMPLE_ROWS_VIEW" && object.Kind == "VIEW" {
			sampleView = object
		}
	}
	if sampleTable.Name == "" || sampleView.Name == "" {
		t.Fatal("missing table/view fixtures; existing DB: make -C test/oracle browser-fixture")
	}
	expectedKinds := map[string]string{
		"TABLE": "SAMPLE_ROWS", "VIEW": "SAMPLE_ROWS_VIEW", "MATERIALIZED VIEW": "BROWSER_MVIEW",
		"FUNCTION": "BROWSER_FUNCTION", "PROCEDURE": "BROWSER_PROCEDURE",
		"PACKAGE": "BROWSER_PACKAGE", "PACKAGE BODY": "BROWSER_PACKAGE",
		"SEQUENCE": "BROWSER_SEQUENCE", "SYNONYM": "BROWSER_SYNONYM", "INDEX": "BROWSER_INDEX",
		"TRIGGER": "BROWSER_TRIGGER", "TYPE": "BROWSER_TYPE", "TYPE BODY": "BROWSER_TYPE",
	}
	foundKinds := make(map[string]bool)
	for _, object := range objects {
		if object.Name == "BROWSER_MVIEW" && object.Kind == "TABLE" {
			t.Fatal("materialized view storage duplicated in Tables")
		}
		if expectedKinds[object.Kind] == object.Name {
			foundKinds[object.Kind] = true
			info, infoErr := s.ObjectInfo(ctx, object)
			if infoErr != nil || info.Status != "VALID" {
				t.Fatalf("object metadata: %+v err=%v", info, infoErr)
			}
			if object.Kind == "MATERIALIZED VIEW" {
				mvColumns, columnErr := s.Columns(ctx, object)
				if columnErr != nil || len(mvColumns) != 2 || mvColumns[0].Name != "ID" || mvColumns[1].Comment != "구체화 뷰 제목 / Materialized view title" {
					t.Fatalf("materialized view columns=%+v err=%v", mvColumns, columnErr)
				}
			}
		}
	}
	for kind := range expectedKinds {
		if !foundKinds[kind] {
			t.Fatalf("missing %s fixture; run browser-fixture", kind)
		}
	}
	columns, err := s.Columns(ctx, sampleTable)
	if err != nil || len(columns) != 6 {
		t.Fatalf("table columns=%+v err=%v", columns, err)
	}
	if columns[0].Name != "ID" || columns[0].DataType != "NUMBER(10,0)" || columns[0].Nullable || columns[1].DataType != "VARCHAR2(100 CHAR)" || !columns[3].Nullable || columns[0].Comment != "식별자 / Identifier" || columns[3].Comment != "" {
		t.Fatalf("unexpected column details: %+v", columns)
	}
	viewColumns, err := s.Columns(ctx, sampleView)
	if err != nil || len(viewColumns) != 3 || viewColumns[0].Name != "ID" || viewColumns[1].Comment != "뷰 제목 / View title" {
		t.Fatalf("view columns=%+v err=%v", viewColumns, err)
	}
	// Inspect only the seed rows; manual TUI tests may have added other rows.
	result, err := s.Execute(ctx, "SELECT title, note, amount, created_at, changed_at FROM sample_rows WHERE id IN (1, 2, 3) ORDER BY id;")
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
	if s.PendingChanges() {
		t.Fatal("dictionary/SELECT queries marked pending changes")
	}
	updated, err := s.Execute(ctx, "UPDATE sample_rows SET title = '롤백 확인' WHERE id = 1;")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Affected != 1 {
		t.Fatalf("expected one updated row, got %d", updated.Affected)
	}
	if !s.PendingChanges() {
		t.Fatal("DML did not mark pending changes")
	}
	result, err = s.Execute(ctx, "SELECT title FROM sample_rows WHERE id = 1")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || result.Rows[0][0] != "롤백 확인" {
		t.Fatal("update was not visible inside the transaction")
	}
	// Browsing during pending DML must not commit or roll it back.
	if _, err = s.Catalog(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Objects(ctx, catalog.Current); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Columns(ctx, sampleTable); err != nil {
		t.Fatal(err)
	}
	result, err = s.Execute(ctx, "SELECT title FROM sample_rows WHERE id = 1")
	if err != nil || len(result.Rows) != 1 || result.Rows[0][0] != "롤백 확인" {
		t.Fatalf("browsing changed pending DML: result=%+v err=%v", result, err)
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
	if s.PendingChanges() {
		t.Fatal("rollback did not clear pending state")
	}
	result, err = s.Execute(ctx, "SELECT title FROM sample_rows WHERE id = 1")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || result.Rows[0][0] != "첫 번째 테스트" {
		t.Fatal("rollback did not restore the sample row")
	}
	// Exercise explicit commit without changing the fixture's values.
	if _, err = s.Execute(ctx, "UPDATE sample_rows SET title = title WHERE id = 1"); err != nil {
		t.Fatal(err)
	}
	if err = s.Finish(true); err != nil || s.PendingChanges() {
		t.Fatalf("commit did not clear pending state: %v", err)
	}
}
