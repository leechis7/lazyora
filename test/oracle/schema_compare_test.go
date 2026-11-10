package oracle_test

import (
	"context"
	"testing"

	"lazyora/internal/oracle"
)

func TestOracleSchemaCompareIntegration(t *testing.T) {
	left, right := workflowSession(t), workflowSession(t)
	ctx := context.Background()
	exec := func(s *oracle.Session, sql string) {
		t.Helper()
		if _, err := s.Execute(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	exec(right, "CREATE TABLE lazyora_cmp_probe (id NUMBER(10) NOT NULL, title VARCHAR2(20 CHAR) DEFAULT 'old')")
	t.Cleanup(func() {
		_ = left.Finish(false)
		_ = right.Finish(false)
		exec(right, "DROP TABLE lazyora_cmp_probe PURGE")
	})
	if err := right.Finish(true); err != nil {
		t.Fatal(err)
	}
	a, err := left.ReadSchemaStructure(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	exec(right, "ALTER TABLE lazyora_cmp_probe MODIFY title VARCHAR2(40 CHAR) DEFAULT 'new'")
	exec(right, "ALTER TABLE lazyora_cmp_probe ADD amount NUMBER(18,4)")
	exec(right, "INSERT INTO lazyora_cmp_probe (id) VALUES (1)")
	b, err := right.ReadSchemaStructure(ctx, a.Owner)
	if err != nil {
		t.Fatal(err)
	}
	if !right.PendingChanges() || left.PendingChanges() {
		t.Fatal("metadata comparison changed a session's pending transaction")
	}
	differences := oracle.CompareSchemas(a, b)
	if len(differences) != 1 || differences[0].Name != "LAZYORA_CMP_PROBE" || len(differences[0].Columns) != 2 {
		t.Fatalf("table changes: %+v", differences)
	}
	added, modified := differences[0].Columns[0], differences[0].Columns[1]
	if added.Name != "AMOUNT" || added.Left != nil || added.Right.Type != "NUMBER(18,4)" ||
		modified.Name != "TITLE" || modified.Left.Type != "VARCHAR2(20 CHAR)" || modified.Right.Type != "VARCHAR2(40 CHAR)" ||
		modified.Left.Default != "'old'" || modified.Right.Default != "'new'" {
		t.Fatalf("column changes: %+v", differences[0].Columns)
	}
	if _, err := right.ReadSchemaStructure(ctx, "LAZYORA_NO_SUCH_SCHEMA"); err == nil {
		t.Fatal("missing schema was treated as empty")
	}
}
