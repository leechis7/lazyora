package oracle

import "testing"

func TestExecutionRisk(t *testing.T) {
	for _, tc := range []struct {
		sql  string
		want SQLRisk
	}{
		{"DROP TABLE demo", RiskDDL},
		{"/* explain */ -- more\n TrUnCaTe TABLE demo;", RiskDDL},
		{"CREATE TABLE demo(id NUMBER)", RiskDDL},
		{"ALTER SESSION SET CURRENT_SCHEMA=app", RiskDDL},
		{"GRANT SELECT ON demo TO app", RiskDDL},
		{"COMMENT ON TABLE demo IS 'where'", RiskDDL},
		{"ANALYZE TABLE demo COMPUTE STATISTICS", RiskDDL},
		{"PURGE RECYCLEBIN", RiskDDL},
		{"UPDATE demo SET name='x'", RiskUnfilteredDML},
		{"DELETE FROM demo;", RiskUnfilteredDML},
		{"UPDATE demo SET name='WHERE id=1'", RiskUnfilteredDML},
		{"UPDATE demo SET name='it''s WHERE'", RiskUnfilteredDML},
		{"UPDATE demo SET name=q'[a 'WHERE' ]'", RiskUnfilteredDML},
		{"UPDATE demo SET name=NQ'{WHERE}'", RiskUnfilteredDML},
		{"UPDATE demo SET name=q'!WHERE!'", RiskUnfilteredDML},
		{"UPDATE demo SET name=q'한WHERE한'", RiskUnfilteredDML},
		{"UPDATE demo SET \"WHERE\"=1", RiskUnfilteredDML},
		{"UPDATE \"WHERE\" SET id=1", RiskUnfilteredDML},
		{"UPDATE demo SET name=:where", RiskUnfilteredDML},
		{"DELETE FROM demo /* WHERE id=1 */", RiskUnfilteredDML},
		{"DELETE FROM demo -- WHERE id=1\n", RiskUnfilteredDML},
		{"UPDATE demo SET id=(SELECT id FROM src WHERE id=1)", RiskUnfilteredDML},
		{"DELETE FROM (SELECT * FROM demo WHERE id=1)", RiskUnfilteredDML},
		{"UPDATE demo SET id=2 WHERE id=1", RiskNone},
		{"DELETE FROM demo WHERE id IN (SELECT id FROM src WHERE id=1)", RiskNone},
		{"UPDATE demo SET name=q'[WHERE]' WHERE id=1", RiskNone},
		{"UPDATE demo SET id=2 WHERE 1=1", RiskNone}, // Predicate truth is outside this check.
		{"UPDATE demo SET name='unterminated", RiskUncertainDML},
		{"DELETE FROM demo /* unterminated", RiskUncertainDML},
		{"UPDATE demo SET id=(SELECT id FROM src", RiskUncertainDML},
		{"UPDATE demo SET id=2; DELETE FROM demo WHERE id=1", RiskUncertainDML},
		{"SELECT 'DROP TABLE demo' FROM dual", RiskNone},
		{"SELECT * FROM demo FOR UPDATE", RiskNone},
		{"WITH x AS (SELECT * FROM demo) SELECT * FROM x", RiskNone},
		{"INSERT INTO demo VALUES(1)", RiskNone},
		{"BEGIN UPDATE demo SET id=2; END;", RiskNone},
		{"EXPLAIN PLAN FOR DELETE FROM demo", RiskNone},
		{"-- comment only", RiskNone},
		{"", RiskNone},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			if got := ExecutionRisk(tc.sql); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
