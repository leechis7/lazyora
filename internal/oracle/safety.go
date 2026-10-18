package oracle

import (
	"strings"
	"unicode"
)

// SQLRisk is a conservative pre-execution hint, not an Oracle SQL parser or an
// estimate of affected rows. In particular, a WHERE clause need not be selective.
type SQLRisk string

const (
	RiskNone          SQLRisk = ""
	RiskDDL           SQLRisk = "ddl"
	RiskUnfilteredDML SQLRisk = "unfiltered_dml"
	RiskUncertainDML  SQLRisk = "uncertain_dml"
)

// ExecutionRisk checks standalone DDL and UPDATE/DELETE statements. It does not
// inspect PL/SQL, dynamically generated SQL, procedures, MERGE or predicate truth.
func ExecutionRisk(text string) SQLRisk {
	first, err := leadingWord(text)
	if err != nil {
		return RiskNone
	}
	switch first {
	case "ALTER", "ANALYZE", "ASSOCIATE", "AUDIT", "COMMENT", "CREATE", "DISASSOCIATE", "DROP", "FLASHBACK", "GRANT", "NOAUDIT", "PURGE", "RENAME", "REVOKE", "TRUNCATE", "UNDROP":
		return RiskDDL
	case "UPDATE", "DELETE":
		tokens, ok := safetyTokens(text)
		if !ok {
			return RiskUncertainDML
		}
		for _, token := range tokens {
			if token.depth == 0 && token.word == "WHERE" {
				return RiskNone
			}
		}
		return RiskUnfilteredDML
	}
	return RiskNone
}

type safetyToken struct {
	word  string
	depth int
}

// safetyTokens ignores comments and quoted values/identifiers, including Oracle
// q/nq quoting. Parenthesis depth keeps a subquery's WHERE out of the outer DML.
func safetyTokens(text string) ([]safetyToken, bool) {
	runes := []rune(text)
	var tokens []safetyToken
	depth := 0
	for i := 0; i < len(runes); {
		r := runes[i]
		if unicode.IsSpace(r) {
			i++
			continue
		}
		if r == '-' && i+1 < len(runes) && runes[i+1] == '-' {
			i += 2
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
			continue
		}
		if r == '/' && i+1 < len(runes) && runes[i+1] == '*' {
			i += 2
			for i+1 < len(runes) && !(runes[i] == '*' && runes[i+1] == '/') {
				i++
			}
			if i+1 >= len(runes) {
				return tokens, false
			}
			i += 2
			continue
		}
		q := i
		if (r == 'n' || r == 'N') && i+1 < len(runes) && (runes[i+1] == 'q' || runes[i+1] == 'Q') {
			q++
		}
		if (runes[q] == 'q' || runes[q] == 'Q') && q+2 < len(runes) && runes[q+1] == '\'' {
			delimiter := runes[q+2]
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
			i = q + 3
			for i+1 < len(runes) && !(runes[i] == delimiter && runes[i+1] == '\'') {
				i++
			}
			if i+1 >= len(runes) {
				return tokens, false
			}
			i += 2
			continue
		}
		if r == '\'' || r == '"' {
			quote := r
			i++
			closed := false
			for i < len(runes) {
				if runes[i] != quote {
					i++
					continue
				}
				if i+1 < len(runes) && runes[i+1] == quote {
					i += 2
					continue
				}
				i++
				closed = true
				break
			}
			if !closed {
				return tokens, false
			}
			continue
		}
		if r == ':' {
			i++
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || strings.ContainsRune("_$#", runes[i])) {
				i++
			}
			continue
		}
		if r == '(' {
			depth++
			i++
			continue
		}
		if r == ')' {
			depth--
			if depth < 0 {
				return tokens, false
			}
			i++
			continue
		}
		if r == ';' { // Scripts are unsupported: don't let a later statement hide a missing WHERE.
			if strings.TrimSpace(string(runes[i+1:])) != "" {
				return tokens, false
			}
			i++
			continue
		}
		if unicode.IsLetter(r) || r == '_' || unicode.IsDigit(r) {
			start := i
			i++
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || strings.ContainsRune("_$#", runes[i])) {
				i++
			}
			tokens = append(tokens, safetyToken{strings.ToUpper(string(runes[start:i])), depth})
			continue
		}
		i++
	}
	return tokens, depth == 0
}
