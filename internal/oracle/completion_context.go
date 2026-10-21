package oracle

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Completion ranges are UTF-8 byte offsets, matching the editor's Replace API.
// This tolerant lexer is for suggestions only and never validates/executes SQL.
type CompletionContext struct {
	Start, End                   int
	Prefix                       string
	Quoted, ObjectsOnly, Blocked bool
	Qualifier                    []CompletionIdentifier
	Sources                      []CompletionSource
}
type CompletionIdentifier struct {
	Name   string
	Quoted bool
}

func (i CompletionIdentifier) SQL() string { return quoteIdentifier(i.Name) }

type CompletionSource struct {
	Owner, Name, Alias CompletionIdentifier
	Derived            bool
}
type completionToken struct {
	text                               string
	start, end, scope                  int
	identifier, quoted, closed, opaque bool
}

func completionIdent(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("_$#", r)
}
func lexCompletion(text string, cursor int) []completionToken {
	return lexEditorRange(text, cursor, 0, len(text))
}
func lexEditorRange(text string, cursor, from, to int) []completionToken {
	var tokens []completionToken
	for i := 0; i < len(text); {
		start := i
		r, size := utf8.DecodeRuneInString(text[i:])
		if unicode.IsSpace(r) {
			i += size
			continue
		}
		token := completionToken{start: start, closed: true}
		switch {
		case strings.HasPrefix(text[i:], "--"):
			i = len(text)
			if n := strings.IndexByte(text[start:], '\n'); n >= 0 {
				i = start + n
				token.closed = true
			} else {
				token.closed = false
			}
			token.opaque = true
		case strings.HasPrefix(text[i:], "/*"):
			i = len(text)
			if n := strings.Index(text[start+2:], "*/"); n >= 0 {
				i = start + 2 + n + 2
			} else {
				token.closed = false
			}
			token.opaque = true
		default:
			q := i
			if (r == 'n' || r == 'N') && q+1 < len(text) && (text[q+1] == 'q' || text[q+1] == 'Q') {
				q++
			}
			switch {
			case q+2 < len(text) && (text[q] == 'q' || text[q] == 'Q') && text[q+1] == '\'':
				delimiter, ds := utf8.DecodeRuneInString(text[q+2:])
				switch delimiter {
				case '[':
					delimiter = ']'
				case '{':
					delimiter = '}'
				case '(':
					delimiter = ')'
				case '<':
					delimiter = '>'
				}
				closing := string(delimiter) + "'"
				i = len(text)
				if n := strings.Index(text[q+2+ds:], closing); n >= 0 {
					i = q + 2 + ds + n + len(closing)
				} else {
					token.closed = false
				}
				token.opaque = true
			case r == '\'' || r == '"':
				i += size
				token.closed = false
				for i < len(text) {
					if text[i] != byte(r) {
						i++
						continue
					}
					i++
					if i < len(text) && text[i] == byte(r) {
						i++
						continue
					}
					token.closed = true
					break
				}
				token.identifier = r == '"'
				token.quoted = r == '"'
				token.opaque = r == '\''
			case completionIdent(r):
				i += size
				for i < len(text) {
					r, size = utf8.DecodeRuneInString(text[i:])
					if !completionIdent(r) {
						break
					}
					i += size
				}
				token.identifier = true
			default:
				i += size
			}
		}
		token.end = i
		token.text = text[start:i]
		if cursor >= 0 && token.text == ";" && !token.opaque && !token.quoted {
			if token.end <= cursor {
				tokens = tokens[:0]
				continue
			}
			if token.start >= cursor {
				break
			}
		}
		if token.start >= to {
			break
		}
		if token.end > from {
			tokens = append(tokens, token)
		}
	}
	return tokens
}
func completionID(t completionToken) CompletionIdentifier {
	if t.quoted {
		name := strings.TrimPrefix(t.text, `"`)
		if t.closed {
			name = strings.TrimSuffix(name, `"`)
		}
		return CompletionIdentifier{Name: strings.ReplaceAll(name, `""`, `"`), Quoted: true}
	}
	return CompletionIdentifier{Name: strings.ToUpper(t.text)}
}
func completionKeyword(t completionToken, word string) bool {
	return !t.quoted && strings.EqualFold(t.text, word)
}

var completionStops = map[string]bool{}

func init() {
	for _, s := range strings.Fields("WHERE GROUP ORDER HAVING CONNECT START MODEL UNION INTERSECT MINUS SET VALUES RETURNING JOIN INNER LEFT RIGHT FULL OUTER CROSS ON USING FETCH OFFSET FOR PARTITION PIVOT UNPIVOT WHEN THEN ELSE END AS") {
		completionStops[s] = true
	}
}
func completionAlias(t completionToken) bool {
	return t.identifier && (t.quoted || !completionStops[strings.ToUpper(t.text)])
}

