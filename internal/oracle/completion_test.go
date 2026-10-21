package oracle

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func expectCompletionSchema(mock sqlmock.Sqlmock, schema string) {
	mock.ExpectQuery("CURRENT_SCHEMA").WillReturnRows(sqlmock.NewRows([]string{"SCHEMA"}).AddRow(schema)).RowsWillBeClosed()
}
func TestCompletionUsesCurrentSchemaCachesAndRefreshes(t *testing.T) {
	s, mock := mockSession(t)
	c := completionContext(t, `SELECT s.ti| FROM sample_rows s`)
	for i := 0; i < 3; i++ {
		expectCompletionSchema(mock, "CURRENT_OWNER")
		if i != 1 {
			mock.ExpectQuery("FROM all_tab_columns").WithArgs("CURRENT_OWNER", "SAMPLE_ROWS", completionColumnLimit+1).WillReturnRows(sqlmock.NewRows([]string{"NAME", "TYPE"}).AddRow("TITLE", "VARCHAR2").AddRow("ID", "NUMBER")).RowsWillBeClosed()
		}
		out, err := s.Complete(context.Background(), c, i == 2)
		if err != nil || len(out.Candidates) != 1 || out.Candidates[0].Text != `"TITLE"` || out.Schema != "CURRENT_OWNER" {
			t.Fatalf("%+v %v", out, err)
		}
	}
	if s.tx != nil || s.pending || len(s.work.Statements) > 0 {
		t.Fatal("completion changed transaction")
	}
}
func TestCompletionObjectsLiteralPrefixAndQuotedIdentifiers(t *testing.T) {
	s, mock := mockSession(t)
	c := completionContext(t, `SELECT * FROM "MixOwner"."ab_|`)
	expectCompletionSchema(mock, "LOGIN")
	mock.ExpectQuery(`object_name LIKE :2`).WithArgs("MixOwner", `ab\_%`, completionObjectLimit+1).WillReturnRows(sqlmock.NewRows([]string{"NAME", "TYPE"}).AddRow(`ab_"Mixed"`, "TABLE"))
	out, err := s.Complete(context.Background(), c, false)
	if err != nil || len(out.Candidates) != 1 || out.Candidates[0].Text != `"ab_""Mixed"""` {
		t.Fatalf("%+v %v", out, err)
	}
}
func TestCompletionDoesNotResolveDerivedOrAmbiguousAliasesAsTables(t *testing.T) {
	s, mock := mockSession(t)
	for _, text := range []string{`WITH c AS (SELECT 1 id FROM dual) SELECT c.| FROM c`, `SELECT a.| FROM x a JOIN y a ON 1=1`} {
		expectCompletionSchema(mock, "APP")
		out, err := s.Complete(context.Background(), completionContext(t, text), false)
		if err != nil || len(out.Candidates) != 0 || len(out.Warnings) == 0 {
			t.Fatalf("%+v %v", out, err)
		}
	}
}
func TestCompletionQualifiesJoinedColumnsAndKeepsPartialWarnings(t *testing.T) {
	s, mock := mockSession(t)
	expectCompletionSchema(mock, "APP")
	mock.ExpectQuery("FROM all_tab_columns").WithArgs("APP", "X", completionColumnLimit+1).WillReturnRows(sqlmock.NewRows([]string{"NAME", "TYPE"}).AddRow("TITLE", "VARCHAR2"))
	mock.ExpectQuery("FROM all_tab_columns").WithArgs("OTHER", "Y", completionColumnLimit+1).WillReturnError(errors.New("ORA-01031: insufficient privileges"))
	mock.ExpectQuery("FROM all_objects").WithArgs("APP", "TI%", completionObjectLimit+1).WillReturnRows(sqlmock.NewRows([]string{"NAME", "TYPE"}).AddRow("TITLES", "TABLE"))
	c := completionContext(t, `SELECT ti| FROM x a JOIN other.y b ON a.id=b.id`)
	out, err := s.Complete(context.Background(), c, false)
	if err != nil || len(out.Candidates) != 2 || out.Candidates[0].Text != `"A"."TITLE"` || out.Candidates[1].Text != `"TITLES"` || len(out.Warnings) != 1 {
		t.Fatalf("%+v %v", out, err)
	}
}
func TestCompletionExpiresCacheAndCapsCandidates(t *testing.T) {
	s, mock := mockSession(t)
	key := completionKey{Kind: "columns", Owner: "APP", Name: "T"}
	s.completionCache = map[completionKey]completionEntry{key: {At: time.Now().Add(-2 * completionCacheTTL), Names: []completionName{{Name: "OLD"}}}}
	expectCompletionSchema(mock, "APP")
	rows := sqlmock.NewRows([]string{"NAME", "TYPE"})
	for i := 0; i < MaxCompletionCandidates+10; i++ {
		rows.AddRow(strings.Repeat("A", i+1), "VARCHAR2")
	}
	mock.ExpectQuery("FROM all_tab_columns").WithArgs("APP", "T", completionColumnLimit+1).WillReturnRows(rows).RowsWillBeClosed()
	out, err := s.Complete(context.Background(), completionContext(t, `SELECT t.| FROM t`), false)
	if err != nil || len(out.Candidates) != MaxCompletionCandidates || !out.Truncated || s.completionCache[key].Names[0].Name == "OLD" {
		t.Fatalf("count %d truncated %v err %v", len(out.Candidates), out.Truncated, err)
	}
}
