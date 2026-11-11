package oracle_test

import (
	"context"
	"testing"
	"time"

	"lazyora/internal/oracle"
)

func TestOracleActualPlanIntegration(t *testing.T) {
	s := workflowSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	execute := func(text string) oracle.Result {
		t.Helper()
		r, err := s.Execute(ctx, text)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	execute("CREATE TABLE lazyora_actual_probe (id NUMBER)")
	t.Cleanup(func() {
		_ = s.Finish(false)
		if _, err := s.Execute(context.Background(), "DROP TABLE lazyora_actual_probe PURGE"); err != nil {
			t.Error(err)
		}
	})
	if err := s.Finish(true); err != nil {
		t.Fatal(err)
	}
	measure := func(sql string, binds map[string]oracle.BindValue) oracle.Result {
		t.Helper()
		result, err := s.ExecuteWithActualPlan(ctx, sql, binds)
		if err != nil || result.ActualPlan == nil {
			t.Fatalf("measured execution: %v", err)
		}
		if result.ActualPlan.Err != nil || result.ActualPlan.SQLID == "" || len(result.ActualPlan.Steps) == 0 {
			t.Fatalf("actual plan: %+v", result.ActualPlan)
		}
		return result
	}
	insert := measure("INSERT INTO lazyora_actual_probe VALUES (:id)", map[string]oracle.BindValue{"ID": {Kind: "number", Text: "42"}})
	if insert.Affected != 1 || !s.PendingChanges() || len(s.PendingWork().Statements) != 1 {
		t.Fatal("measured DML was repeated or changed transaction tracking")
	}
	query := "SELECT /*+ GATHER_PLAN_STATISTICS */ id FROM lazyora_actual_probe WHERE id = :id"
	result := measure(query, map[string]oracle.BindValue{"ID": {Kind: "number", Text: "42"}})
	if len(result.Rows) != 1 || result.Rows[0][0] != "42" || result.Query == nil || result.Query.Text != query ||
		result.ActualPlan.SQLID == insert.ActualPlan.SQLID || result.ActualPlan.Partial {
		t.Fatal("measured query did not preserve SQL, result or identity")
	}
	hasStats := false
	for _, step := range result.ActualPlan.Steps {
		if step.Actual != nil && step.Actual.Rows == "1" && step.Actual.Starts == "1" && step.Actual.Buffers != "" {
			hasStats = true
			break
		}
	}
	if !hasStats {
		t.Fatal("actual row, start and buffer counts were not captured")
	}
	paged := measure("SELECT /*+ GATHER_PLAN_STATISTICS */ LEVEL FROM dual CONNECT BY LEVEL <= 1002", nil)
	if !paged.ActualPlan.Partial || paged.PageID == 0 {
		t.Fatal("partial statistics were not distinguished")
	}
	next, err := s.FetchPage(ctx, paged.PageID)
	if err != nil || len(next.Rows) != 2 || next.Rows[0][0] != "1001" {
		t.Fatalf("capture invalidated the result cursor: %v", err)
	}
	if err := s.Finish(false); err != nil {
		t.Fatal(err)
	}
	if r := execute("SELECT COUNT(*) FROM lazyora_actual_probe"); r.Rows[0][0] != "0" {
		t.Fatal("measured statement was committed")
	}
}
