package ui

import (
	"lazyora/internal/oracle"
	"lazyora/internal/settings"
	"strings"
	"testing"
	"time"

	"github.com/leechis7/tuivory/go"
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
	s.key(tuivory.NewEventKey(tuivory.KeyRune, 'v', tuivory.ModAlt))
	if s.pendingTable == nil || s.pendingSQL.GetText(false) != s.pendingWork.Statements[0].SQL {
		t.Fatal("exit details did not show original SQL")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyF9, 0, 0))
	if len(s.jobs) != 0 {
		t.Fatal("details allowed execution behind overlay")
	}
	s.pendingTable.Select(2, 0)
	if !strings.Contains(s.pendingSQL.GetText(false), "cancelled [red]") {
		t.Fatal("details omitted execution error")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyTab, 0, 0))
	if !s.pendingSQL.HasFocus() {
		t.Fatal("Tab did not reach scrollable SQL")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyEscape, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "exit" || s.exitDialog == nil || !s.pendingChanges || len(s.jobs) != 0 {
		t.Fatal("details changed transaction/exit state")
	}
	s.exitDialog.InputHandler()(tuivory.NewEventKey(tuivory.KeyEnter, 0, 0), func(p tuivory.Primitive) { s.app.SetFocus(p) })
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
	s.key(tuivory.NewEventKey(tuivory.KeyEscape, 0, 0))
	if !s.editor.HasFocus() || !s.pendingChanges || len(s.jobs) != 0 {
		t.Fatal("journal close changed session state")
	}
}

func TestPendingCountRendersWithConnectionContext(t *testing.T) {
	for _, width := range []int{80, 120} {
		m, terminal := workspaceTestSet(t)
		s := m.active
		terminal.SetSize(width, 32)
		s.connected = true
		s.connectionIdentity = settings.Connection{User: "lazyora", Host: "127.0.0.1", Port: 1521, Name: "FREEPDB1"}
		s.pendingChanges = true
		s.pendingWork = oracle.PendingWork{Statements: []oracle.StatementRecord{{SQL: "UPDATE sample_rows SET title=title WHERE id=1", Affected: 1}}}
		s.showExitConfirmation()
		s.exitDialog.Draw(terminal)
		var visible strings.Builder
		for y := 0; y < 32; y++ {
			for x := 0; x < width; x++ {
				r, _, _, _ := terminal.GetContent(x, y)
				visible.WriteRune(r)
			}
			visible.WriteByte('\n')
		}
		if !strings.Contains(visible.String(), "Tracked statements: 1") {
			t.Fatalf("pending count clipped at %d: %s", width, visible.String())
		}
	}
}
