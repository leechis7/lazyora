package ui

import (
	"fmt"
	"lazyora/internal/oracle"

	"github.com/leechis7/tuivory/go"
)

func (s *screen) requestReconnect(config oracle.Config) {
	if s.busy || s.reconnectDialog != nil {
		return
	}
	if s.gridDraftCount() != 0 && !s.transactionUncertain {
		s.setStatus("grid_resolve_drafts")
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
	message := fmt.Sprintf(s.tr(messageKey), tuivory.Escape(config.User), tuivory.Escape(config.Host), config.Port, tuivory.Escape(config.Name))
	s.reconnectDialog = tuivory.NewModal().SetText(message).AddButtons(labels).
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
	s.reconnectDialog.SetMouseCapture(func(action tuivory.MouseAction, event *tuivory.EventMouse) (tuivory.MouseAction, *tuivory.EventMouse) {
		if action == tuivory.MouseLeftDoubleClick {
			action = tuivory.MouseLeftClick
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
