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
	if s.editor.GetText() != "SELECT 1 FROM dual;" || s.resultRows[50][0] != "50" || len(s.jobs) != 0 {
		t.Fatal("layout change altered SQL/data")
	}
	s.invoke("browser")
	s.invoke("maximize")
	s.invoke("maximize")
	if !s.browserHidden {
		t.Fatal("restore opened explicitly hidden browser")
	}
}

func TestEscapeRestoresFullscreenWithoutChangingWork(t *testing.T) {
	for _, panel := range []string{"sql", "results"} {
		for _, keymap := range []string{"", "vim", "notepad++"} {
			t.Run(panel+"/"+keymap, func(t *testing.T) {
				s, _ := resultScreen(t)
				s.prefs.EditorKeymap = keymap
				s.editor.SetText("SELECT '한글' FROM dual;", false).Select(7, 15)
				s.editor.SetOffset(2, 1)
				s.table.Select(2, 1).SetOffset(1, 0)
				rangeSelection := resultSelection{anchorRow: 1, anchorCol: 0, endRow: 2, endCol: 1, active: true}
				s.resultSelection = rangeSelection
				s.browserHidden = keymap == "notepad++"
				hidden := s.browserHidden
				selection, from, to := s.editor.GetSelection()
				node := s.browser.GetCurrentNode()
				rows := fmt.Sprint(s.resultRows)
				s.pendingChanges = true
				key := tcell.NewEventKey(tcell.KeyF2, 0, 0)
				if panel == "results" {
					key = tcell.NewEventKey(tcell.KeyF2, 0, tcell.ModShift)
				}
				s.key(key)
				if !s.fullscreen || s.maximized != panel {
					t.Fatal("fullscreen entry failed")
				}
				s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
				if s.fullscreen || s.maximized != "" || s.browser.GetCurrentNode() != node || s.browserHidden != hidden {
					t.Fatal("Escape did not restore split and schema visibility")
				}
				selected, start, end := s.editor.GetSelection()
				row, col := s.table.GetSelection()
				if s.editor.GetText() != "SELECT '한글' FROM dual;" || selected != selection || start != from || end != to || row != 2 || col != 1 || s.resultSelection != rangeSelection || fmt.Sprint(s.resultRows) != rows || !s.pendingChanges || len(s.jobs) != 0 {
					t.Fatal("Escape changed SQL, selection, results or transaction")
				}
				er, ec := s.editor.GetOffset()
				rr, rc := s.table.GetOffset()
				if er != 2 || ec != 1 || rr != 1 || rc != 0 {
					t.Fatal("Escape lost split scroll positions")
				}
				if panel == "sql" && !s.editor.HasFocus() || panel == "results" && !s.table.HasFocus() {
					t.Fatal("Escape lost the active panel")
				}
			})
		}
	}
}

func TestFullscreenEscapeCancelsRunningQueryBeforeRestoringLayout(t *testing.T) {
	s, _ := resultScreen(t)
	s.key(tcell.NewEventKey(tcell.KeyF2, 0, 0))
	cancelled := false
	s.busy = true
	s.cancel = func() { cancelled = true }
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if !cancelled || !s.fullscreen || s.maximized != "sql" {
		t.Fatal("Escape must cancel the query before restoring fullscreen")
	}
	s.busy = false
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.fullscreen || s.maximized != "" {
		t.Fatal("Escape did not restore after the query finished")
	}
}

func TestResultFullscreenAliasesRestoreAndPreserveModalPriority(t *testing.T) {
	for _, key := range []tcell.Key{tcell.KeyF14, tcell.KeyF12} {
		for _, keymap := range []string{"", "vim", "notepad++"} {
			t.Run(fmt.Sprintf("%v/%s", key, keymap), func(t *testing.T) {
				s, _ := resultScreen(t)
				s.prefs.EditorKeymap = keymap
				s.app.SetFocus(s.editor)
				text := s.editor.GetText()
				s.key(tcell.NewEventKey(key, 0, 0))
				if !s.fullscreen || s.maximized != "results" || !s.table.HasFocus() {
					t.Fatal("result fullscreen alias did not act as Shift+F2")
				}
				s.showHelp()
				s.key(tcell.NewEventKey(key, 0, 0))
				if page, _ := s.pages.GetFrontPage(); page != "help" || !s.fullscreen || s.maximized != "results" {
					t.Fatal("legacy key changed layout behind help")
				}
				s.closeHelp()
				s.key(tcell.NewEventKey(key, 0, 0))
				if s.fullscreen || s.maximized != "" || s.editor.GetText() != text || len(s.jobs) != 0 {
					t.Fatal("legacy key failed to restore split or changed work")
				}
				s.key(tcell.NewEventKey(tcell.KeyF2, 0, 0))
				if s.maximized != "sql" {
					t.Fatal("legacy alias changed normal F2")
				}
				s.key(tcell.NewEventKey(key, 0, 0))
				if s.maximized != "results" || !s.fullscreen {
					t.Fatal("legacy key did not switch fullscreen SQL to results")
				}
				s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
				if s.fullscreen || s.maximized != "" {
					t.Fatal("Escape no longer restores split")
				}
			})
		}
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
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	wait(func() bool { return !s.chromeHidden && !s.fullscreen && s.maximized == "" })
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyF2, 0, 0))
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
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	wait(func() bool { return !s.chromeHidden && !s.fullscreen && s.maximized == "" })
	// Keep the original toggle-key restoration covered as well.
	s.app.QueueEvent(tcell.NewEventKey(tcell.KeyF2, 0, tcell.ModShift))
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
		if s.editor.GetText() != "SELECT sysdate AS server_time FROM dual;" || s.resultRows[0][0] != "2026-10-14 12:00:00" {
			t.Error("fullscreen changed SQL/results")
		}
	})
}
