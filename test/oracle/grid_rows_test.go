package oracle_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

func TestOracleGridRowDraftIntegration(t *testing.T) {
	s, observer := workflowSession(t), workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	execute := func(conn *oracle.Session, text string) oracle.Result {
		t.Helper()
		r, err := conn.Execute(ctx, text)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	finish := func(conn *oracle.Session, commit bool) {
		t.Helper()
		if err := conn.Finish(commit); err != nil {
			t.Fatal(err)
		}
	}
	execute(s, "CREATE TABLE lazyora_row_drafts (id NUMBER PRIMARY KEY, qty NUMBER DEFAULT 42, note VARCHAR2(32) DEFAULT 'default')")
	t.Cleanup(func() {
		_ = observer.Finish(false)
		_ = s.Finish(false)
		if _, err := s.Execute(context.Background(), "DROP TABLE lazyora_row_drafts PURGE"); err != nil {
			t.Error(err)
		}
	})
	execute(s, "INSERT INTO lazyora_row_drafts (id) VALUES (1)")
	execute(s, "INSERT INTO lazyora_row_drafts (id) VALUES (2)")
	finish(s, true)
	query := func(text string) oracle.Result {
		t.Helper()
		r, err := s.ExecutePaged(ctx, text)
		if err != nil || !r.CanInsertRow() {
			t.Fatalf("verified insert target: %+v, %v", r, err)
		}
		return r
	}
	partial := query("SELECT ROWID, id FROM lazyora_row_drafts")
	if partial.EditCanDelete || s.ApplyGridChanges(ctx, nil, []oracle.EditDelete{{ID: partial.EditID, Row: 0}}, nil) == nil {
		t.Fatal("partial snapshot allowed deletion")
	}
	empty := query("SELECT t.ROWID, t.id, t.qty, t.note FROM lazyora_row_drafts t WHERE id = -1")
	if len(empty.Rows) != 0 || empty.EditRowIDColumn != 1 {
		t.Fatal("empty verified result lost its target")
	}
	const all = "SELECT t.ROWID, t.id, t.qty, t.note FROM lazyora_row_drafts t ORDER BY id"
	snap := query(all)
	if !snap.EditCanDelete || snap.EditRowIDColumn != 1 || len(snap.EditRowIDs) != 2 {
		t.Fatal("full verified snapshot lost delete metadata")
	}
	// An earlier change must survive when a later INSERT fails after a DELETE.
	execute(observer, "UPDATE lazyora_row_drafts SET note = 'external' WHERE id = 1")
	finish(observer, true)
	stale := []oracle.EditDelete{{ID: snap.EditID, Row: 0}}
	if err := s.ApplyGridChanges(ctx, nil, stale, nil); !errors.Is(err, oracle.ErrEditConflict) {
		t.Fatalf("changed row must not be deleted: %v", err)
	}
	execute(s, "UPDATE lazyora_row_drafts SET qty = 55 WHERE id = 1")
	snap = query(all)
	update := []oracle.EditChange{{ID: snap.EditID, Row: 0, Column: 3, Text: "edited"}}
	deleteRow := []oracle.EditDelete{{ID: snap.EditID, Row: 1}}
	insert := func(id string) []oracle.EditInsert {
		return []oracle.EditInsert{{ID: snap.EditID, Fields: map[int]oracle.EditInsertField{1: {Text: id}, 3: {Null: true}}}}
	}
	if err := s.ApplyGridChanges(ctx, update, deleteRow, insert("1")); err == nil || !s.PendingChanges() || s.EditID() != snap.EditID {
		t.Fatalf("failed batch didn't retain earlier work and snapshot: %v", err)
	}
	// A retry against the same snapshot proves that the attempted UPDATE
	// and DELETE were undone, while qty=55 still satisfies the predicate.
	if err := s.ApplyGridChanges(ctx, update, deleteRow, insert("3")); err != nil {
		t.Fatal(err)
	}
	read := "SELECT id, qty, note FROM lazyora_row_drafts ORDER BY id"
	before := execute(observer, read).Values
	if len(before) != 2 || before[0][1].Text != "42" || before[0][2].Text != "external" || before[1][0].Text != "2" {
		t.Fatalf("drafts visible to another session before commit: %+v", before)
	}
	finish(s, true)
	after := execute(observer, read).Values
	if len(after) != 2 || after[0][1].Text != "55" || after[0][2].Text != "edited" ||
		after[1][0].Text != "3" || after[1][1].Text != "42" || !after[1][2].Null {
		t.Fatalf("expected committed update, delete, default and explicit NULL: %+v", after)
	}
}
