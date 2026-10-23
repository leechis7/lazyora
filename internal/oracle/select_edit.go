package oracle

import (
	"context"
	"strings"
	"time"
	"unicode"
)

// This is deliberately a small allowlist, not a general SQL lineage parser.
// A column label (including a label named ROWID) never grants write access.
type selectEditColumn struct {
	name, qualifier string
	rowID, star     bool
}
type selectEditPlan struct {
	owner, table, alias string
	columns             []selectEditColumn
}

func editIdentifier(text string, t sqlToken) (string, bool) {
	raw := text[t.start:t.end]
	if strings.HasPrefix(raw, "\"") && len(raw) > 2 {
		return strings.ReplaceAll(raw[1:len(raw)-1], "\"\"", "\""), true
	}
	if t.word == "" || !(unicode.IsLetter(rune(raw[0])) || raw[0] == '_') {
		return "", false
	}
	for _, r := range raw {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("_$#", r) {
			return "", false
		}
	}
	return t.word, true
}

func parseSelectEdit(text string) *selectEditPlan {
	tokens, err := sqlTokens(text)
	if err != nil || len(tokens) < 4 || tokens[0].word != "SELECT" {
		return nil
	}
	from := -1
	// Reject other row sources, set operations and transformations even if
	// nested in a predicate. Quoted identifiers and literals are opaque tokens.
	for i, t := range tokens {
		switch t.word {
		case "SELECT":
			if i != 0 {
				return nil
			}
		case "FROM":
			if from != -1 {
				return nil
			}
			from = i
		case "DISTINCT", "UNIQUE", "ALL", "JOIN", "GROUP", "HAVING", "UNION", "INTERSECT", "MINUS", "CONNECT", "START", "MODEL", "MATCH_RECOGNIZE", "PIVOT", "UNPIVOT", "SAMPLE", "VERSIONS", "AS", "FOR", "FETCH", "OFFSET", "@", ";":
			// AS is allowed only for a projection alias, handled below.
			if t.word != "AS" || from >= 0 {
				return nil
			}
		}
	}
	if from < 2 || from+1 >= len(tokens) {
		return nil
	}
	p := &selectEditPlan{}
	i := from + 1
	name, ok := editIdentifier(text, tokens[i])
	if !ok {
		return nil
	}
	i++
	if i < len(tokens) && tokens[i].word == "." {
		p.owner = name
		i++
		if i >= len(tokens) {
			return nil
		}
		name, ok = editIdentifier(text, tokens[i])
		if !ok {
			return nil
		}
		i++
	}
	p.table = name
	if i < len(tokens) && tokens[i].word != "WHERE" && tokens[i].word != "ORDER" {
		p.alias, ok = editIdentifier(text, tokens[i])
		if !ok {
			return nil
		}
		i++
	}
	if i < len(tokens) {
		if tokens[i].word == "WHERE" {
			if i+1 == len(tokens) {
				return nil
			}
		} else if tokens[i].word != "ORDER" || i+2 >= len(tokens) || tokens[i+1].word != "BY" {
			return nil
		}
	}
	rowIDs := 0
	for start := 1; start < from; {
		end := start
		for end < from && tokens[end].word != "," {
			end++
		}
		part := tokens[start:end]
		if len(part) == 0 {
			return nil
		}
		// Strip a column alias but never use it to resolve a physical column.
		base := part
		for j, t := range part {
			if t.word == "AS" {
				if j != len(part)-2 {
					return nil
				}
				if _, ok := editIdentifier(text, part[j+1]); !ok {
					return nil
				}
				base = part[:j]
				break
			}
		}
		if len(base) == 2 || len(base) == 4 {
			if _, ok := editIdentifier(text, base[len(base)-1]); !ok {
				return nil
			}
			base = base[:len(base)-1]
		}
		c := selectEditColumn{}
		if len(base) == 3 && base[1].word == "." {
			c.qualifier, ok = editIdentifier(text, base[0])
			if !ok {
				return nil
			}
			expected := p.table
			if p.alias != "" {
				expected = p.alias
			}
			if c.qualifier != expected {
				return nil
			}
			base = base[2:]
		}
		if len(base) != 1 {
			return nil
		}
		if base[0].word == "*" {
			if len(part) != 1 && !(len(part) == 3 && part[1].word == ".") {
				return nil
			}
			c.star = true
		} else {
			c.name, ok = editIdentifier(text, base[0])
			if !ok {
				return nil
			}
			c.rowID = base[0].word == "ROWID"
			if c.rowID {
				rowIDs++
			}
		}
		p.columns = append(p.columns, c)
		if end == from {
			break
		}
		start = end + 1
		if start == from {
			return nil
		}
	}
	if rowIDs != 1 {
		return nil
	}
	return p
}

