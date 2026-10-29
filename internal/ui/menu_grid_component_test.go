package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func TestTuivoryMenuGridProfilesPreserveDataAndFocus(t *testing.T) {
	for _, colors := range []int{8, 16, 256, 1 << 24} {
		for _, theme := range []string{"dark", "light", "blackwhite"} {
			for _, language := range []string{"ko", "en"} {
				t.Run(fmt.Sprintf("%s/%d/%s", theme, colors, language), func(t *testing.T) {
					s, terminal := resultScreen(t)
					s.terminal = auditTerminal{SimulationScreen: terminal, count: colors}
					s.limitedColors = colors <= 16
					s.prefs.Theme = theme
					s.prefs.Language = language
					s.render(oracle.Result{Columns: []string{"N", "한글[red]"}, Rows: [][]string{{"1", "한글\t원문\n[red]"}, {"2", "other"}}, Values: [][]oracle.Cell{{{Type: "NUMBER", Text: "1"}, {Type: "VARCHAR2", Text: "한글\t원문\n[red]"}}, {{Type: "NUMBER", Text: "2"}, {Type: "VARCHAR2", Text: "other"}}}})
					s.pendingChanges = true
					s.table.Select(1, 1)
					s.app.SetFocus(s.table)
					for _, width := range []int{40, 80, 120} {
						s.resultGrid().SetRect(0, 0, width, 15)
						s.resultGrid().Draw(terminal)
						x, y, _ := s.resultCellRect(1, 1)
						_, _, style, _ := terminal.GetContent(x, y)
						fg, bg, _ := style.Decompose()
						if fg == bg {
							t.Fatal("active cell is invisible")
						}
					}
					s.showMenu(menuEdit)
					for _, width := range []int{40, 80, 120} {
						_, root := s.pages.GetFrontPage()
						root.SetRect(0, 0, width, 15)
						root.Draw(terminal)
						x, y, _, _ := s.menuList.GetInnerRect()
						_, _, style, _ := terminal.GetContent(x, y)
						fg, bg, _ := style.Decompose()
						if fg == bg {
							t.Fatal("menu is invisible")
						}
					}
					s.closeMenu()
					s.copyResults("cell")
					if !s.table.HasFocus() || s.clipboardText != "\"한글\t원문\n[red]\"" || !s.pendingChanges || len(s.jobs) != 0 {
						t.Fatal("UI integration changed application work")
					}
				})
			}
		}
	}
}
func TestTuivoryGridNavigationResizeAndHostFunctionKeys(t *testing.T) {
	s, terminal := resultScreen(t)
	s.table.SetInputCapture(s.resultGridKey)
	s.app.SetFocus(s.editor)
	s.resultMouse(tview.MouseMove, tcell.NewEventMouse(20, 5, 0, 0))
	if !s.editor.HasFocus() {
		t.Fatal("hover stole cross-component focus")
	}
	s.app.SetFocus(s.table)
	press := func(k tcell.Key) {
		s.table.InputHandler()(tcell.NewEventKey(k, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	}
	press(tcell.KeyDown)
	press(tcell.KeyRight)
	if r, c := s.table.GetSelection(); r != 2 || c != 1 {
		t.Fatalf("common navigation: %d,%d", r, c)
	}
	s.resultGrid().Draw(terminal)
	x, y, width := s.resultCellRect(0, 1)
	before := s.grid.widths[1]
	grid := s.resultGrid()
	handled, capture := grid.MouseHandler()(tview.MouseLeftDown, tcell.NewEventMouse(x+width, y, tcell.Button1, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	if !handled || capture != grid {
		t.Fatal("header drag did not capture pointer")
	}

	s.resultMouse(tview.MouseMove, tcell.NewEventMouse(x+width+6, y, tcell.Button1, 0))
	s.resultMouse(tview.MouseLeftUp, tcell.NewEventMouse(x+width+6, y, 0, 0))
	if s.grid.widths[1] != before+6 {
		t.Fatal("header resize not retained in host state", before, s.grid.widths[1])
	}
	s.key(tcell.NewEventKey(tcell.KeyF10, 0, 0))
	if !s.pages.HasPage("menu") {
		t.Fatal("generic grid range key replaced application F10")
	}
	s.key(tcell.NewEventKey(tcell.KeyF10, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyF12, 0, 0))
	if !s.table.HasFocus() || s.maximized != "results" {
		t.Fatal("generic grid detail key replaced result maximization")
	}
}
func TestTuivoryGridEditedPresentationAndMetadata(t *testing.T) {
	s, terminal := inlineFixture(t)
	key := s.editedCellIdentity(s.resultBrowser.source, 1, 1)
	s.markEditedCell(key)
	s.table.Select(2, 1)
	s.app.SetFocus(s.table)
	s.resultGrid().SetRect(0, 0, 120, 15)
	s.resultGrid().Draw(terminal)
	x, y, _ := s.resultCellRect(2, 1)
	_, _, style, _ := terminal.GetContent(x, y)
	_, bg, _ := style.Decompose()
	_, edited, _ := s.editedCellStyle().Decompose()
	if bg != edited {
		t.Fatal("common renderer lost edited cell background")
	}
	x, y, _ = s.resultCellRect(2, 0)
	_, _, style, _ = terminal.GetContent(x, y)
	_, normal, _ := style.Decompose()
	if normal == edited {
		t.Fatal("row metadata incorrectly marked untouched cells edited")
	}
	_, top, _, height := s.table.GetInnerRect()
	var footer strings.Builder
	for px := 1; px < 119; px++ {
		r, _, _, _ := terminal.GetContent(px, top+height-1)
		footer.WriteRune(r)
	}
	if strings.Contains(footer.String(), "VARCHAR2 / VARCHAR2") {
		t.Fatal("duplicated cell metadata")
	}
}
