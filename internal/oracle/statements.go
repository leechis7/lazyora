package oracle

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Statement offsets are UTF-8 byte offsets into the original editor text.
type Statement struct {
	Text             string
	Start, End, Line int
}

type sqlToken struct {
	word       string
	start, end int
}

// sqlTokens retains source positions while ignoring comments. Quotes, including
// Oracle q/nq literals, are single opaque tokens and never statement separators.
func sqlTokens(text string) ([]sqlToken, error) {
	var out []sqlToken
	for i := 0; i < len(text); {
		start := i
		r, size := utf8.DecodeRuneInString(text[i:])
		if unicode.IsSpace(r) {
			i += size
			continue
		}
		if strings.HasPrefix(text[i:], "--") {
			if n := strings.IndexByte(text[i:], '\n'); n >= 0 {
				i += n + 1
			} else {
				i = len(text)
			}
			continue
		}
		if strings.HasPrefix(text[i:], "/*") {
			n := strings.Index(text[i+2:], "*/")
			if n < 0 {
				return nil, fmt.Errorf("unterminated comment at byte %d", i)
			}
			i += n + 4
			continue
		}
		q := i
		if (r == 'n' || r == 'N') && i+1 < len(text) && (text[i+1] == 'q' || text[i+1] == 'Q') {
			q++
		}
		if q+2 < len(text) && (text[q] == 'q' || text[q] == 'Q') && text[q+1] == '\'' {
			d, ds := utf8.DecodeRuneInString(text[q+2:])
			end := d
			switch d {
			case '[':
				end = ']'
			case '{':
				end = '}'
			case '(':
				end = ')'
			case '<':
				end = '>'
			}
			i = q + 2 + ds
			closing := string(end) + "'"
			n := strings.Index(text[i:], closing)
			if n < 0 {
				return nil, fmt.Errorf("unterminated q literal at byte %d", start)
			}
			i += n + len(closing)
			out = append(out, sqlToken{"", start, i})
			continue
		}
		if r == '\'' || r == '"' {
			i += size
			closed := false
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
				closed = true
				break
			}
			if !closed {
				return nil, fmt.Errorf("unterminated quote at byte %d", start)
			}
			out = append(out, sqlToken{"", start, i})
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("_$#", r) {
			i += size
			for i < len(text) {
				r, size = utf8.DecodeRuneInString(text[i:])
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("_$#", r) {
					break
				}
				i += size
			}
			out = append(out, sqlToken{strings.ToUpper(text[start:i]), start, i})
			continue
		}
		i += size
		out = append(out, sqlToken{string(r), start, i})
	}
	return out, nil
}

func plsqlStatement(tokens []sqlToken) bool {
	if len(tokens) == 0 {
		return false
	}
	if tokens[0].word == "BEGIN" || tokens[0].word == "DECLARE" {
		return true
	}
	if tokens[0].word != "CREATE" {
		return false
	}
	for _, t := range tokens[1:] {
		switch t.word {
		case "OR", "REPLACE", "EDITIONABLE", "NONEDITIONABLE":
			continue
		case "FUNCTION", "PROCEDURE", "PACKAGE", "TRIGGER", "TYPE":
			return true
		default:
			return false
		}
	}
	return false
}

func slashLine(line string) bool {
	if n := strings.Index(line, "--"); n >= 0 {
		line = line[:n]
	}
	return strings.TrimSpace(line) == "/"
}

// Statements splits SQL at semicolons and PL/SQL at a slash on its own line.
// A final PL/SQL unit may omit slash. SQL*Plus directives are not interpreted.
func Statements(text string) ([]Statement, error) {
	tokens, err := sqlTokens(text)
	if err != nil {
		return nil, err
	}
	var statements []Statement
	start, first, depth := 0, 0, 0
	line, lineStart, lineEnd, firstLine := 1, 0, 0, 1
	slashChecked, standaloneSlash := false, false
	nextLineEnd := func(start int) int {
		if n := strings.IndexByte(text[start:], '\n'); n >= 0 {
			return start + n
		}
		return len(text)
	}
	lineEnd = nextLineEnd(0)
	appendStatement := func(end int, part []sqlToken) {
		if len(part) == 0 || (len(part) == 1 && part[0].word == ";") {
			return
		}
		statements = append(statements, Statement{strings.TrimSpace(text[start:end]), start, end, firstLine})
	}
	for i, t := range tokens {
		// Token positions increase. Visit each line only once, including lines
		// inside comments and quoted values, instead of rescanning every prefix.
		for t.start > lineEnd {
			lineStart = lineEnd + 1
			lineEnd = nextLineEnd(lineStart)
			line++
			slashChecked = false
		}
		if i == first {
			firstLine = line
		}
		if t.word == "/" && !slashChecked {
			// A long division expression may contain thousands of slash tokens
			// on one line. Classify that line at most once.
			standaloneSlash = slashLine(text[lineStart:lineEnd])
			slashChecked = true
		}
		if t.word == "/" && standaloneSlash {
			if depth != 0 {
				return nil, fmt.Errorf("unbalanced parentheses before line %d", line)
			}
			appendStatement(t.start, tokens[first:i])
			start = t.end
			first = i + 1
			continue
		}
		if t.word == "(" {
			depth++
		}
		if t.word == ")" {
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("unbalanced parentheses at byte %d", t.start)
			}
		}
		if t.word == ";" && !plsqlStatement(tokens[first:i+1]) {
			if depth != 0 {
				return nil, fmt.Errorf("semicolon inside unclosed parentheses at byte %d", t.start)
			}
			appendStatement(t.end, tokens[first:i+1])
			start = t.end
			first = i + 1
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("unbalanced parentheses")
	}
	appendStatement(len(text), tokens[first:])
	return statements, nil
}

// AtStatement chooses the following statement in a gap and the preceding one
// after the last statement. Ambiguous/unterminated input is never sent to DB.
func AtStatement(text string, offset int) (string, error) {
	statements, err := Statements(text)
	if err != nil {
		return "", err
	}
	if len(statements) == 0 {
		return "", fmt.Errorf("enter a SQL statement")
	}
	for _, statement := range statements {
		if offset < statement.End {
			return statement.Text, nil
		}
	}
	return statements[len(statements)-1].Text, nil
}