// Metadata failure only leaves a SELECT read-only; it never guesses a target.
// Resolve before opening the live result cursor on the same pinned connection.
func (s *Session) prepareSelectEdit(ctx context.Context, text string) *editSnapshot {
	p := parseSelectEdit(text)
	if p == nil {
		return nil
	}
	if p.owner == "" {
		if err := s.tx.QueryRowContext(ctx, "SELECT SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA') FROM dual").Scan(&p.owner); err != nil {
			return nil
		}
	}
	var count int
	err := s.tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM all_tables WHERE owner=:1 AND table_name=:2 AND iot_type IS NULL AND nested='NO' AND temporary='N'`, p.owner, p.table).Scan(&count)
	if err != nil || count != 1 {
		return nil
	}
	object := Object{Owner: p.owner, Name: p.table, Kind: "TABLE"}
	columns, err := s.Columns(ctx, object)
	if err != nil || len(columns) == 0 {
		return nil
	}
	byName := make(map[string]Column)
	for _, c := range columns {
		byName[c.Name] = c
	}
	snap := &editSnapshot{object: object, rowIDColumn: -1}
	seen := make(map[string]bool)
	add := func(c Column) bool {
		if seen[c.Name] {
			return false
		}
		seen[c.Name] = true
		c.DataType = normalizeEditType(c.DataType)
		if !editableType(c.DataType) {
			c.Name = ""
		}
		snap.columns = append(snap.columns, c)
		return true
	}
	for _, c := range p.columns {
		if c.rowID {
			snap.rowIDColumn = len(snap.columns)
			snap.columns = append(snap.columns, Column{})
		} else if c.star {
			for _, column := range columns {
				if !add(column) {
					return nil
				}
			}
		} else {
			column, found := byName[c.name]
			if !found || !add(column) {
				return nil
			}
		}
	}
	if len(snap.columns) > 65 {
		return nil
	}
	writeable := false
	for _, c := range snap.columns {
		writeable = writeable || c.Name != ""
	}
	if !writeable {
		return nil
	}
	s.editSequence++
	snap.id = s.editSequence
	return snap
}

func normalizeEditType(kind string) string {
	if !strings.Contains(kind, "TIME ZONE") {
		kind, _, _ = strings.Cut(kind, "(")
	}
	return kind
}

func (snap *editSnapshot) appendSelectRow(cells []Cell) bool {
	if len(cells) != len(snap.columns) {
		return false
	}
	rid := cells[snap.rowIDColumn]
	if rid.Null || rid.Partial || len(rid.Text) != 18 {
		return false
	}
	for _, c := range rid.Text {
		if !strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/", c) {
			return false
		}
	}
	row := append([]Cell(nil), cells...)
	for i, c := range snap.columns {
		if c.Name == "" {
			continue
		}
		if row[i].Partial || row[i].LOBID != 0 {
			return false
		}
		row[i].Type = c.DataType
		cells[i].Type = c.DataType
		if !row[i].Null && (c.DataType == "DATE" || c.DataType == "TIMESTAMP") {
			date, err := time.Parse("2006-01-02 15:04:05.999999999Z07:00", row[i].Text)
			if err != nil {
				date, err = time.Parse("2006-01-02 15:04:05.999999999", row[i].Text)
			}
			if err == nil {
				// Match the existing TO_CHAR(...FF9) conflict predicate exactly.
				layout := "2006-01-02 15:04:05.000000000"
				if c.DataType == "DATE" {
					layout = "2006-01-02 15:04:05"
				}
				row[i].Text = date.Format(layout)
			}
		}
		if c.DataType == "RAW" {
			row[i].Text = strings.ToUpper(row[i].Text)
		}
		if _, err := editValue(c.DataType, row[i].Text, row[i].Null); err != nil {
			return false
		}
	}
	snap.rowIDs = append(snap.rowIDs, rid.Text)
	snap.rows = append(snap.rows, row)
	return true
}

func (snap *editSnapshot) markSelectResult(result *Result) {
	if snap == nil || len(snap.rows) == 0 {
		return
	}
	result.EditID, result.EditObject = snap.id, snap.object
	result.EditRowIDColumn = snap.rowIDColumn + 1
	result.EditRowIDs = append([]string(nil), snap.rowIDs[len(snap.rowIDs)-len(result.Rows):]...)
	result.EditColumnNames = make([]string, len(snap.columns))
	result.EditColumns = make([]bool, len(snap.columns))
	for i, c := range snap.columns {
		result.EditColumns[i] = c.Name != ""
		result.EditColumnNames[i] = c.Name
	}
}

func (r Result) CanEditColumn(column int) bool {
	return r.EditID != 0 && column >= 0 && column < len(r.Columns) && (r.EditColumns == nil || column < len(r.EditColumns) && r.EditColumns[column])
}
