package oracle

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
)

var objectReference = regexp.MustCompile(`^\s*("(?:""|[^"])+"|[\pL][\pL\pN_$#]*)(?:\s*\.\s*("(?:""|[^"])+"|[\pL][\pL\pN_$#]*))?\s*$`)

// ParseObjectReference accepts a name or schema.name, with quoted identifiers.
func ParseObjectReference(reference string) (owner, name string, err error) {
	match := objectReference.FindStringSubmatch(reference)
	if match == nil {
		return "", "", errors.New("expected object name or schema.object")
	}
	name = identifierValue(match[1])
	if match[2] != "" {
		owner, name = name, identifierValue(match[2])
	}
	return owner, name, nil
}

func (s *Session) FindObject(ctx context.Context, reference string) (Object, error) {
	owner, name, err := ParseObjectReference(reference)
	if err != nil {
		return Object{}, err
	}
	rows, err := s.dictionaryQuery(ctx, `SELECT owner, object_name, object_type, status FROM all_objects
WHERE owner = NVL(:1, SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA')) AND object_name = :2
AND subobject_name IS NULL AND object_type IN (
'TABLE','VIEW','MATERIALIZED VIEW','FUNCTION','PROCEDURE','PACKAGE','PACKAGE BODY',
'SEQUENCE','SYNONYM','INDEX','TRIGGER','TYPE','TYPE BODY')
ORDER BY CASE object_type WHEN 'MATERIALIZED VIEW' THEN 0 WHEN 'PACKAGE' THEN 1 WHEN 'TYPE' THEN 2 ELSE 3 END, object_type`, owner, name)
	if err != nil {
		return Object{}, err
	}
	defer rows.Close()
	var object Object
	if !rows.Next() {
		if err = rows.Err(); err == nil {
			err = sql.ErrNoRows
		}
		return object, err
	}
	err = rows.Scan(&object.Owner, &object.Name, &object.Kind, &object.Status)
	return object, err
}

func QualifiedObject(object Object) string {
	return strings.Join([]string{quoteIdentifier(object.Owner), quoteIdentifier(object.Name)}, ".")
}
