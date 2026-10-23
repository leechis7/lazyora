package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"testing"
)

func TestBrowserFilterMatchesOriginalObjectsWithoutQueries(t *testing.T) {
	s := demoBrowserScreen(t)
	s.browserObjects = []oracle.Object{{Owner: "LAZYORA", Name: "EMP", Kind: "TABLE"}, {Owner: "LAZYORA", Name: "Temp한글", Kind: "TABLE"}, {Owner: "LAZYORA", Name: "EMP_VIEW", Kind: "VIEW"}}
	s.renderBrowserObjects()
	s.browserSearch.SetText("emp")
	children := s.browser.GetRoot().GetChildren()
	if len(children) != 2 || children[1].GetReference().(*browserNode).object.Name != "Temp한글" || len(s.jobs) != 0 {
		t.Fatal("wrong filter/source mapping or unexpected query")
	}
	s.browserSearch.SetText("한글")
	if len(s.browser.GetRoot().GetChildren()) != 1 {
		t.Fatal("unicode filter")
	}
	s.selectBrowserType(1)
	if s.browser.GetCurrentNode() != nil {
		t.Fatal("no-result state selectable")
	}
	s.browserSearch.SetText("")
	if s.browser.GetCurrentNode().GetReference().(*browserNode).object.Name != "EMP_VIEW" {
		t.Fatal("type filter lost")
	}
	s.invoke("browser_search")
	if !s.browserSearch.HasFocus() {
		t.Fatal("search shortcut target")
	}
	s.browserSearch.InputHandler()(tcell.NewEventKey(tcell.KeyDown, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	if !s.browser.HasFocus() {
		t.Fatal("cannot reach results")
	}
	s.browserSearch.SetText("emp")
	s.loadBrowserUser("OTHER")
	if s.browserSearch.GetText() != "" {
		t.Fatal("other owner inherited search")
	}
}
