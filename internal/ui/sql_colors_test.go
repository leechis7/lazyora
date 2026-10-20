package ui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rivo/uniseg"
)

func TestOracleSQLColorTokens(t *testing.T) {
	for _, tc := range []struct {
		text string
		kind sqlColorKind
	}{
		{"sElEcT", sqlKeyword}, {"VARCHAR2", sqlType}, {"sysdate", sqlFunction},
		{"'SELECT :x it''s 한글'", sqlString}, {"N'한글'", sqlString},
		{"q'[it''s; :x\n/ SELECT]'", sqlString}, {"nq'{한글 :n\n}'", sqlString},
		{"q'!/* SELECT */!'", sqlString}, {"q'한SELECT한'", sqlString},
		{"'unfinished", sqlString}, {"q'[unfinished\nSELECT", sqlString},
		{"-- SELECT :x", sqlComment}, {"/* SELECT\n:x */", sqlComment}, {"/* unfinished", sqlComment},
		{":id", sqlBind}, {":1", sqlBind}, {":\"한글\"", sqlBind},
		{"42", sqlNumber}, {".5", sqlNumber}, {"1.2e-3", sqlNumber}, {"2D", sqlNumber},
	} {
		t.Run(tc.text, func(t *testing.T) {
			got := colorSQL(tc.text, nil)
			want := []sqlColorSpan{{0, len(tc.text), tc.kind}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got=%v want=%v", got, want)
			}
		})
	}
	for _, text := range []string{`"SELECT"`, `"it""SELECT"`, "my_select", "표SELECT", ":=", "/* SELECT */"} {
		got := colorSQL(text, nil)
		for _, span := range got {
			if span.kind == sqlKeyword {
				t.Fatalf("keyword inside identifier/comment: %q", text)
			}
		}
	}
	text := "BEGIN n := 1..5; x := :id; END;"
	for _, span := range colorSQL(text, nil) {
		if span.kind == sqlNumber && strings.Contains(text[span.start:span.end], "..") {
			t.Fatal("PL/SQL range consumed as one numeric literal")
		}
	}
}

func colorTestTerminal(t *testing.T, width, height int) tcell.SimulationScreen {
	t.Helper()
	terminal := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.Init(); err != nil {
		t.Fatal(err)
	}
	terminal.SetSize(width, height)
	t.Cleanup(terminal.Fini)
	return terminal
}

func TestSQLColorsRenderWithoutChangingTextAndSelection(t *testing.T) {
	for _, limited := range []bool{false, true} {
		t.Run(fmt.Sprint(limited), func(t *testing.T) {
			terminal := colorTestTerminal(t, 100, 8)
			s := shortcutScreen(t)
			s.limitedColors = limited
			editor := s.editor
			editor.SetRect(0, 0, 100, 8)
			editor.SetBackgroundColor(palette.field).SetBorder(false)
			editor.SetTextStyle(fieldStyle()).SetSelectedStyle(focusStyle())
			changes := 0
			s.colorEditor(editor, func() { changes++ })
			text := "SELECT '한글' AS x, :id, 12.5 FROM dual; -- SELECT"
			editor.SetText(text, false)
			editor.Draw(terminal)
			for _, tc := range []struct {
				text string
				kind sqlColorKind
			}{{"SELECT", sqlKeyword}, {"'한글'", sqlString}, {":id", sqlBind}, {"12.5", sqlNumber}, {"-- SELECT", sqlComment}} {
				x := uniseg.StringWidth(text[:strings.Index(text, tc.text)])
				_, _, style, _ := terminal.GetContent(x, 0)
				fg, bg, _ := style.Decompose()
				want, _, _ := sqlTokenStyle(tc.kind, limited, fieldStyle()).Decompose()
				if fg != want || bg != palette.field {
					t.Fatalf("%q: fg=%v bg=%v want=%v", tc.text, fg, bg, want)
				}
			}
			editor.Select(0, 6)
			editor.Draw(terminal)
			_, _, style, _ := terminal.GetContent(0, 0)
			if style != focusStyle() {
				t.Fatal("syntax colors overrode selection highlighting")
			}
			if selected, start, end := editor.GetSelection(); selected != "SELECT" || start != 0 || end != 6 || editor.GetText() != text || changes != 1 {
				t.Fatal("drawing changed text, selection or fired changed callbacks")
			}
		})
	}
}

func TestSQLColorsFollowScrolledEditedPieceOffsets(t *testing.T) {
	terminal := colorTestTerminal(t, 30, 1)
	s := shortcutScreen(t)
	editor := s.editor
	editor.SetTextStyle(fieldStyle()).SetWrap(false).SetText("한글\nSELECT :id FROM dual;", false)
	editor.SetRect(0, 0, 30, 1)
	s.colorEditor(editor, nil)
	editor.Draw(terminal)
	// Force multiple pieces before the visible row; the absolute offset must
	// include both the UTF-8 original and inserted/deleted pieces.
	editor.Replace(0, 0, "-- prefix\n")
	editor.Replace(3, 9, "changed")
	editor.SetOffset(2, 0)
	editor.Draw(terminal)
	r, _, style, _ := terminal.GetContent(0, 0)
	fg, _, _ := style.Decompose()
	if r != 'S' || fg != sqlRichColors[sqlKeyword] {
		t.Fatalf("scrolled SELECT lost its absolute source style: %q %v", r, fg)
	}
	_, _, style, _ = terminal.GetContent(7, 0)
	fg, _, _ = style.Decompose()
	if fg != sqlRichColors[sqlBind] {
		t.Fatal("scrolled bind variable lost its style")
	}
	editor.SetOffset(2, 8)
	editor.Draw(terminal)
	r, _, style, _ = terminal.GetContent(0, 0)
	fg, _, _ = style.Decompose()
	if r != 'i' || fg != sqlRichColors[sqlBind] {
		t.Fatalf("horizontal scroll lost bind style: %q %v", r, fg)
	}
}

