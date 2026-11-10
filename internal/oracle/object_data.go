package oracle

import (
	"context"
	"fmt"
)

const MaxObjectPreviewRows = 1000
const maxObjectPreviewBytes = 8 << 20

// ObjectPreview reads from the current session without replacing the SQL
// editor's cursor, edit snapshot, LOB references, output, or transaction.
func (s *Session) ObjectPreview(ctx context.Context, object Object, limit int) (Result, error) {
	var result Result
	if s.finishErr != nil {
		return result, s.finishErr
	}
	if !object.HasColumns() || object.Owner == "" || object.Name == "" || limit < 1 || limit > MaxObjectPreviewRows {
		return result, fmt.Errorf("invalid object or preview row limit")
	}
	return s.readObjectPreview(ctx, "SELECT * FROM "+QualifiedObject(object)+" WHERE ROWNUM <= :1", limit, []any{limit + 1})
}

func (s *Session) readObjectPreview(ctx context.Context, query string, limit int, args []any) (Result, error) {
	var result Result
	rows, err := s.dictionaryQuery(ctx, query, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	result.Columns, err = rows.Columns()
	if err != nil {
		return result, err
	}
	types, err := rows.ColumnTypes()
	if err != nil {
		return result, err
	}
	stop, err := s.pageFetchContext(ctx)
	if err != nil {
		return result, err
	}
	defer stop()
	ctx = context.WithValue(ctx, fetchWatchKey{}, true)
	readLOB := func(ref lobReference) (Cell, error) {
		return s.scanLOBValue(ctx, ref, false, true)
	}
	size := 0
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		if len(result.Values) >= limit {
			result.Truncated = true
			break
		}
		cells, err := scanCellValues(rows, types, readLOB)
		if err != nil {
			return Result{}, err
		}
		for _, cell := range cells {
			size += len(cell.Text) + len(cell.Type) + 96
		}
		if size > maxObjectPreviewBytes {
			result.Truncated, result.PageLimited = true, true
			break
		}
		result.Values = append(result.Values, cells)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	return result, rows.Err()
}
