package oracle

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const MaxFormatBytes = 4 * 1024 * 1024
const maxFormattedBytes = 16 * 1024 * 1024
const maxFormatDepth = 64

type formatAtom struct {
	text                                    string
	start, end                              int
	comment, literal, identifier, delimiter bool
}

var formatNumber = regexp.MustCompile(`^(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?[fFdD]?`)
var formatOperators = []string{"(+)", ":=", "=>", "<=", ">=", "<>", "!=", "~=", "^=", "||", "**", "..", "<<", ">>"}

func formatAtoms(text string) ([]formatAtom, error) {
	tokens := lexCompletion(text, -1)
	for _, token := range tokens {
		if !token.closed && !strings.HasPrefix(token.text, "--") {
			return nil, fmt.Errorf("unterminated quote or comment at byte %d", token.start)
		}
	}
	atoms := make([]formatAtom, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		atom := formatAtom{text: t.text, start: t.start, end: t.end, identifier: t.identifier}
		atom.comment = t.opaque && (strings.HasPrefix(t.text, "--") || strings.HasPrefix(t.text, "/*"))
		atom.literal = t.opaque && !atom.comment || t.quoted
		// National literals are one lexical unit: adding a space after N changes
		// their meaning. q/nq literals and quoted identifiers are already opaque.
		if !atom.literal && !atom.comment && strings.EqualFold(t.text, "N") && i+1 < len(tokens) && tokens[i+1].start == t.end && strings.HasPrefix(tokens[i+1].text, "'") {
			i++
			atom.end = tokens[i].end
			atom.text = text[atom.start:atom.end]
			atom.literal = true
			atom.identifier = false
		} else if !atom.literal && !atom.comment {
			end := t.end
			if (t.text[0] >= '0' && t.text[0] <= '9') || t.text == "." {
				if number := formatNumber.FindString(text[t.start:]); number != "" {
					if strings.HasSuffix(number, ".") && t.start+len(number) < len(text) && text[t.start+len(number)] == '.' {
						number = strings.TrimSuffix(number, ".")
					}
					end = t.start + len(number)
				}
			}
			for _, op := range formatOperators {
				if strings.HasPrefix(text[t.start:], op) {
					end = t.start + len(op)
					break
				}
			}
			j := i
			for j+1 < len(tokens) && tokens[j+1].start == tokens[j].end && tokens[j+1].end <= end {
				j++
			}
			if tokens[j].end == end && j > i {
				i = j
				atom.end = end
				atom.text = text[atom.start:end]
			}
		}
		if atom.text == "/" {
			from := strings.LastIndexByte(text[:atom.start], '\n') + 1
			to := len(text)
			if n := strings.IndexByte(text[atom.end:], '\n'); n >= 0 {
				to = atom.end + n
			}
			atom.delimiter = strings.TrimSpace(text[from:to]) == "/"
		}
		atoms = append(atoms, atom)
	}
	return atoms, nil
}

type formatWriter struct {
	base          string
	b             strings.Builder
	level, indent int
	lineStart     bool
	previous      string
}

func (w *formatWriter) line(indent int) {
	if w.b.Len() > 0 && !w.lineStart {
		w.b.WriteByte('\n')
	}
	w.lineStart = true
	w.indent = max(0, indent)
}
func (w *formatWriter) emit(text string, tight bool) {
	if w.lineStart {
		if w.b.Len() > 0 {
			w.b.WriteString(w.base)
		}
		w.b.WriteString(strings.Repeat("  ", w.indent))
		w.lineStart = false
	} else if w.b.Len() > 0 && !tight {
		w.b.WriteByte(' ')
	}
	w.b.WriteString(text)
	w.previous = text
}

type formatBlock struct {
	kind              string
	base, restore     int
	exception, branch bool
}
type formatParen struct {
	multiline  bool
	level      int
	clause     string
	clauseBase int
}

// FormatSQL changes whitespace only. It is an editor aid, not an Oracle parser.
// The final lexical equivalence check rejects any output changing opaque data,
// compound operators, numeric literals, bind adjacency or script delimiters.
func FormatSQL(text string) (string, error) { return formatSQL(text, "") }

