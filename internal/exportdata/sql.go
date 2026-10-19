package exportdata

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"lazyora/internal/oracle"
)

var numericLiteral = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`)

func quoteIdentifier(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }
func targetIdentifier(name string) (string, error) {
	name = strings.TrimSpace(name)
	if !utf8.ValidString(name) || strings.IndexByte(name, 0) >= 0 {
		return "", fmt.Errorf("invalid target identifier")
	}
	if name == "" {
		return "", fmt.Errorf("enter a target table/schema name")
	}
	if strings.HasPrefix(name, `"`) && strings.HasSuffix(name, `"`) && len(name) >= 2 {
		inner := strings.ReplaceAll(name[1:len(name)-1], `""`, `"`)
		if strings.ReplaceAll(inner, `"`, `""`) != name[1:len(name)-1] || inner == "" {
			return "", fmt.Errorf("invalid quoted identifier")
		}
		return quoteIdentifier(inner), nil
	}
	for i, r := range name {
		if !unicode.IsLetter(r) && (i == 0 || (!unicode.IsDigit(r) && !strings.ContainsRune("_$#", r))) {
			return "", fmt.Errorf("invalid unquoted identifier; use double quotes for exact names")
		}
	}
	return quoteIdentifier(strings.ToUpper(name)), nil
}

func SQLLiteral(cell oracle.Cell) (string, error) {
	if cell.Null {
		return "NULL", nil
	}
	if !utf8.ValidString(cell.Text) || strings.IndexByte(cell.Text, 0) >= 0 {
		return "", fmt.Errorf("invalid text for INSERT SQL")
	}
	kind := strings.ToUpper(cell.Type)
	switch {
	case kind == "NUMBER" || kind == "FLOAT" || strings.Contains(kind, "BINARY_FLOAT") || strings.Contains(kind, "BINARY_DOUBLE"):
		if !numericLiteral.MatchString(cell.Text) {
			return "", fmt.Errorf("invalid numeric literal")
		}
		return cell.Text, nil
	case kind == "DATE" || strings.Contains(kind, "TIMESTAMP"):
		var value time.Time
		var err error
		for _, layout := range []string{"2006-01-02 15:04:05.999999999Z07:00", time.RFC3339Nano, "2006-01-02 15:04:05.999999999", "2006-01-02"} {
			value, err = time.Parse(layout, cell.Text)
			if err == nil {
				break
			}
		}
		if err != nil {
			return "", fmt.Errorf("unsupported date/time representation")
		}
		if kind == "DATE" {
			return "TO_DATE('" + value.Format("2006-01-02 15:04:05") + "','YYYY-MM-DD HH24:MI:SS')", nil
		}
		if strings.Contains(kind, "TIME ZONE") || strings.Contains(kind, "TIMESTAMPTZ") || strings.Contains(kind, "TIMESTAMPLTZ") {
			return "TO_TIMESTAMP_TZ('" + value.Format("2006-01-02 15:04:05.000000000 -07:00") + "','YYYY-MM-DD HH24:MI:SS.FF9 TZH:TZM')", nil
		}
		return "TO_TIMESTAMP('" + value.Format("2006-01-02 15:04:05.000000000") + "','YYYY-MM-DD HH24:MI:SS.FF9')", nil
	case kind == "RAW":
		if len(cell.Text) > 4000 {
			return "", fmt.Errorf("RAW literal exceeds Oracle 10g limit")
		}
		if _, err := hex.DecodeString(cell.Text); err != nil {
			return "", fmt.Errorf("invalid RAW hex")
		}
		return "HEXTORAW('" + cell.Text + "')", nil
	case kind == "" || strings.Contains(kind, "CHAR") || kind == "LONG":
		if len(cell.Text) > 4000 {
			return "", fmt.Errorf("string literal exceeds Oracle 10g 4000-byte limit")
		}
		prefix := ""
		if strings.HasPrefix(kind, "N") {
			prefix = "N"
		}
		return prefix + "'" + strings.ReplaceAll(cell.Text, "'", "''") + "'", nil
	default:
		return "", fmt.Errorf("unsupported INSERT data type %q", cell.Type)
	}
}

func writeSQL(ctx context.Context, w io.Writer, data Dataset, options Options) error {
	table, err := targetIdentifier(options.Table)
	if err != nil {
		return err
	}
	if options.Schema != "" {
		schema, err := targetIdentifier(options.Schema)
		if err != nil {
			return err
		}
		table = schema + "." + table
	}
	columns := options.Columns
	if len(columns) == 0 {
		columns = data.Columns
	}
	if len(columns) != len(data.Columns) {
		return fmt.Errorf("target column count must match exported columns")
	}
	quoted := make([]string, len(columns))
	seen := map[string]bool{}
	for i, name := range columns {
		if name == "" || strings.IndexByte(name, 0) >= 0 || !utf8.ValidString(name) || seen[name] {
			return fmt.Errorf("empty, duplicate or invalid target column")
		}
		seen[name] = true
		quoted[i] = quoteIdentifier(name)
	}
	if _, err = io.WriteString(w, "-- Generated INSERT statements only; no COMMIT. Review target names before executing.\n"); err != nil {
		return err
	}
	for row, values := range data.Rows {
		if err = ctx.Err(); err != nil {
			return err
		}
		literals := make([]string, len(values))
		for col, value := range values {
			literals[col], err = SQLLiteral(value)
			if err != nil {
				return fmt.Errorf("row %d column %d: %w", row+1, col+1, err)
			}
		}
		if _, err = fmt.Fprintf(w, "INSERT INTO %s (%s) VALUES (%s);\n", table, strings.Join(quoted, ", "), strings.Join(literals, ", ")); err != nil {
			return err
		}
	}
	return nil
}
