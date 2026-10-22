package ui

import (
	"fmt"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestDocumentFocusBordersAndArrowNavigation(t *testing.T) {
	for _, kind := range []string{"help", "guide", "release_history"} {
		for _, theme := range []string{"dark", "light"} {
			for _, limited := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/ansi=%v", kind, theme, limited), func(t *testing.T) {
					spaces, terminal := workspaceTestSet(t)
					s := spaces.active
					s.prefs.Theme = theme
					s.limitedColors = limited
					switch kind {
					case "help":
						s.showHelp()
					case "guide":
						s.showGuide()
					case "release_history":
						s.showReleaseHistory()
					}
					list, text := s.helpTopics, s.helpText
					if kind != "help" {
						list, text = s.reference.topics, s.reference.text
					}
					if !list.HasFocus() {
						t.Fatal("contents not initially focused")
					}
					check := func() {
						s.refreshBorders()
						s.rootLayout.SetRect(0, 0, 120, 40)
						s.rootLayout.Draw(terminal)
						colors := make([]tcell.Color, 0, 2)
						for _, widget := range []tview.Primitive{list, text} {
							x, y, _, _ := widget.GetRect()
							r, _, st, _ := terminal.GetContent(x, y)
							want := tview.Borders.TopLeft
							if widget.HasFocus() {
								want = tview.Borders.TopLeftFocus
							}
							if r != want {
								t.Fatalf("%T focus=%v corner=%q expected=%q", widget, widget.HasFocus(), r, want)
							}
							fg, _, _ := st.Decompose()
							colors = append(colors, fg)
						}
						if colors[0] == colors[1] {
							t.Fatal("focused and inactive borders share a color")
						}
					}
					deliver := func(key tcell.Key) {
						e := s.key(tcell.NewEventKey(key, 0, 0))
						if e != nil {
							s.app.GetFocus().InputHandler()(e, func(p tview.Primitive) { s.app.SetFocus(p) })
						}
					}
					check()
					before := list.GetCurrentItem()
					deliver(tcell.KeyDown)
					if list.GetCurrentItem() != before+1 {
						t.Fatal("Down did not change topic")
					}
					deliver(tcell.KeyRight)
					if !text.HasFocus() {
						t.Fatal("Right did not enter article")
					}
					check()
					deliver(tcell.KeyLeft)
					if !list.HasFocus() {
						t.Fatal("Left did not return to contents")
					}
					check()
					deliver(tcell.KeyTab)
					if !text.HasFocus() {
						t.Fatal("Tab did not enter article")
					}
					check()
					deliver(tcell.KeyBacktab)
					if !list.HasFocus() {
						t.Fatal("Backtab did not return to contents")
					}
					check()
					if len(s.jobs) != 0 {
						t.Fatal("navigation submitted SQL")
					}
				})
			}
		}
	}
}

func TestPopupAndMainFocusBorderConsistency(t *testing.T) {
	spaces, terminal := workspaceTestSet(t)
	s := spaces.active
	check := func(p tview.Primitive) {
		s.refreshBorders()
		s.rootLayout.SetRect(0, 0, 120, 40)
		s.rootLayout.Draw(terminal)
		x, y, _, _ := p.GetRect()
		r, _, _, _ := terminal.GetContent(x, y)
		if r != tview.Borders.TopLeftFocus {
			t.Fatalf("%T does not use focused double border: %q", p, r)
		}
	}
	s.app.SetFocus(s.editor)
	check(s.editor)
	s.app.SetFocus(s.table)
	check(s.table)
	s.showAbout()
	_, p := s.pages.GetFrontPage()
	check(p.(*formPopupLayer).content)
	s.closeAbout()
	s.connectionForm()
	check(s.connectForm)
	s.closeConnection()
	s.showWorkspaces()
	check(s.workspaceList)
}
