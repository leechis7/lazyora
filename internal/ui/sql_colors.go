package ui

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type sqlColorKind uint8

const (
	sqlKeyword sqlColorKind = iota + 1
	sqlFunction
	sqlType
	sqlString
	sqlNumber
	sqlComment
	sqlBind
)

type sqlColorSpan struct {
	start, end int
	kind       sqlColorKind
}

var sqlWords = func() map[string]sqlColorKind {
	words := make(map[string]sqlColorKind)
	for kind, list := range map[sqlColorKind]string{
		sqlKeyword:  "SELECT FROM WHERE AS DISTINCT ALL WITH RECURSIVE UNION INTERSECT MINUS JOIN INNER LEFT RIGHT FULL OUTER CROSS ON USING AND OR NOT IN IS NULL LIKE BETWEEN EXISTS ESCAPE GROUP BY HAVING ORDER ASC DESC NULLS FIRST LAST FETCH OFFSET NEXT ROW ROWS ONLY FOR UPDATE NOWAIT SKIP LOCKED INSERT INTO VALUES DELETE MERGE MATCHED WHEN THEN ELSE END CASE SET RETURNING CREATE REPLACE ALTER DROP TRUNCATE TABLE VIEW MATERIALIZED INDEX SEQUENCE SYNONYM TRIGGER TYPE BODY PACKAGE PROCEDURE FUNCTION DECLARE BEGIN EXCEPTION WHEN OTHERS RAISE RETURN IF ELSIF LOOP WHILE EXIT CONTINUE CURSOR OPEN CLOSE BULK COLLECT LIMIT EXECUTE IMMEDIATE COMMIT ROLLBACK SAVEPOINT GRANT REVOKE TO CONNECT START PRIOR NOCYCLE OVER PARTITION RANGE UNBOUNDED PRECEDING FOLLOWING CURRENT ROWTYPE CONSTANT DEFAULT IN OUT NOCOPY AUTHID DEFINER CURRENT_USER PRAGMA EXCEPTION_INIT AUTONOMOUS_TRANSACTION TRUE FALSE SQL SQLCODE SQLERRM PIVOT UNPIVOT MODEL CONSTRAINT PRIMARY KEY FOREIGN REFERENCES UNIQUE CHECK ENABLE DISABLE ADD MODIFY COMMENT ANALYZE EXPLAIN PLAN LEVEL ROWNUM",
		sqlFunction: "SYSDATE SYSTIMESTAMP CURRENT_DATE CURRENT_TIMESTAMP SESSIONTIMEZONE DBTIMEZONE USER SYS_CONTEXT COUNT SUM AVG MIN MAX LISTAGG ROW_NUMBER RANK DENSE_RANK LAG LEAD FIRST_VALUE LAST_VALUE NVL NVL2 NULLIF COALESCE DECODE TO_CHAR TO_DATE TO_TIMESTAMP TO_TIMESTAMP_TZ TO_NUMBER CAST CONVERT TRUNC ROUND CEIL FLOOR ABS MOD POWER SQRT SUBSTR INSTR LENGTH LENGTHB UPPER LOWER INITCAP TRIM LTRIM RTRIM LPAD RPAD TRANSLATE CONCAT REGEXP_LIKE REGEXP_REPLACE REGEXP_SUBSTR REGEXP_INSTR REGEXP_COUNT ADD_MONTHS MONTHS_BETWEEN LAST_DAY NEXT_DAY EXTRACT CHR ASCII HEXTORAW RAWTOHEX XMLTYPE JSON_VALUE JSON_QUERY JSON_TABLE DBMS_OUTPUT PUT_LINE",
		sqlType:     "NUMBER INTEGER INT SMALLINT NUMERIC DECIMAL FLOAT BINARY_FLOAT BINARY_DOUBLE PLS_INTEGER BINARY_INTEGER SIMPLE_INTEGER BOOLEAN CHAR NCHAR VARCHAR VARCHAR2 NVARCHAR2 LONG RAW DATE TIMESTAMP INTERVAL YEAR MONTH DAY HOUR MINUTE SECOND TIME ZONE CLOB NCLOB BLOB BFILE ROWID UROWID RECORD REF VARRAY",
	} {
		for _, word := range strings.Fields(list) {
			words[word] = kind
		}
	}
	return words
}()

