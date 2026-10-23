package ui

import (
	"github.com/gdamore/tcell/v2"
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
