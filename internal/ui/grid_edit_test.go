package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"strings"
	"testing"
)

func TestGridEditMapsSortedFilteredRowsAndRequiresConfirmation(t *testing.T) {
	s, terminal := resultScreen(t)
	s.connected = true
	object := oracle.Object{Owner: "SCOTT", Name: "TABLE_WITH_QUOTES\"", Kind: "TABLE"}
	s.render(oracle.Result{EditID: 77, EditObject: object, Columns: []string{"N", "TITLE"}, Rows: [][]string{{"1", "first"}, {"2", "second"}}, Values: [][]oracle.Cell{{{Type: "NUMBER", Text: "1"}, {Type: "VARCHAR2", Text: "first"}}, {{Type: "NUMBER", Text: "2"}, {Type: "VARCHAR2", Text: "second"}}}})
	s.table.Select(1, 0)
	s.sortResult(true)
	s.table.Select(1, 1)
	s.app.SetFocus(s.table)
	if !s.commandEnabled("grid_edit") {
		t.Fatal("editable table disabled")
	}
	s.invoke("grid_edit")
	if s.editChange.Row != 1 || s.editChange.ID != 77 {
		t.Fatal("sorted view changed another source row")
	}
	s.editForm.GetFormItem(0).(*tview.InputField).SetText("' ; DELETE FROM x; --")
	s.previewGridEdit()
	if s.programModal == nil || len(s.jobs) != 0 {
		t.Fatal("preview ran SQL")
	}
	// Default cancel must never submit an update.
	s.programModal.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	if len(s.jobs) != 0 || !s.pages.HasPage("grid_edit") {
		t.Fatal("default confirmation applied changes")
	}
	s.previewGridEdit()
	s.programModal.InputHandler()(tcell.NewEventKey(tcell.KeyLeft, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	s.programModal.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	select {
	case j := <-s.jobs:
		if j.action != "edit_apply" || j.edit.Row != 1 || !strings.Contains(j.edit.Text, "DELETE") {
			t.Fatal("change snapshot corrupted")
		}
	default:
		t.Fatal("confirmed edit not queued")
	}
	s.cancel()
	s.cancel = nil
	s.busy = false
	s.closeGridEdit()
	s.invalidateEditResults(0)
	if s.commandEnabled("grid_edit") {
		t.Fatal("expired snapshot editable")
	}
	s.render(oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"1"}}})
	if s.commandEnabled("grid_edit") {
		t.Fatal("arbitrary SELECT editable")
	}
	_ = terminal
}
func TestGridEditModalNarrowDrawAndButtonArrows(t *testing.T) {
	s, terminal := resultScreen(t)
	s.render(oracle.Result{EditID: 1, EditObject: oracle.Object{Owner: "A", Name: "B", Kind: "TABLE"}, Columns: []string{"VALUE"}, Rows: [][]string{{"old"}}, Values: [][]oracle.Cell{{{Type: "VARCHAR2", Text: "old"}}}})
	s.table.Select(1, 0)
	s.showGridEdit()
	for _, width := range []int{40, 80, 120} {
		terminal.SetSize(width, 30)
		_, root := s.pages.GetFrontPage()
		root.SetRect(0, 0, width, 28)
		root.Draw(terminal)
	}
	focusForm(s, s.editForm, 2)
	s.gridEditKey(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	if !s.editForm.GetButton(1).HasFocus() {
		t.Fatal("button arrows failed")
	}
	s.gridEditKey(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.app.GetFocus() != s.table {
		t.Fatal("edit close focus")
	}
}
