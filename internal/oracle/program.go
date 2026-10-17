package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

func (o Object) HasSource() bool {
	switch o.Kind {
	case "FUNCTION", "PROCEDURE", "PACKAGE", "PACKAGE BODY", "TYPE", "TYPE BODY", "TRIGGER":
		return true
	}
	return false
}

func (o Object) HasArguments() bool {
	return o.Kind == "FUNCTION" || o.Kind == "PROCEDURE" || o.Kind == "PACKAGE" || o.Kind == "PACKAGE BODY"
}

type Argument struct {
	Member, Overload, Name, DataType, Direction string
	Position, Depth                             int
}

type Diagnostic struct {
	Line, Position  int
	Attribute, Text string
}

type Compilation struct {
	Object      Object
	Diagnostics []Diagnostic
	Applied     bool
}

func quoteIdentifier(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }

var sourceHeader = regexp.MustCompile(`(?is)^\s*(?:CREATE\s+OR\s+REPLACE\s+)?(FUNCTION|PROCEDURE|PACKAGE\s+BODY|PACKAGE|TYPE\s+BODY|TYPE|TRIGGER)\s+("(?:""|[^"])+"|[\pL][\pL\pN_$#]*)(?:\s*\.\s*("(?:""|[^"])+"|[\pL][\pL\pN_$#]*))?`)

func identifierValue(name string) string {
	if strings.HasPrefix(name, `"`) {
		return strings.ReplaceAll(name[1:len(name)-1], `""`, `"`)
	}
	return strings.ToUpper(name)
}

// SourceDDL pins the editable header to the selected object. The PL/SQL body's
// semicolons remain intact; a SQL*Plus trailing slash is only a client delimiter.
func SourceDDL(object Object, source string) (string, error) {
	if !object.HasSource() {
		return "", errors.New("object does not support PL/SQL source")
	}
	source = strings.TrimSpace(strings.ReplaceAll(source, "\r\n", "\n"))
	if strings.HasSuffix(source, "\n/") {
		source = strings.TrimSpace(strings.TrimSuffix(source, "/"))
	}
	match := sourceHeader.FindStringSubmatchIndex(source)
	if match == nil {
		return "", errors.New("source must begin with the selected object type and name")
	}
	kind := strings.ToUpper(strings.Join(strings.Fields(source[match[2]:match[3]]), " "))
	first := identifierValue(source[match[4]:match[5]])
	owner, name := object.Owner, first
	if match[6] >= 0 {
		owner, name = first, identifierValue(source[match[6]:match[7]])
	}
	if kind != object.Kind || owner != object.Owner || name != object.Name {
		return "", errors.New("source header must match the selected schema, object name and type")
	}
	return "CREATE OR REPLACE " + kind + " " + quoteIdentifier(owner) + "." + quoteIdentifier(name) + source[match[1]:], nil
}

func (s *Session) Source(ctx context.Context, object Object) (string, error) {
	if !object.HasSource() {
		return "", errors.New("source is not available for this object type")
	}
	rows, err := s.dictionaryQuery(ctx, `SELECT text FROM all_source
WHERE owner = :1 AND name = :2 AND type = :3 ORDER BY line`, object.Owner, object.Name, object.Kind)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var source strings.Builder
	for rows.Next() {
		var line sql.NullString
		if err = rows.Scan(&line); err != nil {
			return "", err
		}
		source.WriteString(line.String)
	}
	if err = rows.Err(); err != nil {
		return "", err
	}
	if source.Len() == 0 {
		return "", sql.ErrNoRows
	}
	return SourceDDL(object, source.String())
}

func (s *Session) Arguments(ctx context.Context, object Object) ([]Argument, error) {
	if !object.HasArguments() {
		return nil, errors.New("arguments are not available for this object type")
	}
	filter := "package_name IS NULL AND object_name = :2"
	if object.Kind == "PACKAGE" || object.Kind == "PACKAGE BODY" {
		filter = "package_name = :2"
	}
	rows, err := s.dictionaryQuery(ctx, `SELECT object_name, NVL(overload, '-'), NVL(argument_name, '-'),
position, data_level, NVL(data_type, '-'), NVL(in_out, '-'), type_owner, type_name, type_subname
FROM all_arguments WHERE owner = :1 AND `+filter+` ORDER BY object_name, overload, sequence`, object.Owner, object.Name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var arguments []Argument
	for rows.Next() {
		var argument Argument
		var owner, name, subname sql.NullString
		if err = rows.Scan(&argument.Member, &argument.Overload, &argument.Name, &argument.Position, &argument.Depth, &argument.DataType, &argument.Direction, &owner, &name, &subname); err != nil {
			return nil, err
		}
		if name.Valid {
			argument.DataType = name.String
			if owner.Valid {
				argument.DataType = owner.String + "." + argument.DataType
			}
			if subname.Valid {
				argument.DataType += "." + subname.String
			}
		}
		arguments = append(arguments, argument)
	}
	return arguments, rows.Err()
}

func (s *Session) Diagnostics(ctx context.Context, object Object) ([]Diagnostic, error) {
	rows, err := s.dictionaryQuery(ctx, `SELECT line, position, NVL(attribute, 'ERROR'), text
FROM all_errors WHERE owner = :1 AND name = :2 AND type = :3 ORDER BY sequence`, object.Owner, object.Name, object.Kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var diagnostics []Diagnostic
	for rows.Next() {
		var diagnostic Diagnostic
		if err = rows.Scan(&diagnostic.Line, &diagnostic.Position, &diagnostic.Attribute, &diagnostic.Text); err != nil {
			return nil, err
		}
		diagnostics = append(diagnostics, diagnostic)
	}
	return diagnostics, rows.Err()
}

func compileDDL(object Object) (string, error) {
	if !object.HasSource() {
		return "", errors.New("object cannot be compiled")
	}
	kind, clause := object.Kind, " COMPILE"
	switch kind {
	case "PACKAGE", "TYPE":
		clause += " SPECIFICATION"
	case "PACKAGE BODY", "TYPE BODY":
		kind = strings.TrimSuffix(kind, " BODY")
		clause += " BODY"
	}
	return "ALTER " + kind + " " + quoteIdentifier(object.Owner) + "." + quoteIdentifier(object.Name) + clause, nil
}

// Compile executes exactly one object DDL, never a script. Oracle DDL can commit
// prior DML even when compilation fails; the UI must confirm this before calling.
func (s *Session) Compile(ctx context.Context, object Object, source string) (Compilation, error) {
	result := Compilation{Object: object}
	if s.finishErr != nil {
		return result, s.finishErr
	}
	statement, err := compileDDL(object)
	if source != "" {
		statement, err = SourceDDL(object, source)
	}
	if err != nil {
		return result, err
	}
	s.pending = true
	var execErr error
	if s.tx != nil {
		_, execErr = s.tx.ExecContext(ctx, statement)
	} else {
		_, execErr = s.conn.ExecContext(ctx, statement)
	}
	result.Applied = source != "" && execErr == nil
	compiledObject, err := s.ObjectInfo(ctx, object)
	if err != nil {
		return result, errors.Join(execErr, fmt.Errorf("read compiled object status: %w", err))
	}
	result.Object = compiledObject
	result.Diagnostics, err = s.Diagnostics(ctx, object)
	if source != "" && execErr != nil {
		// Some drivers return a warning for CREATE with compilation errors,
		// although Oracle has already replaced the object.
		stored, readErr := s.Source(ctx, object)
		result.Applied = readErr == nil && strings.TrimSpace(stored) == strings.TrimSpace(statement)
	}
	return result, errors.Join(execErr, err)
}
