package oracle_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

func TestOracleRelationsIntegration(t *testing.T) {
	admin := workflowSession(t)
	s := workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	t.Cleanup(func() {
		_ = s.Finish(false)
		_ = admin.Finish(false)
		for _, text := range []string{`DROP VIEW "Rel Bad"`, `DROP VIEW "Rel View"`, `DROP TABLE "Rel Table" PURGE`, `DROP ROLE LORA_REL_TEST`} {
			if _, err := admin.Execute(context.Background(), text); err != nil && !strings.Contains(err.Error(), "ORA-00942") && !strings.Contains(err.Error(), "ORA-01919") {
				t.Error(err)
			}
		}
	})
	for _, text := range []string{
		`CREATE ROLE LORA_REL_TEST`,
		`CREATE TABLE "Rel Table" ("ID" NUMBER, "Value" VARCHAR2(20))`,
		`CREATE VIEW "Rel View" AS SELECT "ID" FROM "Rel Table"`,
		`CREATE VIEW "Rel Bad" AS SELECT "Value" FROM "Rel Table"`,
		`GRANT SELECT ON "Rel Table" TO LORA_REL_TEST`,
		`GRANT UPDATE ("ID") ON "Rel Table" TO LORA_REL_TEST`,
		`ALTER TABLE "Rel Table" DROP COLUMN "Value"`,
	} {
		if _, err := admin.Execute(ctx, text); err != nil {
			t.Fatalf("fixture %s: %v", text, err)
		}
	}
	// Disable all roles (including the fixture user's DBA role); dictionary
	// reads below must work using the user's own object visibility.
	if _, err := s.Execute(ctx, "SET ROLE NONE"); err != nil {
		t.Fatal(err)
	}
	if err := s.Finish(false); err != nil {
		t.Fatal(err)
	}
	roles, err := s.Execute(ctx, "SELECT role FROM session_roles")
	if err != nil || len(roles.Rows) != 0 {
		t.Fatalf("roles still enabled: %+v %v", roles, err)
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
	object := oracle.Object{Owner: "LAZYORA", Name: "Rel Table", Kind: "TABLE"}
	grants, err := s.Definition(ctx, object, "grants")
	if err != nil {
		t.Fatal(err)
	}
	selectGrant, columnGrant := false, false
	for _, g := range grants.Grants {
		if g.Grantee == "LORA_REL_TEST" && g.Privilege == "SELECT" && g.Column == "" {
			selectGrant = true
		}
		if g.Grantee == "LORA_REL_TEST" && g.Privilege == "UPDATE" && g.Column == "ID" {
			columnGrant = true
		}
	}
	if !selectGrant || !columnGrant || s.PendingChanges() {
		t.Fatalf("visible owner grants: %+v", grants)
	}
	view := oracle.Object{Owner: "LAZYORA", Name: "Rel View", Kind: "VIEW"}
	deps, err := s.Definition(ctx, view, "dependencies")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range deps.Relations {
		if r.Object.Name == object.Name && r.Object.Owner == object.Owner && r.Navigable() {
			found = true
		}
	}
	if !found {
		t.Fatalf("forward dependency: %+v", deps)
	}
	reverse, err := s.Definition(ctx, object, "dependents")
	if err != nil {
		t.Fatal(err)
	}
	valid, invalid := false, false
	for _, r := range reverse.Relations {
		if r.Object.Name == "Rel View" {
			valid = true
		}
		if r.Object.Name == "Rel Bad" && r.Object.Status == "INVALID" {
			invalid = true
		}
	}
	if !valid || !invalid {
		t.Fatalf("reverse/status: %+v", reverse)
	}
	if _, err = s.Execute(ctx, "UPDATE sample_rows SET title='relations pending' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	for _, section := range []string{"grants", "dependencies", "dependents"} {
		if _, err = s.Definition(ctx, object, section); err != nil {
			t.Fatal(err)
		}
	}
	if !s.PendingChanges() {
		t.Fatal("metadata lost pending changes")
	}
	observed, err := admin.Execute(ctx, "SELECT title FROM sample_rows WHERE id=1")
	if err != nil || observed.Rows[0][0] == "relations pending" {
		t.Fatal("metadata committed pending changes")
	}
	if err = s.Finish(false); err != nil {
		t.Fatal(err)
	}
}
