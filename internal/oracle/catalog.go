package oracle

import (
	"context"
	"database/sql"
	"fmt"
)

type Catalog struct {
	Current string // Login user (SESSION_USER), the default browser owner.
	Schemas []string
}

type Object struct {
	Owner, Name, Kind string
	Status            string
}

func (o Object) HasColumns() bool {
	return o.Kind == "TABLE" || o.Kind == "VIEW" || o.Kind == "MATERIALIZED VIEW"
}

type Column struct {
	Name, DataType, Comment string
	Nullable                bool
}

// Dictionary queries use the existing transaction, if any, without starting or
// finishing one. Only the UI worker may call these methods, just like Execute.
func (s *Session) dictionaryQuery(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if s.tx != nil {
		return s.tx.QueryContext(ctx, query, args...)
	}
	return s.conn.QueryContext(ctx, query, args...)
}

func (s *Session) Catalog(ctx context.Context) (Catalog, error) {
	var catalog Catalog
	rows, err := s.dictionaryQuery(ctx, "SELECT SYS_CONTEXT('USERENV', 'SESSION_USER') FROM dual")
	if err != nil {
		return catalog, err
	}
	if rows.Next() {
		err = rows.Scan(&catalog.Current)
	} else {
		err = rows.Err()
		if err == nil {
			err = sql.ErrNoRows
		}
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return catalog, err
	}
	rows, err = s.dictionaryQuery(ctx, "SELECT username FROM all_users ORDER BY username")
	if err != nil {
		return catalog, err
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return catalog, err
		}
		catalog.Schemas = append(catalog.Schemas, name)
		found = found || name == catalog.Current
	}
	if !found && catalog.Current != "" {
		catalog.Schemas = append(catalog.Schemas, catalog.Current)
	}
	return catalog, rows.Err()
}

func (s *Session) Objects(ctx context.Context, owner string) ([]Object, error) {
	rows, err := s.dictionaryQuery(ctx, `SELECT object_name, object_type, status FROM all_objects
WHERE owner = :1 AND object_type IN (
'TABLE', 'VIEW', 'MATERIALIZED VIEW', 'FUNCTION', 'PROCEDURE', 'PACKAGE', 'PACKAGE BODY',
'SEQUENCE', 'SYNONYM', 'INDEX', 'TRIGGER', 'TYPE', 'TYPE BODY') AND subobject_name IS NULL
ORDER BY object_type, object_name`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var objects []Object
	materialized := make(map[string]bool)
	for rows.Next() {
		object := Object{Owner: owner}
		if err = rows.Scan(&object.Name, &object.Kind, &object.Status); err != nil {
			return nil, err
		}
		objects = append(objects, object)
		if object.Kind == "MATERIALIZED VIEW" {
			materialized[object.Name] = true
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	// Oracle also lists a materialized view's storage table. Present it once.
	filtered := objects[:0]
	for _, object := range objects {
		if object.Kind != "TABLE" || !materialized[object.Name] {
			filtered = append(filtered, object)
		}
	}
	return filtered, nil
}

func (s *Session) ObjectInfo(ctx context.Context, object Object) (Object, error) {
	rows, err := s.dictionaryQuery(ctx, `SELECT status FROM all_objects
WHERE owner = :1 AND object_name = :2 AND object_type = :3 AND subobject_name IS NULL`, object.Owner, object.Name, object.Kind)
	if err != nil {
		return object, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err = rows.Err(); err != nil {
			return object, err
		}
		return object, sql.ErrNoRows
	}
	if err = rows.Scan(&object.Status); err != nil {
		return object, err
	}
	return object, rows.Err()
}

func (s *Session) Columns(ctx context.Context, object Object) ([]Column, error) {
	rows, err := s.dictionaryQuery(ctx, `SELECT c.column_name, c.data_type, c.data_length,
c.data_precision, c.data_scale, c.char_length, c.char_used, c.nullable, m.comments
FROM all_tab_columns c LEFT JOIN all_col_comments m
ON m.owner = c.owner AND m.table_name = c.table_name AND m.column_name = c.column_name
WHERE c.owner = :1 AND c.table_name = :2 ORDER BY c.column_id`, object.Owner, object.Name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var columns []Column
	for rows.Next() {
		var column Column
		var length, precision, scale, chars sql.NullInt64
		var semantics, comment sql.NullString
		var nullable string
		if err = rows.Scan(&column.Name, &column.DataType, &length, &precision, &scale, &chars, &semantics, &nullable, &comment); err != nil {
			return nil, err
		}
		column.DataType = columnType(column.DataType, length, precision, scale, chars, semantics)
		column.Nullable = nullable == "Y"
		column.Comment = comment.String
		columns = append(columns, column)
	}
	return columns, rows.Err()
}

func columnType(kind string, length, precision, scale, chars sql.NullInt64, semantics sql.NullString) string {
	switch kind {
	case "NUMBER":
		if precision.Valid {
			if scale.Valid {
				return fmt.Sprintf("NUMBER(%d,%d)", precision.Int64, scale.Int64)
			}
			return fmt.Sprintf("NUMBER(%d)", precision.Int64)
		}
		if scale.Valid {
			return fmt.Sprintf("NUMBER(*,%d)", scale.Int64)
		}
	case "FLOAT":
		if precision.Valid {
			return fmt.Sprintf("FLOAT(%d)", precision.Int64)
		}
	case "CHAR", "VARCHAR2":
		if semantics.String == "C" && chars.Valid {
			return fmt.Sprintf("%s(%d CHAR)", kind, chars.Int64)
		}
		if length.Valid {
			return fmt.Sprintf("%s(%d BYTE)", kind, length.Int64)
		}
	case "NCHAR", "NVARCHAR2":
		if chars.Valid {
			return fmt.Sprintf("%s(%d)", kind, chars.Int64)
		}
	case "RAW":
		if length.Valid {
			return fmt.Sprintf("RAW(%d)", length.Int64)
		}
	}
	return kind
}
