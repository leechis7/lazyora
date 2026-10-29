package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestTuivory040DisabledMenuAndSurfaceText(t *testing.T) {
	for _, colors := range []int{8, 16, 256, 1 << 24} {
		for _, theme := range []string{"dark", "light", "blackwhite"} {
			t.Run(fmt.Sprintf("%s/%d", theme, colors), func(t *testing.T) {
				s, terminal := resultScreen(t)
				s.terminal = auditTerminal{SimulationScreen: terminal, count: colors}
				s.limitedColors = colors <= 16
				s.prefs.Theme = theme
				s.connected = false
				s.app.SetFocus(s.editor)
				before := s.editor.GetText()
				s.showMenu(menuExecute)
				_, root := s.pages.GetFrontPage()
				root.SetRect(0, 0, 100, 25)
				root.Draw(terminal)
				x, y, w, h := s.menuList.GetInnerRect()
				focus := s.menuModel.FocusedID()
				found := false
				for at, line := range s.menuModel.Layout(w, h, s.asciiDisplay()) {
					if !strings.HasSuffix(line.ID, "/execute") {
						continue
					}
					found = true
					if line.Role != "disabled" || strings.HasPrefix(strings.TrimSpace(line.Spans[0].Text), "-") {
						t.Fatal("ordinary disabled prefix retained")
					}
					_, _, style, _ := terminal.GetContent(x, y+at)
					fg, bg, attrs := style.Decompose()
					normal := s.componentPalette()["body"].FG
					if fg == bg || fg == normal && attrs&tcell.AttrDim == 0 {
						t.Fatal("disabled control has no visible gray/Dim cue")
					}
					s.menuList.MouseHandler()(tview.MouseLeftClick, tcell.NewEventMouse(x+2, y+at, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
					if s.menuModel.FocusedID() != focus || !s.pages.HasPage("menu") || len(s.jobs) != 0 || s.editor.GetText() != before {
						t.Fatal("disabled click focused or executed a command")
					}
				}
				if !found {
					t.Fatal("disabled command missing")
				}
				s.closeMenu()
				for _, surface := range []surfaceRole{surfaceCanvas, surfacePanel, surfaceField, surfaceFooter, surfaceEditor} {
					text := s.newTextView(textHint).SetText("한글 label")
					s.textSurface(text, surface)
					text.SetRect(0, 0, 30, 2)
					text.Draw(terminal)
					_, _, style, _ := terminal.GetContent(0, 0)
					fg, bg, _ := style.Decompose()
					if fg == bg || bg != s.surfaceColor(surface) || text.GetText(false) != "한글 label" {
						t.Fatal("plain text mismatched actual surface or changed content", surface)
					}
				}
			})
		}
	}
}
