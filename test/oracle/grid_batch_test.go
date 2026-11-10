package oracle_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

func TestOracleGridBatchEditIntegration(t *testing.T) {
	writer, observer := workflowSession(t), workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	execute := func(session *oracle.Session, statement string) oracle.Result {
		t.Helper()
		result, err := session.Execute(ctx, statement)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	finish := func(session *oracle.Session, commit bool) {
		t.Helper()
		if err := session.Finish(commit); err != nil {
			t.Fatal(err)
		}
	}
	execute(observer, "CREATE TABLE lazyora_grid_batch_test (id NUMBER PRIMARY KEY, title VARCHAR2(20) NOT NULL, amount NUMBER CHECK(amount>=0))")
	t.Cleanup(func() {
		_ = writer.Finish(false)
		_ = observer.Finish(false)
		if _, err := observer.Execute(context.Background(), "DROP TABLE lazyora_grid_batch_test PURGE"); err != nil {
			t.Error(err)
		}
	})
	execute(observer, "INSERT INTO lazyora_grid_batch_test VALUES (1,'first',10)")
	execute(observer, "INSERT INTO lazyora_grid_batch_test VALUES (2,'second',20)")
	finish(observer, true)
	const projection = "SELECT t.ROWID,t.id,t.title,t.amount FROM lazyora_grid_batch_test t ORDER BY t.ROWID"
	load := func() oracle.Result {
		t.Helper()
		r, err := writer.ExecutePaged(ctx, projection, nil)
		if err != nil || r.EditID == 0 || len(r.Rows) != 2 {
			t.Fatalf("invalid editable result: %v", err)
		}
		return r
	}
	read := func(session *oracle.Session) [][]string {
		t.Helper()
		return execute(session, "SELECT id,title,amount FROM lazyora_grid_batch_test ORDER BY id").Rows
	}
	original := [][]string{{"1", "first", "10"}, {"2", "second", "20"}}
	want := [][]string{{"1", "updated", "11"}, {"2", "second", "NULL"}}
	r := load()
	changes := []oracle.EditChange{
		{ID: r.EditID, Row: 0, Column: 2, Text: "updated"},
		{ID: r.EditID, Row: 0, Column: 3, Text: "11"},
		{ID: r.EditID, Row: 1, Column: 3, Null: true},
	}
	if r.Rows[0][1] != "1" {
		t.Fatal("fixture must be ordered by inserted row identity")
	}
	if err := writer.ApplyEdits(ctx, changes); err != nil {
		t.Fatal(err)
	}
	if got := read(observer); !reflect.DeepEqual(got, original) {
		t.Fatalf("uncommitted batch visible to observer: %v", got)
	}
	finish(writer, false)
	if got := read(observer); !reflect.DeepEqual(got, original) {
		t.Fatalf("rollback changed observer result: %v", got)
	}
	r = load()
	for i := range changes {
		changes[i].ID = r.EditID
	}
	changes[2].Text, changes[2].Null = "-1", false
	if err := writer.ApplyEdits(ctx, changes); err == nil {
		t.Fatal("constraint failure accepted")
	}
	if writer.EditID() != r.EditID || writer.PendingChanges() {
		t.Fatal("failed batch lost retry snapshot or left dirty SQL")
	}
	if got := read(writer); !reflect.DeepEqual(got, original) {
		t.Fatalf("failed batch left part of update in writer: %v", got)
	}
	// Verify an external update on the second row prevents the whole batch.
	r = load()
	for i := range changes {
		changes[i].ID = r.EditID
	}
	changes[2].Text, changes[2].Null = "", true
	execute(observer, "UPDATE lazyora_grid_batch_test SET title='external' WHERE id=2")
	finish(observer, true)
	if err := writer.ApplyEdits(ctx, changes); !errors.Is(err, oracle.ErrEditConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if got := read(writer); got[0][1] != "first" || got[1][1] != "external" {
		t.Fatalf("conflict left partial writes: %v", got)
	}
	execute(observer, "UPDATE lazyora_grid_batch_test SET title='second' WHERE id=2")
	finish(observer, true)
	r = load()
	for i := range changes {
		changes[i].ID = r.EditID
	}
	execute(observer, "UPDATE lazyora_grid_batch_test SET amount=amount WHERE id=2")
	if err := writer.ApplyEdits(ctx, changes); err == nil || writer.EditID() != r.EditID || writer.PendingChanges() {
		t.Fatalf("lock failure did not preserve retryable batch: %v", err)
	}
	finish(observer, false)
	if err := writer.ApplyEdits(ctx, changes); err != nil {
		t.Fatal(err)
	}
	finish(writer, true)
	if got := read(observer); !reflect.DeepEqual(got, want) {
		t.Fatalf("committed batch: got %v want %v", got, want)
	}
}
