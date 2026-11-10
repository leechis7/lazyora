package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type ForeignKeyColumn struct {
	Source, Target string
}

type ForeignKeyLink struct {
	Name     string
	Target   Object
	ToParent bool
	Columns  []ForeignKeyColumn
}

type ForeignKeyLinks struct {
	Links   []ForeignKeyLink
	Limited bool
}

const foreignKeyColumns = `SELECT c.owner AS child_owner, c.constraint_name, c.table_name AS child_table,
    p.owner AS parent_owner, p.table_name AS parent_table,
    cc.position, cc.column_name AS child_column, pc.column_name AS parent_column
FROM all_constraints c
JOIN all_constraints p ON p.owner = c.r_owner AND p.constraint_name = c.r_constraint_name
JOIN all_cons_columns cc ON cc.owner = c.owner AND cc.constraint_name = c.constraint_name
JOIN all_cons_columns pc ON pc.owner = p.owner AND pc.constraint_name = p.constraint_name AND pc.position = cc.position
WHERE c.constraint_type = 'R' AND c.status = 'ENABLED' AND p.status = 'ENABLED'
AND %s
ORDER BY c.owner, c.constraint_name, cc.position`

// ForeignKeys returns relationships visible to the current user, including
// incoming, outgoing and self-referencing foreign keys.
func (s *Session) ForeignKeys(ctx context.Context, object Object) (ForeignKeyLinks, error) {
	var result ForeignKeyLinks
	if s.finishErr != nil {
		return result, s.finishErr
	}
	if object.Kind != "TABLE" || object.Owner == "" || object.Name == "" {
		return result, errors.New("foreign-key navigation requires a table")
	}
	for _, toParent := range []bool{true, false} {
		predicate := "p.owner = :1 AND p.table_name = :2"
		if toParent {
			predicate = "c.owner = :1 AND c.table_name = :2"
		}
		query := "SELECT * FROM (" + fmt.Sprintf(foreignKeyColumns, predicate) + ") WHERE ROWNUM <= 2049"
		var links []ForeignKeyLink
		var lastOwner, lastName string
		limited := false
		err := s.definitionQuery(ctx, query, []any{object.Owner, object.Name}, definitionColumns, &limited, func(rows *sql.Rows) error {
			var owner, name, table, parentOwner, parentTable string
			var position int
			var col ForeignKeyColumn
			if err := rows.Scan(&owner, &name, &table, &parentOwner, &parentTable, &position, &col.Source, &col.Target); err != nil {
				return err
			}
			if owner != lastOwner || name != lastName {
				lastOwner, lastName = owner, name
				target := Object{Owner: parentOwner, Name: parentTable, Kind: "TABLE"}
				if !toParent {
					target.Owner, target.Name = owner, table
				}
				links = append(links, ForeignKeyLink{Name: name, Target: target, ToParent: toParent})
			}
			link := &links[len(links)-1]
			if position != len(link.Columns)+1 {
				return errors.New("incomplete foreign-key column metadata")
			}
			if !toParent {
				col.Source, col.Target = col.Target, col.Source
			}
			link.Columns = append(link.Columns, col)
			return nil
		})
		if err != nil {
			return ForeignKeyLinks{}, err
		}
		if limited && len(links) > 0 {
			// The final relationship may be split at the dictionary row limit.
			links = links[:len(links)-1]
		}
		result.Limited = result.Limited || limited
		result.Links = append(result.Links, links...)
	}
	return result, nil
}

var ErrForeignKeyNull = errors.New("NULL foreign-key components have no matching row")

// RelatedRows reads a bounded preview without changing the SQL editor's state.
func (s *Session) RelatedRows(ctx context.Context, link ForeignKeyLink, headers []string, row []Cell, limit int) (Result, error) {
	if s.finishErr != nil {
		return Result{}, s.finishErr
	}
	if link.Target.Kind != "TABLE" || link.Target.Owner == "" || link.Target.Name == "" ||
		len(link.Columns) == 0 || len(link.Columns) > MaxBindVariables || len(headers) != len(row) ||
		limit < 1 || limit > MaxObjectPreviewRows {
		return Result{}, errors.New("invalid foreign-key preview")
	}
	predicates := make([]string, 0, len(link.Columns)+1)
	args := make([]any, 0, len(link.Columns)+1)
	for _, mapping := range link.Columns {
		index := -1
		for i, name := range headers {
			if name == mapping.Source {
				if index != -1 {
					return Result{}, fmt.Errorf("ambiguous key column %q", name)
				}
				index = i
			}
		}
		if index < 0 || mapping.Target == "" {
			return Result{}, fmt.Errorf("missing key column %q", mapping.Source)
		}
		cell := row[index]
		if cell.Null {
			return Result{}, ErrForeignKeyNull
		}
		kind := strings.ToUpper(cell.Type)
		switch kind {
		case "NUMBER", "VARCHAR2", "NVARCHAR2", "CHAR", "NCHAR", "RAW":
		default:
			return Result{}, fmt.Errorf("unsupported key type %s for %q", kind, mapping.Source)
		}
		if cell.Partial {
			return Result{}, fmt.Errorf("partial key column %q", mapping.Source)
		}
		value, err := editValue(kind, cell.Text, false)
		if err != nil {
			return Result{}, fmt.Errorf("invalid key column %q: %w", mapping.Source, err)
		}
		args = append(args, value)
		predicates = append(predicates, quoteIdentifier(mapping.Target)+" = "+editBind(fmt.Sprintf(":%d", len(args)), kind))
	}
	args = append(args, limit+1)
	predicates = append(predicates, fmt.Sprintf("ROWNUM <= :%d", len(args)))
	query := "SELECT * FROM " + QualifiedObject(link.Target) + " WHERE " + strings.Join(predicates, " AND ")
	return s.readObjectPreview(ctx, query, limit, args)
}
