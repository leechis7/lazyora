package oracle

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const MaxCompletionCandidates = 200
const completionCacheEntries = 48
const completionCacheTTL = 5 * time.Minute
const completionObjectLimit = 2000
const completionColumnLimit = 4096

type CompletionCandidate struct{ Text, Detail, Kind, Name string }
type Completions struct {
	Candidates []CompletionCandidate
	Schema     string
	Warnings   []string
	Truncated  bool
}
type completionKey struct {
	Kind, Owner, Name, Prefix string
	Quoted                    bool
}
type completionName struct{ Name, Kind string }
type completionEntry struct {
	At        time.Time
	Names     []completionName
	Truncated bool
}

// Metadata stays on the worker-owned Session: reconnect cannot reuse another
// connection's privileges/schema, and no keystroke starts a parallel SQL query.
func (s *Session) clearCompletionCache() { s.completionCache = nil }
func (s *Session) completionNames(ctx context.Context, key completionKey) (completionEntry, error) {
	if cached, ok := s.completionCache[key]; ok && time.Since(cached.At) < completionCacheTTL {
		return cached, nil
	}
	var query string
	var args []any
	limit := completionColumnLimit
	if key.Kind == "objects" {
		limit = completionObjectLimit
		prefix := key.Prefix
		column := "UPPER(object_name)"
		if key.Quoted {
			column = "object_name"
		} else {
			prefix = strings.ToUpper(prefix)
		}
		// Escape LIKE metacharacters: a typed underscore is a literal identifier.
		prefix = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(prefix) + "%"
		query = `SELECT object_name, object_type FROM (
SELECT DISTINCT object_name, object_type FROM all_objects
WHERE owner = :1 AND ` + column + ` LIKE :2 ESCAPE '\' AND subobject_name IS NULL
AND object_type IN ('TABLE','VIEW','MATERIALIZED VIEW','FUNCTION','PROCEDURE','PACKAGE','SEQUENCE','SYNONYM')
ORDER BY object_name, object_type) WHERE ROWNUM <= :3`
		args = []any{key.Owner, prefix, limit + 1}
	} else {
		query = `SELECT column_name, data_type FROM (
SELECT column_name, data_type FROM all_tab_columns
WHERE owner = :1 AND table_name = :2 ORDER BY column_id) WHERE ROWNUM <= :3`
		args = []any{key.Owner, key.Name, limit + 1}
	}
	rows, err := s.dictionaryQuery(ctx, query, args...)
	if err != nil {
		return completionEntry{}, err
	}
	defer rows.Close()
	entry := completionEntry{At: time.Now()}
	seen := map[string]bool{}
	for rows.Next() {
		var name completionName
		if err = rows.Scan(&name.Name, &name.Kind); err != nil {
			return entry, err
		}
		if seen[name.Name] {
			continue
		}
		seen[name.Name] = true
		if len(entry.Names) == limit {
			entry.Truncated = true
			break
		}
		entry.Names = append(entry.Names, name)
	}
	if err = rows.Err(); err != nil {
		return entry, err
	}
	if err = rows.Close(); err != nil {
		return entry, err
	}
	if s.completionCache == nil {
		s.completionCache = make(map[completionKey]completionEntry)
	}
	if len(s.completionCache) >= completionCacheEntries {
		var oldest completionKey
		var at time.Time
		for k, v := range s.completionCache {
			if at.IsZero() || v.At.Before(at) {
				oldest, at = k, v.At
			}
		}
		delete(s.completionCache, oldest)
	}
	s.completionCache[key] = entry
	return entry, nil
}
func completionMatches(name string, c CompletionContext) bool {
	if c.Quoted {
		return strings.HasPrefix(name, c.Prefix)
	}
	return strings.HasPrefix(strings.ToUpper(name), strings.ToUpper(c.Prefix))
}
func (s *Session) Complete(ctx context.Context, c CompletionContext, refresh bool) (out Completions, err error) {
	if c.Blocked {
		return out, nil
	}
	if refresh {
		s.clearCompletionCache()
	}
	rows, err := s.dictionaryQuery(ctx, `SELECT SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA') FROM dual`)
	if err != nil {
		return out, err
	}
	if !rows.Next() {
		err = rows.Err()
		rows.Close()
		if err == nil {
			err = fmt.Errorf("current schema unavailable")
		}
		return out, err
	}
	err = rows.Scan(&out.Schema)
	if err == nil {
		err = rows.Err()
	}
	closeErr := rows.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return out, err
	}
	seen := map[string]bool{}
	add := func(candidate CompletionCandidate) {
		if seen[candidate.Text] {
			return
		}
		seen[candidate.Text] = true
		if len(out.Candidates) == MaxCompletionCandidates {
			out.Truncated = true
			return
		}
		out.Candidates = append(out.Candidates, candidate)
	}
	objects := func(owner CompletionIdentifier) error {
		if owner.Name == "" {
			owner.Name = out.Schema
		}
		entry, e := s.completionNames(ctx, completionKey{Kind: "objects", Owner: owner.Name, Prefix: c.Prefix, Quoted: c.Quoted})
		if e != nil {
			return e
		}
		out.Truncated = out.Truncated || entry.Truncated
		for _, name := range entry.Names {
			if completionMatches(name.Name, c) {
				add(CompletionCandidate{Text: quoteIdentifier(name.Name), Detail: owner.Name + " · " + name.Kind, Kind: name.Kind, Name: name.Name})
			}
		}
		return nil
	}
	sources := c.Sources
	qualified := len(c.Qualifier) > 0
	if len(c.Qualifier) == 2 {
		sources = []CompletionSource{{Owner: c.Qualifier[0], Name: c.Qualifier[1]}}
	} else if len(c.Qualifier) == 1 && !c.ObjectsOnly {
		var matched []CompletionSource
		for _, source := range sources {
			name := source.Alias
			if name.Name == "" {
				name = source.Name
			}
			if name.Name == c.Qualifier[0].Name {
				matched = append(matched, source)
			}
		}
		if len(matched) > 1 {
			out.Warnings = append(out.Warnings, "ambiguous alias: "+c.Qualifier[0].Name)
			return out, nil
		}
		if len(matched) == 0 {
			err = objects(c.Qualifier[0])
			return out, err
		}
		sources = matched
	} else if c.ObjectsOnly {
		owner := CompletionIdentifier{Name: out.Schema}
		if len(c.Qualifier) == 1 {
			owner = c.Qualifier[0]
		}
		err = objects(owner)
		return out, err
	}
	if len(sources) > 8 {
		sources = sources[:8]
		out.Truncated = true
	}
	for _, source := range sources {
		if source.Derived {
			out.Warnings = append(out.Warnings, "derived table / CTE columns require explicit names: "+source.Alias.Name+source.Name.Name)
			continue
		}
		owner := source.Owner.Name
		if owner == "" {
			owner = out.Schema
		}
		entry, e := s.completionNames(ctx, completionKey{Kind: "columns", Owner: owner, Name: source.Name.Name})
		if e != nil {
			if ctx.Err() != nil {
				return out, ctx.Err()
			}
			out.Warnings = append(out.Warnings, e.Error())
			continue
		}
		out.Truncated = out.Truncated || entry.Truncated
		prefix := ""
		if !qualified && len(sources) > 1 {
			if source.Alias.Name != "" {
				prefix = source.Alias.SQL() + "."
			} else {
				prefix = source.Name.SQL() + "."
				if source.Owner.Name != "" {
					prefix = source.Owner.SQL() + "." + prefix
				}
			}
		}
		for _, name := range entry.Names {
			if completionMatches(name.Name, c) {
				add(CompletionCandidate{Text: prefix + quoteIdentifier(name.Name), Detail: owner + "." + source.Name.Name + " · " + name.Kind, Kind: "COLUMN", Name: name.Name})
			}
		}
	}
	if !qualified {
		if e := objects(CompletionIdentifier{Name: out.Schema}); e != nil {
			if ctx.Err() != nil {
				return out, ctx.Err()
			}
			out.Warnings = append(out.Warnings, e.Error())
		}
	}
	return out, nil
}
