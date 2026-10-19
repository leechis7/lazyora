package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"lazyora/internal/oracle"
)

func TestSQLTabsPreserveWidgetsAndResults(t *testing.T) {
	s := fileTestScreen(t)
	s.initTabs()
	firstEditor, firstTable := s.editor, s.table
	s.editor.SetText("SELECT 111 FROM dual;", false).Select(7, 10).SetOffset(2, 0)
	s.render(oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"111"}, {"222"}}})
	s.table.Select(2, 0).SetOffset(1, 0)
	s.newDocument()
	if len(s.tabs) != 2 || s.activeTab != 1 || s.editor == firstEditor || s.editor.GetText() != "" || len(s.resultRows) != 0 {
		t.Fatal("New did not create isolated tab")
	}
	s.editor.SetText("SELECT 333 FROM dual;", false)
	s.render(oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"333"}}})
	s.key(tcell.NewEventKey(tcell.KeyPgUp, 0, tcell.ModAlt))
	selected, _, _ := s.editor.GetSelection()
	row, _ := s.table.GetSelection()
	if s.editor != firstEditor || s.table != firstTable || selected != "111" || row != 2 || s.resultRows[1][0] != "222" {
		t.Fatal("switch lost editor/result state")
	}
	s.key(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModCtrl))
	if s.activeTab != 1 || s.resultRows[0][0] != "333" {
		t.Fatal("next tab failed")
	}
	s.key(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModCtrl|tcell.ModShift))
	if s.activeTab != 0 {
		t.Fatal("previous tab failed")
	}
	if !s.pendingChanges || len(s.jobs) != 0 {
		t.Fatal("switch resolved DB transaction")
	}
}

func TestSQLTabCloseAndQuitProtectAllDocuments(t *testing.T) {
	s := fileTestScreen(t)
	s.initTabs()
	s.editor.SetText("first dirty", false)
	s.newDocument()
	s.editor.SetText("second dirty", false)
	s.closeDocument()
	modalChoice(s, s.documentModal, 2)
	if len(s.tabs) != 2 {
		t.Fatal("cancel closed tab")
	}
	s.requestQuit()
	if s.activeTab != 0 || s.documentModal == nil || s.exitDialog != nil {
		t.Fatal("quit did not check first dirty tab")
	}
	modalChoice(s, s.documentModal, 1)
	if s.activeTab != 1 || s.documentModal == nil || s.exitDialog != nil {
		t.Fatal("quit did not check second dirty tab")
	}
	modalChoice(s, s.documentModal, 1)
	if s.exitDialog == nil || !s.pendingChanges || len(s.jobs) != 0 {
		t.Fatal("file decisions skipped DB confirmation")
	}
	s.cancelExit()
	s.closeDocument()
	modalChoice(s, s.documentModal, 1)
	if len(s.tabs) != 1 || s.editor.GetText() != "first dirty" {
		t.Fatal("wrong tab removed")
	}
	s.closeDocument()
	modalChoice(s, s.documentModal, 1)
	if len(s.tabs) != 1 || s.editor.GetText() != "" || s.document.Dirty("") {
		t.Fatal("last closed tab not reset")
	}
}

func TestSQLTabsLockDuringExecution(t *testing.T) {
	s := fileTestScreen(t)
	s.initTabs()
	s.newDocument()
	s.busy = true
	s.switchTab(0)
	s.newDocument()
	s.closeDocument()
	if s.activeTab != 1 || len(s.tabs) != 2 || s.documentModal != nil {
		t.Fatal("execution did not lock tab ownership")
	}
	s.busy = false
}
