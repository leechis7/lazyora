package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func reconnectScreen(t *testing.T) *screen {
	t.Helper()
	s := shortcutScreen(t)
	s.pendingChanges = true
	s.connectionForm()
	fillConnection(s, "test", "service")
	s.connectAction()
	if s.reconnectDialog == nil || len(s.jobs) != 0 {
		t.Fatal("pending changes did not require confirmation")
	}
	return s
}

func TestReconnectDefaultAndEscapePreserveInputs(t *testing.T) {
	for _, key := range []tcell.Key{tcell.KeyEnter, tcell.KeyEscape} {
		s := reconnectScreen(t)
		if key == tcell.KeyEnter {
			s.reconnectDialog.InputHandler()(tcell.NewEventKey(key, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
		} else {
			s.key(tcell.NewEventKey(key, 0, 0))
		}
		if page, _ := s.pages.GetFrontPage(); page != "connect" || s.reconnectDialog != nil || len(s.jobs) != 0 || !s.pendingChanges {
			t.Fatal("cancel changed connection or submitted SQL")
		}
		if s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).GetText() != "secret-only-in-memory" || s.editor.GetText() != "SELECT 1 FROM dual;" {
			t.Fatal("cancel discarded inputs")
		}
	}
}

func TestReconnectSelectionQueuesOneExplicitChoice(t *testing.T) {
	for i, choice := range []oracle.ReconnectResolution{oracle.ReconnectCommit, oracle.ReconnectRollback} {
		s := reconnectScreen(t)
		// Use the same arrows and Enter handled by tview's actual modal.
		for move := 0; move < 2-i; move++ {
			s.reconnectDialog.InputHandler()(tcell.NewEventKey(tcell.KeyLeft, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
		}
		s.reconnectDialog.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
		if len(s.jobs) != 1 || !s.busy || s.reconnectDialog != nil || s.connectForm != nil {
			t.Fatal("confirmation did not submit exactly one connect job")
		}
		j := <-s.jobs
		if j.action != "connect" || j.resolution != choice || j.config.Host != "db.example" || j.config.Password != "secret-only-in-memory" {
			t.Fatal("connection snapshot or transaction choice lost")
		}
	}
}

func TestReconnectDialogRejectsOtherActions(t *testing.T) {
	s := reconnectScreen(t)
	dialog := s.reconnectDialog
	for _, event := range []*tcell.EventKey{
		tcell.NewEventKey(tcell.KeyCtrlQ, 0, 0), tcell.NewEventKey(tcell.KeyF1, 0, 0),
		tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModCtrl), tcell.NewEventKey(tcell.KeyF9, 0, 0),
	} {
		s.key(event)
	}
	s.connectAction()
	if s.reconnectDialog != dialog || len(s.jobs) != 0 || s.exitDialog != nil || s.busy {
		t.Fatal("other shortcut bypassed confirmation")
	}
}

func TestCleanConnectDoesNotPrompt(t *testing.T) {
	for _, connected := range []bool{false, true} {
		s := shortcutScreen(t)
		s.connected = connected
		s.connectionForm()
		fillConnection(s, "test", "sid")
		s.connectAction()
		if s.reconnectDialog != nil || len(s.jobs) != 1 {
			t.Fatal("clean connection was blocked")
		}
		j := <-s.jobs
		if j.resolution != "" || j.config.Mode != "sid" {
			t.Fatal("clean connection resolved a transaction")
		}
	}
}

func TestUncertainReconnectDefaultsToCancelAndRequiresAbandonment(t *testing.T) {
	s := shortcutScreen(t)
	s.pendingChanges, s.transactionUncertain = true, true
	s.connectionForm()
	fillConnection(s, "test", "service")
	s.connectAction()
	s.reconnectDialog.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	if len(s.jobs) != 0 || !s.transactionUncertain || s.connectForm == nil {
		t.Fatal("default action discarded uncertain transaction")
	}
	s.connectAction()
	s.reconnectDialog.InputHandler()(tcell.NewEventKey(tcell.KeyLeft, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	s.reconnectDialog.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	j := <-s.jobs
	if j.resolution != oracle.ReconnectDiscardUncertain {
		t.Fatal("uncertain recovery was misrepresented as commit or rollback")
	}
}
