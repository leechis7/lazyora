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
			if inputBackground != palette.field {
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
		if background == palette.field {
			t.Fatal("language choice resembles text input")
		}
		terminal.Fini()
	}
}
