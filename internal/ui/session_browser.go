package ui

import (
	"errors"
	"fmt"
	"lazyora/internal/oracle"
	"strings"

	"github.com/leechis7/tuivory/go"
)

type sessionBrowserView struct {
	entries       []oracle.SessionEntry
	list          *tuivory.WidgetList
	info, detail  *tuivory.WidgetTextView
	refresh, kill *tuivory.WidgetButton
	focus         tuivory.Primitive
}

func (s *screen) showSessionBrowser() {
	if s.sessionBrowser != nil {
		return
	}
	v := &sessionBrowserView{
		list:   tuivory.NewWidgetList().ShowSecondaryText(false),
		info:   s.newTextView(textHint).SetDynamicColors(false).SetWrap(true),
		detail: s.newTextView().SetDynamicColors(false).SetWrap(true).SetScrollable(true),
		focus:  s.app.GetFocus(),
	}
	s.sessionBrowser = v
	s.styleList(v.list)
	v.list.SetBorder(true).SetTitle(s.tr("session_browser"))
	v.detail.SetBorder(true).SetTitle(s.tr("object_details"))
	s.styleBox(v.detail.Box)
	v.list.SetChangedFunc(func(index int, _, _ string, _ rune) { s.describeSession(index) })
	v.list.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		if index >= 0 && index < len(v.entries) {
			s.app.SetFocus(v.detail)
		}
	})
	v.refresh = s.clickable(tuivory.NewWidgetButton(s.tr("session_refresh")).SetSelectedFunc(s.refreshSessions))
	v.kill = s.clickable(tuivory.NewWidgetButton(s.tr("session_kill")).SetSelectedFunc(s.confirmSessionTermination))
	close := s.clickable(tuivory.NewWidgetButton(s.tr("close")).SetSelectedFunc(s.closeSessionBrowser))
	buttons := s.newFlex().AddItem(v.refresh, 0, 1, false).AddItem(v.kill, 0, 1, false).AddItem(close, 0, 1, false)
	panels := s.newFlex().AddItem(v.list, 0, 3, true).AddItem(v.detail, 0, 2, false)
	root := s.newFlex().SetDirection(tuivory.FlexRow).AddItem(v.info, 3, 0, false).AddItem(panels, 0, 1, true).AddItem(buttons, 1, 0, false)
	s.pages.AddPage("session_browser", s.formPopup(root, 108, 30, nil), true, true)
	s.app.SetFocus(v.list)
	s.refreshSessions()
}

func (s *screen) refreshSessions() {
	v := s.sessionBrowser
	if v == nil || s.busy {
		return
	}
	v.refresh.SetDisabled(true)
	v.kill.SetDisabled(true)
	v.info.SetText(s.tr("browser_loading"))
	s.app.SetFocus(v.list)
	s.submit(job{action: "session_list"})
}

func (s *screen) describeSession(index int) {
	v := s.sessionBrowser
	if index < 0 || index >= len(v.entries) {
		v.detail.SetText("")
		v.kill.SetDisabled(true)
		return
	}
	item := v.entries[index]
	blocker := "-"
	if item.BlockedBy.SID != 0 {
		blocker = item.BlockedBy.String()
	}
	v.detail.SetText(fmt.Sprintf(s.tr("session_detail"), item.ID, strings.TrimSpace(item.User), item.Status,
		item.Kind, blocker, item.Waiters, strings.TrimSpace(item.Event), strings.TrimSpace(item.SQLID),
		strings.TrimSpace(item.Machine), strings.TrimSpace(item.Program))).ScrollToBeginning()
	v.kill.SetDisabled(s.busy || s.transactionUncertain || !item.CanTerminate)
}

func (s *screen) acceptSessions(list oracle.SessionList, err error) {
	v := s.sessionBrowser
	if v == nil {
		return
	}
	v.entries = nil
	v.list.Clear()
	v.refresh.SetDisabled(false)
	s.describeSession(-1)
	if err != nil {
		v.info.SetText(s.tr("session_read_failed") + "\n" + err.Error())
		return
	}
	v.entries = list.Entries
	for _, item := range v.entries {
		label := fmt.Sprintf("%s  %s  %s", item.ID, strings.TrimSpace(item.User), item.Status)
		switch {
		case item.Self:
			label += " · " + s.tr("session_current")
		case item.Waiters > 0:
			label += " · " + fmt.Sprintf(s.tr("session_waiter_count"), item.Waiters)
		case item.BlockedBy.SID != 0:
			label += " ← " + item.BlockedBy.String()
		}
		v.list.AddItem(tuivory.Escape(label), "", 0, nil)
	}
	hint := fmt.Sprintf(s.tr("session_summary"), s.connectionBadge(), len(v.entries))
	if list.Truncated {
		hint += "\n" + s.tr("session_truncated")
	}
	v.info.SetText(hint)
	s.describeSession(v.list.GetCurrentItem())
}

func (s *screen) confirmSessionTermination() {
	v := s.sessionBrowser
	if v == nil || s.busy || s.transactionUncertain {
		return
	}
	i := v.list.GetCurrentItem()
	if i < 0 || i >= len(v.entries) || !v.entries[i].CanTerminate {
		return
	}
	target := v.entries[i]
	message := fmt.Sprintf(s.tr("session_kill_warning"), tuivory.Escape(s.connectionBadge()))
	preview := fmt.Sprintf(s.tr("session_kill_target"), target.User, target.ID, target.Waiters)
	s.confirmActionPreview(preview, message, s.tr("session_browser"), s.tr("session_kill"), func() {
		v.kill.SetDisabled(true)
		v.refresh.SetDisabled(true)
		v.info.SetText(s.tr("session_kill_running"))
		s.app.SetFocus(v.list)
		s.submit(job{action: "session_kill", sessionTarget: target})
	})
}

func (s *screen) acceptSessionTermination(marked bool, err error) {
	v := s.sessionBrowser
	if v == nil {
		return
	}
	for i := range v.entries {
		v.entries[i].CanTerminate = false
	}
	v.kill.SetDisabled(true)
	v.refresh.SetDisabled(false)
	switch {
	case errors.Is(err, oracle.ErrSessionChanged), errors.Is(err, oracle.ErrSessionProtected):
		v.info.SetText(s.tr("session_kill_changed"))
	case err != nil:
		v.info.SetText(s.tr("session_kill_failed") + "\n" + err.Error())
	case marked:
		v.info.SetText(s.tr("session_kill_marked"))
	default:
		v.info.SetText(s.tr("session_kill_sent"))
	}
}

func (s *screen) closeSessionBrowser() {
	v := s.sessionBrowser
	if v == nil || s.busy {
		return
	}
	s.pages.RemovePage("session_browser")
	s.sessionBrowser = nil
	s.app.SetFocus(v.focus)
}

func (s *screen) sessionBrowserKey(event *tuivory.EventKey) *tuivory.EventKey {
	if s.buttonArrow(event) {
		return nil
	}
	switch event.Key() {
	case tuivory.KeyEscape:
		s.closeSessionBrowser()
		return nil
	case tuivory.KeyTab, tuivory.KeyBacktab:
		s.dialogTab(event, s.sessionBrowser.list, s.sessionBrowser.detail)
		return nil
	}
	return event
}