func formatSQL(text, base string) (string, error) {
	if len(text) > MaxFormatBytes {
		return "", fmt.Errorf("format selection must be at most %d MiB", MaxFormatBytes/(1024*1024))
	}
	atoms, err := formatAtoms(text)
	if err != nil {
		return "", err
	}
	if len(atoms) == 0 {
		return text, nil
	}
	w := formatWriter{base: base}
	w.lineStart = true
	var blocks []formatBlock
	var parens []formatParen
	clause := ""
	clauseBase := 0
	pendingHeader := false
	pendingIf := false
	pendingIfDepth := 0
	elsif := false
	withClause := false
	tableDDL := false
	previousWord := ""
	for i, a := range atoms {
		if w.b.Len() > maxFormattedBytes || w.level > maxFormatDepth || len(blocks)+len(parens) > maxFormatDepth {
			return "", fmt.Errorf("formatting exceeds size/nesting limit")
		}
		word := strings.ToUpper(a.text)
		keyword := !a.literal && !a.comment && a.identifier
		next := ""
		if i+1 < len(atoms) {
			next = strings.ToUpper(atoms[i+1].text)
		}
		if a.comment {
			w.emit(a.text, false)
			if strings.HasPrefix(a.text, "--") || i+1 < len(atoms) && strings.Contains(text[a.end:atoms[i+1].start], "\n") {
				w.line(w.level)
			}
			continue
		}
		if a.delimiter {
			w.line(0)
			w.emit(a.text, true)
			w.line(0)
			previousWord = ""
			continue
		}
		if keyword && (word == "FUNCTION" || word == "PROCEDURE" || word == "PACKAGE" || word == "TRIGGER") {
			pendingHeader = true
		}
		if keyword && word == "TABLE" && previousWord == "CREATE" {
			tableDDL = true
		}
		if keyword && (word == "AS" || word == "IS") && pendingHeader {
			w.emit(a.text, false)
			blocks = append(blocks, formatBlock{kind: "DECLARE", base: w.level, restore: w.level})
			w.level++
			w.line(w.level)
			pendingHeader = false
			previousWord = word
			continue
		}
		if keyword && word == "END" {
			if len(blocks) > 0 {
				block := blocks[len(blocks)-1]
				blocks = blocks[:len(blocks)-1]
				w.level = block.restore
				w.line(block.base)
			} else {
				w.line(w.level)
			}
			w.emit(a.text, true)
			previousWord = word
			continue
		}
		if keyword && previousWord != "END" {
			switch word {
			case "DECLARE", "BEGIN":
				if word == "BEGIN" && len(blocks) > 0 && blocks[len(blocks)-1].kind == "DECLARE" {
					w.level = blocks[len(blocks)-1].restore
					blocks = blocks[:len(blocks)-1]
				}
				w.line(w.level)
				w.emit(a.text, true)
				blocks = append(blocks, formatBlock{kind: word, base: w.level, restore: w.level})
				w.level++
				w.line(w.level)
				pendingHeader = false
				previousWord = word
				continue
			case "LOOP":
				w.emit(a.text, false)
				blocks = append(blocks, formatBlock{kind: "LOOP", base: w.level, restore: w.level})
				w.level++
				w.line(w.level)
				previousWord = word
				continue
			case "IF":
				pendingIf = true
				pendingIfDepth = len(blocks)
			case "CASE":
				w.emit(a.text, false)
				base := max(w.level, w.indent)
				blocks = append(blocks, formatBlock{kind: "CASE", base: base, restore: w.level})
				w.level = base + 1
				previousWord = word
				continue
			case "ELSIF":
				w.line(max(0, w.level-1))
				w.emit(a.text, true)
				pendingIf = true
				pendingIfDepth = len(blocks)
				elsif = true
				previousWord = word
				continue
			case "ELSE":
				if len(blocks) > 0 && blocks[len(blocks)-1].kind == "CASE" {
					w.line(w.level)
					w.emit(a.text, true)
				} else {
					w.line(max(0, w.level-1))
					w.emit(a.text, true)
					w.line(w.level)
				}
				previousWord = word
				continue
			case "EXCEPTION":
				w.line(max(0, w.level-1))
				w.emit(a.text, true)
				w.line(w.level)
				if len(blocks) > 0 {
					blocks[len(blocks)-1].exception = true
				}
				previousWord = word
				continue
			case "WHEN":
				if len(blocks) > 0 {
					block := &blocks[len(blocks)-1]
					if block.branch {
						w.level--
						block.branch = false
					}
					w.line(w.level)
				}
			case "THEN":
				if pendingIf && len(blocks) == pendingIfDepth {
					w.emit(a.text, false)
					if !elsif {
						blocks = append(blocks, formatBlock{kind: "IF", base: w.level, restore: w.level})
						w.level++
					}
					pendingIf = false
					elsif = false
					w.line(w.level)
					previousWord = word
					continue
				}
				if len(blocks) > 0 && blocks[len(blocks)-1].exception {
					w.emit(a.text, false)
					blocks[len(blocks)-1].branch = true
					w.level++
					w.line(w.level)
					previousWord = word
					continue
				}
			case "WITH":
				withClause = true
			}
		}
		structuralSQL := len(parens) == 0 || parens[len(parens)-1].multiline
		if keyword && structuralSQL && (word == "SELECT" || word == "FROM" || word == "WHERE" || word == "HAVING" || word == "JOIN" || word == "GROUP" || word == "ORDER" || word == "SET" || word == "UNION" || word == "INTERSECT" || word == "MINUS" || word == "FETCH" || word == "OFFSET") {
			// LEFT/RIGHT/INNER/... JOIN is one heading, and GROUP/ORDER's BY is inline.
			joinPrefix := word == "JOIN" && (previousWord == "LEFT" || previousWord == "RIGHT" || previousWord == "INNER" || previousWord == "OUTER" || previousWord == "FULL" || previousWord == "CROSS")
			if !joinPrefix {
				w.line(w.level)
			}
			clause = word
			clauseBase = w.level
			w.emit(a.text, !joinPrefix)
			if word == "SELECT" {
				w.line(w.level + 1)
			}
			previousWord = word
			continue
		}
		if keyword && (word == "LEFT" || word == "RIGHT" || word == "INNER" || word == "FULL" || word == "CROSS") && (next == "JOIN" || next == "OUTER") {
			w.line(w.level)
		}
		if keyword && (word == "AND" || word == "OR") && (clause == "WHERE" || clause == "HAVING" || clause == "JOIN") {
			w.line(clauseBase + 1)
		}
		tight := w.previous == "." || w.previous == "@" || w.previous == "%" || w.previous == ":" || w.previous == "(" || a.text == "." || a.text == "@" || a.text == "%"
		switch a.text {
		case "(":
			multi := next == "SELECT" || next == "WITH" || tableDDL
			if tableDDL {
				tableDDL = false
			}
			tight = !multi && (w.previous == ")" || i > 0 && atoms[i-1].identifier && previousWord != "IN" && previousWord != "VALUES" && previousWord != "EXISTS" && previousWord != "USING")
			w.emit(a.text, tight)
			parens = append(parens, formatParen{multiline: multi, level: w.level, clause: clause, clauseBase: clauseBase})
			if multi {
				w.level++
				clause = ""
				w.line(w.level)
			}
		case ")":
			if len(parens) > 0 {
				p := parens[len(parens)-1]
				parens = parens[:len(parens)-1]
				if p.multiline {
					w.level = p.level
					clause = p.clause
					clauseBase = p.clauseBase
					w.line(w.level)
				}
			}
			w.emit(a.text, true)
		case ",":
			w.emit(a.text, true)
			if len(parens) > 0 && parens[len(parens)-1].multiline {
				w.line(w.level)
			} else if len(parens) == 0 && (clause == "SELECT" || clause == "SET" || clause == "ORDER") {
				w.line(clauseBase + 1)
			} else if len(parens) == 0 && withClause {
				w.line(w.level)
			}
		case ";":
			w.emit(a.text, true)
			w.line(w.level)
			clause = ""
			pendingHeader = false
			withClause = false
		default:
			w.emit(a.text, tight)
		}
		previousWord = word
	}
	if w.level > maxFormatDepth || len(blocks)+len(parens) > maxFormatDepth {
		return "", fmt.Errorf("formatting exceeds nesting limit")
	}
	formatted := strings.TrimRight(w.b.String(), "\n")
	if strings.HasSuffix(text, "\n") || strings.HasPrefix(atoms[len(atoms)-1].text, "--") {
		formatted += "\n"
	}
	if len(formatted) > maxFormattedBytes {
		return "", fmt.Errorf("formatted output is too large")
	}
	check, err := formatAtoms(formatted)
	if err != nil {
		return "", fmt.Errorf("cannot preserve SQL tokens: %w", err)
	}
	if len(check) != len(atoms) {
		return "", fmt.Errorf("formatting would change SQL tokens")
	}
	for i, a := range atoms {
		if a.text != check[i].text || a.delimiter != check[i].delimiter {
			return "", fmt.Errorf("formatting would change SQL token at byte %d", a.start)
		}
	}
	// Bind names must retain their original adjacency to the colon. The raw
	// atom comparison above already preserves every name and value; do not
	// tokenize the entire file again just to reconstruct the same bind list.
	for i := 0; i+1 < len(atoms); i++ {
		if atoms[i].text == ":" && (atoms[i].end == atoms[i+1].start) != (check[i].end == check[i+1].start) {
			return "", fmt.Errorf("formatting would change bind adjacency")
		}
	}
	return formatted, nil
}

