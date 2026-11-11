package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

const (
	MaxDataCompareRows    = 5000
	MaxDataCompareColumns = 64
	MaxDataCompareBytes   = 16 << 20
	maxDataCompareKeys    = 16
)

type DataValue struct {
	Text string
	Null bool
}

type TableData struct {
	Object  Object
	Columns []Column
	Key     []int
	Rows    map[string][]DataValue
}

type DataDifference struct {
	Left, Right []DataValue
	Columns     []int // different column indexes; empty when a whole row is missing
}

type DataComparison struct {
	Rows                []DataDifference
	LeftOnly, RightOnly int
	Changed             int
}

func ParseDataCompareKeys(text string) ([]string, error) {
	tokens, err := sqlTokens(text)
	if err != nil || len(tokens)%2 != 1 || len(tokens)/2 >= maxDataCompareKeys {
		return nil, fmt.Errorf("enter 1–%d comma-separated column names", maxDataCompareKeys)
	}
	var keys []string
	for i, token := range tokens {
		if i%2 == 1 {
			if token.word != "," {
				return nil, errors.New("separate comparison key columns with commas")
			}
			continue
		}
		owner, name, err := ParseObjectReference(text[token.start:token.end])
		if err != nil || owner != "" || slices.Contains(keys, name) {
			return nil, errors.New("use distinct, unqualified key column names")
		}
		keys = append(keys, name)
	}
	return keys, nil
}

// ReadTableData reads one complete, bounded snapshot using the workspace's
// existing transaction, without modifying its SQL result, page, or edit state.
func (s *Session) ReadTableData(ctx context.Context, reference string, keys []string) (TableData, error) {
	if s.finishErr != nil {
		return TableData{}, s.finishErr
	}
	if len(keys) == 0 || len(keys) > maxDataCompareKeys {
		return TableData{}, fmt.Errorf("choose 1–%d key columns", maxDataCompareKeys)
	}
	object, err := s.FindObject(ctx, reference)
	if err != nil {
		return TableData{}, fmt.Errorf("resolve table %s: %w", reference, err)
	}
	if object.Kind != "TABLE" {
		return TableData{}, errors.New("choose a base table")
	}
	columns, err := s.Columns(ctx, object)
	if err != nil {
		return TableData{}, err
	}
	if len(columns) == 0 || len(columns) > MaxDataCompareColumns {
		return TableData{}, fmt.Errorf("the comparison supports 1–%d columns per table", MaxDataCompareColumns)
	}
	// Columns can appear in a different physical order on the other side.
	slices.SortFunc(columns, func(a, b Column) int { return strings.Compare(a.Name, b.Name) })
	indexes := make(map[string]int, len(columns))
	out := TableData{Object: object, Columns: columns, Rows: make(map[string][]DataValue)}
	for i, c := range columns {
		indexes[c.Name] = i
	}
	for _, key := range keys {
		index, exists := indexes[key]
		if !exists {
			return TableData{}, fmt.Errorf("key column %s was not found", key)
		}
		if slices.Contains(out.Key, index) {
			return TableData{}, fmt.Errorf("key column %s was specified more than once", key)
		}
		out.Key = append(out.Key, index)
	}

	exprs := make([]string, len(columns))
	dates := make([]sql.NullTime, len(columns))
	dateColumn := make([]bool, len(columns))
	values := make([]sql.NullString, len(columns))
	args := make([]any, len(columns))
	for i, c := range columns {
		kind := normalizeEditType(c.DataType)
		if !editableType(kind) {
			return TableData{}, fmt.Errorf("unsupported comparison column %s (%s)", c.Name, c.DataType)
		}
		column := quoteIdentifier(c.Name)
		exprs[i] = column
		args[i] = &values[i]
		switch kind {
		case "NUMBER":
			// Oracle's TM9 returns a canonical decimal or scientific representation
			// without narrowing NUMBER to float64 or inheriting decimal separators.
			exprs[i] = "TO_CHAR(" + column + ", 'TM9', 'NLS_NUMERIC_CHARACTERS=''.,''')"
		case "RAW":
			exprs[i] = "RAWTOHEX(" + column + ")"
		case "DATE", "TIMESTAMP":
			dateColumn[i] = true
			args[i] = &dates[i]
		}
	}
	query := fmt.Sprintf("SELECT %s FROM %s WHERE ROWNUM <= %d", strings.Join(exprs, ", "), QualifiedObject(object), MaxDataCompareRows+1)
	rows, err := s.dictionaryQuery(ctx, query)
	if err != nil {
		return TableData{}, err
	}
	defer rows.Close()
	size := 0
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return TableData{}, err
		}
		rowNumber := len(out.Rows) + 1
		if rowNumber > MaxDataCompareRows {
			return TableData{}, fmt.Errorf("table exceeds the %d-row comparison limit", MaxDataCompareRows)
		}
		if err := rows.Scan(args...); err != nil {
			return TableData{}, err
		}
		cells := make([]DataValue, len(columns))
		for i := range cells {
			if dateColumn[i] {
				d := dates[i]
				if d.Valid {
					cells[i].Text = d.Time.Format("2006-01-02 15:04:05.999999999")
				}
				cells[i].Null = !d.Valid
			} else {
				cells[i] = DataValue{Text: values[i].String, Null: !values[i].Valid}
			}
			size += len(cells[i].Text) + 24 // approximate per-cell storage overhead
		}
		var key strings.Builder
		for _, i := range out.Key {
			value := cells[i]
			if value.Null {
				return TableData{}, fmt.Errorf("NULL in key column %s (read row %d)", columns[i].Name, rowNumber)
			}
			// Length prefixes distinguish values containing any separators.
			key.WriteString(strconv.Itoa(len(value.Text)))
			key.WriteByte(':')
			key.WriteString(value.Text)
		}
		size += key.Len()
		if size > MaxDataCompareBytes {
			return TableData{}, fmt.Errorf("table exceeds the %d MiB comparison limit", MaxDataCompareBytes>>20)
		}
		id := key.String()
		if _, exists := out.Rows[id]; exists {
			return TableData{}, fmt.Errorf("duplicate comparison key (read row %d)", rowNumber)
		}
		out.Rows[id] = cells
	}
	if err := rows.Err(); err != nil {
		return TableData{}, err
	}
	return out, ctx.Err()
}

func CompareTableData(left, right TableData) (DataComparison, error) {
	if !slices.Equal(left.Key, right.Key) || len(left.Columns) != len(right.Columns) {
		return DataComparison{}, errors.New("the tables have different columns or comparison keys; compare table structures first")
	}
	for i, c := range left.Columns {
		other := right.Columns[i]
		if c.Name != other.Name || c.DataType != other.DataType {
			return DataComparison{}, fmt.Errorf("incompatible column: %s (%s) / %s (%s)", c.Name, c.DataType, other.Name, other.DataType)
		}
	}
	ids := make([]string, 0, len(left.Rows)+len(right.Rows))
	for id := range left.Rows {
		ids = append(ids, id)
	}
	for id := range right.Rows {
		if _, found := left.Rows[id]; !found {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	var out DataComparison
	for _, id := range ids {
		a, b := left.Rows[id], right.Rows[id]
		diff := DataDifference{Left: a, Right: b}
		switch {
		case a == nil:
			out.RightOnly++
		case b == nil:
			out.LeftOnly++
		default:
			for i, value := range a {
				if value != b[i] {
					diff.Columns = append(diff.Columns, i)
				}
			}
			if len(diff.Columns) == 0 {
				continue
			}
			out.Changed++
		}
		out.Rows = append(out.Rows, diff)
	}
	return out, nil
}
