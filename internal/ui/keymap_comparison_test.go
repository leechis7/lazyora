package ui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
	"testing"
)

func TestKeymapComparisonMatchesAssignmentsAndRestoresHelp(t *testing.T) {
	s, terminal := resultScreen(t)
	s.prefs.EditorKeymap = "intellij"
	before := s.editor.GetText()
	s.showHelp()
	s.helpJump(8)
	focus := s.app.GetFocus()
	s.key(tcell.NewEventKey(tcell.KeyRune, 'k', tcell.ModAlt))
	if s.comparison == nil || !s.comparison.table.HasFocus() {
		t.Fatal("help cannot open comparison")
	}
	table := s.comparison.table
	if table.GetColumnCount() != 6 || table.GetRowCount() < 25 || !strings.Contains(table.GetCell(0, 5).Text, "*") {
		t.Fatal("map headers/current selection missing")
	}
	if s.comparisonBinding("", "select_all") != "Ctrl+L" || s.comparisonBinding("intellij", "commit") != "Alt+C" {
		t.Fatal("common/default assigned keys omitted")
	}
	if s.comparisonBinding("intellij", "delete_line") != "Ctrl+Y" || s.comparisonBinding("vscode", "format_code") != "Ctrl+Shift+I / Alt+Shift+F" || s.comparisonBinding("", "editor_document_start") != "—" {
		t.Fatal("comparison not reflecting runtime")
	}
	for _, size := range [][2]int{{40, 16}, {80, 24}, {160, 40}} {
		terminal.SetSize(size[0], size[1])
		_, root := s.pages.GetFrontPage()
		root.SetRect(0, 0, size[0], size[1])
		root.Draw(terminal)
		table.Select(3, 5)
		root.Draw(terminal)
		x, y, _ := table.GetCell(3, 0).GetLastPosition()
		actual, _ := table.CellAt(x, y)
		if actual != 3 || !table.InRect(x, y) {
			t.Fatal("pinned function column lost")
		}
	}
	s.key(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0))
	if !strings.Contains(s.clipboardText, "IntelliJ IDEA") || !strings.Contains(s.clipboardText, "Ctrl+Space") || s.editor.GetText() != before {
		t.Fatal("comparison copy changed SQL or omitted columns")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.comparison != nil || !s.pages.HasPage("help") || s.app.GetFocus() != focus {
		t.Fatal("comparison failed to restore help")
	}
}

