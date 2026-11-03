package ui

import (
	"sort"
	"sync"

	"github.com/leechis7/tuivory/go"
)

// Keep the existing display adapter's seven roles; the SDK owns classification.
type sqlColorKind = tuivory.SyntaxKind

const (
	sqlKeyword  = tuivory.SyntaxKeyword
	sqlFunction = tuivory.SyntaxFunction
	sqlType     = tuivory.SyntaxType
	sqlString   = tuivory.SyntaxString
	sqlNumber   = tuivory.SyntaxNumber
	sqlComment  = tuivory.SyntaxComment
	sqlBind     = tuivory.SyntaxParameter
)

type sqlColorSpan struct {
	start, end int
	kind       sqlColorKind
}

// Initialize optional SQL vocabulary only when a SQL editor needs highlighting.
var oracleHighlight = sync.OnceValue(tuivory.NewSQLAnalyzer)

// colorSQL adapts public semantic byte ranges to the retained editor's style
// callback. Execution parsing and dangerous-SQL decisions never use this result.
func colorSQL(text string, spans []sqlColorSpan) []sqlColorSpan {
	spans = spans[:0]
	analysis, err := oracleHighlight().AnalyzeSyntax(text, tuivory.SyntaxRequest{Language: "sql", Dialect: "oracle"})
	if err != nil {
		return spans
	} // Render plain text instead of stale colors.
	for _, token := range analysis.Tokens {
		var kind sqlColorKind
		switch tuivory.SyntaxRoleStyle(token.Role) {
		case "syntax.keyword":
			kind = sqlKeyword
		case "syntax.function":
			kind = sqlFunction
		case "syntax.type":
			kind = sqlType
		case "syntax.string":
			kind = sqlString
		case "syntax.number":
			kind = sqlNumber
		case "syntax.comment":
			kind = sqlComment
		case "syntax.parameter":
			kind = sqlBind
		default:
			continue
		}
		spans = append(spans, sqlColorSpan{token.Start, token.End, kind})
	}
	return spans
}

func (s *screen) colorEditor(editor *tuivory.TextArea, changed func()) {
	dirty := true
	var spans []sqlColorSpan
	editor.SetTextStyleFunc(func(offset int, base tuivory.Style) tuivory.Style {
		if dirty {
			spans = colorSQL(editor.GetText(), spans)
			dirty = false
		}
		index := sort.Search(len(spans), func(i int) bool { return spans[i].end > offset })
		if index < len(spans) && spans[index].start <= offset {
			return s.profileSQLStyle(spans[index].kind, base)
		}
		return base
	}).SetChangedFunc(func() {
		dirty = true
		if changed != nil {
			changed()
		}
	})
}

func (s *screen) profileSQLStyle(kind sqlColorKind, base tuivory.Style) tuivory.Style {
	return s.tuivoryStyles().SyntaxStyle(tuivory.SyntaxKind(kind), base)
}
