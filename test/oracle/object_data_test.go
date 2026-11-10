package oracle_test

import (
	"context"
	"strings"
	"testing"

	"lazyora/internal/oracle"
)

func TestOracleObjectPreviewIntegration(t *testing.T) {
	s := workflowSession(t)
	ctx := context.Background()
	if _, err := s.Execute(ctx, "CREATE TABLE lazyora_obj_preview (n NUMBER, content CLOB)"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = s.Finish(false)
		if _, err := s.Execute(ctx, "DROP TABLE lazyora_obj_preview PURGE"); err != nil {
			t.Error(err)
		}
	})
	if err := s.Finish(true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Execute(ctx, "INSERT INTO lazyora_obj_preview VALUES (1, TO_CLOB(RPAD('x',4000,'x')) || RPAD('y',4000,'y'))"); err != nil {
		t.Fatal(err)
	}
	catalog, err := s.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.ExecutePaged(ctx, "SELECT LEVEL AS N FROM dual CONNECT BY LEVEL <= 1002")
	if err != nil || first.PageID == 0 {
		t.Fatalf("create retained cursor: %v", err)
	}
	preview, err := s.ObjectPreview(ctx, oracle.Object{Owner: catalog.Current, Name: "LAZYORA_OBJ_PREVIEW", Kind: "TABLE"}, 100)
	if err != nil || len(preview.Values) != 1 || len(preview.Values[0]) != 2 {
		t.Fatalf("preview: %v, %v", len(preview.Values), err)
	}
	number, lob := preview.Values[0][0], preview.Values[0][1]
	if number.Text != "1" || !lob.Partial || lob.LOBID != 0 || !strings.HasPrefix(lob.Text, "xxx") || !s.PendingChanges() {
		t.Fatal("preview did not preserve the transaction or bound the LOB")
	}
	next, err := s.FetchPage(ctx, first.PageID)
	if err != nil || len(next.Rows) != 2 || next.Rows[0][0] != "1001" {
		t.Fatalf("preview invalidated the active cursor: %v", err)
	}
}
