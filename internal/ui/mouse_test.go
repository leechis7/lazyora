package ui

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/settings"
)

func TestMouseControlsAndSavedPreference(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	t.Setenv("LAZYORA_CONFIG", path)
	t.Setenv("LC_ALL", "C")
	terminal := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.Init(); err != nil {
		t.Fatal(err)
	}
	terminal.SetSize(80, 24)
	drawn, finished := make(chan struct{}), make(chan error, 1)
	var once sync.Once
	var s *screen
	go func() {
		finished <- run(true, terminal, func(ui *screen) {
			s = ui
			ui.app.SetAfterDrawFunc(func(tcell.Screen) { once.Do(func() { close(drawn) }) })
		})
	}()
	select {
	case <-drawn:
	case err := <-finished:
		t.Fatalf("app stopped before drawing: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("app did not draw")
	}
	t.Cleanup(func() {
		s.app.QueueEvent(tcell.NewEventKey(tcell.KeyCtrlQ, 0, tcell.ModNone))
		s.app.QueueEvent(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
		s.app.QueueEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
		select {
		case err := <-finished:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(5 * time.Second):
			t.Error("app did not stop")
		}
	})
	wait := func(check func() bool) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			var ok bool
			s.app.QueueUpdate(func() { ok = check() })
			if ok {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		s.app.QueueUpdate(func() {
			current, _ := s.pages.GetFrontPage()
			t.Logf("page=%s prefs=%+v", current, s.prefs)
			if s.prefsForm != nil {
				t.Logf("settings title=%s", s.prefsForm.GetTitle())
			}
		})
		t.Fatal("UI condition was not reached")
	}
	page := func(name string) bool { current, _ := s.pages.GetFrontPage(); return current == name }
	click := func(primitive func() tview.Primitive) {
		t.Helper()
		var x, y int
		s.app.QueueUpdate(func() { left, top, width, height := primitive().GetRect(); x, y = left+width/2, top+height/2 })
		s.app.QueueEvent(tcell.NewEventMouse(x, y, tcell.Button1, tcell.ModNone))
		s.app.QueueEvent(tcell.NewEventMouse(x, y, tcell.ButtonNone, tcell.ModNone))
	}
	wait(func() bool { return strings.Contains(s.footer.GetText(false), "F9 / Ctrl+Enter") })
	s.app.QueueUpdate(func() {
		_, editorY, _, _ := s.editor.GetRect()
		_, buttonsY, _, _ := s.buttons[0].GetRect()
		_, height := terminal.Size()
		if editorY != 2 || buttonsY != height-1 {
			t.Errorf("footer was not at bottom: editor=%d footer=%d", editorY, buttonsY)
		}
	})
	s.app.QueueUpdateDraw(func() { s.focusPanel("results") })
	wait(func() bool {
		return strings.Contains(s.footer.GetText(false), "Ctrl+C") && !strings.Contains(s.footer.GetText(false), "Ctrl+Enter")
	})
	click(func() tview.Primitive { return s.editor })
	wait(func() bool { return strings.Contains(s.footer.GetText(false), "F9 / Ctrl+Enter") })
	click(func() tview.Primitive { return s.table })
	wait(func() bool { return strings.Contains(s.footer.GetText(false), "Alt+Y") })

	s.app.QueueUpdateDraw(func() { s.invoke("browser") })
	wait(func() bool {
		x, _, width, _ := s.editor.GetRect()
		terminalWidth, _ := terminal.Size()
		return s.browserHidden && x == 0 && width == terminalWidth
	})
	s.app.QueueUpdateDraw(func() { s.invoke("browser") })
	wait(func() bool { return s.browser.HasFocus() && strings.Contains(s.footer.GetText(false), "Ctrl+R") })
	var objectX, objectY int
	s.app.QueueUpdate(func() {
		x, y, _, _ := s.browser.GetInnerRect()
		objectX, objectY = x+4, y // SAMPLE_ROWS in the flat object list
	})
	s.app.QueueEvent(tcell.NewEventMouse(objectX, objectY, tcell.Button1, tcell.ModNone))
	s.app.QueueEvent(tcell.NewEventMouse(objectX, objectY, tcell.ButtonNone, tcell.ModNone))
	wait(func() bool { return page("object") && s.object.Name == "SAMPLE_ROWS" })
	wait(func() bool { return strings.Contains(s.footer.GetText(false), "Esc Back") })
	click(func() tview.Primitive { return s.buttons[3] })
	wait(func() bool { return page("main") && s.browser.HasFocus() })
	s.app.QueueUpdateDraw(func() { s.focusPanel("results") })
	wait(func() bool { return s.table.HasFocus() })
	s.app.QueueUpdateDraw(func() { s.invoke("settings") })
	wait(func() bool { return page("settings") })
	s.app.QueueUpdateDraw(func() {
		s.prefsForm.GetFormItem(0).(*choiceField).SetCurrentOption(1)
		s.prefsForm.GetFormItem(1).(*tview.Checkbox).SetChecked(false)
	})
	click(func() tview.Primitive { return s.prefsForm.GetButton(0) })
	wait(func() bool { return page("main") && s.prefs.Language == "ko" && !s.prefs.Mouse })
	wait(func() bool { return strings.Contains(s.footer.GetText(false), "복사") })
	click(func() tview.Primitive { return s.buttons[1] })
	s.app.QueueUpdateDraw(func() { s.focusPanel("results") })
	wait(func() bool { return s.app.GetFocus() == s.table })
	s.app.QueueUpdate(func() {
		if !page("main") {
			t.Error("disabled mouse still opened connection")
		}
	})
	s.app.QueueUpdateDraw(func() { s.invoke("settings") })
	wait(func() bool { return page("settings") })
	s.app.QueueUpdate(func() { s.prefsForm.GetFormItem(1).(*tview.Checkbox).SetChecked(true) })
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyCtrlS, 0, tcell.ModCtrl))
	wait(func() bool { return page("main") && s.prefs.Mouse })
	click(func() tview.Primitive { return s.buttons[1] })
	wait(func() bool { return page("connect") })
	wait(func() bool {
		return strings.Contains(s.footer.GetText(false), "별칭") && strings.Contains(s.buttons[2].GetLabel(), "저장")
	})
	click(func() tview.Primitive { return s.connectProfiles.GetFormItem(0) })
	wait(func() bool { return s.connectProfiles.GetFormItem(0).(*choiceField).IsOpen() })
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	wait(func() bool { return !s.connectProfiles.GetFormItem(0).(*choiceField).IsOpen() })
	s.app.QueueUpdateDraw(func() { fillConnection(s, "마우스 저장 시험", "service") })
	click(func() tview.Primitive { return s.connectForm.GetButton(2) })
	wait(func() bool { return strings.Contains(s.connectForm.GetTitle(), "저장했습니다") })
	profiles, err := settings.LoadConnections(settings.ConnectionsPath(path))
	if err != nil || len(profiles) != 1 || profiles[0].Alias != "마우스 저장 시험" {
		t.Fatalf("mouse save did not persist the profile: %+v %v", profiles, err)
	}
	click(func() tview.Primitive { return s.connectForm.GetFormItem(connectionUser) })
	wait(func() bool { return s.connectForm.GetFormItem(connectionUser).HasFocus() })
	click(func() tview.Primitive { return s.connectForm.GetButton(1) })
	wait(func() bool { return page("main") })
	click(func() tview.Primitive { return s.buttons[0] })
	wait(func() bool { return page("help") })
	wait(func() bool {
		return strings.Contains(s.footer.GetText(false), "닫기") && strings.Contains(s.buttons[2].GetLabel(), "PgDn")
	})
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyRune, '1', 0))
	wait(func() bool { return s.helpTopics.GetCurrentItem() == 1 })
	s.app.QueueEvent(tcell.NewEventMouse(40, 10, tcell.WheelDown, tcell.ModNone))
	wait(func() bool {
		row, _ := s.helpText.GetScrollOffset()
		return row > 0
	})
	click(func() tview.Primitive { _, root := s.pages.GetFrontPage(); return root.(*formPopupLayer).GetItem(1) })
	wait(func() bool { return page("main") })
	loaded, err := settings.Load(path)
	if err != nil || loaded.Language != "ko" || !loaded.Mouse {
		t.Fatalf("saved settings were not restored: %+v %v", loaded, err)
	}
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyCtrlQ, 0, tcell.ModNone))
	wait(func() bool { return page("exit") })
	click(func() tview.Primitive { return s.editor })
	wait(func() bool { return s.exitDialog != nil && s.exitDialog.HasFocus() })
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	wait(func() bool { return page("main") })
}
