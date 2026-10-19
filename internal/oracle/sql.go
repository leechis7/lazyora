package oracle

import (
	"errors"
	"strings"
	"unicode"
)

// Prepare accepts one statement. It preserves semicolons within PL/SQL blocks.
func Prepare(text string) (string, bool, error) {
	text = strings.TrimSpace(text)
	if strings.HasSuffix(text, "\n/") {
		text = strings.TrimSpace(strings.TrimSuffix(text, "/"))
	}
	first, err := leadingWord(text)
	if err != nil {
		return "", false, err
	}
	tokens, tokenErr := sqlTokens(text)
	if tokenErr != nil {
		return "", false, tokenErr
	}
	plsql := plsqlStatement(tokens)
	if !plsql {
		text = strings.TrimSpace(strings.TrimSuffix(text, ";"))
	}
	return text, first == "SELECT" || first == "WITH", nil
}

func leadingWord(text string) (string, error) {
	for {
		text = strings.TrimSpace(text)
		if strings.HasPrefix(text, "--") {
			end := strings.IndexByte(text, '\n')
			if end < 0 {
				return "", errors.New("enter a SQL statement")
			}
			text = text[end+1:]
			continue
		}
		if strings.HasPrefix(text, "/*") {
			end := strings.Index(text[2:], "*/")
			if end < 0 {
				return "", errors.New("unterminated SQL comment")
			}
			text = text[end+4:]
			continue
		}
		break
	}
	end := strings.IndexFunc(text, func(r rune) bool { return !unicode.IsLetter(r) })
	if end < 0 {
		end = len(text)
	}
	if end == 0 {
		return "", errors.New("enter a SQL statement")
	}
	return strings.ToUpper(text[:end]), nil
}
