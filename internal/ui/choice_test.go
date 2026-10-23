package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"testing"
)

func formFieldCell(form *tview.Form, index int) (int, int) {
	labelWidth := 0
	for i := 0; i < form.GetFormItemCount(); i++ {
		if width := tview.TaggedStringWidth(form.GetFormItem(i).GetLabel()); width > labelWidth {
			labelWidth = width
		}
	}
	x, y, _, _ := form.GetFormItem(index).GetRect()
	return x + labelWidth + 1, y
}

func TestRenderedChoicesStayDistinctFromTextInputs(t *testing.T) {
	for _, limited := range []bool{false, true} {
		s := shortcutScreen(t)
		s.limitedColors = limited
		s.connectionForm()
		terminal := tcell.NewSimulationScreen("UTF-8")
		if err := terminal.Init(); err != nil {
			t.Fatal(err)
		}
		terminal.SetSize(80, 24)
		s.pages.SetRect(0, 0, 80, 22)
		for _, language := range []string{"en", "ko"} {
			s.prefs.Language = language
			s.localizeConnection()
			s.app.SetFocus(s.editor)
			// Repeated draws must not let Form's shared white field style win.
			for draw := 0; draw < 2; draw++ {
				s.pages.Draw(terminal)
			}
			x, y := formFieldCell(s.connectForm, connectionHost)
			_, _, style, _ := terminal.GetContent(x, y)
			_, inputBackground, _ := style.Decompose()
			if inputBackground != s.colors().field {
				t.Fatal("text input lost white background")
			}
			for _, form := range []*tview.Form{s.connectForm, s.connectProfiles} {
				index := 0
				if form == s.connectForm {
					index = connectionMode
				}
				x, y := formFieldCell(form, index)
				_, _, normal, _ := terminal.GetContent(x, y)
				_, choiceBackground, _ := normal.Decompose()
				if choiceBackground == inputBackground {
					t.Fatalf("choice resembles text input (limited=%v, language=%s)", limited, language)
				}
				marker := false
				for col := x; col < x+40 && col < 80; col++ {
					r, _, _, _ := terminal.GetContent(col, y)
					marker = marker || r == '▾'
				}
				if !marker {
					t.Fatal("choice indicator missing")
				}
				focusForm(s, form, index)
				s.pages.Draw(terminal)
				_, _, focused, _ := terminal.GetContent(x, y)
				if focused == normal {
					t.Fatal("choice focus was not visible")
				}
				s.app.SetFocus(s.editor)
				s.pages.Draw(terminal)
				_, _, restored, _ := terminal.GetContent(x, y)
				if restored != normal {
					t.Fatal("choice style changed after focus moved away")
				}
			}
		}
		s.settingsForm()
		s.app.SetFocus(s.editor)
		s.pages.Draw(terminal)
		x, y := formFieldCell(s.prefsForm, 0)
		_, _, style, _ := terminal.GetContent(x, y)
		_, background, _ := style.Decompose()
		if background == s.colors().field {
			t.Fatal("language choice resembles text input")
		}
		terminal.Fini()
	}
}

func TestOpenChoiceErasesWideCharacterCrossingItsLeftEdge(t *testing.T) {
	s, terminal := resultScreen(t)
	terminal.SetSize(80, 20)
	c := s.newChoice("", []string{"다크", "라이트", "흑백 (Black / White)"}, 1, nil)
	c.SetRect(10, 2, 35, 1)
	s.app.SetFocus(c)
	c.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	// A button behind the list ends in a two-cell Hangul glyph straddling
	// the list's left edge. The physical terminal skips the overlaid first
	// character unless the old glyph's first cell is erased too.
	terminal.SetContent(9, 5, '취', nil, tcell.StyleDefault)
	c.Draw(terminal)
	terminal.Show()
	cells, width, _ := terminal.GetContents()
	if len(cells[5*width+10].Runes) == 0 || cells[5*width+10].Runes[0] != '흑' {
		t.Fatal("the first Hangul character is hidden by the underlying button")
	}
	if _, _, _, previousWidth := terminal.GetContent(9, 5); previousWidth > 1 {
		t.Fatal("a partial wide glyph remains across the popup boundary")
	}
}
