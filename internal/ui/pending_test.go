package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func TestExitPendingDetailsPreserveSQLAndReturnToSafeDefault(t *testing.T) {
	s := shortcutScreen(t)
	s.pendingChanges = true
	s.pendingWork = oracle.PendingWork{Statements: []oracle.StatementRecord{
		{At: time.Now(), SQL: "UPDATE items SET name='한글 [red]'\nWHERE id=1", Affected: 1},
		{At: time.Now(), SQL: "BEGIN do_work; END;", Affected: -1, Error: "cancelled [red]"},
	}}
	s.app.SetFocus(s.table)
	s.requestQuit()
	if !strings.Contains(s.exitMessage("exit_pending"), "2") {
		t.Fatal("exit did not show statement count")
	}
	s.key(tcell.NewEventKey(tcell.KeyRune, 'v', tcell.ModAlt))
	if s.pendingTable == nil || s.pendingSQL.GetText(false) != s.pendingWork.Statements[0].SQL {
		t.Fatal("exit details did not show original SQL")
	}
	s.key(tcell.NewEventKey(tcell.KeyF9, 0, 0))
	if len(s.jobs) != 0 {
		t.Fatal("details allowed execution behind overlay")
	}
	s.pendingTable.Select(2, 0)
	if !strings.Contains(s.pendingSQL.GetText(false), "cancelled [red]") {
		t.Fatal("details omitted execution error")
	}
	s.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	if !s.pendingSQL.HasFocus() {
		t.Fatal("Tab did not reach scrollable SQL")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "exit" || s.exitDialog == nil || !s.pendingChanges || len(s.jobs) != 0 {
		t.Fatal("details changed transaction/exit state")
	}
	s.exitDialog.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	if s.exitDialog != nil || len(s.jobs) != 0 || !s.table.HasFocus() {
		t.Fatal("return from details lost default Keep working")
	}
	if s.editor.GetText() != "SELECT 1 FROM dual;" {
		t.Fatal("journal replaced editor")
	}
}

func TestPendingJournalCanOpenFromSessionAndDisplaysOmissions(t *testing.T) {
	s := shortcutScreen(t)
	s.pendingChanges = true
	s.pendingWork = oracle.PendingWork{Omitted: 7, Statements: []oracle.StatementRecord{{SQL: "UPDATE t SET x=1", Affected: 0}}}
	s.invoke("pending_work")
	if s.pendingTable.GetCell(1, 0).Text != "8" || s.pendingTable.GetCell(1, 3).Text != "0" {
		t.Fatal("omitted numbering/zero-row count was lost")
	}
	s.selectPendingStatement(0)
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if !s.editor.HasFocus() || !s.pendingChanges || len(s.jobs) != 0 {
		t.Fatal("journal close changed session state")
	}
}