func TestColoringPreservesWrappedGraphemesAndUndo(t *testing.T) {
	for _, width := range []int{8, 17, 40} {
		plain, colored := tview.NewTextArea(), tview.NewTextArea()
		s := shortcutScreen(t)
		s.colorEditor(colored, nil)
		text := "SELECT\t'한글 e\u0301 👨‍👩‍👧' AS 값;\r\n/* 주석\nSELECT */\nSELECT :id FROM dual;"
		for _, editor := range []*tview.TextArea{plain, colored} {
			editor.SetRect(0, 0, width, 12)
			editor.SetTextStyle(fieldStyle()).SetSelectedStyle(focusStyle()).SetText(text, false)
			editor.Replace(7, 7, "42 + ")
			editor.Select(0, 6)
		}
		a, b := colorTestTerminal(t, width, 12), colorTestTerminal(t, width, 12)
		for _, offset := range []int{0, 1, 3} {
			plain.SetOffset(offset, 0).Draw(a)
			colored.SetOffset(offset, 0).Draw(b)
			for y := 0; y < 12; y++ {
				for x := 0; x < width; x++ {
					ra, ca, _, wa := a.GetContent(x, y)
					rb, cb, _, wb := b.GetContent(x, y)
					if ra != rb || !reflect.DeepEqual(ca, cb) || wa != wb {
						t.Fatalf("coloring changed wrapped characters at width=%d cell=%d,%d", width, x, y)
					}
				}
			}
		}
		for _, editor := range []*tview.TextArea{plain, colored} {
			editor.InputHandler()(tcell.NewEventKey(tcell.KeyCtrlZ, 0, 0), func(tview.Primitive) {})
			if editor.GetText() != text {
				t.Fatal("color rendering broke undo")
			}
		}
	}
}

func TestSourceEditorHasSyntaxColors(t *testing.T) {
	s := programScreen(t)
	editor := s.sourceEditor
	editor.SetBorder(false)
	editor.SetRect(0, 0, 60, 4)
	editor.SetText("BEGIN NULL; END;", false)
	terminal := colorTestTerminal(t, 60, 4)
	editor.Draw(terminal)
	_, _, style, _ := terminal.GetContent(0, 0)
	fg, _, _ := style.Decompose()
	if fg != sqlRichColors[sqlKeyword] {
		t.Fatal("source editor did not use SQL syntax coloring")
	}
}

func TestColoredTabsSubmitOriginalSQL(t *testing.T) {
	s := fileTestScreen(t)
	s.initTabs()
	terminal := colorTestTerminal(t, 80, 6)
	for i := range 2 {
		if i > 0 {
			s.newDocument()
		}
		text := "SELECT '-- :not_a_bind [red] 한글' AS text FROM dual;"
		s.editor.SetText(text, false)
		s.editor.SetRect(0, 0, 80, 6)
		s.editor.Draw(terminal)
		if len(s.jobs) != 0 {
			t.Fatal("drawing submitted SQL")
		}
		s.executeSQL()
		if len(s.jobs) != 1 || s.bindForm != nil {
			t.Fatal("coloring introduced a bind or changed execution")
		}
		if j := <-s.jobs; j.text != text || s.editor.GetText() != text {
			t.Fatal("coloring changed the SQL sent to the database")
		}
		s.cancel()
		s.busy = false
	}
}

func BenchmarkSQLColors(b *testing.B) {
	text := strings.Repeat("SELECT :id, q'[한글]' FROM dual; -- comment\n", 80000)
	var spans []sqlColorSpan
	b.SetBytes(int64(len(text)))
	b.ReportAllocs()
	for b.Loop() {
		spans = colorSQL(text, spans)
	}
}

func BenchmarkSQLColorRedraw(b *testing.B) {
	terminal := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.Init(); err != nil {
		b.Fatal(err)
	}
	defer terminal.Fini()
	terminal.SetSize(100, 20)
	editor := tview.NewTextArea().SetText(strings.Repeat("SELECT :id FROM dual; -- comment\n", 80000), false)
	editor.SetRect(0, 0, 100, 20)
	s := &screen{}
	s.colorEditor(editor, nil)
	editor.Draw(terminal) // Build tokens once; cursor-only redraws reuse them.
	b.ReportAllocs()
	for b.Loop() {
		editor.Draw(terminal)
	}
}

func FuzzSQLColorRanges(f *testing.F) {
	for _, text := range []string{"SELECT :x, .5 FROM dual", "nq'[한글;\n/]' /* comment */", "BEGIN n := 1..5; END;", "'unfinished"} {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		end := 0
		for _, span := range colorSQL(text, nil) {
			if span.start < end || span.end <= span.start || span.end > len(text) || span.kind < sqlKeyword || span.kind > sqlBind {
				t.Fatalf("invalid color span: %+v", span)
			}
			end = span.end
		}
	})
}
