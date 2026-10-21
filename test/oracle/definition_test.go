package oracle_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

func TestOracleDefinitionIntegration(t *testing.T) {
	s := workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	start := time.Now()
	initial, err := s.Definition(ctx, oracle.Object{Owner: "LAZYORA", Name: "SAMPLE_ROWS", Kind: "TABLE"}, "ddl")
	if err != nil || initial.DDL == "" || s.PendingChanges() {
		t.Fatalf("fresh metadata: %v", err)
	}
	t.Logf("fresh DDL read (without user transaction): %s", time.Since(start).Round(time.Millisecond))
	t.Cleanup(func() {
		if err := s.Finish(false); err != nil {
			t.Error(err)
		}
		for _, sql := range []string{`DROP PROCEDURE def_long`, `DROP TABLE "Def Child" PURGE`, `DROP TABLE "Def Parent" PURGE`} {
			if _, err := s.Execute(context.Background(), sql); err != nil && !strings.Contains(err.Error(), "ORA-00942") && !strings.Contains(err.Error(), "ORA-04043") {
				t.Error(err)
			}
		}
	})
	// Disposable quoted fixtures cover LONG checks, composite FKs and function indexes.
	for _, sql := range []string{
		`CREATE TABLE "Def Parent" ("鍵" NUMBER, "Part" NUMBER, CONSTRAINT "Def PK" PRIMARY KEY ("鍵","Part"))`,
		`CREATE TABLE "Def Child" ("鍵" NUMBER, "Part" NUMBER, "Title" VARCHAR2(30), CONSTRAINT "Def FK" FOREIGN KEY ("鍵","Part") REFERENCES "Def Parent"("鍵","Part") ON DELETE CASCADE, CONSTRAINT "Def Check" CHECK ("鍵">0))`,
		`CREATE INDEX "Def Expr" ON "Def Child"(UPPER("Title"), "Part" DESC)`,
	} {
		if _, err := s.Execute(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}

	object := oracle.Object{Owner: "LAZYORA", Name: "Def Child", Kind: "TABLE"}
	got, err := s.Definition(ctx, object, "ddl")
	if err != nil || !strings.Contains(got.DDL, `"Def Child"`) {
		t.Fatalf("DDL: %q %v", got.DDL, err)
	}
	got, err = s.Definition(ctx, object, "constraints")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range got.Constraints {
		if c.Name == "Def FK" {
			found = len(c.Columns) == 2 && len(c.ReferenceColumns) == 2 && c.ReferenceTable == "Def Parent" && c.DeleteRule == "CASCADE"
		}
		if c.Name == "Def Check" && c.Condition == "" {
			t.Fatal("check text missing")
		}
	}
	if !found {
		t.Fatalf("FK: %+v", got.Constraints)
	}
	got, err = s.Definition(ctx, object, "indexes")
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, index := range got.Indexes {
		if index.Name == "Def Expr" {
			found = len(index.Columns) == 2 && strings.Contains(index.Columns[0].Expression, "UPPER") && index.Columns[1].Direction == "DESC"
		}
	}
	if !found {
		t.Fatalf("function index: %+v", got.Indexes)
	}
	// Exercise multiple UTF-8 pieces and reject oversized output, never copy
	// a silently truncated executable-looking DDL into the UI.
	longComments := strings.Repeat("-- 한글 원문 👩‍💻 조각 경계 e\u0301\n", 1500)
	if _, err = s.Execute(ctx, "CREATE OR REPLACE PROCEDURE def_long AS\n"+longComments+"BEGIN NULL; END;"); err != nil {
		t.Fatal(err)
	}
	longObject := oracle.Object{Owner: "LAZYORA", Name: "DEF_LONG", Kind: "PROCEDURE"}
	large, err := s.Definition(ctx, longObject, "ddl")
	if err != nil || !strings.Contains(large.DDL, longComments) {
		t.Fatalf("UTF8 DDL pieces: %v", err)
	}
	tooLarge := strings.Repeat("-- xxxxxxxx\n", 26000)
	if _, err = s.Execute(ctx, "CREATE OR REPLACE PROCEDURE def_long AS\n"+tooLarge+"BEGIN NULL; END;"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Definition(ctx, longObject, "ddl"); err == nil || !strings.Contains(err.Error(), "262144") {
		t.Fatalf("DDL cap: %v", err)
	}
	// Reading all supported kinds must not resolve pending DML.
	if _, err = s.Execute(ctx, "UPDATE sample_rows SET title='definition pending' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ name, kind string }{
		{"SAMPLE_ROWS", "TABLE"}, {"SAMPLE_ROWS_VIEW", "VIEW"}, {"BROWSER_FUNCTION", "FUNCTION"}, {"BROWSER_PROCEDURE", "PROCEDURE"}, {"BROWSER_PACKAGE", "PACKAGE"}, {"BROWSER_PACKAGE", "PACKAGE BODY"}, {"BROWSER_TYPE", "TYPE"}, {"BROWSER_TYPE", "TYPE BODY"}, {"BROWSER_TRIGGER", "TRIGGER"}, {"BROWSER_SYNONYM", "SYNONYM"}, {"BROWSER_SEQUENCE", "SEQUENCE"}, {"BROWSER_INDEX", "INDEX"}, {"BROWSER_MVIEW", "MATERIALIZED VIEW"},
	} {
		result, err := s.Definition(ctx, oracle.Object{Owner: "LAZYORA", Name: item.name, Kind: item.kind}, "ddl")
		if err != nil || result.DDL == "" {
			t.Fatalf("%s DDL: %v", item.kind, err)
		}
	}
	if !s.PendingChanges() {
		t.Fatal("metadata lost pending work")
	}
	observer := workflowSession(t)
	row, err := observer.Execute(ctx, "SELECT title FROM sample_rows WHERE id=1")
	if err != nil || row.Rows[0][0] == "definition pending" {
		t.Fatal("metadata committed DML")
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
}
