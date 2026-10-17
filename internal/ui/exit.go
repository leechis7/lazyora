package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (s *screen) requestQuit() {
	if s.exitDialog != nil || s.programModal != nil || s.quitting {
		return
	}
	if s.busy {
		s.quitting = true
		s.cancel()
		s.setStatus("cancelling")
		return
	}
	s.guardSourceDiscard(s.showExitConfirmation)
}

func (s *screen) showExitConfirmation() {
	if s.exitDialog != nil {
		return
	}
	s.exitFocus = s.app.GetFocus()
	message := "exit_clean"
	labels := []string{s.tr("exit_now"), s.tr("exit_cancel")}
	if s.pendingChanges {
		message = "exit_pending"
		labels = []string{s.tr("exit_commit"), s.tr("exit_rollback"), s.tr("pending_work"), s.tr("exit_cancel")}
	}
	s.exitDialog = tview.NewModal().SetText(s.exitMessage(message)).AddButtons(labels).
		SetBackgroundColor(palette.background).SetTextColor(palette.text).
		SetButtonStyle(buttonStyle()).SetButtonActivatedStyle(buttonFocusStyle())
	s.exitDialog.SetDoneFunc(func(index int, _ string) {
		if index < 0 || index == len(labels)-1 {
			s.cancelExit()
			return
		}
		if s.pendingChanges && index == 2 {
			s.showPendingWork()
			return
		}
		action := "exit_now"
		if s.pendingChanges {
			if index == 0 {
				action = "exit_commit"
			} else {
				action = "exit_rollback"
			}
		}
		s.confirmExit(action)
	})
	styleBox(s.exitDialog.Box)
	s.exitDialog.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if s.busy {
			return action, nil
		}
		if action == tview.MouseLeftDoubleClick {
			action = tview.MouseLeftClick
		}
		return action, event
	})
	// Enter after the quit shortcut must never accidentally commit or exit.
	s.exitDialog.SetFocus(len(labels) - 1)
	s.pages.AddPage("exit", s.exitDialog, true, true)
	s.app.SetFocus(s.exitDialog)
}

func (s *screen) cancelExit() {
	if s.busy {
		return
	}
	s.pages.RemovePage("exit")
	s.exitDialog = nil
	if s.exitFocus != nil {
		s.app.SetFocus(s.exitFocus)
	}
	s.exitFocus = nil
}

func (s *screen) confirmExit(action string) {
	if s.busy {
		return
	}
	if action == "exit_now" && !s.pendingChanges {
		s.app.Stop()
		return
	}
	if s.pendingChanges && (action == "exit_commit" || action == "exit_rollback") {
		s.exitDialog.SetText(s.tr("exit_finishing"))
		s.submit(job{action: action})
	}
}

func (s *screen) exitKey(event *tcell.EventKey) *tcell.EventKey {
	if s.busy {
		return nil
	}
	if event.Modifiers()&tcell.ModAlt != 0 && event.Rune() == 'v' {
		if s.pendingChanges {
			s.showPendingWork()
		}
		return nil
	}
	if event.Modifiers()&tcell.ModAlt != 0 && event.Rune() == 'c' {
		s.confirmExit("exit_commit")
		return nil
	}
	if event.Modifiers()&tcell.ModAlt != 0 && event.Rune() == 'r' {
		s.confirmExit("exit_rollback")
		return nil
	}
	switch event.Key() {
	case tcell.KeyEscape:
		s.cancelExit()
		return nil

	}
	return event
}