// FormatSQLRange returns only the replacement for [start,end). Boundaries may
// not split tokens or opaque text, so formatting part of a string/comment
// cannot silently change its value. Whitespace outside the code is preserved.
func FormatSQLRange(text string, start, end int) (string, error) {
	if start < 0 || end < start || end > len(text) || start < len(text) && !utf8.RuneStart(text[start]) || end < len(text) && !utf8.RuneStart(text[end]) {
		return "", fmt.Errorf("invalid formatting selection")
	}
	if end-start > MaxFormatBytes {
		return "", fmt.Errorf("format selection must be at most %d MiB", MaxFormatBytes/(1024*1024))
	}
	if start == end {
		return "", nil
	}
	tokens := lexEditorRange(text, -1, start, end)
	if len(tokens) == 0 {
		return text[start:end], nil
	}
	for _, t := range tokens {
		if start > t.start && start < t.end || end > t.start && end < t.end {
			return "", fmt.Errorf("selection cuts a SQL token or literal; select complete code")
		}
	}
	selected := text[start:end]
	bodyStart := tokens[0].start
	bodyEnd := tokens[len(tokens)-1].end
	leading := text[start:bodyStart]
	trailing := text[bodyEnd:end]
	lineStart := strings.LastIndexByte(text[:bodyStart], '\n') + 1
	base := text[lineStart:bodyStart]
	if strings.IndexFunc(base, func(r rune) bool { return r != ' ' && r != '\t' }) >= 0 {
		base = ""
	}
	formatted, err := formatSQL(text[bodyStart:bodyEnd], base)
	if err != nil {
		return "", err
	}
	// A generated newline after an EOF-style line comment must not accumulate
	// when the original line break is retained at the selection boundary.
	if strings.Contains(trailing, "\n") || end < len(text) && text[end] == '\n' {
		formatted = strings.TrimSuffix(formatted, "\n")
	}
	if strings.TrimSpace(selected) == "" {
		return selected, nil
	}
	if len(leading)+len(formatted)+len(trailing) > maxFormattedBytes {
		return "", fmt.Errorf("formatted output is too large")
	}
	return leading + formatted + trailing, nil
}

// FormattedCursor maps an editor byte offset by the unchanged lexical atoms.
func FormattedCursor(before, after string, offset int) int {
	original, e1 := formatAtoms(before)
	formatted, e2 := formatAtoms(after)
	if e1 != nil || e2 != nil || len(original) != len(formatted) {
		return min(offset, len(after))
	}
	for i, t := range original {
		if offset < t.start {
			return formatted[i].start
		}
		if offset <= t.end {
			return formatted[i].start + min(offset-t.start, len(formatted[i].text))
		}
	}
	return len(after)
}
