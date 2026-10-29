package ui

import (
	"encoding/csv"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"strings"
	"testing"
)

func resultScreen(t *testing.T) (*screen, tcell.SimulationScreen) {
	s := demoBrowserScreen(t)
	terminal := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(terminal.Fini)
	terminal.SetSize(120, 30)
	s.terminal = terminal
	s.render(oracle.Result{Columns: []string{"ID", "VALUE", "NOTE"}, Rows: [][]string{{"1", "한글 [red]", "NULL"}, {"2", strings.Repeat("긴 값", 60), "tab\tquote\"\nline"}, {"3", "third", "end"}}})
	s.table.SetRect(0, 0, 120, 15)
	s.table.SetMouseCapture(s.resultMouse)
	s.app.SetFocus(s.table)
	s.resultGrid().Draw(terminal)
	return s, terminal
}

func TestCopyRawRowsAndKeyboardRange(t *testing.T) {
	s, terminal := resultScreen(t)
	s.table.Select(2, 2)
	s.key(tcell.NewEventKey(tcell.KeyRune, 'y', tcell.ModAlt))
	// Parse as TSV, including embedded tabs/newlines and quotes.
	reader := csv.NewReader(strings.NewReader(s.clipboardText))
	reader.Comma = '\t'
	records, err := reader.ReadAll()
	if err != nil || len(records) != 1 || len(records[0]) != 3 || records[0][1] != s.resultRows[1][1] || records[0][2] != s.resultRows[1][2] {
		t.Fatalf("row copy corrupted data: %v %v", records, err)
	}
	if string(terminal.GetClipboardData()) != s.clipboardText {
		t.Fatal("clipboard payload differs")
	}
	s.table.Select(1, 1)
	s.key(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModShift))
	s.key(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModShift))
	s.key(tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModCtrl))
	reader = csv.NewReader(strings.NewReader(s.clipboardText))
	reader.Comma = '\t'
	records, err = reader.ReadAll()
	if err != nil || len(records) != 2 || records[0][0] != "한글 [red]" || records[0][1] != "NULL" || records[1][1] != s.resultRows[1][2] {
		t.Fatalf("range copy corrupted values: %v %v", records, err)
	}
	if s.exitDialog != nil {
		t.Fatal("Ctrl+C attempted exit")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.resultSelection.active {
		t.Fatal("Escape did not clear range")
	}
}

func TestDragRangeCopyAndPreviewRestoresFocus(t *testing.T) {
	s, terminal := resultScreen(t)
	handler := s.table.MouseHandler()
	x1, y1, _ := s.resultCellRect(1, 0)
	x2, y2, _ := s.resultCellRect(3, 1)
	focus := func(p tview.Primitive) { s.app.SetFocus(p) }
	handler(tview.MouseLeftDown, tcell.NewEventMouse(x1, y1, tcell.Button1, 0), focus)
	handler(tview.MouseMove, tcell.NewEventMouse(x2, y2, tcell.Button1, 0), focus)
	handler(tview.MouseLeftUp, tcell.NewEventMouse(x2, y2, 0, 0), focus)
	handler(tview.MouseLeftClick, tcell.NewEventMouse(x2, y2, 0, 0), focus)
	r1, c1, r2, c2 := s.resultBounds()
	if r1 != 1 || c1 != 0 || r2 != 3 || c2 != 1 {
		t.Fatalf("drag bounds: %d,%d-%d,%d", r1, c1, r2, c2)
	}
	s.copyResults("selection")
	before := s.clipboardText
	s.copyPreview()
	s.copyText.Draw(terminal)
	if page, _ := s.pages.GetFrontPage(); page != "copy" {
		t.Fatal("copy preview missing")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if !s.table.HasFocus() || s.clipboardText != before {
		t.Fatal("preview changed selection/copy")
	}
	s.prefs.Mouse = false
	handler(tview.MouseLeftDown, tcell.NewEventMouse(x2, y2, tcell.Button1, 0), focus)
	if a, _, _, _ := s.resultBounds(); a != 1 {
		t.Fatal("disabled mouse changed range")
	}
	s.render(oracle.Result{Columns: []string{"X"}, Rows: [][]string{{"fresh"}}})
	if s.resultSelection.active {
		t.Fatal("new result retained stale range")
	}
}

func TestCopyFromScrolledResultUsesActualRowAndFullValue(t *testing.T) {
	s, terminal := resultScreen(t)
	var rows [][]string
	for i := 0; i < 1000; i++ {
		rows = append(rows, []string{fmt.Sprint(i), strings.Repeat("한글", 80)})
	}
	s.render(oracle.Result{Columns: []string{"ID", "VALUE"}, Rows: rows})
	s.table.Select(501, 0).SetOffset(500, 0)
	s.resultGrid().Draw(terminal)
	x, y, _ := s.resultCellRect(501, 1)
	s.table.MouseHandler()(tview.MouseLeftDown, tcell.NewEventMouse(x, y, tcell.Button1, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	s.table.MouseHandler()(tview.MouseLeftUp, tcell.NewEventMouse(x, y, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	s.copyResults("row")
	reader := csv.NewReader(strings.NewReader(s.clipboardText))
	reader.Comma = '\t'
	values, err := reader.Read()
	if err != nil || len(values) != 2 || values[0] != "500" || values[1] != rows[500][1] {
		t.Fatalf("scrolled row copy failed: %v %v", values, err)
	}
}
