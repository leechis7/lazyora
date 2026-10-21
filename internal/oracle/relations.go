package oracle

import (
	"context"
	"database/sql"
)

// These are visible grants, not a calculation of effective access through
// system privileges, role hierarchies or object ownership.
type ObjectGrant struct {
	Grantor, Grantee, Privilege, Grantable, Column string
}

type ObjectRelation struct {
	Object           Object
	Link, Dependency string
	Visible          bool
}

func (r ObjectRelation) Navigable() bool {
	if !r.Visible || r.Link != "" || r.Object.Owner == "" || r.Object.Name == "" {
		return false
	}
	_, err := metadataKind(r.Object.Kind)
	return err == nil
}

func (s *Session) objectGrants(ctx context.Context, object Object) (ObjectDefinition, error) {
	var result ObjectDefinition
	err := s.definitionQuery(ctx, `SELECT grantor, grantee, privilege, grantable, column_name FROM (
SELECT grantor, grantee, privilege, grantable, CAST(NULL AS VARCHAR2(128)) column_name
FROM all_tab_privs WHERE table_schema = :1 AND table_name = :2
UNION ALL
SELECT grantor, grantee, privilege, grantable, column_name
FROM all_col_privs WHERE table_schema = :3 AND table_name = :4
ORDER BY grantee, privilege, column_name, grantor) WHERE ROWNUM <= 257`,
		[]any{object.Owner, object.Name, object.Owner, object.Name}, definitionRows, &result.Limited, func(rows *sql.Rows) error {
			var g ObjectGrant
			var grantable, column sql.NullString
			if err := rows.Scan(&g.Grantor, &g.Grantee, &g.Privilege, &grantable, &column); err != nil {
				return err
			}
			g.Grantable, g.Column = grantable.String, column.String
			result.Grants = append(result.Grants, g)
			return nil
		})
	return result, err
}

func (s *Session) objectRelations(ctx context.Context, object Object, reverse bool) (ObjectDefinition, error) {
	var result ObjectDefinition
	owner, name, kind := "d.referenced_owner", "d.referenced_name", "d.referenced_type"
	filter := "d.owner = :1 AND d.name = :2 AND d.type = :3"
	join := " AND d.referenced_link_name IS NULL"
	if reverse {
		owner, name, kind = "d.owner", "d.name", "d.type"
		filter = "d.referenced_owner = :1 AND d.referenced_name = :2 AND d.referenced_type = :3 AND d.referenced_link_name IS NULL"
		join = ""
	}
	// LEFT JOIN preserves remote, missing and invisible targets. A relation in
	// the dictionary does not guarantee that its target can be inspected.
	query := `SELECT target_owner, target_name, target_type, referenced_link_name, dependency_type, status FROM (
SELECT DISTINCT ` + owner + ` target_owner, ` + name + ` target_name, ` + kind + ` target_type,
d.referenced_link_name, d.dependency_type, o.status
FROM all_dependencies d LEFT JOIN all_objects o ON o.owner = ` + owner + ` AND o.object_name = ` + name +
		` AND o.object_type = ` + kind + ` AND o.subobject_name IS NULL` + join + `
WHERE ` + filter + ` ORDER BY target_owner, target_type, target_name, referenced_link_name, dependency_type)
WHERE ROWNUM <= 257`
	err := s.definitionQuery(ctx, query, []any{object.Owner, object.Name, object.Kind}, definitionRows, &result.Limited, func(rows *sql.Rows) error {
		var r ObjectRelation
		var owner, name, kind, link, dependency, status sql.NullString
		if err := rows.Scan(&owner, &name, &kind, &link, &dependency, &status); err != nil {
			return err
		}
		r.Object = Object{Owner: owner.String, Name: name.String, Kind: kind.String, Status: status.String}
		r.Link, r.Dependency, r.Visible = link.String, dependency.String, status.Valid
		result.Relations = append(result.Relations, r)
		return nil
	})
	return result, err
}
