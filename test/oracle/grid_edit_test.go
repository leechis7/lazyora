package oracle_test

import (
	"context"
	"errors"
	"lazyora/internal/oracle"
	"strings"
	"testing"
	"time"
)

func TestOracleGridEditIntegration(t *testing.T) {
	s, admin := workflowSession(t), workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	for _, name := range []string{"grid_edit_fixture", "grid_edit_duplicate"} {
		_, err := admin.Execute(ctx, "DROP TABLE "+name+" PURGE")
		if err != nil && !strings.Contains(err.Error(), "ORA-00942") {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		s.Finish(false)
		admin.Finish(false)
		for _, sql := range []string{"DROP TABLE grid_edit_fixture PURGE", "DROP TABLE grid_edit_duplicate PURGE"} {
			if _, err := admin.Execute(context.Background(), sql); err != nil {
				t.Error(err)
			}
		}
	})
	for _, sql := range []string{
		`CREATE TABLE grid_edit_fixture(id NUMBER(8) PRIMARY KEY,title VARCHAR2(100), amount NUMBER(38,4),d DATE,ts TIMESTAMP(6),r RAW(8),flag CHAR(3))`,
		`INSERT INTO grid_edit_fixture VALUES(1,'original',123456789012345678901234567890.1234,TO_DATE('2026-10-21 12:34:56','YYYY-MM-DD HH24:MI:SS'),TIMESTAMP '2026-10-21 12:34:56.123456',HEXTORAW('0001FF'),'a')`,
		`CREATE TABLE grid_edit_duplicate(title VARCHAR2(100))`,
		`INSERT INTO grid_edit_duplicate SELECT 'same' FROM dual CONNECT BY LEVEL<=2`,
	} {
		if _, err := admin.Execute(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	if err := admin.Finish(true); err != nil {
		t.Fatal(err)
	}
	object := oracle.Object{Owner: "LAZYORA", Name: "GRID_EDIT_FIXTURE", Kind: "TABLE"}
	load := func() oracle.Result {
		t.Helper()
		r, err := s.EditableTable(ctx, object)
		if err != nil || len(r.Rows) != 1 || r.EditID == 0 || len(r.Columns) != 7 {
			t.Fatalf("load: %+v %v", r, err)
		}
		return r
	}
	// Earlier work survives both a successful cell update and a failed attempt.
	if _, err := s.Execute(ctx, "UPDATE sample_rows SET amount=amount+12345 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	r := load()
	text := "O'Reilly'; DELETE FROM grid_edit_fixture; --"
	if err := s.ApplyEdit(ctx, oracle.EditChange{ID: r.EditID, Row: 0, Column: 1, Text: text}); err != nil {
		t.Fatal(err)
	}
	if !s.PendingChanges() || s.EditID() != 0 {
		t.Fatal("update autocommitted or retained stale edit capability")
	}
	other, err := admin.Execute(ctx, "SELECT title FROM grid_edit_fixture")
	if err != nil || other.Rows[0][0] != "original" {
		t.Fatal("update leaked before commit")
	}
	if err = s.ApplyEdit(ctx, oracle.EditChange{ID: r.EditID, Row: 0, Column: 1, Text: "stale"}); err == nil {
		t.Fatal("stale edit accepted")
	}
	r = load()
	if r.Rows[0][1] != text {
		t.Fatal("bound text changed")
	}
	if err = s.ApplyEdit(ctx, oracle.EditChange{ID: r.EditID, Row: 0, Column: 0, Text: "bad number"}); err == nil {
		t.Fatal("bad NUMBER accepted")
	}
	if !s.PendingChanges() {
		t.Fatal("validation cleared previous pending changes")
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
	r = load()
	if r.Rows[0][1] != "original" {
		t.Fatal("rollback did not revert edit")
	}
	// A different column changed remotely: the entire old row must match.
	if _, err = admin.Execute(ctx, "UPDATE grid_edit_fixture SET amount=42 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	admin.Finish(true)
	if err = s.ApplyEdit(ctx, oracle.EditChange{ID: r.EditID, Row: 0, Column: 1, Text: "lost update"}); !errors.Is(err, oracle.ErrEditConflict) {
		t.Fatalf("conflict: %v", err)
	}
	if s.PendingChanges() {
		t.Fatal("conflict marked a clean transaction dirty")
	}
	// Conversion/constraint failure rolls back only this attempt.
	if _, err = s.Execute(ctx, "UPDATE sample_rows SET amount=amount+12345 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	r = load()
	if err = s.ApplyEdit(ctx, oracle.EditChange{ID: r.EditID, Row: 0, Column: 1, Text: strings.Repeat("x", 101)}); err == nil {
		t.Fatal("oversized value accepted")
	}
	if !s.PendingChanges() || s.TransactionUncertain() {
		t.Fatal("failed attempt damaged earlier work")
	}
	s.Finish(false)
	// ROWID identifies one row even when a table has no primary key and duplicates.
	dup := oracle.Object{Owner: "LAZYORA", Name: "GRID_EDIT_DUPLICATE", Kind: "TABLE"}
	r, err = s.EditableTable(ctx, dup)
	if err != nil || len(r.Rows) != 2 {
		t.Fatalf("duplicates %v", err)
	}
	if err = s.ApplyEdit(ctx, oracle.EditChange{ID: r.EditID, Row: 0, Column: 0, Text: "changed"}); err != nil {
		t.Fatal(err)
	}
	if err = s.Finish(true); err != nil {
		t.Fatal(err)
	}
	observed, err := admin.Execute(ctx, "SELECT COUNT(*) FROM grid_edit_duplicate WHERE title='changed'")
	if err != nil || observed.Rows[0][0] != "1" {
		t.Fatal("ROWID updated multiple rows")
	}
	// A lock wait can be cancelled; the failed attempt must not resolve earlier work.
	r = load()
	if _, err = admin.Execute(ctx, "UPDATE grid_edit_fixture SET title='locked' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	blocked, stop := context.WithTimeout(ctx, 2*time.Second)
	err = s.ApplyEdit(blocked, oracle.EditChange{ID: r.EditID, Row: 0, Column: 1, Text: "blocked"})
	stop()
	admin.Finish(false)
	if err == nil || time.Since(started) > time.Second || s.TransactionUncertain() || s.PendingChanges() {
		t.Fatalf("cancelled lock attempt: %v", err)
	}
	// Explicit NULL, DATE, timestamp and binary values remain typed and exact.
	for _, tc := range []struct {
		col  int
		text string
		null bool
	}{{1, "", true}, {3, "2026-11-01 23:59:58", false}, {4, "2026-11-01 23:59:58.987654", false}, {5, "FF0002", false}} {
		r = load()
		if err = s.ApplyEdit(ctx, oracle.EditChange{ID: r.EditID, Row: 0, Column: tc.col, Text: tc.text, Null: tc.null}); err != nil {
			t.Fatal(err)
		}
	}
	if err = s.Finish(true); err != nil {
		t.Fatal(err)
	}
	checked := load()
	if !checked.Values[0][1].Null || checked.Rows[0][3] != "2026-11-01 23:59:58" || checked.Rows[0][4] != "2026-11-01 23:59:58.987654000" || checked.Rows[0][5] != "FF0002" {
		t.Fatalf("types: %+v", checked.Rows)
	}
	if _, err = s.Execute(ctx, "SELECT title FROM grid_edit_fixture"); err != nil {
		t.Fatal(err)
	}
	if s.EditID() != 0 {
		t.Fatal("arbitrary SQL retained edit permission")
	}
	if _, err = s.EditableTable(ctx, oracle.Object{Owner: "LAZYORA", Name: "GRID_EDIT_FIXTURE", Kind: "VIEW"}); err == nil {
		t.Fatal("view allowed editing")
	}
}
