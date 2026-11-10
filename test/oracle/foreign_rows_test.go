package oracle_test

import (
	"context"
	"errors"
	"testing"

	"lazyora/internal/oracle"
)

func TestOracleRelatedRowsIntegration(t *testing.T) {
	s := workflowSession(t)
	ctx := context.Background()
	exec := func(sql string) {
		t.Helper()
		if _, err := s.Execute(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	exec("CREATE TABLE lazyora_fk_parent (n NUMBER(38), code VARCHAR2(40), PRIMARY KEY(n, code))")
	t.Cleanup(func() {
		_ = s.Finish(false)
		exec("DROP TABLE lazyora_fk_parent PURGE")
	})
	exec("CREATE TABLE lazyora_fk_child (id NUMBER PRIMARY KEY, parent_code VARCHAR2(40), parent_n NUMBER(38), CONSTRAINT lazyora_fk_pair FOREIGN KEY(parent_n, parent_code) REFERENCES lazyora_fk_parent(n, code))")
	t.Cleanup(func() {
		_ = s.Finish(false)
		exec("DROP TABLE lazyora_fk_child PURGE")
	})
	if err := s.Finish(true); err != nil {
		t.Fatal(err)
	}
	const large = "12345678901234567890123456789012345678"
	exec("INSERT INTO lazyora_fk_parent VALUES (" + large + ", 'O''Brien')")
	exec("INSERT INTO lazyora_fk_parent VALUES (" + large + ", 'Other')")
	exec("INSERT INTO lazyora_fk_child VALUES (1, 'O''Brien', " + large + ")")
	exec("INSERT INTO lazyora_fk_child VALUES (2, 'Other', " + large + ")")
	catalog, err := s.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	parent := oracle.Object{Owner: catalog.Current, Name: "LAZYORA_FK_PARENT", Kind: "TABLE"}
	child := oracle.Object{Owner: catalog.Current, Name: "LAZYORA_FK_CHILD", Kind: "TABLE"}
	for _, tc := range []struct {
		object   oracle.Object
		headers  []string
		values   []oracle.Cell
		toParent bool
		want     string
	}{
		{child, []string{"PARENT_CODE", "PARENT_N"}, []oracle.Cell{{Text: "O'Brien", Type: "VARCHAR2"}, {Text: large, Type: "NUMBER"}}, true, large},
		{parent, []string{"CODE", "N"}, []oracle.Cell{{Text: "O'Brien", Type: "VARCHAR2"}, {Text: large, Type: "NUMBER"}}, false, "1"},
	} {
		links, err := s.ForeignKeys(ctx, tc.object)
		if err != nil || len(links.Links) != 1 || links.Links[0].ToParent != tc.toParent || len(links.Links[0].Columns) != 2 {
			t.Fatalf("foreign keys: %+v, %v", links, err)
		}
		rows, err := s.RelatedRows(ctx, links.Links[0], tc.headers, tc.values, 100)
		if err != nil || len(rows.Values) != 1 || rows.Values[0][0].Text != tc.want {
			t.Fatalf("related rows: %+v, %v", rows.Values, err)
		}
		tc.values[0].Null = true
		if _, err := s.RelatedRows(ctx, links.Links[0], tc.headers, tc.values, 100); !errors.Is(err, oracle.ErrForeignKeyNull) {
			t.Fatalf("NULL key: %v", err)
		}
	}
	if !s.PendingChanges() {
		t.Fatal("navigation resolved the transaction")
	}
}
