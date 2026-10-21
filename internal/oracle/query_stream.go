package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"sync"
	"time"
)

// QuerySpec is an in-memory replay recipe, never a saved credential or history
// record. It belongs to the connection which produced the visible result.
type QuerySpec struct {
	Text     string
	Bindings map[string]BindValue
}

func NewQuerySpec(text string, bindings map[string]BindValue) (*QuerySpec, error) {
	statements, err := Statements(text)
	if err != nil {
		return nil, err
	}
	if len(statements) != 1 {
		return nil, fmt.Errorf("export requires one SELECT statement")
	}
	statement, query, err := Prepare(statements[0].Text)
	if err != nil {
		return nil, err
	}
	if !query {
		return nil, fmt.Errorf("only SELECT results can be queried for export")
	}
	tokens, _ := sqlTokens(statement)
	for i := 1; i < len(tokens); i++ {
		if tokens[i-1].word == "FOR" && tokens[i].word == "UPDATE" {
			return nil, fmt.Errorf("SELECT FOR UPDATE cannot be replayed for export")
		}
	}
	names, err := BindNames(statement)
	if err != nil {
		return nil, err
	}
	spec := &QuerySpec{Text: statement, Bindings: make(map[string]BindValue, len(names))}
	for _, name := range names {
		value, ok := bindings[name]
		if !ok {
			return nil, fmt.Errorf("missing IN bind :%s", name)
		}
		spec.Bindings[name] = value
	}
	if _, _, err = BindArguments(statement, spec.Bindings); err != nil {
		return nil, err
	}
	return spec, nil
}

// QueryRows holds a single live cursor. Call Close before running another job.
// It retains only the current row; callers must not accumulate rows for export.
type QueryRows struct {
	owner    *Session
	stop     func()
	stopOnce sync.Once
	Columns  []string
	rows     *sql.Rows
	types    []*sql.ColumnType
	ctx      context.Context
}

func (s *Session) OpenQuery(ctx context.Context, spec *QuerySpec) (*QueryRows, error) {
	s.ClosePage()
	s.clearLOBs()
	if s.finishErr != nil {
		return nil, s.finishErr
	}
	if spec == nil {
		return nil, fmt.Errorf("original query is no longer available")
	}
	validated, err := NewQuerySpec(spec.Text, spec.Bindings)
	if err != nil {
		return nil, err
	}
	statement, args, err := BindArguments(validated.Text, validated.Bindings)
	if err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if s.tx == nil {
		// The export's cancellation must never resolve the shared transaction.
		s.tx, err = s.conn.BeginTx(context.Background(), nil)
		if err != nil {
			return nil, err
		}
	}
	rows, err := s.tx.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, err
	}
	columns, err := rows.Columns()
	if err != nil {
		rows.Close()
		return nil, err
	}
	types, err := rows.ColumnTypes()
	if err != nil {
		rows.Close()
		return nil, err
	}
	stop, err := s.pageFetchContext(ctx)
	if err != nil {
		rows.Close()
		return nil, err
	}
	return &QueryRows{Columns: columns, rows: rows, types: types, ctx: context.WithValue(ctx, fetchWatchKey{}, true), owner: s, stop: stop}, nil
}

func (r *QueryRows) Next() ([]Cell, error) {
	if err := r.ctx.Err(); err != nil {
		return nil, err
	}
	if !r.rows.Next() {
		if err := r.rows.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	return r.owner.scanCells(r.ctx, r.rows, r.types, false)
}
func (r *QueryRows) Close() error {
	r.stopOnce.Do(func() {
		if r.stop != nil {
			r.stop()
		}
	})
	return r.rows.Close()
}

func (s *Session) scanCells(ctx context.Context, rows *sql.Rows, columnTypes []*sql.ColumnType, retain bool) ([]Cell, error) {
	values := make([]any, len(columnTypes))
	pointers := make([]any, len(values))
	for i := range values {
		pointers[i] = &values[i]
	}
	if err := rows.Scan(pointers...); err != nil {
		return nil, err
	}
	cells := make([]Cell, len(values))
	for i, value := range values {
		if reference, ok := lobValue(value); ok {
			cell, err := s.scanLOB(ctx, reference, retain)
			if err != nil {
				return nil, err
			}
			cells[i] = cell
			continue
		}
		kind := columnTypes[i].DatabaseTypeName()
		// go-ora can omit DatabaseTypeName and return Oracle NUMBER as text.
		// Precision/scale metadata identifies numeric columns without guessing
		// from text values (which would corrupt numeric-looking VARCHAR2 data).
		if kind == "" {
			if _, _, numeric := columnTypes[i].DecimalSize(); numeric {
				kind = "NUMBER"
			}
		}
		if kind == "" {
			switch value.(type) {
			case int64, int32, int, float64, float32:
				kind = "NUMBER"
			case time.Time:
				kind = "TIMESTAMP"
			case []byte:
				kind = "RAW"
			default:
				kind = "VARCHAR2"
			}
		}
		cells[i] = Cell{Text: formatValue(value), Type: kind, Null: value == nil}
		cells[i].Size = int64(len(cells[i].Text))
		if kind == "RAW" {
			cells[i].Size /= 2
		}
	}
	return cells, nil
}
