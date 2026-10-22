package oracle_test

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

// No dependency on the Free sample schema, DBA views, DBMS_SESSION.SLEEP,
// or 12c+ syntax: usable with an explicitly supplied disposable test account.
func compatibilityConfig(t *testing.T) oracle.Config {
	t.Helper()
	if os.Getenv("LAZYORA_COMPAT") != "1" {
		t.Skip("run test/oracle/compat-test.sh")
	}
	value := func(key string) string {
		v := os.Getenv("LAZYORA_COMPAT_" + key)
		if v == "" {
			t.Fatalf("LAZYORA_COMPAT_%s is required", key)
		}
		return v
	}
	port, err := oracle.ParsePort(value("PORT"))
	if err != nil {
		t.Fatal(err)
	}
	return oracle.Config{Host: value("HOST"), Port: port, User: value("USER"), Password: value("PASSWORD"), Mode: value("MODE"), Name: value("NAME")}
}

func TestOracleCompatibilityIntegration(t *testing.T) {
	config := compatibilityConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	s, err := oracle.Open(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	peer, err := oracle.Open(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	version, err := s.Execute(ctx, "SELECT product, version FROM product_component_version WHERE product LIKE 'Oracle%'")
	if err != nil || len(version.Rows) == 0 {
		t.Fatalf("server version: %v", err)
	}
	t.Logf("server=%v OS=%s/%s mode=%s", version.Rows, runtime.GOOS, runtime.GOARCH, config.Mode)
	// Fits the old 30-byte identifier limit; never replaces an existing object.
	table := fmt.Sprintf("LORA_COMP_%x", time.Now().UnixNano())
	if _, err = s.Execute(ctx, "CREATE TABLE "+table+" (id NUMBER PRIMARY KEY, title VARCHAR2(100 CHAR))"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = s.Finish(false)
		_ = peer.Finish(false)
		cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		if _, err := peer.Execute(cleanup, "DROP TABLE "+table+" PURGE"); err != nil {
			t.Error("fixture cleanup:", err)
		}
	}()
	if _, err = s.Execute(ctx, "INSERT INTO "+table+" VALUES (1, :text)", map[string]oracle.BindValue{"TEXT": {Kind: "text", Text: "한글 기준값"}}); err != nil {
		t.Fatal(err)
	}
	if err = s.Finish(true); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Execute(ctx, "UPDATE "+table+" SET title='pending' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if err = s.SetOutput(ctx, true); err != nil {
		t.Fatal(err)
	}
	block, err := s.Execute(ctx, "BEGIN DBMS_OUTPUT.PUT_LINE('한글 출력'); END;")
	if err != nil || block.Output.Error != "" || len(block.Output.Lines) != 1 || block.Output.Lines[0] != "한글 출력" {
		t.Fatalf("PL/SQL output: %+v %v", block.Output, err)
	}
	if err = s.SetOutput(ctx, false); err != nil {
		t.Fatal(err)
	}
	object, err := s.FindObject(ctx, table)
	if err != nil {
		t.Fatal(err)
	}
	columns, err := s.Columns(ctx, object)
	if err != nil || len(columns) != 2 || columns[0].Name != "ID" {
		t.Fatalf("dictionary columns: %+v %v", columns, err)
	}
	binds := map[string]oracle.BindValue{"N": {Kind: "number", Text: "123456789012345678901234567890.12345678"}, "T": {Kind: "text", Text: "한글 : 값"}, "X": {Null: true}}
	bound, err := s.Execute(ctx, "SELECT :n, :t, :x, TO_CLOB(:t) FROM dual", binds)
	if err != nil || len(bound.Rows) != 1 || bound.Rows[0][0] != binds["N"].Text || bound.Rows[0][1] != binds["T"].Text || !bound.Values[0][2].Null || bound.Rows[0][3] != binds["T"].Text {
		t.Fatalf("number/text/NULL/CLOB bind roundtrip: %v", err)
	}
	page, err := s.ExecutePaged(ctx, "SELECT LEVEL n FROM dual CONNECT BY LEVEL<=2507 ORDER BY LEVEL")
	count := len(page.Rows)
	for err == nil && page.PageID != 0 {
		page, err = s.FetchPage(ctx, page.PageID)
		count += len(page.Rows)
	}
	if err != nil || count != 2507 || page.Truncated {
		t.Fatalf("same-cursor paging: rows=%d err=%v", count, err)
	}
	observed, err := peer.Execute(ctx, "SELECT title FROM "+table+" WHERE id=1")
	if err != nil || len(observed.Rows) != 1 || observed.Rows[0][0] != "한글 기준값" || !s.PendingChanges() {
		t.Fatal("metadata/output/paging committed pending DML", err)
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
	after, err := s.Execute(ctx, "SELECT title FROM "+table+" WHERE id=1")
	if err != nil || len(after.Rows) != 1 || after.Rows[0][0] != "한글 기준값" {
		t.Fatal("rollback did not restore baseline", err)
	}
}

func TestOracleWideIntegration(t *testing.T) {
	s := workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	columns := make([]string, 16)
	for i := range columns {
		columns[i] = fmt.Sprintf("RPAD(TO_CHAR(LEVEL)||'x',4000,'x') c%d", i)
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	started := time.Now()
	page, err := s.ExecutePaged(ctx, "SELECT "+strings.Join(columns, ",")+" FROM dual CONNECT BY LEVEL<=2000")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Values) == 0 {
		t.Fatal("missing wide values")
	}
	// Documented UI-retention estimate, not a process RSS limit.
	cost := 64
	for _, cell := range page.Values[0] {
		cost += len(cell.Text) + len(cell.Type) + 512 + min(len(cell.Text), 2048)*2
	}
	if !page.PageLimited || page.PageID != 0 || !page.Truncated || len(page.Rows) != oracle.MaxGridBytes/cost || s.PageID() != 0 {
		t.Fatalf("wide retention cap: rows=%d cost=%d limited=%v cursor=%d", len(page.Rows), cost, page.PageLimited, page.PageID)
	}
	for _, row := range page.Rows {
		if len(row) != 16 || len(row[15]) != 4000 {
			t.Fatal("wide retained values were corrupted")
		}
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	t.Logf("wide grid: rows=%d columns=16 bytes/cell=4000 estimate=%d payload=%d heap_delta=%d allocated=%d elapsed=%s", len(page.Rows), len(page.Rows)*cost, len(page.Rows)*16*4000, int64(after.HeapAlloc)-int64(before.HeapAlloc), after.TotalAlloc-before.TotalAlloc, time.Since(started).Round(time.Millisecond))
	runtime.KeepAlive(page)
}
