package ui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func browserPress(s *screen, key tcell.Key, mods tcell.ModMask) {
	s.browser.InputHandler()(tcell.NewEventKey(key, 0, mods), func(p tview.Primitive) { s.app.SetFocus(p) })
}

func TestTuivoryBrowserExpandsColumnsWithoutChangingSQL(t *testing.T) {
	s := demoBrowserScreen(t)
	s.pendingChanges = true
	s.render(oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"keep"}}})
	n := s.browser.GetCurrentNode()
	browserPress(s, tcell.KeyRight, 0)
	if !n.IsExpanded() || len(n.GetChildren()) != 3 || len(s.jobs) != 0 {
		t.Fatal("demo expansion did not load columns locally")
	}
	browserPress(s, tcell.KeyRight, 0)
	selected := s.browser.GetCurrentNode()
	if selected == nil || selected.GetReference().(*browserNode).column.Name != "ID" {
		t.Fatal("right did not enter columns")
	}
	browserPress(s, tcell.KeyLeft, 0)
	if s.browser.GetCurrentNode() != n {
		t.Fatal("left did not return to table")
	}
	browserPress(s, tcell.KeyLeft, 0)
	if n.IsExpanded() {
		t.Fatal("left did not collapse")
	}
	browserPress(s, tcell.KeyRight, 0)
	if n.GetChildren()[0] != selected {
		t.Fatal("re-expansion replaced cached columns")
	}
	s.browser.SetCurrentNode(selected)
	browserPress(s, tcell.KeyEnter, 0)
	if name, _ := s.pages.GetFrontPage(); name != "object" || s.object.Name != "SAMPLE_ROWS" {
		t.Fatal("column activation did not open parent details")
	}
	if !s.pendingChanges || s.editor.GetText() != "SELECT 1 FROM dual;" || s.resultRows[0][0] != "keep" {
		t.Fatal("tree navigation changed user work")
	}
}

func TestTuivoryBrowserColumnsRetryAndRejectStaleResponses(t *testing.T) {
	s := demoBrowserScreen(t)
	s.demo, s.connected = false, true
	n := s.browser.GetCurrentNode()
	browserPress(s, tcell.KeyRight, 0)
	j := <-s.jobs
	if j.action != "browser_columns" || j.node != n || j.object.Name != "SAMPLE_ROWS" {
		t.Fatal("wrong column request")
	}
	s.cancel()
	s.cancel = nil
	s.busy = false
	s.acceptBrowserColumns(n, nil, errors.New("canceled"))
	browserPress(s, tcell.KeyLeft, 0)
	browserPress(s, tcell.KeyRight, 0)
	retry := <-s.jobs
	if retry.node != n {
		t.Fatal("failed column lookup was cached")
	}
	s.cancel()
	s.cancel = nil
	s.busy = false
	s.loadBrowserUser("LAZYORA") // Same name, different generation.
	objects := <-s.jobs
	s.cancel()
	s.cancel = nil
	s.busy = false
	s.populateObjects(objects.node, demoObjects("LAZYORA"))
	s.acceptBrowserColumns(n, []oracle.Column{{Name: "STALE"}}, nil)
	fresh := s.browser.GetCurrentNode()
	if fresh == n || len(fresh.GetChildren()) != 0 {
		t.Fatal("late columns populated refreshed tree")
	}
	oldRoot := s.browser.GetRoot()
	s.loadBrowserUser("OTHER")
	<-s.jobs
	s.cancel()
	s.cancel = nil
	s.busy = false
	s.populateObjects(oldRoot, demoObjects("LAZYORA"))
	if s.browserOwner != "OTHER" || s.browser.GetCurrentNode() != nil {
		t.Fatal("late object response replaced current owner")
	}
}

func TestTuivoryBrowserMultipleEmptyBranchesAndFilterCache(t *testing.T) {
	s := demoBrowserScreen(t)
	s.browserObjects = []oracle.Object{{Owner: "LAZYORA", Name: "A", Kind: "TABLE"}, {Owner: "LAZYORA", Name: "B", Kind: "TABLE"}}
	s.renderBrowserObjects()
	for _, n := range s.browser.GetRoot().GetChildren() {
		n.SetExpanded(true)
		s.acceptBrowserColumns(n, nil, nil)
	}
	rows := s.browser.model.Visible(&s.browser.state)
	if len(rows) != 4 {
		t.Fatal("empty branches not independently visible")
	}
	first := s.browser.GetRoot().GetChildren()[0]
	s.browserSearch.SetText("B")
	s.browserSearch.SetText("")
	if s.browser.GetRoot().GetChildren()[0] != first || !first.IsExpanded() {
		t.Fatal("filter discarded cached expansion")
	}
}