func SQLCompletionContext(text string, cursor int) (CompletionContext, error) {
	c := CompletionContext{Start: cursor, End: cursor}
	if cursor < 0 || cursor > len(text) || cursor < len(text) && !utf8.RuneStart(text[cursor]) {
		return c, fmt.Errorf("invalid completion cursor")
	}
	tokens := lexCompletion(text, cursor)
	// Choose one statement. A semicolon in quotes/comments is opaque.
	first, last := 0, len(tokens)
	for i, t := range tokens {
		if t.text == ";" && !t.opaque && !t.quoted {
			if t.end <= cursor {
				first = i + 1
			} else if t.start >= cursor {
				last = i
				break
			}
		}
	}
	tokens = tokens[first:last]
	scopes := []int{0}
	parents := map[int]int{}
	nextScope := 0
	activeScope := 0
	for i := range tokens {
		t := &tokens[i]
		if t.text == ")" && !t.opaque && !t.quoted && len(scopes) > 1 {
			scopes = scopes[:len(scopes)-1]
		}
		t.scope = scopes[len(scopes)-1]
		if t.start < cursor {
			activeScope = t.scope
		}
		if t.text == "(" && !t.opaque && !t.quoted {
			nextScope++
			parents[nextScope] = t.scope
			scopes = append(scopes, nextScope)
			if t.end <= cursor {
				activeScope = nextScope
			}
		}
	}
	selected := -1
	for i, t := range tokens {
		if t.opaque && cursor > t.start && (cursor < t.end || cursor == t.end && !t.closed) {
			c.Blocked = true
			return c, nil
		}
		if t.identifier && cursor > t.start && cursor <= t.end {
			selected = i
			activeScope = t.scope
			break
		}
	}
	var scope []completionToken
	for _, t := range tokens {
		if t.scope == activeScope && !t.opaque {
			scope = append(scope, t)
		}
	}
	// UNION branches and sibling subqueries must not leak aliases to each other.
	first, last = 0, len(scope)
	for i, t := range scope {
		if completionKeyword(t, "UNION") || completionKeyword(t, "INTERSECT") || completionKeyword(t, "MINUS") {
			if t.end <= cursor {
				first = i + 1
			} else {
				last = i
				break
			}
		}
	}
	scope = scope[first:last]
	prefixIndex := len(scope)
	if selected >= 0 {
		t := tokens[selected]
		c.Start = t.start
		c.End = t.end
		c.Quoted = t.quoted
		if !t.closed {
			c.End = cursor
		}
		prefix := text[t.start:cursor]
		if t.quoted {
			prefix = strings.TrimPrefix(prefix, `"`)
			if t.closed && cursor == t.end {
				prefix = strings.TrimSuffix(prefix, `"`)
			}
			prefix = strings.ReplaceAll(prefix, `""`, `"`)
		}
		c.Prefix = prefix
	}
	for i, t := range scope {
		if t.start >= c.Start {
			prefixIndex = i
			break
		}
	}
	before := prefixIndex
	for before >= 2 && scope[before-1].text == "." && scope[before-2].identifier {
		c.Qualifier = append([]CompletionIdentifier{completionID(scope[before-2])}, c.Qualifier...)
		before -= 2
	}
	if len(c.Qualifier) > 2 {
		c.Blocked = true
		return c, nil
	}
	if before > 0 {
		word := strings.ToUpper(scope[before-1].text)
		if scope[before-1].text == ":" {
			c.Blocked = true
			return c, nil
		}
		c.ObjectsOnly = word == "FROM" || word == "JOIN" || word == "UPDATE" || word == "INTO" || word == "TABLE" || word == "VIEW"
	}
	// CTE definitions live in the current or an ancestor query block, including
	// definitions with explicit column lists. Do not mistake them for real tables.
	ancestors := map[int]bool{}
	for scopeID := activeScope; ; scopeID = parents[scopeID] {
		ancestors[scopeID] = true
		if scopeID == 0 {
			break
		}
	}
	grouped := map[int][]completionToken{}
	for _, t := range tokens {
		if ancestors[t.scope] && !t.opaque {
			grouped[t.scope] = append(grouped[t.scope], t)
		}
	}
	ctes := map[string]bool{}
	for _, group := range grouped {
		for i := 0; i < len(group); i++ {
			if !completionKeyword(group[i], "WITH") {
				continue
			}
			j := i + 1
			for j < len(group) && group[j].identifier {
				name := completionID(group[j]).Name
				j++
				if j+1 < len(group) && group[j].text == "(" && group[j+1].text == ")" {
					j += 2
				}
				if j+2 >= len(group) || !completionKeyword(group[j], "AS") || group[j+1].text != "(" || group[j+2].text != ")" {
					break
				}
				ctes[name] = true
				j += 3
				if j >= len(group) || group[j].text != "," {
					break
				}
				j++
			}
		}
	}
	inFrom := false
	statementKind := ""
	for i := 0; i < len(scope); i++ {
		t := scope[i]
		word := strings.ToUpper(t.text)
		if !t.quoted && (word == "SELECT" || word == "INSERT" || word == "MERGE" || word == "UPDATE" || word == "DELETE") {
			statementKind = word
		}
		begin := !t.quoted && (word == "FROM" || word == "JOIN" || word == "UPDATE" || word == "INTO" && (statementKind == "INSERT" || statementKind == "MERGE")) || inFrom && t.text == ","
		if !begin {
			if !t.quoted && completionStops[word] && word != "ON" && word != "USING" {
				inFrom = false
			}
			continue
		}
		inFrom = true
		i++
		if i >= len(scope) {
			break
		}
		source := CompletionSource{}
		if scope[i].text == "(" {
			source.Derived = true
			for i < len(scope) && scope[i].text != ")" {
				i++
			}
			if i >= len(scope) {
				continue
			}
		} else {
			if !scope[i].identifier {
				continue
			}
			source.Name = completionID(scope[i])
			if i+2 < len(scope) && scope[i+1].text == "." && scope[i+2].identifier {
				source.Owner = source.Name
				source.Name = completionID(scope[i+2])
				i += 2
			}
			source.Derived = source.Owner.Name == "" && ctes[source.Name.Name]
		}
		if i+1 < len(scope) && completionKeyword(scope[i+1], "AS") {
			i++
		}
		if i+1 < len(scope) && completionAlias(scope[i+1]) {
			i++
			source.Alias = completionID(scope[i])
		}
		if source.Name.Name != "" || source.Alias.Name != "" {
			c.Sources = append(c.Sources, source)
		}
	}
	return c, nil
}
