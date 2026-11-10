package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
)

const (
	maxCompareColumns  = 12000
	maxCompareDefaults = 8 << 20
)

type SchemaColumn struct {
	Position int
	Type     string
	Nullable bool
	Default  string
}

type SchemaTable map[string]SchemaColumn

type SchemaStructure struct {
	Owner  string
	Tables map[string]SchemaTable
}

type SchemaColumnChange struct {
	Name        string
	Left, Right *SchemaColumn
}

type SchemaTableChange struct {
	Name                string
	LeftOnly, RightOnly bool
	Columns             []SchemaColumnChange
}

// ReadSchemaStructure collects only metadata visible to this session. An
// incomplete snapshot returns an error so missing tables are never inferred.
func (s *Session) ReadSchemaStructure(ctx context.Context, owner string) (SchemaStructure, error) {
	if s.finishErr != nil {
		return SchemaStructure{}, s.finishErr
	}
	rows, err := s.dictionaryQuery(ctx, `SELECT username FROM all_users WHERE username =
NVL(:1, SYS_CONTEXT('USERENV','CURRENT_SCHEMA'))`, owner)
	if err != nil {
		return SchemaStructure{}, err
	}
	if rows.Next() {
		err = rows.Scan(&owner)
	} else if err = rows.Err(); err == nil {
		err = errors.New("schema is missing or not visible")
	}
	rows.Close()
	if err != nil {
		return SchemaStructure{}, err
	}

	result := SchemaStructure{Owner: owner, Tables: make(map[string]SchemaTable)}
	query := fmt.Sprintf(`SELECT * FROM (
SELECT t.table_name, c.column_name, c.column_id, c.data_type, c.data_type_owner, c.data_length,
c.data_precision, c.data_scale, c.char_length, c.char_used, c.nullable, c.default_length
FROM all_tables t
LEFT JOIN all_tab_columns c ON c.owner = t.owner AND c.table_name = t.table_name
WHERE t.owner = :1
AND NOT EXISTS (SELECT 1 FROM all_mviews m WHERE m.owner = t.owner AND m.mview_name = t.table_name)
ORDER BY t.table_name, c.column_id
) WHERE ROWNUM <= %d`, maxCompareColumns+1)
	limited := false
	var defaults []longDefinitionRow
	err = s.definitionQuery(ctx, query, []any{owner}, maxCompareColumns, &limited, func(rows *sql.Rows) error {
		var table string
		var name, kind, typeOwner, semantics, nullable sql.NullString
		var pos, length, precision, scale, chars, defaultLength sql.NullInt64
		if err := rows.Scan(&table, &name, &pos, &kind, &typeOwner, &length, &precision, &scale, &chars, &semantics, &nullable, &defaultLength); err != nil {
			return err
		}
		columns := result.Tables[table]
		if columns == nil {
			columns = make(SchemaTable)
			result.Tables[table] = columns
		}
		if !name.Valid {
			return nil
		}
		typeName := columnType(kind.String, length, precision, scale, chars, semantics)
		if typeOwner.Valid {
			typeName = quoteIdentifier(typeOwner.String) + "." + quoteIdentifier(typeName)
		}
		columns[name.String] = SchemaColumn{Position: int(pos.Int64), Type: typeName, Nullable: nullable.String == "Y"}
		if defaultLength.Int64 > 0 {
			defaults = append(defaults, longDefinitionRow{owner: owner, table: table, name: name.String, position: int(defaultLength.Int64)})
		}
		return nil
	})
	if err != nil {
		return SchemaStructure{}, err
	}
	if limited {
		return SchemaStructure{}, fmt.Errorf("schema metadata exceeds the %d-row comparison limit", maxCompareColumns)
	}
	size := 0
	for _, lookup := range defaults {
		if lookup.position > maxCompareDefaults-size {
			return SchemaStructure{}, errors.New("schema defaults exceed the comparison size limit")
		}
		size += lookup.position
	}
	values, err := s.definitionLong(ctx, "default", defaults)
	if err != nil {
		return SchemaStructure{}, fmt.Errorf("read column defaults: %w", err)
	}
	for i, lookup := range defaults {
		column := result.Tables[lookup.table][lookup.name]
		column.Default = strings.TrimSpace(values[i])
		result.Tables[lookup.table][lookup.name] = column
	}
	return result, nil
}

func CompareSchemas(left, right SchemaStructure) []SchemaTableChange {
	var differences []SchemaTableChange
	for name, a := range left.Tables {
		b, ok := right.Tables[name]
		if !ok {
			differences = append(differences, SchemaTableChange{Name: name, LeftOnly: true})
			continue
		}
		change := SchemaTableChange{Name: name}
		for column, first := range a {
			second, found := b[column]
			if !found {
				change.Columns = append(change.Columns, SchemaColumnChange{Name: column, Left: &first})
			} else if first != second {
				change.Columns = append(change.Columns, SchemaColumnChange{Name: column, Left: &first, Right: &second})
			}
		}
		for column, second := range b {
			if _, found := a[column]; !found {
				change.Columns = append(change.Columns, SchemaColumnChange{Name: column, Right: &second})
			}
		}
		if len(change.Columns) > 0 {
			slices.SortFunc(change.Columns, func(a, b SchemaColumnChange) int { return strings.Compare(a.Name, b.Name) })
			differences = append(differences, change)
		}
	}
	for name := range right.Tables {
		if _, found := left.Tables[name]; !found {
			differences = append(differences, SchemaTableChange{Name: name, RightOnly: true})
		}
	}
	slices.SortFunc(differences, func(a, b SchemaTableChange) int { return strings.Compare(a.Name, b.Name) })
	return differences
}
