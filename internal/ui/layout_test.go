package ui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"path/filepath"
	"testing"
	"time"
)

func TestMaximizeSwitchRestorePreservesPanelState(t *testing.T) {
	s := demoBrowserScreen(t)
	var rows [][]string
	for i := 0; i < 100; i++ {
		rows = append(rows, []string{fmt.Sprint(i)})
	}
	s.render(oracle.Result{Columns: []string{"N"}, Rows: rows})
	s.table.Select(51, 0).SetOffset(40, 0)
	node := s.browser.GetCurrentNode()
	s.invoke("maximize")
	if s.maximized != "sql" || s.browserHidden {
		t.Fatal("SQL maximize changed browser preference")
	}
	terminal := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.Init(); err != nil {
		t.Fatal(err)
	}
	defer terminal.Fini()
	terminal.SetSize(100, 30)
	s.mainLayout.SetRect(0, 0, 100, 28)
	s.mainLayout.Draw(terminal)
	x, _, w, _ := s.editor.GetRect()
	if x != 0 || w != 100 {
		t.Fatal("SQL did not fill work area")
	}
	s.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	if s.maximized != "results" || !s.table.HasFocus() {
		t.Fatal("Tab did not switch maximized panel")
	}
	terminal.SetSize(80, 24)
	s.mainLayout.SetRect(0, 0, 80, 22)
	s.mainLayout.Draw(terminal)
	x, _, w, _ = s.table.GetRect()
	if x != 0 || w != 80 {
		t.Fatal("maximized result did not resize")
	}
	s.key(tcell.NewEventKey(tcell.KeyF1, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.maximized != "results" || !s.table.HasFocus() {
		t.Fatal("help lost panel focus")
	}
	s.invoke("maximize")
	row, _ := s.table.GetSelection()
	offset, _ := s.table.GetOffset()
	if s.maximized != "" || s.browserHidden || row != 51 || offset != 40 || s.browser.GetCurrentNode() != node {
		t.Fatal("restore lost split/selection/scroll/tree")
	}
	if s.editor.GetText() != "SELECT 1 FROM dual;" || s.table.GetCell(51, 0).Text != "50" || len(s.jobs) != 0 {
		t.Fatal("layout change altered SQL/data")
	}
	s.invoke("browser")
	s.invoke("maximize")
	s.invoke("maximize")
	if !s.browserHidden {
		t.Fatal("restore opened explicitly hidden browser")
	}
}

func TestFullscreenOccupiesEntireTerminalAndRestoresChrome(t *testing.T) {
	t.Setenv("LAZYORA_CONFIG", filepath.Join(t.TempDir(), "settings.json"))
	terminal := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.Init(); err != nil {
		t.Fatal(err)
	}
	terminal.SetSize(100, 30)
	ready := make(chan *screen, 1)
	finished := make(chan error, 1)
	go func() { finished <- run(true, terminal, func(s *screen) { ready <- s }) }()
	var s *screen
	select {
	case s = <-ready:
	case <-time.After(3 * time.Second):
		t.Fatal("app did not initialize")
	}
	t.Cleanup(func() {
		s.app.QueueUpdate(func() { s.app.Stop() })
		select {
		case err := <-finished:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(3 * time.Second):
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
		t.Fatal("layout condition not reached")
	}
	s.app.QueueUpdateDraw(func() { terminal.SetSize(100, 30) })
	fills := func(p tview.Primitive, w, h int) bool {
		x, y, width, height := p.GetRect()
		return x == 0 && y == 0 && width == w && height == h
	}
	// The menu-only maximize command retains chrome.
	s.app.QueueUpdateDraw(func() { s.invoke("maximize") })
	wait(func() bool {
		_, y, _, h := s.editor.GetRect()
		return s.maximized == "sql" && !s.fullscreen && !s.chromeHidden && y == 2 && h == 24
	})
	s.app.QueueUpdate(func() {
		_, _, _, h := s.status.GetRect()
		if h != 2 {
			t.Error("maximize hid status")
		}
		_, _, _, h = s.menuLayout.GetRect()
		if h != 1 {
			t.Error("maximize hid menus")
		}
	})
	// Switching modes keeps the same panel; pressing its key again restores split.
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyF2, 0, 0))
	wait(func() bool { return s.fullscreen && fills(s.editor, 100, 30) })
	s.app.QueueUpdateDraw(func() { s.invoke("maximize") })
	wait(func() bool { return !s.fullscreen && !s.chromeHidden && s.maximized == "sql" })
	s.app.QueueUpdateDraw(func() { s.invoke("maximize") })
	wait(func() bool { return s.maximized == "" })
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyF2, 0, 0))
	wait(func() bool { return s.chromeHidden && fills(s.editor, 100, 30) })
	s.app.QueueUpdate(func() {
		for _, p := range []tview.Primitive{s.menuLayout, s.tabBar, s.footer, s.footerBar, s.status, s.table, s.browser} {
			_, _, w, h := p.GetRect()
			if w != 0 || h != 0 {
				t.Error("unrelated UI still occupies fullscreen")
			}
		}
	})
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyF1, 0, 0))
	wait(func() bool { page, _ := s.pages.GetFrontPage(); return page == "help" && !s.chromeHidden })
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	wait(func() bool { return s.chromeHidden && fills(s.editor, 100, 30) })
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyRune, 'm', tcell.ModAlt))
	wait(func() bool { return s.menuList != nil && !s.chromeHidden })
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	wait(func() bool { return s.chromeHidden && fills(s.editor, 100, 30) })
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyF6, 0, 0))
	wait(func() bool { return s.maximized == "results" && fills(s.table, 100, 30) })
	s.app.QueueUpdateDraw(func() { terminal.SetSize(80, 24) })
	wait(func() bool { return fills(s.table, 80, 24) })
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyCtrlQ, 0, 0))
	wait(func() bool { return s.exitDialog != nil && !s.chromeHidden })
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	wait(func() bool { return s.chromeHidden && fills(s.table, 80, 24) })
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyF2, 0, tcell.ModShift))
	wait(func() bool {
		_, y, _, h := s.footerBar.GetRect()
		return !s.chromeHidden && s.maximized == "" && y == 23 && h == 1
	})
	s.app.QueueUpdate(func() {
		x, y, w, h := s.editor.GetRect()
		if x <= 0 || y != 2 || w == 0 || h == 0 {
			t.Error("split SQL/browser layout not restored")
		}
		_, _, _, h = s.status.GetRect()
		if h != 2 {
			t.Error("status not restored")
		}
		if s.editor.GetText() != "SELECT sysdate AS server_time FROM dual;" || s.table.GetCell(1, 0).Text != "2026-10-14 12:00:00" {
			t.Error("fullscreen changed SQL/results")
		}
	})
}
