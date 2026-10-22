package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func (s *screen) requestReconnect(config oracle.Config) {
	if s.busy || s.reconnectDialog != nil {
		return
	}
	identity := s.pendingIdentity
	connect := func(resolution oracle.ReconnectResolution) {
		s.closeConnection()
		s.submit(job{action: "connect", config: config, resolution: resolution, identity: identity})
	}
	if !s.pendingChanges {
		connect("")
		return
	}
	labels := []string{s.tr("reconnect_commit"), s.tr("reconnect_rollback"), s.tr("cancel")}
	messageKey := "reconnect_pending"
	uncertain := s.transactionUncertain
	if uncertain {
		messageKey = "reconnect_uncertain"
		labels = []string{s.tr("reconnect_discard"), s.tr("cancel")}
	}
	message := fmt.Sprintf(s.tr(messageKey), tview.Escape(config.User), tview.Escape(config.Host), config.Port, tview.Escape(config.Name))
	s.reconnectDialog = tview.NewModal().SetText(message).AddButtons(labels).
		SetBackgroundColor(s.colors().background).SetTextColor(s.colors().text).
		SetButtonStyle(s.buttonStyle()).SetButtonActivatedStyle(s.buttonFocusStyle())
	s.styleModal(s.reconnectDialog)
	s.reconnectDialog.SetDoneFunc(func(index int, _ string) {
		if index < 0 || index >= len(labels)-1 {
			s.cancelReconnect()
			return
		}
		s.cancelReconnect()
		resolution := oracle.ReconnectCommit
		if index == 1 {
			resolution = oracle.ReconnectRollback
		}
		if uncertain {
			resolution = oracle.ReconnectDiscardUncertain
		}
		connect(resolution)
	})
	s.reconnectDialog.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftDoubleClick {
			action = tview.MouseLeftClick
		}
		return action, event
	})
	// Repeating Enter after the connection shortcut must default to cancellation.
	s.reconnectDialog.SetFocus(len(labels) - 1)
	s.pages.AddPage("reconnect", s.reconnectDialog, true, true)
	s.app.SetFocus(s.reconnectDialog)
}

func (s *screen) cancelReconnect() {
	s.pages.RemovePage("reconnect")
	s.reconnectDialog = nil
	if s.connectForm != nil {
		s.app.SetFocus(s.connectForm)
	}
}
