package ui

import (
	"lazyora/internal/oracle"
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestSessionTerminationRequiresExplicitApproval(t *testing.T) {
	s := shortcutScreen(t)
	s.setStatus("connected", "operator", "db", 1521, "pdb", "service")
	s.invoke("session_browser")
	j := <-s.jobs
	jCancel := s.cancel
	jCancel()
	s.busy = false
	if j.action != "session_list" {
		t.Fatalf("opening browser submitted %q", j.action)
	}
	blocker := oracle.SessionEntry{ID: oracle.SessionID{Instance: 1, SID: 70, Serial: 541}, User: "APP", Logon: "20261111080000", Waiters: 2, CanTerminate: true}
	s.acceptSessions(oracle.SessionList{Entries: []oracle.SessionEntry{{Self: true}, blocker}}, nil)
	v := s.sessionBrowser
	if !v.kill.IsDisabled() {
		t.Fatal("self session offered termination")
	}
	v.list.SetCurrentItem(1)
	s.confirmSessionTermination()
	button := func() *tuivory.WidgetButton {
		focus, ok := s.app.GetFocus().(*tuivory.WidgetButton)
		if !ok {
			t.Fatal("confirmation button is not focused")
		}
		return focus
	}
	if got := button().GetLabel(); got != s.tr("cancel") {
		t.Fatalf("default action = %q", got)
	}
	if preview := s.sqlConfirmText.GetText(false); !strings.Contains(preview, "APP") || !strings.Contains(preview, "1:70,541") {
		t.Fatalf("confirmation does not identify target: %q", preview)
	}
	press := func(key tuivory.Key) *tuivory.EventKey { return tuivory.NewEventKey(key, 0, 0) }
	enter := func() { button().InputHandler()(press(tuivory.KeyEnter), func(tuivory.Primitive) {}) }
	enter()
	if page, _ := s.pages.GetFrontPage(); page != "session_browser" || len(s.jobs) != 0 {
		t.Fatal("default cancellation submitted a termination")
	}
	s.confirmSessionTermination()
	s.key(press(tuivory.KeyLeft))
	if got := button().GetLabel(); got != s.tr("session_kill") {
		t.Fatalf("approval action = %q", got)
	}
	enter()
	accepted := <-s.jobs
	s.cancel()
	if accepted.action != "session_kill" || accepted.sessionTarget != blocker {
		t.Fatalf("termination did not use confirmed session: %+v", accepted)
	}
}
