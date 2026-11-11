package oracle_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

func TestOracleDataCompareIntegration(t *testing.T) {
	left, right := workflowSession(t), workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	exec := func(s *oracle.Session, sql string) {
		t.Helper()
		if _, err := s.Execute(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	const a = "lazyora_data_cmp_a"
	const b = "lazyora_data_cmp_b"
	exec(right, "CREATE TABLE "+a+" (k VARCHAR2(10), n VARCHAR2(10), v NUMBER, txt NVARCHAR2(20), dt DATE, ts TIMESTAMP(9), r RAW(2))")
	exec(right, "CREATE TABLE "+b+" (r RAW(2), ts TIMESTAMP(9), dt DATE, txt NVARCHAR2(20), v NUMBER, n VARCHAR2(10), k VARCHAR2(10))")
	t.Cleanup(func() {
		_ = left.Finish(false)
		_ = right.Finish(false)
		for _, table := range []string{a, b} {
			if _, err := right.Execute(context.Background(), "DROP TABLE "+table+" PURGE"); err != nil {
				t.Error(err)
			}
		}
	})
	insert := func(s *oracle.Session, table, k, n, num, value string) {
		t.Helper()
		exec(s, "INSERT INTO "+table+" (k,n,v,txt,dt,ts,r) VALUES ("+k+","+n+","+num+","+value+
			",DATE '2026-11-12',TIMESTAMP '2026-11-12 12:03:04.123456789',HEXTORAW('00AF'))")
	}
	insert(left, a, "'a'", "'bc'", "12345678901234567890123456789012345678", "NULL")
	insert(right, b, "'a'", "'bc'", "12345678901234567890123456789012345679", "N'NULL'")
	insert(left, a, "'ab'", "'c'", "3.25", "N'한글'")
	insert(right, b, "'z'", "'z'", "3.25", "N'한글'")
	insert(left, a, "'same'", "'key'", "0.123", "N'같음'")
	insert(right, b, "'same'", "'key'", "0.123", "N'같음'")
	exec(right, "ALTER SESSION SET NLS_NUMERIC_CHARACTERS = ',.'")
	page, err := left.ExecutePaged(ctx, "SELECT LEVEL FROM dual CONNECT BY LEVEL <= 1002")
	if err != nil || page.PageID == 0 {
		t.Fatalf("setup live cursor: %v", err)
	}

	keys := []string{"K", "N"}
	read := func(s *oracle.Session, table string) oracle.TableData {
		t.Helper()
		data, err := s.ReadTableData(ctx, table, keys)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	first, second := read(left, a), read(right, b)
	diff, err := oracle.CompareTableData(first, second)
	if err != nil || diff.LeftOnly != 1 || diff.RightOnly != 1 || diff.Changed != 1 {
		t.Fatalf("comparison = %+v, err = %v", diff, err)
	}
	for _, row := range diff.Rows {
		if row.Left != nil && row.Right != nil {
			var names []string
			for _, index := range row.Columns {
				names = append(names, first.Columns[index].Name)
			}
			if !slices.Equal(names, []string{"TXT", "V"}) || !row.Left[row.Columns[0]].Null || row.Right[row.Columns[0]].Text != "NULL" {
				t.Fatalf("incorrect type/value comparison: %v, %+v", names, row)
			}
		}
	}
	if !left.PendingChanges() || !right.PendingChanges() || left.PageID() != page.PageID {
		t.Fatal("comparison did not preserve pending work or the open result page")
	}
	if next, err := left.FetchPage(ctx, page.PageID); err != nil || len(next.Rows) != 2 {
		t.Fatalf("could not resume original query cursor: %v", err)
	}
	if _, err := left.ReadTableData(ctx, a, []string{"NO_COLUMN"}); err == nil {
		t.Fatal("missing comparison key accepted")
	}
	insert(left, a, "'a'", "'bc'", "1", "NULL")
	if _, err := left.ReadTableData(ctx, a, keys); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate key accepted: %v", err)
	}
	insert(right, b, "NULL", "'nullkey'", "1", "NULL")
	if _, err := right.ReadTableData(ctx, b, keys); err == nil || !strings.Contains(err.Error(), "NULL in key") {
		t.Fatalf("NULL key accepted: %v", err)
	}
	if err := left.Finish(false); err != nil {
		t.Fatal(err)
	}
	exec(left, "INSERT INTO "+a+" (k,n) SELECT TO_CHAR(LEVEL), 'x' FROM dual CONNECT BY LEVEL <= 5001")
	if data, err := left.ReadTableData(ctx, a, keys); err == nil || data.Rows != nil || !strings.Contains(err.Error(), "5"+"000-row") {
		t.Fatalf("over-limit input yielded a partial comparison: %v", err)
	}
	stopped, stop := context.WithCancel(ctx)
	stop()
	if _, err := left.ReadTableData(stopped, a, keys); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled read did not stop: %v", err)
	}
}
