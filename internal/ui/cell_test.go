package ui

import (
	"github.com/gdamore/tcell/v2"
	"lazyora/internal/oracle"
	"strings"
	"testing"
)

func TestCellDetailUsesTypedRawValueAndRestoresFocus(t *testing.T) {
	s, terminal := resultScreen(t)
	s.render(oracle.Result{Columns: []string{"BYTES"}, Rows: [][]string{{"0001ff"}}, Values: [][]oracle.Cell{{{Type: "BLOB", Text: "0001ff", Size: 3}}}})
	s.table.Select(1, 0)
	s.app.SetFocus(s.table)
	s.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	if !s.pages.HasPage("cell") || !strings.Contains(s.cellText.GetText(false), "00000000") || !strings.Contains(s.cellInfo.GetText(false), "BLOB") {
		t.Fatal("typed binary detail missing")
	}
	for _, width := range []int{40, 80, 120} {
		terminal.SetSize(width, 30)
		_, root := s.pages.GetFrontPage()
		root.SetRect(0, 0, width, 28)
		root.Draw(terminal)
	}
	s.cellKey(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0))
	if s.clipboardText != "0001ff" {
		t.Fatal("copied dump instead of raw hex")
	}
	s.cellKey(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.pages.HasPage("cell") || s.app.GetFocus() != s.table {
		t.Fatal("focus not restored")
	}
}
func TestPartialCellCannotBeCopiedAsWholeAndBusyEscapeCancels(t *testing.T) {
	s, _ := resultScreen(t)
	s.render(oracle.Result{Columns: []string{"LARGE"}, Rows: [][]string{{"part"}}, Values: [][]oracle.Cell{{{Type: "CLOB", Text: "part", Partial: true, LOBID: 1, Size: 300000}}}})
	s.table.Select(1, 0)
	s.showCellDetail()
	s.clipboardText = "original"
	s.copyCellDetail()
	if s.clipboardText != "original" {
		t.Fatal("partial value copied as complete")
	}
	canceled := false
	s.busy = true
	s.cancel = func() { canceled = true }
	s.closeCellDetail()
	if !canceled || !s.pages.HasPage("cell") {
		t.Fatal("cancel closed live cell prematurely")
	}
	s.busy = false
	s.closeCellDetail()
}
