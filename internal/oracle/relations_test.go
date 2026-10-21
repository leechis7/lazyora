package oracle

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRelationsPreservePendingWorkAndBindQuotedNames(t *testing.T) {
	s, m := mockSession(t)
	ctx := context.Background()
	m.ExpectBegin()
	m.ExpectExec("UPDATE items").WillReturnResult(sqlmock.NewResult(0, 1))
	if _, err := s.Execute(ctx, "UPDATE items SET n=2"); err != nil {
		t.Fatal(err)
	}
	tx := s.tx
	object := Object{Owner: "O'wner", Name: "한글[red]", Kind: "TABLE"}
	m.ExpectQuery("SELECT grantor, grantee.*all_tab_privs.*all_col_privs").WithArgs(object.Owner, object.Name, object.Owner, object.Name).
		WillReturnRows(sqlmock.NewRows([]string{"GRANTOR", "GRANTEE", "PRIVILEGE", "GRANTABLE", "COL"}).AddRow(object.Owner, "PUBLIC", "SELECT", "NO", nil).AddRow(object.Owner, "ROLE", "UPDATE", "NO", "한글")).RowsWillBeClosed()
	g, err := s.Definition(ctx, object, "grants")
	if err != nil || len(g.Grants) != 2 || g.Grants[1].Column != "한글" {
		t.Fatalf("grants: %+v %v", g, err)
	}
	for _, section := range []string{"dependencies", "dependents"} {
		m.ExpectQuery("SELECT target_owner").WithArgs(object.Owner, object.Name, object.Kind).WillReturnRows(sqlmock.NewRows([]string{"O", "N", "T", "L", "D", "S"}).
			AddRow("APP", "Broken", "VIEW", nil, "HARD", "INVALID").AddRow("APP", "Remote", "TABLE", "LINK", "HARD", nil).AddRow("APP", "Missing", "TABLE", nil, "HARD", nil)).RowsWillBeClosed()
		got, err := s.Definition(ctx, object, section)
		if err != nil || len(got.Relations) != 3 || !got.Relations[0].Navigable() || got.Relations[1].Navigable() || got.Relations[2].Navigable() {
			t.Fatalf("relations: %+v %v", got, err)
		}
	}
	if s.tx != tx || !s.PendingChanges() || len(s.PendingWork().Statements) != 1 {
		t.Fatal("metadata resolved or polluted pending transaction")
	}
	m.ExpectRollback()
	if err := s.Finish(false); err != nil {
		t.Fatal(err)
	}
}

func TestRelationsLimitsFailuresAndUnsupportedNavigation(t *testing.T) {
	s, m := mockSession(t)
	rows := sqlmock.NewRows([]string{"O", "N", "T", "L", "D", "S"})
	for i := 0; i < definitionRows+1; i++ {
		rows.AddRow("APP", "T", "TABLE", nil, "HARD", "VALID")
	}
	m.ExpectQuery("SELECT target_owner").WillReturnRows(rows).RowsWillBeClosed()
	got, err := s.Definition(context.Background(), Object{Owner: "APP", Name: "T", Kind: "TABLE"}, "dependencies")
	if err != nil || !got.Limited || len(got.Relations) != definitionRows {
		t.Fatalf("limit: %+v %v", got, err)
	}
	m.ExpectQuery("SELECT grantor").WillReturnError(errors.New("ORA-00942"))
	if _, err = s.Definition(context.Background(), Object{Owner: "APP", Name: "T"}, "grants"); err == nil || s.PendingChanges() {
		t.Fatal("permission error hidden or pending state changed")
	}
	if (ObjectRelation{Object: Object{Owner: "APP", Name: "X", Kind: "JAVA CLASS"}, Visible: true}).Navigable() {
		t.Fatal("unsupported target navigable")
	}
	s.finishErr = errors.New("transaction outcome unknown")
	if _, err = s.Definition(context.Background(), Object{Owner: "APP", Name: "T"}, "grants"); err != s.finishErr {
		t.Fatal("metadata bypassed unresolved transaction guard")
	}
}