func TestTuivoryBrowserThemesWideTextAndNavigation(t *testing.T) {
	for _, colors := range []int{8, 16, 256, 1 << 24} {
		for _, theme := range []string{"dark", "light", "blackwhite"} {
			t.Run(fmt.Sprintf("%s/%d", theme, colors), func(t *testing.T) {
				s := demoBrowserScreen(t)
				terminal := tcell.NewSimulationScreen("UTF-8")
				if err := terminal.Init(); err != nil {
					t.Fatal(err)
				}
				defer terminal.Fini()
				terminal.SetSize(64, 12)
				s.terminal = auditTerminal{SimulationScreen: terminal, count: colors}
				s.prefs.Theme = theme
				s.browserObjects = []oracle.Object{{Owner: "LAZYORA", Name: "긴한글[red]테이블이름_ABCDEFG", Kind: "TABLE"}}
				s.renderBrowserObjects()
				s.browser.SetRect(0, 0, 34, 10)
				s.app.SetFocus(s.browser)
				browserPress(s, tcell.KeyRight, 0)
				browserPress(s, tcell.KeyRight, tcell.ModAlt)
				scroll := s.browser.state.Horizontal
				s.refreshBorders()
				s.browser.Draw(terminal)
				fg, bg, _ := func() tcell.Style { _, _, st, _ := terminal.GetContent(1, 1); return st }().Decompose()
				if fg == bg {
					t.Fatal("selection text invisible")
				}
				ch, _, _, _ := terminal.GetContent(0, 0)
				if ch != '╔' {
					t.Fatalf("focused border is not double: %q", ch)
				}
				selected := s.browser.state.Selected
				s.app.SetFocus(s.editor)
				s.browser.Draw(terminal)
				if s.browser.state.Selected != selected || s.browser.state.Horizontal != scroll {
					t.Fatal("focus/style redraw lost state")
				}
				if !strings.Contains(s.browser.GetCurrentNode().GetReference().(*browserNode).object.Name, "[red]") {
					t.Fatal("literal Oracle name changed")
				}
			})
		}
	}
}

func TestTuivoryBrowserLargeListKeepsSelectionAcrossFilter(t *testing.T) {
	s := demoBrowserScreen(t)
	s.browserObjects = nil
	for i := 0; i < 5000; i++ {
		s.browserObjects = append(s.browserObjects, oracle.Object{Owner: "LAZYORA", Name: fmt.Sprintf("TABLE_%04d", i), Kind: "TABLE"})
	}
	s.renderBrowserObjects()
	s.browser.SetRect(0, 0, 32, 20)
	browserPress(s, tcell.KeyEnd, 0)
	last := s.browser.GetCurrentNode()
	if last.GetReference().(*browserNode).object.Name != "TABLE_4999" || s.browser.state.Scroll == 0 {
		t.Fatal("End did not scroll to last object")
	}
	browserPress(s, tcell.KeyRight, 0)
	s.browserSearch.SetText("4999")
	if len(s.browser.GetRoot().GetChildren()) != 1 || s.browser.GetCurrentNode() != last || !last.IsExpanded() {
		t.Fatal("filter discarded selected expanded table")
	}
	s.browserSearch.SetText("")
	if s.browser.GetCurrentNode() != last || len(s.jobs) != 0 {
		t.Fatal("clearing filter lost identity or submitted query")
	}
}

func TestTuivoryBrowserColumnActionsTargetItsParent(t *testing.T) {
	s := demoBrowserScreen(t)
	s.app.SetFocus(s.browser)
	browserPress(s, tcell.KeyRight, 0)
	browserPress(s, tcell.KeyRight, 0)
	if target := s.editTableTarget(); target.Name != "SAMPLE_ROWS" || target.Kind != "TABLE" {
		t.Fatal("column selection lost parent table for edit action")
	}
	s.describeObject()
	if s.object.Name != "SAMPLE_ROWS" {
		t.Fatal("describe did not target column parent")
	}
}
