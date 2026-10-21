package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	go_ora "github.com/sijms/go-ora/v3"
)

const definitionRows = 256
const definitionColumns = 2048
const ddlCharacters = 262144 // Oracle CLOB/UCS2 units, bounded before fetching data.

type Constraint struct {
	Name, Kind, Status, Validated, Deferrable, Deferred, DeleteRule, Condition string
	ReferenceOwner, ReferenceName, ReferenceTable                              string
	Columns, ReferenceColumns                                                  []string
}
type IndexColumn struct {
	Name, Direction, Expression string
	Position                    int
}
type Index struct {
	Owner, Name, Kind, Uniqueness, Status, TableOwner, TableName string
	Columns                                                      []IndexColumn
}
type ObjectDefinition struct {
	DDL         string
	Constraints []Constraint
	Indexes     []Index
	Grants      []ObjectGrant
	Relations   []ObjectRelation
	Limited     bool
}

func metadataKind(kind string) (string, error) {
	switch kind {
	case "PACKAGE":
		return "PACKAGE_SPEC", nil
	case "TYPE":
		return "TYPE_SPEC", nil
	case "TABLE", "VIEW", "MATERIALIZED VIEW", "FUNCTION", "PROCEDURE", "PACKAGE BODY", "SEQUENCE", "SYNONYM", "INDEX", "TRIGGER", "TYPE BODY":
		return strings.ReplaceAll(kind, " ", "_"), nil
	}
	return "", errors.New("DDL is not supported for this object type")
}

// Definition only reads on the pinned connection/current transaction. It never
// calls the execute pipeline or mutates DBMS_METADATA session transforms.
func (s *Session) Definition(ctx context.Context, object Object, section string) (ObjectDefinition, error) {
	var result ObjectDefinition
	if s.finishErr != nil {
		return result, s.finishErr
	}
	if object.Owner == "" || object.Name == "" {
		return result, errors.New("schema and object name are required")
	}
	switch section {
	case "grants":
		return s.objectGrants(ctx, object)
	case "dependencies", "dependents":
		return s.objectRelations(ctx, object, section == "dependents")
	case "ddl":
		ddl, err := s.definitionDDL(ctx, object)
		result.DDL = ddl
		return result, err
	case "constraints":
		if !object.HasColumns() {
			return result, nil
		}
		err := s.constraintDefinition(ctx, object, &result)
		return result, err
	case "indexes":
		if !object.HasColumns() && object.Kind != "INDEX" {
			return result, nil
		}
		err := s.indexDefinition(ctx, object, &result)
		return result, err
	default:
		return result, errors.New("unknown definition section")
	}
}

// Read a bounded CLOB through PL/SQL VARCHAR2 output pieces. The driver cannot
// decode DBMS_METADATA's temporary CLOB directly on supported Free versions.
// Fetch it once, and always release the temporary LOB on the server.
func (s *Session) definitionDDL(ctx context.Context, object Object) (string, error) {
	kind, err := metadataKind(object.Kind)
	if err != nil {
		return "", err
	}
	const pieceChars = 4000
	parts := make([]sql.NullString, (ddlCharacters+pieceChars-1)/pieceChars)
	length := 0
	args := []any{sql.Named("kind", kind), sql.Named("name", object.Name), sql.Named("owner", object.Owner), sql.Named("length", go_ora.Out{Dest: &length})}
	var code strings.Builder
	code.WriteString("DECLARE v_ddl CLOB; v_len PLS_INTEGER; v_pos PLS_INTEGER := 1; v_piece VARCHAR2(16000); BEGIN v_ddl := DBMS_METADATA.GET_DDL(:kind, :name, :owner); v_len := DBMS_LOB.GETLENGTH(v_ddl); :length := v_len; IF v_len <= 262144 THEN\n")
	for i := range parts {
		name := fmt.Sprintf("part%d", i)
		fmt.Fprintf(&code, "IF v_pos <= v_len THEN v_piece := DBMS_LOB.SUBSTR(v_ddl, %d, v_pos); IF v_piece IS NULL THEN RAISE_APPLICATION_ERROR(-20001, 'Incomplete DDL preview'); END IF; :%s := v_piece; v_pos := v_pos + LENGTH2(v_piece); END IF;\n", pieceChars, name)
		args = append(args, sql.Named(name, go_ora.Out{Dest: &parts[i], Size: pieceChars * 4}))
	}
	code.WriteString("END IF; IF DBMS_LOB.ISTEMPORARY(v_ddl) = 1 THEN DBMS_LOB.FREETEMPORARY(v_ddl); END IF; EXCEPTION WHEN OTHERS THEN IF DBMS_LOB.ISTEMPORARY(v_ddl) = 1 THEN DBMS_LOB.FREETEMPORARY(v_ddl); END IF; RAISE; END;")
	err = s.dictionaryExec(ctx, code.String(), args...)
	if err != nil {
		return "", err
	}
	if length > ddlCharacters {
		return "", fmt.Errorf("DDL exceeds the %d-character preview limit", ddlCharacters)
	}
	if length == 0 {
		return "", sql.ErrNoRows
	}
	var ddl strings.Builder
	for _, part := range parts {
		ddl.WriteString(part.String)
	}
	if ddl.Len() == 0 {
		return "", sql.ErrNoRows
	}
	units := 0
	for _, r := range ddl.String() {
		units++
		if r > 0xFFFF {
			units++
		}
	}
	if units != length {
		return "", errors.New("incomplete DDL preview: character count mismatch")
	}
	return ddl.String(), nil
}

