package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestQuitAlwaysConfirmsAndEscapePreservesFocus(t *testing.T) {
	for _, pending := range []bool{false, true} {
		for _, key := range []tcell.Key{tcell.KeyCtrlQ} {
			s := shortcutScreen(t)
			s.pendingChanges = pending
			s.app.SetFocus(s.table)
			s.key(tcell.NewEventKey(key, 0, tcell.ModNone))
			if page, _ := s.pages.GetFrontPage(); page != "exit" || s.exitDialog == nil {
				t.Fatal("quit did not show confirmation")
			}
			dialog := s.exitDialog
			s.key(tcell.NewEventKey(key, 0, tcell.ModNone))
			if s.exitDialog != dialog {
				t.Fatal("repeated quit replaced confirmation")
			}
			s.key(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
			if page, _ := s.pages.GetFrontPage(); page != "main" || !s.table.HasFocus() || s.exitDialog != nil || s.pendingChanges != pending {
				t.Fatal("cancel changed focus, page or transaction state")
			}
			if len(s.jobs) != 0 || s.editor.GetText() != "SELECT 1 FROM dual;" {
				t.Fatal("quit/cancel submitted SQL or changed editor")
			}
		}
	}
}

func TestPendingQuitActionsUseWorkerAndDefaultEnterCancels(t *testing.T) {
	for _, action := range []string{"exit_commit", "exit_rollback"} {
		s := shortcutScreen(t)
		s.pendingChanges = true
		s.requestQuit()
		s.confirmExit(action)
		j := <-s.jobs
		if j.action != action || !s.busy {
			t.Fatal("quit action did not use worker")
		}
		s.cancelExit()
		if s.exitDialog == nil {
			t.Fatal("closed dialog during transaction resolution")
		}
	}
	s := shortcutScreen(t)
	s.pendingChanges = true
	s.requestQuit()
	s.exitDialog.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) { s.app.SetFocus(p) })
	if s.exitDialog != nil || len(s.jobs) != 0 {
		t.Fatal("default Enter did not cancel safely")
	}
}

func TestBusyQuitRequestsCancellationBeforeConfirmation(t *testing.T) {
	s := shortcutScreen(t)
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel, s.busy = cancel, true
	s.requestQuit()
	if !s.quitting || ctx.Err() == nil || s.exitDialog != nil {
		t.Fatal("busy quit did not cancel before prompting")
	}
	// The worker completes the operation before constructing this dialog.
	s.busy, s.quitting, s.pendingChanges = false, false, true
	s.showExitConfirmation()
	s.cancelExit()
	if strings.Contains(s.editor.GetText(), "COMMIT") || s.pendingChanges != true {
		t.Fatal("busy cancellation resolved transaction implicitly")
	}
}
