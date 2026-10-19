package oracle

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestBindNamesIgnoreQuotesCommentsAndAssignment(t *testing.T) {
	text := "DECLARE n NUMBER; BEGIN n := :value; /* :ignored */ IF :VALUE IS NULL THEN n:=:1; END IF; -- :no\n n:=length(q'[:not_a_bind]')+length(nq'{:no}'); END;"
	names, err := BindNames(text)
	if err != nil || !reflect.DeepEqual(names, []string{"VALUE", "1"}) {
		t.Fatalf("%v %v", names, err)
	}
	for _, bad := range []string{`SELECT :"quoted" FROM dual`, "BEGIN :out := 1; END;", "INSERT INTO x VALUES(1) RETURNING id INTO :out;"} {
		if _, err := BindNames(bad); err == nil {
			t.Fatalf("OUT/quoted bind accepted: %s", bad)
		}
	}
}

func TestBindArgumentsNeverInterpolateValues(t *testing.T) {
	secret := "x'); DROP TABLE private_data; --"
	text := "SELECT :x, :X, :1, ':x' FROM dual"
	got, args, err := BindArguments(text, map[string]BindValue{"X": {Kind: "text", Text: secret}, "1": {Null: true}})
	if err != nil || strings.Contains(got, secret) || got != "SELECT :LAZYORA_B1, :LAZYORA_B1, :LAZYORA_B2, ':x' FROM dual" || len(args) != 2 {
		t.Fatalf("%q %v %v", got, args, err)
	}
	if args[0].(sql.NamedArg).Value != secret || args[1].(sql.NamedArg).Value != nil {
		t.Fatal("driver argument values lost")
	}
	if _, _, err = BindArguments(text, nil); err == nil {
		t.Fatal("missing values accepted")
	}
	for _, v := range []BindValue{{Kind: "number", Text: "bad secret number"}, {Kind: "date", Text: "bad secret date"}, {Kind: "timestamp", Text: "bad secret timestamp"}} {
		if _, err := v.Value(); err == nil || strings.Contains(err.Error(), v.Text) {
			t.Fatal("conversion accepted invalid input or leaked it")
		}
	}
	for _, text := range []string{"0", "-0.0001", "+1.25E2", "123456789012345678901234567890.12345678"} {
		if _, err := (BindValue{Kind: "number", Text: text}).Value(); err != nil {
			t.Fatalf("NUMBER %s: %v", text, err)
		}
	}
}

func TestBoundExecutionKeepsValuesOutOfJournal(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE items SET n=:LAZYORA_B1 WHERE name=:LAZYORA_B2").WithArgs(sql.Named("LAZYORA_B1", "bound secret"), sql.Named("LAZYORA_B2", "name")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectRollback()
	_, err := s.Execute(context.Background(), "UPDATE items SET n=:n WHERE name=:name", map[string]BindValue{"N": {Text: "bound secret"}, "NAME": {Text: "name"}})
	if err != nil {
		t.Fatal(err)
	}
	journal := s.PendingWork().Statements
	if len(journal) != 1 || strings.Contains(journal[0].SQL, "bound secret") || !strings.Contains(journal[0].SQL, ":n") {
		t.Fatal("journal stored values or replaced source placeholders")
	}
}
