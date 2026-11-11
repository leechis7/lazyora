package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// An omitted field uses the column default. An explicit Null binds SQL NULL.
type EditInsert struct {
	ID     uint64
	Fields map[int]EditInsertField
}

type EditInsertField struct {
	Text string
	Null bool
}

type EditDelete struct {
	ID  uint64
	Row int // index in the verified snapshot, never the displayed order
}

type editInsertSQL struct {
	statement string
	args      []any
}

func (r Result) CanInsertRow() bool {
	return r.EditID != 0 && r.EditObject.Kind == "TABLE" && len(r.EditColumnTypes) == len(r.Columns) && len(r.Columns) != 0
}

// ApplyGridChanges applies all row and cell drafts together under one savepoint.
func (s *Session) ApplyGridChanges(ctx context.Context, changes []EditChange, deletes []EditDelete, inserts []EditInsert) error {
	return s.applyEdits(ctx, changes, deletes, inserts, true)
}

func (snap *editSnapshot) prepareInserts(inserts []EditInsert) ([]editInsertSQL, error) {
	var prepared []editInsertSQL
	table := QualifiedObject(snap.object)
	for n, insert := range inserts {
		if insert.ID != snap.id {
			return nil, errors.New("editable result expired; reload table data")
		}
		for i := range insert.Fields {
			if i < 0 || i >= len(snap.columns) || snap.columns[i].Name == "" {
				return nil, errors.New("insert field is not a verified original column")
			}
		}
		var out editInsertSQL
		var cols, expr []string
		defaultColumn := ""
		for i, c := range snap.columns {
			if c.Name == "" {
				continue
			}
			if defaultColumn == "" {
				defaultColumn = quoteIdentifier(c.Name)
			}
			f, ok := insert.Fields[i]
			if !ok {
				continue
			}
			null := f.Null || f.Text == ""
			if null && !c.Nullable {
				return nil, fmt.Errorf("new row %d, %s: column does not allow NULL", n+1, c.Name)
			}
			value, err := editValue(c.DataType, f.Text, null)
			if err != nil {
				return nil, fmt.Errorf("new row %d, %s: %w", n+1, c.Name, err)
			}
			name := fmt.Sprintf("value_%d", i)
			cols = append(cols, quoteIdentifier(c.Name))
			expr = append(expr, editBind(":"+name, c.DataType))
			out.args = append(out.args, sql.Named(name, value))
		}
		if len(cols) == 0 {
			if defaultColumn == "" {
				return nil, errors.New("insert needs at least one verified original column")
			}
			cols, expr = []string{defaultColumn}, []string{"DEFAULT"}
		}
		out.statement = "INSERT INTO " + table + " (" + strings.Join(cols, ", ") + ") VALUES (" + strings.Join(expr, ", ") + ")"
		prepared = append(prepared, out)
	}
	return prepared, nil
}