func sqlWordKind(word string) sqlColorKind {
	var upper [64]byte
	if len(word) > len(upper) {
		return 0
	}
	for i := range word {
		c := word[i]
		if c >= utf8.RuneSelf {
			return 0 // Unicode identifiers remain literal identifiers.
		}
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		upper[i] = c
	}
	return sqlWords[string(upper[:len(word)])]
}

func sqlIdent(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("_$#", r)
}
func sqlDigit(c byte) bool { return c >= '0' && c <= '9' }

func sqlQuoteEnd(text string, start int, quote byte) int {
	for i := start + 1; i < len(text); i++ {
		if text[i] == quote {
			if i+1 < len(text) && text[i+1] == quote {
				i++
				continue
			}
			return i + 1
		}
	}
	return len(text) // Incomplete literals stay colored while being typed.
}

// colorSQL is a tolerant display lexer, never an SQL validator or rewriter.
// Every emitted range is an original UTF-8 byte range. Each byte is scanned a
// constant number of times, including multiline comments and Oracle q quotes.
func colorSQL(text string, spans []sqlColorSpan) []sqlColorSpan {
	spans = spans[:0]
	for i := 0; i < len(text); {
		start := i
		kind := sqlColorKind(0)
		switch {
		case strings.HasPrefix(text[i:], "--"):
			i = len(text)
			if n := strings.IndexByte(text[start:], '\n'); n >= 0 {
				i = start + n
			}
			kind = sqlComment
		case strings.HasPrefix(text[i:], "/*"):
			i = len(text)
			if n := strings.Index(text[start+2:], "*/"); n >= 0 {
				i = start + 2 + n + 2
			}
			kind = sqlComment
		default:
			q := i
			if (text[i] == 'n' || text[i] == 'N') && i+1 < len(text) && (text[i+1] == 'q' || text[i+1] == 'Q') {
				q++
			}
			switch {
			case q+2 < len(text) && (text[q] == 'q' || text[q] == 'Q') && text[q+1] == '\'':
				delim, size := utf8.DecodeRuneInString(text[q+2:])
				switch delim {
				case '[':
					delim = ']'
				case '{':
					delim = '}'
				case '(':
					delim = ')'
				case '<':
					delim = '>'
				}
				i = len(text)
				if n := strings.Index(text[q+2+size:], string(delim)+"'"); n >= 0 {
					i = q + 2 + size + n + len(string(delim)) + 1
				}
				kind = sqlString
			case (text[i] == 'n' || text[i] == 'N') && i+1 < len(text) && text[i+1] == '\'':
				i, kind = sqlQuoteEnd(text, i+1, '\''), sqlString
			case text[i] == '\'':
				i, kind = sqlQuoteEnd(text, i, '\''), sqlString
			case text[i] == '"':
				i = sqlQuoteEnd(text, i, '"')
			case text[i] == ':' && i+1 < len(text) && text[i+1] != '=':
				i++
				if text[i] == '"' {
					i = sqlQuoteEnd(text, i, '"')
				} else {
					for i < len(text) {
						r, size := utf8.DecodeRuneInString(text[i:])
						if !sqlIdent(r) {
							break
						}
						i += size
					}
				}
				if i > start+1 {
					kind = sqlBind
				}
			case sqlDigit(text[i]) || text[i] == '.' && i+1 < len(text) && sqlDigit(text[i+1]) && (i == 0 || text[i-1] != '.'):
				for i < len(text) && sqlDigit(text[i]) {
					i++
				}
				if i < len(text) && text[i] == '.' && !(i+1 < len(text) && text[i+1] == '.') {
					i++
					for i < len(text) && sqlDigit(text[i]) {
						i++
					}
				}
				if i < len(text) && (text[i] == 'e' || text[i] == 'E') {
					end := i + 1
					if end < len(text) && (text[end] == '+' || text[end] == '-') {
						end++
					}
					digits := end
					for end < len(text) && sqlDigit(text[end]) {
						end++
					}
					if end > digits {
						i = end
					}
				}
				if i < len(text) && strings.ContainsRune("fFdD", rune(text[i])) {
					i++
				}
				kind = sqlNumber
			default:
				r, size := utf8.DecodeRuneInString(text[i:])
				i += size
				if sqlIdent(r) {
					for i < len(text) {
						r, size = utf8.DecodeRuneInString(text[i:])
						if !sqlIdent(r) {
							break
						}
						i += size
					}
					kind = sqlWordKind(text[start:i])
				}
			}
		}
		if kind != 0 {
			spans = append(spans, sqlColorSpan{start, i, kind})
		}
	}
	return spans
}

