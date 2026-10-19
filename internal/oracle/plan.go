package oracle

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type PlanStep struct {
	ID, Parent                                                    int
	HasParent                                                     bool
	Operation, Options, Owner, Object, Cost, Rows, Access, Filter string
}

func PlanSQL(text string) (string, error) {
	statements, err := Statements(text)
	if err != nil {
		return "", err
	}
	if len(statements) != 1 {
		return "", fmt.Errorf("select one SQL statement for EXPLAIN PLAN")
	}
	statement, _, err := Prepare(statements[0].Text)
	if err != nil {
		return "", err
	}
	word, _ := leadingWord(statement)
	switch word {
	case "SELECT", "WITH", "INSERT", "UPDATE", "DELETE", "MERGE":
		return statement, nil
	}
	return "", fmt.Errorf("EXPLAIN PLAN supports SELECT/WITH/INSERT/UPDATE/DELETE/MERGE here; PL/SQL and DDL are not executed")
}

// Explain only inserts/selects/deletes its own random statement_id in PLAN_TABLE.
// The existing transaction is never committed, rolled back or replaced.
func (s *Session) Explain(ctx context.Context, text string, bindings map[string]BindValue) (steps []PlanStep, err error) {
	if s.finishErr != nil {
		return nil, s.finishErr
	}
	statement, err := PlanSQL(text)
	if err != nil {
		return nil, err
	}
	statement, args, err := BindArguments(statement, bindings)
	if err != nil {
		return nil, err
	}
	if s.tx == nil {
		s.tx, err = s.conn.BeginTx(context.Background(), nil)
		if err != nil {
			return nil, err
		}
	}
	var random [10]byte
	if _, err = rand.Read(random[:]); err != nil {
		return nil, err
	}
	id := "LAZYORA_" + hex.EncodeToString(random[:])
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, cleanupErr := s.tx.ExecContext(cleanup, "DELETE FROM plan_table WHERE statement_id = :1", id)
		if cleanupErr != nil {
			if err == nil {
				err = fmt.Errorf("PLAN_TABLE cleanup failed: %w", cleanupErr)
			} else {
				err = fmt.Errorf("%w; PLAN_TABLE cleanup failed: %v", err, cleanupErr)
			}
		}
	}()
	if _, err = s.tx.ExecContext(ctx, "EXPLAIN PLAN SET STATEMENT_ID = '"+id+"' FOR "+statement, args...); err != nil {
		return nil, fmt.Errorf("EXPLAIN PLAN requires PLAN_TABLE and INSERT/SELECT/DELETE privileges: %w", err)
	}
	rows, err := s.tx.QueryContext(ctx, `SELECT id,parent_id,operation,options,object_owner,object_name,
TO_CHAR(cost),TO_CHAR(cardinality),access_predicates,filter_predicates
FROM plan_table WHERE statement_id = :1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		if len(steps) >= MaxRows {
			return nil, fmt.Errorf("execution plan exceeds %d steps", MaxRows)
		}
		var step PlanStep
		var parent sql.NullInt64
		var values [8]sql.NullString
		if err = rows.Scan(&step.ID, &parent, &values[0], &values[1], &values[2], &values[3], &values[4], &values[5], &values[6], &values[7]); err != nil {
			return nil, err
		}
		step.Parent, step.HasParent = int(parent.Int64), parent.Valid
		step.Operation, step.Options, step.Owner, step.Object = values[0].String, values[1].String, values[2].String, values[3].String
		step.Cost, step.Rows, step.Access, step.Filter = values[4].String, values[5].String, values[6].String, values[7].String
		steps = append(steps, step)
	}
	return steps, rows.Err()
}

func (p PlanStep) Description() string {
	object := p.Object
	if p.Owner != "" {
		object = p.Owner + "." + object
	}
	return fmt.Sprintf("%d · %s %s · %s\nCost: %s   Rows: %s\n\nAccess:\n%s\n\nFilter:\n%s", p.ID, p.Operation, p.Options, strings.TrimSuffix(object, "."), p.Cost, p.Rows, p.Access, p.Filter)
}
