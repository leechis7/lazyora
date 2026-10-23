package oracle_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

func TestOracleSelectRowIDEditIntegration(t *testing.T) {
	s, other := workflowSession(t), workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	const table = "select_rowid_edit_fixture"
	if _, err := other.Execute(ctx, "DROP TABLE "+table+" PURGE"); err != nil && !strings.Contains(err.Error(), "ORA-00942") {
		t.Fatal(err)
	}
	if _, err := other.Execute(ctx, `CREATE TABLE `+table+`(id NUMBER PRIMARY KEY,title VARCHAR2(100),d DATE,ts TIMESTAMP(6),r RAW(8),n NVARCHAR2(100),note CLOB)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		s.Finish(false)
		other.Finish(false)
		if _, err := other.Execute(context.Background(), "DROP TABLE "+table+" PURGE"); err != nil {
			t.Error(err)
		}
	})
	if _, err := other.Execute(ctx, `INSERT INTO `+table+` SELECT LEVEL,'old'||LEVEL,DATE '2026-10-22',TIMESTAMP '2026-10-22 12:34:56.123456',HEXTORAW('0102FF'),N'한글',TO_CLOB('read only') FROM dual CONNECT BY LEVEL<=1002`); err != nil {
		t.Fatal(err)
	}
	if err := other.Finish(true); err != nil {
		t.Fatal(err)
	}
	query := `SELECT t.ROWID AS rid,t.* FROM ` + table + ` t WHERE t.id=:id ORDER BY t.id`
	binds := map[string]oracle.BindValue{"ID": {Kind: "number", Text: "1"}}
	load := func() oracle.Result {
		t.Helper()
		r, err := s.ExecutePaged(ctx, query, binds)
		if err != nil || len(r.Rows) != 1 || r.EditID == 0 {
			t.Fatalf("editable SELECT: rows=%d edit=%d error=%v", len(r.Rows), r.EditID, err)
		}
		return r
	}
	r := load()
	if r.CanEditColumn(0) || r.CanEditColumn(7) || !r.CanEditColumn(2) {
		t.Fatal("ROWID/LOB protection or column mapping incorrect")
	}
	if err := s.ApplyEdit(ctx, oracle.EditChange{ID: r.EditID, Row: 0, Column: 0, Text: "not allowed"}); err == nil {
		t.Fatal("ROWID editable")
	}
	if err := s.ApplyEdit(ctx, oracle.EditChange{ID: r.EditID, Row: 0, Column: 7, Text: "not allowed"}); err == nil {
		t.Fatal("CLOB editable")
	}
	for _, change := range []struct {
		col   int
		value string
	}{{2, "O'Reilly'; DELETE FROM anything; --"}, {3, "2026-10-23 13:14:15"}, {4, "2026-10-23 13:14:15.654321"}, {5, "AABBCC"}, {6, "수정한 한글"}} {
		r = load()
		if err := s.ApplyEdit(ctx, oracle.EditChange{ID: r.EditID, Row: 0, Column: change.col, Text: change.value}); err != nil {
			t.Fatalf("column %d: %v", change.col, err)
		}
	}
	if !s.PendingChanges() {
		t.Fatal("SELECT edit autocommitted")
	}
	seen, err := other.Execute(ctx, "SELECT title FROM "+table+" WHERE id=1")
	if err != nil || seen.Rows[0][0] != "old1" {
		t.Fatal("other session saw uncommitted change", err)
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
	if r = load(); r.Rows[0][2] != "old1" {
		t.Fatal("rollback lost original value")
	}
	// A projected alias maps back to TITLE, and re-execution retains the query.
	aliasQuery := `SELECT x.ROWID rid,x.title "Shown" FROM ` + table + ` x WHERE id=1`
	r, err = s.ExecutePaged(ctx, aliasQuery)
	if err != nil || r.EditID == 0 {
		t.Fatal("alias result read-only", err)
	}
	if err = s.ApplyEdit(ctx, oracle.EditChange{ID: r.EditID, Row: 0, Column: 1, Text: "committed"}); err != nil {
		t.Fatal(err)
	}
	r, err = s.ExecutePaged(ctx, r.Query.Text, r.Query.Bindings)
	if err != nil || len(r.Columns) != 2 || r.Columns[1] != "Shown" || r.Rows[0][1] != "committed" {
		t.Fatal("original projection/filter not retained", err)
	}
	if err = s.Finish(true); err != nil {
		t.Fatal(err)
	}
	seen, err = other.Execute(ctx, "SELECT title FROM "+table+" WHERE id=1")
	if err != nil || seen.Rows[0][0] != "committed" {
		t.Fatal("explicit commit not visible", err)
	}
	// Earlier work survives a conflict in a different row.
	if _, err = s.Execute(ctx, "UPDATE "+table+" SET title='previous work' WHERE id=2"); err != nil {
		t.Fatal(err)
	}
	r = load()
	if _, err = other.Execute(ctx, "UPDATE "+table+" SET title='concurrent' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if err = other.Finish(true); err != nil {
		t.Fatal(err)
	}
	if err = s.ApplyEdit(ctx, oracle.EditChange{ID: r.EditID, Row: 0, Column: 2, Text: "overwrite"}); !errors.Is(err, oracle.ErrEditConflict) {
		t.Fatalf("conflict not detected: %v", err)
	}
	seen, err = s.Execute(ctx, "SELECT title FROM "+table+" WHERE id=2")
	if err != nil || seen.Rows[0][0] != "previous work" {
		t.Fatal("conflict rolled back previous work", err)
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
	// Fetching another page must retain the same source-row numbering.
	r, err = s.ExecutePaged(ctx, "SELECT t.ROWID,t.id,t.title FROM "+table+" t ORDER BY id")
	if err != nil || r.PageID == 0 || r.EditID == 0 {
		t.Fatal("editable paged query missing", err)
	}
	page, err := s.FetchPage(ctx, r.PageID)
	if err != nil || len(page.Rows) != 2 || page.EditID != r.EditID {
		t.Fatal("editable page failed", err)
	}
	if err = s.ApplyEdit(ctx, oracle.EditChange{ID: r.EditID, Row: 1001, Column: 2, Text: "last row"}); err != nil {
		t.Fatal(err)
	}
	seen, err = s.Execute(ctx, "SELECT id FROM "+table+" WHERE title='last row'")
	if err != nil || len(seen.Rows) != 1 || seen.Rows[0][0] != "1002" {
		t.Fatal("page edit modified wrong row", err)
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		`SELECT title AS "ROWID",id FROM ` + table,
		"SELECT t.ROWID,t.id+1 id FROM " + table + " t",
		"SELECT t.ROWID,t.title FROM " + table + " t JOIN " + table + " x ON t.id=x.id",
		"SELECT DISTINCT ROWID,title FROM " + table,
	} {
		r, err = s.ExecutePaged(ctx, sql)
		if err != nil || r.EditID != 0 {
			t.Fatalf("unsupported SELECT gained edit permission: %s / %v", sql, err)
		}
	}
}
