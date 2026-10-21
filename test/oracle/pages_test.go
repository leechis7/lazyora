package oracle_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

func TestOraclePagesIntegration(t *testing.T) {
	s, admin := workflowSession(t), workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	t.Cleanup(func() {
		_ = s.Finish(false)
		_ = admin.Finish(false)
		for _, text := range []string{"DROP TABLE page_fixture PURGE", "DROP FUNCTION page_slow"} {
			if _, err := admin.Execute(context.Background(), text); err != nil && !strings.Contains(err.Error(), "ORA-00942") && !strings.Contains(err.Error(), "ORA-04043") {
				t.Error(err)
			}
		}
	})
	if _, err := admin.Execute(ctx, "CREATE TABLE page_fixture AS SELECT LEVEL id, CAST('old' AS VARCHAR2(20)) title FROM dual CONNECT BY LEVEL<=2507"); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Execute(ctx, `CREATE OR REPLACE FUNCTION page_slow(n NUMBER) RETURN NUMBER IS BEGIN IF n>=1200 THEN DBMS_SESSION.SLEEP(0.05); END IF; RETURN n; END;`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Execute(ctx, "UPDATE sample_rows SET title='pages pending' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	job, finishJob := context.WithCancel(ctx)
	first, err := s.ExecutePaged(job, "SELECT id,title FROM page_fixture ORDER BY id")
	finishJob()
	if err != nil || len(first.Rows) != 1000 || first.PageID == 0 || first.Rows[999][0] != "1000" || first.Values[0][0].Type != "NUMBER" {
		t.Fatalf("first page: %d %v", len(first.Rows), err)
	}
	metaJob, stopMeta := context.WithCancel(ctx)
	_, err = s.Definition(metaJob, oracle.Object{Owner: "LAZYORA", Name: "SAMPLE_ROWS", Kind: "TABLE"}, "grants")
	stopMeta()
	if err != nil {
		t.Fatal(err)
	}
	// A real requery would see 'new'; the retained cursor must keep its original
	// read-consistent snapshot even after another session commits an update.
	if _, err = admin.Execute(ctx, "UPDATE page_fixture SET title='new' WHERE id=1500"); err != nil {
		t.Fatal(err)
	}
	if err = admin.Finish(true); err != nil {
		t.Fatal(err)
	}
	second, err := s.FetchPage(ctx, first.PageID)
	if err != nil || len(second.Rows) != 1000 || second.Rows[0][0] != "1001" || second.Rows[499][1] != "old" {
		t.Fatalf("snapshot/second page: %v", err)
	}
	last, err := s.FetchPage(ctx, second.PageID)
	if err != nil || len(last.Rows) != 507 || last.Rows[506][0] != "2507" || last.PageID != 0 {
		t.Fatalf("last page: %v", err)
	}
	// A different execution invalidates the previous token, never resumes the
	// new query from an older tab or silently reruns SQL.
	old, err := s.ExecutePaged(ctx, "SELECT id FROM page_fixture ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Execute(ctx, "SELECT 1 FROM dual"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.FetchPage(ctx, old.PageID); err == nil {
		t.Fatal("stale token fetched another execution")
	}
	limited, err := s.ExecutePaged(ctx, fmt.Sprintf("SELECT LEVEL n FROM dual CONNECT BY LEVEL<=%d", oracle.MaxGridRows+1))
	count := len(limited.Rows)
	for err == nil && limited.PageID != 0 {
		limited, err = s.FetchPage(ctx, limited.PageID)
		count += len(limited.Rows)
	}
	if err != nil || count != oracle.MaxGridRows || !limited.PageLimited {
		t.Fatalf("row cap: %d %+v %v", count, limited, err)
	}
	active, err := s.ExecutePaged(ctx, "SELECT id FROM page_fixture ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	stopped, stop := context.WithCancel(ctx)
	stop()
	if _, err = s.FetchPage(stopped, active.PageID); err == nil || s.PageID() != 0 {
		t.Fatal("cancel did not release cursor")
	}
	blocked, err := s.ExecutePaged(ctx, "SELECT page_slow(LEVEL) n, RPAD('x',4000,'x') padding FROM dual CONNECT BY LEVEL<=2507")
	if err != nil || blocked.PageID == 0 {
		t.Fatalf("slow fetch setup: %v", err)
	}
	deadline, stopDeadline := context.WithTimeout(ctx, 200*time.Millisecond)
	started := time.Now()
	_, err = s.FetchPage(deadline, blocked.PageID)
	stopDeadline()
	if err == nil || time.Since(started) > 3*time.Second || s.PageID() != 0 {
		t.Fatalf("blocked fetch cancellation: %s %v", time.Since(started), err)
	}
	t.Logf("blocked network fetch cancelled in %s", time.Since(started).Round(time.Millisecond))
	if !s.PendingChanges() {
		t.Fatal("page operations resolved pending work")
	}
	observed, err := admin.Execute(ctx, "SELECT title FROM sample_rows WHERE id=1")
	if err != nil || observed.Rows[0][0] == "pages pending" {
		t.Fatal("page operations committed DML")
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
}
