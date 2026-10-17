package ui

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func TestBrowserSelectorsLoadOnlySelectedUserAndPreserveSessionState(t *testing.T) {
	s := demoBrowserScreen(t)
	s.demo, s.connected = false, true
	s.pendingChanges = true
	s.render(oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"keep"}}})
	s.browserOwner = ""
	catalog := oracle.Catalog{Current: "LOGIN", Schemas: []string{"OTHER", "LOGIN"}}
	s.populateSchemas(catalog)
	j := <-s.jobs
	if index, _ := s.browserUsers.GetCurrentOption(); index != 1 || j.object.Owner != "LOGIN" {
		t.Fatal("default user does not match login")
	}
	s.cancel()
	s.cancel = nil
	s.busy = false
	s.populateObjects(j.node, []oracle.Object{{Owner: "LOGIN", Name: "T", Kind: "TABLE"}, {Owner: "LOGIN", Name: "V", Kind: "VIEW"}})
	s.browserTypes.SetCurrentOption(1)
	if s.browser.GetCurrentNode().GetReference().(*browserNode).object.Name != "V" || len(s.jobs) != 0 {
		t.Fatal("type selector did not filter cached objects")
	}
	s.browserUsers.SetCurrentOption(0)
	j = <-s.jobs
	if j.action != "objects" || j.object.Owner != "OTHER" || s.browser.GetCurrentNode() != nil {
		t.Fatal("user selection retained old objects or queried wrong user")
	}
	s.browserUsers.SetCurrentOption(1) // Loading prevents a second concurrent owner change.
	if index, _ := s.browserUsers.GetCurrentOption(); index != 0 || s.browserOwner != "OTHER" || len(s.jobs) != 0 {
		t.Fatal("busy owner selection was not reverted")
	}
	s.cancel()
	s.cancel = nil
	s.busy = false
	s.populateObjects(j.node, nil)
	if len(s.browser.GetRoot().GetChildren()) != 1 || s.browser.GetRoot().GetChildren()[0].GetText() != "(empty)" {
		t.Fatal("empty owner did not show empty state")
	}
	s.populateSchemas(catalog)
	j = <-s.jobs
	if j.object.Owner != "OTHER" {
		t.Fatal("refresh reset selected user")
	}
	s.cancel()
	s.cancel = nil
	s.busy = false
	s.resetBrowser()
	s.populateSchemas(catalog)
	j = <-s.jobs
	if j.object.Owner != "LOGIN" || s.browserKind != 0 {
		t.Fatal("new connection did not reset user/type")
	}
	if !s.pendingChanges || s.editor.GetText() != "SELECT 1 FROM dual;" || s.table.GetCell(1, 0).Text != "keep" {
		t.Fatal("browsing changed SQL/results/transaction")
	}
}

func TestBrowserSelectorKeysAndMouseLayout(t *testing.T) {
	s := demoBrowserScreen(t)
	terminal := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.Init(); err != nil {
		t.Fatal(err)
	}
	defer terminal.Fini()
	terminal.SetSize(100, 30)
	s.mainLayout.SetRect(0, 0, 100, 28)
	s.mainLayout.Draw(terminal)
	_, uy, _, _ := s.browserUsers.GetRect()
	_, ty, _, _ := s.browserTypes.GetRect()
	_, ly, _, _ := s.browser.GetRect()
	if uy >= ty || ty >= ly {
		t.Fatal("user/type selectors are not above the object list")
	}
	s.key(tcell.NewEventKey(tcell.KeyRune, 'u', tcell.ModAlt))
	if !s.browserUsers.HasFocus() {
		t.Fatal("Alt+U did not focus user selector")
	}
	event := tcell.NewEventKey(tcell.KeyEnter, 0, 0)
	s.browserUsers.InputHandler()(s.key(event), func(p tview.Primitive) { s.app.SetFocus(p) })
	event = tcell.NewEventKey(tcell.KeyEscape, 0, 0)
	forwarded := s.key(event)
	if forwarded == nil {
		t.Fatal("Escape did not reach open selector")
	}
	s.browserUsers.InputHandler()(forwarded, func(p tview.Primitive) { s.app.SetFocus(p) })
	if s.browserUsers.IsOpen() {
		t.Fatal("Escape did not close selector")
	}
	s.key(tcell.NewEventKey(tcell.KeyRune, 'o', tcell.ModAlt))
	if !s.browserTypes.HasFocus() {
		t.Fatal("Alt+O did not focus type selector")
	}
	s.invoke("browser")
	if !s.browserHidden || !s.editor.HasFocus() {
		t.Fatal("hiding browser left focus in selector")
	}
	s.key(tcell.NewEventKey(tcell.KeyRune, 'u', tcell.ModAlt))
	if s.browserHidden || !s.browserUsers.HasFocus() {
		t.Fatal("Alt+U did not reopen browser")
	}
	s.key(tcell.NewEventKey(tcell.KeyF2, 0, 0))
	if !s.editor.HasFocus() || !s.fullscreen {
		t.Fatal("F2 from user selector did not open SQL fullscreen")
	}
}

func TestNormalStartupShowsLoginAndAltLReopensIt(t *testing.T) {
	t.Setenv("LAZYORA_CONFIG", filepath.Join(t.TempDir(), "settings.json"))
	terminal := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.Init(); err != nil {
		t.Fatal(err)
	}
	terminal.SetSize(100, 30)
	ready := make(chan *screen, 1)
	finished := make(chan error, 1)
	go func() { finished <- run(false, terminal, func(s *screen) { ready <- s }) }()
	var s *screen
	select {
	case s = <-ready:
	case <-time.After(3 * time.Second):
		t.Fatal("startup timed out")
	}
	defer func() {
		s.app.QueueUpdate(func() { s.app.Stop() })
		select {
		case err := <-finished:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(3 * time.Second):
			t.Error("shutdown timed out")
		}
	}()
	s.app.QueueUpdateDraw(func() {
		if page, _ := s.pages.GetFrontPage(); page != "connect" || s.connected || s.busy {
			t.Error("startup did not show login without auto-connecting")
		}
		s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
		s.key(tcell.NewEventKey(tcell.KeyRune, 'l', tcell.ModAlt))
		if page, _ := s.pages.GetFrontPage(); page != "connect" || s.connectForm == nil {
			t.Error("Alt+L did not reopen login")
		}
	})
}
