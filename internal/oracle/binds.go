package oracle

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sijms/go-ora/v3/types"
)

const MaxBindVariables = 32

// Only IN values are supported. Text is never interpolated into SQL or written
// to history/configuration; the driver receives typed named arguments.
type BindValue struct {
	Kind, Text string
	Null       bool
}

var numberPattern = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`)
var bindNamePattern = regexp.MustCompile(`^[A-Z0-9_$#]+$`)

func (v BindValue) Value() (any, error) {
	if v.Null {
		return nil, nil
	}
	switch v.Kind {
	case "", "text":
		return v.Text, nil
	case "number":
		if !numberPattern.MatchString(v.Text) || len(v.Text) > 128 {
			return nil, fmt.Errorf("invalid NUMBER")
		}
		normalized := strings.TrimPrefix(strings.ToLower(v.Text), "+")
		mantissa, exponentText, hasExponent := strings.Cut(strings.TrimPrefix(normalized, "-"), "e")
		exponent := 0
		if hasExponent {
			var err error
			exponent, err = strconv.Atoi(exponentText)
			if err != nil || exponent < -256 || exponent > 256 {
				return nil, fmt.Errorf("NUMBER exponent out of range")
			}
		}
		whole, _, _ := strings.Cut(mantissa, ".")
		digits := strings.ReplaceAll(mantissa, ".", "")
		first := strings.IndexFunc(digits, func(r rune) bool { return r != '0' })
		if first < 0 {
			normalized = "0"
		} else {
			decimalExponent := exponent + len(whole) - first - 1
			if decimalExponent < -130 || decimalExponent >= 126 || len(strings.Trim(digits, "0")) > 38 {
				return nil, fmt.Errorf("NUMBER exceeds Oracle precision/range (38 digits)")
			}
		}
		value, err := types.NewNumber(normalized)
		if err != nil {
			return nil, fmt.Errorf("invalid NUMBER")
		}
		return value, nil
	case "date":
		value, err := time.Parse("2006-01-02", v.Text)
		if err != nil {
			return nil, fmt.Errorf("invalid DATE; use YYYY-MM-DD")
		}
		return value, nil
	case "timestamp":
		value, err := time.Parse("2006-01-02 15:04:05.999999999", v.Text)
		if err != nil {
			return nil, fmt.Errorf("invalid TIMESTAMP; use YYYY-MM-DD HH:MM:SS[.fraction]")
		}
		return types.NewTimeStamp(value), nil
	default:
		return nil, fmt.Errorf("unsupported bind type")
	}
}

type bindOccurrence struct {
	name       string
	start, end int
}

func bindOccurrences(text string) ([]bindOccurrence, error) {
	tokens, err := sqlTokens(text)
	if err != nil {
		return nil, err
	}
	var occurrences []bindOccurrence
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		if t.word != ":" {
			continue
		}
		if i+1 < len(tokens) && tokens[i+1].word == "=" {
			continue
		}
		if i+1 >= len(tokens) || tokens[i+1].start != t.end || !bindNamePattern.MatchString(tokens[i+1].word) {
			return nil, fmt.Errorf("bind variables must use an unquoted :name or :number")
		}
		if i > 0 && tokens[i-1].word == "INTO" {
			return nil, fmt.Errorf("OUT / INTO bind variables are not supported")
		}
		i++
		if i+2 < len(tokens) && tokens[i+1].word == ":" && tokens[i+2].word == "=" {
			return nil, fmt.Errorf("OUT assignment bind variables are not supported")
		}
		occurrences = append(occurrences, bindOccurrence{tokens[i].word, t.start, tokens[i].end})
	}
	return occurrences, nil
}

func BindNames(text string) ([]string, error) {
	occurrences, err := bindOccurrences(text)
	if err != nil {
		return nil, err
	}
	var names []string
	seen := map[string]bool{}
	for _, o := range occurrences {
		if !seen[o.name] {
			names = append(names, o.name)
			seen[o.name] = true
		}
	}
	if len(names) > MaxBindVariables {
		return nil, fmt.Errorf("at most %d distinct bind variables are supported", MaxBindVariables)
	}
	return names, nil
}

// BindArguments renames placeholders, never values. Generated alphabetic names
// also support Oracle numeric binds through database/sql's named-argument API.
func BindArguments(text string, values map[string]BindValue) (string, []any, error) {
	occurrences, err := bindOccurrences(text)
	if err != nil {
		return "", nil, err
	}
	names, err := BindNames(text)
	if err != nil {
		return "", nil, err
	}
	aliases := map[string]string{}
	var args []any
	for i, name := range names {
		input, ok := values[name]
		if !ok {
			return "", nil, fmt.Errorf("missing IN bind :%s", name)
		}
		value, err := input.Value()
		if err != nil {
			return "", nil, fmt.Errorf(":%s: %w", name, err)
		}
		alias := fmt.Sprintf("LAZYORA_B%d", i+1)
		aliases[name] = alias
		args = append(args, sql.Named(alias, value))
	}
	var result strings.Builder
	start := 0
	for _, o := range occurrences {
		result.WriteString(text[start:o.start])
		result.WriteString(":" + aliases[o.name])
		start = o.end
	}
	result.WriteString(text[start:])
	return result.String(), args, nil
}