var sqlRichColors = [...]tcell.Color{0, tcell.GetColor("#204a87"), tcell.GetColor("#006b73"), tcell.GetColor("#5951a0"), tcell.GetColor("#a3273a"), tcell.GetColor("#6945a3"), tcell.GetColor("#526a43"), tcell.GetColor("#8b4f00")}
var sqlANSIColors = [...]tcell.Color{0, tcell.ColorNavy, tcell.ColorTeal, tcell.ColorPurple, tcell.ColorMaroon, tcell.ColorPurple, tcell.ColorGreen, tcell.ColorMaroon}

var sqlDarkColors = [...]tcell.Color{0, tcell.GetColor("#8db8ff"), tcell.GetColor("#73d7ce"), tcell.GetColor("#c7adff"), tcell.GetColor("#efa3aa"), tcell.GetColor("#ebcd79"), tcell.GetColor("#a2bb98"), tcell.GetColor("#f2bc84")}
var sqlDarkANSIColors = [...]tcell.Color{0, tcell.ColorAqua, tcell.ColorWhite, tcell.ColorFuchsia, tcell.ColorYellow, tcell.ColorYellow, tcell.ColorLime, tcell.ColorWhite}

func sqlTokenStyle(kind sqlColorKind, limited bool, base tcell.Style) tcell.Style {
	colors := sqlRichColors
	if limited {
		colors = sqlANSIColors
	}
	_, bg, _ := base.Decompose()
	r, g, b := bg.RGB()
	if r >= 0 && 299*r+587*g+114*b < 128000 {
		colors = sqlDarkColors
		if limited {
			colors = sqlDarkANSIColors
		}
	}
	style := base.Foreground(colors[kind])
	if !limited && (kind == sqlKeyword || kind == sqlBind) {
		style = style.Bold(true)
	}
	return style
}

func (s *screen) colorEditor(editor *tview.TextArea, changed func()) {
	dirty := true
	var spans []sqlColorSpan
	editor.SetTextStyleFunc(func(offset int, base tcell.Style) tcell.Style {
		if dirty {
			spans = colorSQL(editor.GetText(), spans)
			dirty = false
		}
		index := sort.Search(len(spans), func(i int) bool { return spans[i].end > offset })
		if index < len(spans) && spans[index].start <= offset {
			if s.monochrome() {
				kind := spans[index].kind
				return base.Bold(false).Underline(kind == sqlKeyword || kind == sqlBind).Italic(kind == sqlComment)
			}
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

func (s *screen) profileSQLStyle(kind sqlColorKind, base tcell.Style) tcell.Style {
	if s.monochrome() {
		return base.Bold(false).Underline(kind == sqlKeyword || kind == sqlBind).Italic(kind == sqlComment)
	}
	p := s.colorProfile()
	_, bg, _ := base.Decompose()
	r, g, b := bg.RGB()
	dark := r >= 0 && 299*r+587*g+114*b < 128000
	if p == profileANSI8 {
		fg := indexed(7)
		if !dark {
			fg = indexed(0)
		}
		if kind == sqlKeyword && !dark {
			fg = indexed(4)
		}
		if kind == sqlBind {
			fg = indexed(3)
			if !dark {
				fg = indexed(1)
			}
		}
		return base.Foreground(fg).Bold(false).Underline(kind == sqlKeyword)
	}
	if p == profile256 {
		values := [...]int{0, 25, 23, 54, 124, 54, 22, 94}
		if dark {
			values = [...]int{0, 117, 117, 183, 217, 222, 114, 222}
		}
		return base.Foreground(indexed(values[kind])).Bold(false)
	}
	return sqlTokenStyle(kind, p == profileANSI16, base)
}