func TestComparisonFullTextFollowsSelectionAndScrolls(t *testing.T) {
	for _, language := range []string{"en", "ko"} {
		t.Run(language, func(t *testing.T) {
			s, terminal := resultScreen(t)
			s.prefs.Language = language
			s.showKeymapComparison()
			c := s.comparison
			row := -1
			for index, values := range c.rows {
				if len(values) == 6 && values[0] == s.tr(commandFor("execute").label) {
					row = index
					break
				}
			}
			if row < 0 || len(c.rows[row][4]) <= 22 {
				t.Fatal("missing long assigned binding for regression")
			}
			c.table.Select(row, 4)
			if text := c.detail.GetText(false); !strings.Contains(text, c.rows[row][0]) || !strings.Contains(text, "VS Code") || !strings.Contains(text, c.rows[row][4]) {
				t.Fatal("detail omitted the full function/keymap/key binding")
			}
			delegate := func(p tview.Primitive) { s.app.SetFocus(p) }
			_, root := s.pages.GetFrontPage()
			for _, size := range [][2]int{{40, 16}, {80, 24}, {160, 40}} {
				terminal.SetSize(size[0], size[1])
				root.SetRect(0, 0, size[0], size[1])
				root.Draw(terminal)
				x, y, width, height := c.detail.GetInnerRect()
				if width < 1 || height < 1 || x < 0 || y < 0 || x+width > size[0] || y+height > size[1] {
					t.Fatal("detail is outside the popup")
				}
				s.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
				if !c.detail.HasFocus() {
					t.Fatal("Enter did not focus the full-text area")
				}
				c.detail.InputHandler()(tcell.NewEventKey(tcell.KeyEnd, 0, 0), delegate)
				root.Draw(terminal)
				var visible strings.Builder
				for py := y; py < y+height; py++ {
					for px := x; px < x+width; px++ {
						r, _, _, _ := terminal.GetContent(px, py)
						visible.WriteRune(r)
					}
				}
				if !strings.Contains(visible.String(), "Shift+F9") {
					t.Fatalf("cannot read the hidden tail at %v: %q", size, visible.String())
				}
				s.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
				if !c.table.HasFocus() {
					t.Fatal("Enter did not return to the table")
				}
			}
			// A real cell click updates the full text and resets the detail scroll.
			x, y, _ := c.table.GetCell(row, 5).GetLastPosition()
			c.table.MouseHandler()(tview.MouseLeftClick, tcell.NewEventMouse(x+1, y, 0, 0), delegate)
			if !strings.Contains(c.detail.GetText(false), "IntelliJ IDEA") {
				t.Fatal("mouse selection did not update detail")
			}
			if offset, _ := c.detail.GetScrollOffset(); offset != 0 {
				t.Fatal("new selection retained the old scroll position")
			}
			c.table.Select(row, 0)
			if !strings.Contains(c.detail.GetText(false), c.rows[row][0]) || strings.Contains(c.detail.GetText(false), "IntelliJ IDEA") {
				t.Fatal("function-column selection retained another cell's detail")
			}
			s.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
			if !c.detail.HasFocus() {
				t.Fatal("Tab skipped the detail area")
			}
			s.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
			if _, ok := s.app.GetFocus().(*tview.Button); !ok {
				t.Fatal("Tab did not reach Close")
			}
			s.key(tcell.NewEventKey(tcell.KeyBacktab, 0, 0))
			if !c.detail.HasFocus() || len(s.jobs) != 0 {
				t.Fatal("Shift+Tab lost detail focus or navigation executed SQL")
			}
			s.key(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0))
			if !strings.Contains(s.clipboardText, c.rows[row][4]) {
				t.Fatal("table copy lost the full cell value")
			}
		})
	}
}

func TestKeymapComparisonPaintsReadableHeadersAcrossThemes(t *testing.T) {
	for _, limited := range []bool{false, true} {
		for current, preset := range comparisonMaps {
			t.Run(fmt.Sprintf("limited=%v/%s", limited, preset.name), func(t *testing.T) {
				s, terminal := resultScreen(t)
				s.limitedColors = limited
				s.prefs.EditorKeymap = preset.id
				s.showKeymapComparison()
				table := s.comparison.table
				terminal.SetSize(220, 40)
				table.SetRect(0, 0, 220, 38)
				for _, theme := range []string{"dark", "light", "blackwhite", "dark"} {
					previous := s.prefs.Theme
					s.prefs.Theme = theme
					s.applyTheme(previous)
					table.Draw(terminal)
					for col := range table.GetColumnCount() {
						cell := table.GetCell(0, col)
						x, y, _ := cell.GetLastPosition()
						ch, _, actual, _ := terminal.GetContent(x, y)
						expectedFG, expectedBG, _ := cell.Style.Decompose()
						fg, bg, attrs := actual.Decompose()
						if ch == ' ' || fg != expectedFG || bg != expectedBG || fg == bg || cell.Transparent {
							t.Fatalf("unpainted/unreadable header in %s col=%d: fg=%v bg=%v declared=%v", theme, col, fg, bg, cell.Style)
						}
						bx, by, _ := table.GetCell(3, col).GetLastPosition()
						_, _, body, _ := terminal.GetContent(bx, by)
						_, bodyBG, _ := body.Decompose()
						if bg == bodyBG {
							t.Fatalf("header merges into body in %s col=%d", theme, col)
						}
						if col == current+1 && (!strings.Contains(cell.Text, "*") || attrs&tcell.AttrBold == 0 || attrs&tcell.AttrUnderline == 0) {
							t.Fatal("current keymap lost its visible marker")
						}
					}
					row, col := table.GetSelection()
					if row != 2 || col != current+1 || !table.HasFocus() {
						t.Fatal("theme switch changed selection or focus")
					}
				}
			})
		}
	}
}
