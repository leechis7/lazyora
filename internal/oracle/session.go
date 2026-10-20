package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const MaxRows = 1000

type Result struct {
	Output    Output
	Query     *QuerySpec
	Values    [][]Cell
	Columns   []string
	Rows      [][]string
	Affected  int64
	Truncated bool
	Elapsed   time.Duration
}

// Cell retains NULL and database type separately from the display text.
type Cell struct {
	Text, Type string
	Null       bool
}

// Session is used only by the UI's single worker. A pinned connection preserves
// ALTER SESSION state, and the explicit transaction prevents DML auto-commit.
type Session struct {
	outputEnabled, outputReset bool

	db        *sql.DB
	conn      *sql.Conn
	tx        *sql.Tx
	pending   bool
	work      PendingWork
	finishErr error
}

func Open(ctx context.Context, config Config) (*Session, error) {
	dsn, err := config.DSN()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	conn, err := db.Conn(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}
	if err = conn.PingContext(ctx); err != nil {
		conn.Close()
		db.Close()
		return nil, err
	}
	return &Session{db: db, conn: conn}, nil
}

func (s *Session) Execute(ctx context.Context, text string, bindings ...map[string]BindValue) (result Result, err error) {
	start := time.Now()
	result = Result{Affected: -1}
	if s.finishErr != nil {
		return result, s.finishErr
	}
	statement, query, err := Prepare(text)
	if err != nil {
		return result, err
	}
	word, _ := leadingWord(statement)
	if word == "COMMIT" || word == "ROLLBACK" {
		return result, errors.New("use the session Commit or Rollback actions")
	}
	originalStatement := statement
	var values map[string]BindValue
	if len(bindings) > 0 {
		values = bindings[0]
	}
	statement, args, err := BindArguments(statement, values)
	if err != nil {
		return result, err
	}
	if err = s.resetOutput(ctx); err != nil {
		return result, err
	}
	if s.tx == nil {
		// Do not use the per-query context here: database/sql would roll the
		// entire transaction back when that context's deadline expires.
		s.tx, err = s.conn.BeginTx(context.Background(), nil)
		if err != nil {
			return result, err
		}
	}
	if s.outputEnabled {
		defer func() { result.Output = s.collectOutput() }()
	}
	if !query {
		// A failed/cancelled statement or PL/SQL block may have executed part
		// of its work. Keep a conservative warning until explicit resolution.
		s.pending = true
		var exec sql.Result
		exec, err = s.tx.ExecContext(ctx, statement, args...)
		if err == nil {
			if affected, rowsErr := exec.RowsAffected(); rowsErr == nil {
				result.Affected = affected
			}
		}
		result.Elapsed = time.Since(start)
		s.recordStatement(start, originalStatement, result.Affected, err)
		return result, err
	}
	rows, err := s.tx.QueryContext(ctx, statement, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	result.Columns, err = rows.Columns()
	if err != nil {
		return result, err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return result, err
	}
	result.Query, _ = NewQuerySpec(originalStatement, values)
	for rows.Next() {
		if len(result.Rows) == MaxRows {
			result.Truncated = true
			break
		}
		cells, scanErr := scanCells(rows, columnTypes)
		if scanErr != nil {
			return result, scanErr
		}
		row := make([]string, len(cells))
		for i, cell := range cells {
			row[i] = cell.Text
		}
		result.Rows = append(result.Rows, row)
		result.Values = append(result.Values, cells)
	}
	err = rows.Err()
	result.Elapsed = time.Since(start)
	return result, err
}

func formatValue(value any) string {
	switch v := value.(type) {
	case nil:
		return "NULL"
	case []byte:
		return fmt.Sprintf("%x", v)
	case time.Time:
		return v.Format("2006-01-02 15:04:05.999999999Z07:00")
	default:
		return fmt.Sprint(v)
	}
}

func (s *Session) Finish(commit bool) error {
	if s.tx == nil {
		if s.finishErr == nil {
			s.pending = false
			s.work = PendingWork{}
		}
		return s.finishErr
	}
	tx := s.tx
	s.tx = nil
	var err error
	if commit {
		err = tx.Commit()
	} else {
		err = tx.Rollback()
	}
	if err != nil {
		s.pending = true
		s.finishErr = fmt.Errorf("transaction outcome is uncertain; reconnect before executing more SQL: %w", err)
		return s.finishErr
	}
	s.pending = false
	s.work = PendingWork{}
	return nil
}

// PendingChanges reports statements that may have changed data and have not
// been resolved by this client. DDL/PLSQL may commit internally, so this is a
// conservative client-side warning, not a database transaction inventory.
func (s *Session) PendingChanges() bool { return s.pending }

// TransactionUncertain reports a failed transaction resolution. Repeating
// Commit/Rollback cannot establish its outcome; recovery needs explicit consent.
func (s *Session) TransactionUncertain() bool { return s.finishErr != nil }

func (s *Session) Close() {
	_ = s.Finish(false)
	_ = s.conn.Close()
	_ = s.db.Close()
}
