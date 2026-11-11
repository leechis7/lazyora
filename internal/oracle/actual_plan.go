package oracle

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

type PlanStatistics struct {
	Rows, Starts, Buffers, Reads, ElapsedMicros string
}

type ActualPlan struct {
	SQLID   string
	Child   int
	Steps   []PlanStep
	Partial bool
	Err     error
}

var (
	ErrActualPlanPrivilege   = errors.New("cannot read V$SQL or V$SQL_PLAN_STATISTICS_ALL")
	ErrActualPlanUnavailable = errors.New("cursor or execution plan no longer available")
	ErrActualPlanAmbiguous   = errors.New("cursor did not identify exactly one execution")
)

// ExecuteWithActualPlan runs the statement once. The opt-in diagnostic tag
// gives this execution a dedicated cursor while leaving SQL history and replay
// text unmodified. A plan error is separate from the statement outcome.
func (s *Session) ExecuteWithActualPlan(ctx context.Context, text string, bindings map[string]BindValue) (Result, error) {
	statement, err := PlanSQL(text)
	if err != nil {
		return Result{}, err
	}
	var nonce [12]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return Result{}, err
	}
	tag := "/* lazyora-plan-" + hex.EncodeToString(nonce[:]) + " */ "
	result, err := s.execute(ctx, statement, true, tag, bindings)
	if err != nil {
		return result, err
	}
	plan, err := s.readActualPlan(ctx, tag)
	plan.Partial = result.Truncated
	plan.Err = err
	result.ActualPlan = &plan
	return result, nil
}

func (s *Session) readActualPlan(ctx context.Context, tag string) (plan ActualPlan, err error) {
	defer func() {
		if err != nil && (strings.Contains(err.Error(), "ORA-00942") || strings.Contains(err.Error(), "ORA-01031")) {
			err = fmt.Errorf("%w: %v", ErrActualPlanPrivilege, err)
		}
	}()
	cursors, err := s.dictionaryQuery(ctx, `SELECT sql_id, child_number, NVL(executions, 0)
FROM v$sql WHERE sql_text LIKE :1 AND ROWNUM <= 2`, tag+"%")
	if err != nil {
		return plan, err
	}
	defer cursors.Close()
	if !cursors.Next() {
		if err := cursors.Err(); err != nil {
			return plan, err
		}
		return plan, ErrActualPlanUnavailable
	}
	var executions int
	if err := cursors.Scan(&plan.SQLID, &plan.Child, &executions); err != nil {
		return plan, err
	}
	if cursors.Next() || executions != 1 {
		return plan, ErrActualPlanAmbiguous
	}
	if err := cursors.Err(); err != nil {
		return plan, err
	}
	cursors.Close()
	rows, err := s.dictionaryQuery(ctx, `SELECT id, parent_id, operation, options, object_owner,
object_name, TO_CHAR(cost), TO_CHAR(cardinality), access_predicates, filter_predicates,
TO_CHAR(last_output_rows), TO_CHAR(last_starts),
TO_CHAR(CASE WHEN last_cr_buffer_gets IS NOT NULL OR last_cu_buffer_gets IS NOT NULL
  THEN NVL(last_cr_buffer_gets, 0) + NVL(last_cu_buffer_gets, 0) END),
TO_CHAR(last_disk_reads), TO_CHAR(last_elapsed_time)
FROM v$sql_plan_statistics_all
WHERE sql_id = :1 AND child_number = :2
ORDER BY id`, plan.SQLID, plan.Child)
	if err != nil {
		return plan, err
	}
	defer rows.Close()
	for rows.Next() {
		if len(plan.Steps) >= MaxRows {
			return plan, fmt.Errorf("actual execution plan exceeds %d steps", MaxRows)
		}
		var step PlanStep
		var parent sql.NullInt64
		var v [13]sql.NullString
		if err := rows.Scan(&step.ID, &parent, &v[0], &v[1], &v[2], &v[3], &v[4], &v[5], &v[6], &v[7],
			&v[8], &v[9], &v[10], &v[11], &v[12]); err != nil {
			return plan, err
		}
		step.Parent, step.HasParent = int(parent.Int64), parent.Valid
		step.Operation, step.Options, step.Owner, step.Object = v[0].String, v[1].String, v[2].String, v[3].String
		step.Cost, step.Rows, step.Access, step.Filter = v[4].String, v[5].String, v[6].String, v[7].String
		if v[8].Valid || v[9].Valid || v[10].Valid || v[11].Valid || v[12].Valid {
			step.Actual = &PlanStatistics{Rows: v[8].String, Starts: v[9].String, Buffers: v[10].String, Reads: v[11].String, ElapsedMicros: v[12].String}
		}
		plan.Steps = append(plan.Steps, step)
	}
	if err := rows.Err(); err != nil {
		return plan, err
	}
	if len(plan.Steps) == 0 {
		return plan, ErrActualPlanUnavailable
	}
	return plan, nil
}