// Close each result before opening the next dictionary cursor. Limits bound
// both result count and Go-side retention on unusually large schemas.
func (s *Session) definitionQuery(ctx context.Context, query string, args []any, limit int, limited *bool, scan func(*sql.Rows) error) error {
	rows, err := s.dictionaryQuery(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		if count >= limit {
			*limited = true
			break
		}
		if err = scan(rows); err != nil {
			return err
		}
		count++
	}
	return rows.Err()
}

func (s *Session) constraintDefinition(ctx context.Context, object Object, result *ObjectDefinition) error {
	args := []any{object.Owner, object.Name}
	byName := make(map[string]int)
	err := s.definitionQuery(ctx, `SELECT constraint_name, constraint_type, status, validated, deferrable,
deferred, r_owner, r_constraint_name, delete_rule
FROM (SELECT * FROM all_constraints WHERE owner = :1 AND table_name = :2 ORDER BY constraint_name)
WHERE ROWNUM <= 257`, args, definitionRows, &result.Limited, func(rows *sql.Rows) error {
		var c Constraint
		var ro, rn, rule sql.NullString
		if err := rows.Scan(&c.Name, &c.Kind, &c.Status, &c.Validated, &c.Deferrable, &c.Deferred, &ro, &rn, &rule); err != nil {
			return err
		}
		c.ReferenceOwner, c.ReferenceName, c.DeleteRule = ro.String, rn.String, rule.String
		byName[c.Name] = len(result.Constraints)
		result.Constraints = append(result.Constraints, c)
		return nil
	})
	if err != nil || len(result.Constraints) == 0 {
		return err
	}
	err = s.definitionQuery(ctx, `SELECT constraint_name, column_name FROM (
SELECT constraint_name, column_name, position FROM all_cons_columns
WHERE owner = :1 AND table_name = :2 ORDER BY constraint_name, position) WHERE ROWNUM <= 2049`, args, definitionColumns, &result.Limited, func(rows *sql.Rows) error {
		var name, column string
		if err := rows.Scan(&name, &column); err != nil {
			return err
		}
		if i, ok := byName[name]; ok {
			result.Constraints[i].Columns = append(result.Constraints[i].Columns, column)
		}
		return nil
	})
	if err != nil {
		return err
	}
	var checks []longDefinitionRow
	var checkIndexes []int
	for i, c := range result.Constraints {
		if c.Kind == "C" {
			checks = append(checks, longDefinitionRow{owner: object.Owner, name: c.Name, table: object.Name})
			checkIndexes = append(checkIndexes, i)
		}
	}
	conditions, err := s.definitionLong(ctx, "condition", checks)
	for i, condition := range conditions {
		result.Constraints[checkIndexes[i]].Condition = condition
	}
	if err != nil {
		return err
	}
	hasReference := false
	for _, c := range result.Constraints {
		hasReference = hasReference || c.Kind == "R"
	}
	if !hasReference {
		return nil
	}
	return s.definitionQuery(ctx, `SELECT constraint_name, reference_table, column_name FROM (
SELECT c.constraint_name, r.table_name reference_table, cc.column_name, cc.position
FROM all_constraints c JOIN all_constraints r ON r.owner = c.r_owner AND r.constraint_name = c.r_constraint_name
JOIN all_cons_columns cc ON cc.owner = r.owner AND cc.constraint_name = r.constraint_name AND cc.table_name = r.table_name
WHERE c.owner = :1 AND c.table_name = :2 AND c.constraint_type = 'R'
ORDER BY c.constraint_name, cc.position) WHERE ROWNUM <= 2049`, args, definitionColumns, &result.Limited, func(rows *sql.Rows) error {
		var name, table, column string
		if err := rows.Scan(&name, &table, &column); err != nil {
			return err
		}
		if i, ok := byName[name]; ok {
			c := &result.Constraints[i]
			c.ReferenceTable = table
			c.ReferenceColumns = append(c.ReferenceColumns, column)
		}
		return nil
	})
}

