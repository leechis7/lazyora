package oracle_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

func TestOracleCompletionIntegration(t *testing.T) {
	s := workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	request := func(marked string, refresh bool) oracle.Completions {
		t.Helper()
		cursor := strings.Index(marked, "|")
		text := strings.Replace(marked, "|", "", 1)
		c, err := oracle.SQLCompletionContext(text, cursor)
		if err != nil {
			t.Fatal(err)
		}
		out, err := s.Complete(ctx, c, refresh)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	contains := func(out oracle.Completions, text string) bool {
		for _, c := range out.Candidates {
			if c.Text == text {
				return true
			}
		}
		return false
	}
	out := request(`SELECT s.ti| FROM sample_rows s`, false)
	if out.Schema != "LAZYORA" || !contains(out, `"TITLE"`) || s.PendingChanges() {
		t.Fatalf("alias/schema: %+v", out)
	}
	if _, err := s.Execute(ctx, `CREATE TABLE "completion_44.Mixed" ("MixedCase" VARCHAR2(20),"한글 컬럼" NUMBER)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = s.Execute(context.Background(), `DROP TABLE "completion_44.Mixed" PURGE`) })
	if err := s.Finish(false); err != nil {
		t.Fatal(err)
	}
	out = request(`SELECT c."Mi|" FROM "completion_44.Mixed" c`, false)
	if !contains(out, `"MixedCase"`) {
		t.Fatalf("quoted identifier: %+v", out)
	}
	out = request(`SELECT c.| FROM "completion_44.Mixed" c`, false)
	if !contains(out, `"한글 컬럼"`) {
		t.Fatalf("Unicode column: %+v", out)
	}
	if _, err := s.Execute(ctx, `ALTER TABLE "completion_44.Mixed" ADD "AddedLater" NUMBER`); err != nil {
		t.Fatal(err)
	}
	if err := s.Finish(false); err != nil {
		t.Fatal(err)
	}
	if contains(request(`SELECT c.| FROM "completion_44.Mixed" c`, false), `"AddedLater"`) {
		t.Fatal("cache unexpectedly changed")
	}
	if !contains(request(`SELECT c.| FROM "completion_44.Mixed" c`, true), `"AddedLater"`) {
		t.Fatal("explicit refresh missed column")
	}
	out = request(`SELECT * FROM "completion_44.|`, false)
	if !contains(out, `"completion_44.Mixed"`) {
		t.Fatalf("literal underscore/period prefix: %+v", out)
	}
	if _, err := s.Execute(ctx, "ALTER SESSION SET CURRENT_SCHEMA=SYS"); err != nil {
		t.Fatal(err)
	}
	out = request(`SELECT * FROM DU|`, false)
	if out.Schema != "SYS" || !contains(out, `"DUAL"`) {
		t.Fatalf("current schema not login schema: %+v", out)
	}
	if _, err := s.Execute(ctx, "ALTER SESSION SET CURRENT_SCHEMA=LAZYORA"); err != nil {
		t.Fatal(err)
	}
	if err := s.Finish(false); err != nil {
		t.Fatal(err)
	}
	baseline, err := s.Execute(ctx, "SELECT title FROM sample_rows WHERE id=1")
	if err != nil {
		t.Fatal(err)
	}
	marked := `UPDATE sample_rows SET ti| = :new_value WHERE id=1`
	cursor := strings.Index(marked, "|")
	text := strings.Replace(marked, "|", "", 1)
	updateContext, err := oracle.SQLCompletionContext(text, cursor)
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := s.Complete(ctx, updateContext, false)
	if err != nil {
		t.Fatal(err)
	}
	replacement := ""
	for _, candidate := range candidates.Candidates {
		if candidate.Name == "TITLE" && candidate.Kind == "COLUMN" {
			replacement = candidate.Text
		}
	}
	if replacement != `"TITLE"` {
		t.Fatalf("invalid qualified SET target: %q", replacement)
	}
	update := text[:updateContext.Start] + replacement + text[updateContext.End:]
	if _, err = s.Execute(ctx, update, map[string]oracle.BindValue{"NEW_VALUE": {Text: "completion pending"}}); err != nil {
		t.Fatal(err)
	}
	before := len(s.PendingWork().Statements)
	request(`SELECT s.| FROM sample_rows s`, true)
	if !s.PendingChanges() || len(s.PendingWork().Statements) != before {
		t.Fatal("completion changed pending work")
	}
	observer := workflowSession(t)
	other, err := observer.Execute(ctx, "SELECT title FROM sample_rows WHERE id=1")
	if err != nil || other.Rows[0][0] != baseline.Rows[0][0] {
		t.Fatalf("completion committed: %v", err)
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
}