func (s *Session) indexDefinition(ctx context.Context, object Object, result *ObjectDefinition) error {
	filter := "table_owner = :1 AND table_name = :2"
	if object.Kind == "INDEX" {
		filter = "owner = :1 AND index_name = :2"
	}
	args := []any{object.Owner, object.Name}
	byName := make(map[string]int)
	err := s.definitionQuery(ctx, `SELECT owner, index_name, index_type, uniqueness, status, table_owner, table_name FROM (
SELECT owner, index_name, index_type, uniqueness, status, table_owner, table_name
FROM all_indexes WHERE `+filter+` ORDER BY owner, index_name) WHERE ROWNUM <= 257`, args, definitionRows, &result.Limited, func(rows *sql.Rows) error {
		var index Index
		var status sql.NullString
		if err := rows.Scan(&index.Owner, &index.Name, &index.Kind, &index.Uniqueness, &status, &index.TableOwner, &index.TableName); err != nil {
			return err
		}
		index.Status = status.String
		byName[index.Owner+"\x00"+index.Name] = len(result.Indexes)
		result.Indexes = append(result.Indexes, index)
		return nil
	})
	if err != nil || len(result.Indexes) == 0 {
		return err
	}
	if object.Kind == "INDEX" {
		filter = "index_owner = :1 AND index_name = :2"
	}
	err = s.definitionQuery(ctx, `SELECT index_owner, index_name, column_position, column_name, descend FROM (
SELECT index_owner, index_name, column_position, column_name, descend FROM all_ind_columns
WHERE `+filter+` ORDER BY index_owner, index_name, column_position) WHERE ROWNUM <= 2049`, args, definitionColumns, &result.Limited, func(rows *sql.Rows) error {
		var owner, name string
		var col IndexColumn
		if err := rows.Scan(&owner, &name, &col.Position, &col.Name, &col.Direction); err != nil {
			return err
		}
		if i, ok := byName[owner+"\x00"+name]; ok {
			result.Indexes[i].Columns = append(result.Indexes[i].Columns, col)
		}
		return nil
	})
	if err != nil {
		return err
	}
	var expressionRows []longDefinitionRow
	var expressionIndexes [][2]int
	for i, index := range result.Indexes {
		if strings.HasPrefix(index.Kind, "FUNCTION-BASED") {
			for j, col := range index.Columns {
				expressionRows = append(expressionRows, longDefinitionRow{owner: index.Owner, name: index.Name, position: col.Position})
				expressionIndexes = append(expressionIndexes, [2]int{i, j})
			}
		}
	}
	expressions, err := s.definitionLong(ctx, "expression", expressionRows)
	for i, expression := range expressions {
		pair := expressionIndexes[i]
		result.Indexes[pair[0]].Columns[pair[1]].Expression = expression
	}

	for i := range result.Indexes {
		sort.SliceStable(result.Indexes[i].Columns, func(a, b int) bool {
			return result.Indexes[i].Columns[a].Position < result.Indexes[i].Columns[b].Position
		})
	}
	return err
}

type longDefinitionRow struct {
	owner, name, table string
	position           int
}

func (s *Session) dictionaryExec(ctx context.Context, text string, args ...any) error {
	if s.finishErr != nil {
		return s.finishErr
	}
	var err error
	if s.tx != nil {
		_, err = s.tx.ExecContext(ctx, text, args...)
	} else {
		_, err = s.conn.ExecContext(ctx, text, args...)
	}
	return err
}

// LONG dictionary values are converted on the server; sending LONG directly
// hits the same driver decoder limitation as temporary CLOBs. Batch a bounded
// set of lookups in one round trip, using only 10g-compatible static SQL.
func (s *Session) definitionLong(ctx context.Context, kind string, rows []longDefinitionRow) ([]string, error) {
	var result []string
	for from := 0; from < len(rows); from += 16 {
		batch := rows[from:min(from+16, len(rows))]
		values := make([]sql.NullString, len(batch))
		var code strings.Builder
		code.WriteString("DECLARE v_text VARCHAR2(32767); BEGIN\n")
		var args []any
		for i, row := range batch {
			suffix := fmt.Sprint(i)
			args = append(args, sql.Named("owner"+suffix, row.owner), sql.Named("name"+suffix, row.name))
			if kind == "condition" {
				args = append(args, sql.Named("table"+suffix, row.table))
				fmt.Fprintf(&code, "SELECT search_condition INTO v_text FROM all_constraints WHERE owner = :owner%s AND constraint_name = :name%s AND table_name = :table%s;\n", suffix, suffix, suffix)
			} else {
				args = append(args, sql.Named("position"+suffix, row.position))
				fmt.Fprintf(&code, "BEGIN SELECT column_expression INTO v_text FROM all_ind_expressions WHERE index_owner = :owner%s AND index_name = :name%s AND column_position = :position%s; EXCEPTION WHEN NO_DATA_FOUND THEN v_text := NULL; END;\n", suffix, suffix, suffix)
			}
			fmt.Fprintf(&code, ":value%s := v_text;\n", suffix)
			args = append(args, sql.Named("value"+suffix, go_ora.Out{Dest: &values[i], Size: 32767}))
		}
		code.WriteString("END;")
		if err := s.dictionaryExec(ctx, code.String(), args...); err != nil {
			return result, err
		}
		for _, value := range values {
			result = append(result, value.String)
		}
	}
	return result, nil
}
