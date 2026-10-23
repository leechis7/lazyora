package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func demoBrowserScreen(t *testing.T) *screen {
	t.Helper()
	s := shortcutScreen(t)
	s.demo, s.connected = true, false
	s.newBrowser()
	s.queryLayout = tview.NewFlex().SetDirection(tview.FlexRow).AddItem(s.editor, 0, 2, true).AddItem(s.table, 0, 3, false)
	s.mainLayout = tview.NewFlex().AddItem(s.browserPanel, 0, 1, false).AddItem(s.queryLayout, 0, 3, true)
	s.pages.AddPage("main", s.mainLayout, true, true)
	s.showDemoBrowser()
	s.footer = tview.NewTextView()
	s.toolbar()
	return s
}

func TestBrowserFocusCycleAndContext(t *testing.T) {
	s := demoBrowserScreen(t)
	for _, panel := range []tview.Primitive{s.table, s.browserUsers, s.browserTypes, s.browserSearch, s.browser, s.editor} {
		s.key(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone))
		if s.app.GetFocus() != panel {
			t.Fatal("Tab did not cycle SQL/results/schemas")
		}
	}
	s.key(tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModShift))
	if s.app.GetFocus() != s.browser {
		t.Fatal("Shift+Tab did not cycle in reverse")
	}
	s.refreshFooter()
	if !strings.Contains(s.footer.GetText(false), "Ctrl+R") {
		t.Fatal("browser context shortcuts missing")
	}
	s.key(tcell.NewEventKey(tcell.KeyF6, 0, tcell.ModNone))
	s.key(tcell.NewEventKey(tcell.KeyRune, 'b', tcell.ModAlt))
	if !s.browserHidden || !s.editor.HasFocus() {
		t.Fatal("Browser menu did not hide the browser while preserving SQL focus")
	}
	s.key(tcell.NewEventKey(tcell.KeyRune, 'b', tcell.ModAlt))
	if s.app.GetFocus() != s.browser {
		t.Fatal("Browser menu did not show and focus the browser")
	}
}

func TestBrowserToggleRestoresLayoutAndState(t *testing.T) {
	s := demoBrowserScreen(t)
	s.render(oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"keep"}}})
	terminal := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.Init(); err != nil {
		t.Fatal(err)
	}
	defer terminal.Fini()
	terminal.SetSize(100, 30)
	s.mainLayout.SetRect(0, 0, 100, 28)
	schema := s.browser.GetRoot()
	selected := schema.GetChildren()[0]
	s.browser.SetCurrentNode(selected)
	s.app.SetFocus(s.browser)
	s.mainLayout.Draw(terminal)
	_, _, originalWidth, _ := s.editor.GetRect()
	s.key(tcell.NewEventKey(tcell.KeyRune, 'b', tcell.ModAlt))
	s.mainLayout.Draw(terminal)
	x, _, width, _ := s.editor.GetRect()
	if x != 0 || width != 100 || !s.editor.HasFocus() {
		t.Fatalf("hidden browser did not release width/focus: x=%d width=%d", x, width)
	}
	for _, key := range []tcell.Key{tcell.KeyTab, tcell.KeyTab, tcell.KeyBacktab, tcell.KeyBacktab} {
		s.key(tcell.NewEventKey(key, 0, tcell.ModNone))
		if s.browser.HasFocus() {
			t.Fatal("hidden browser remained in focus cycle")
		}
	}
	s.key(tcell.NewEventKey(tcell.KeyRune, 'b', tcell.ModAlt))
	s.mainLayout.Draw(terminal)
	_, _, restoredWidth, _ := s.editor.GetRect()
	if restoredWidth != originalWidth || !s.browser.HasFocus() || s.browser.GetCurrentNode() != selected || !schema.IsExpanded() {
		t.Fatal("showing browser lost layout, focus or tree state")
	}
	if s.editor.GetText() != "SELECT 1 FROM dual;" || s.resultRows[0][0] != "keep" || len(s.jobs) != 0 {
		t.Fatal("toggle changed SQL/results or submitted a database job")
	}
	s.app.SetFocus(s.table)
	s.key(tcell.NewEventKey(tcell.KeyRune, 'b', tcell.ModAlt))
	if !s.table.HasFocus() {
		t.Fatal("hiding browser changed result focus")
	}
}

func TestBrowserDetailsPreserveSQLAndResults(t *testing.T) {
	s := demoBrowserScreen(t)
	s.render(oracle.Result{Columns: []string{"VALUE"}, Rows: [][]string{{"keep"}}})
	schema := s.browser.GetRoot()
	if schema.GetReference().(*browserNode).object.Owner != "LAZYORA" {
		t.Fatal("current schema not selected")
	}
	s.app.SetFocus(s.browser)
	object := schema.GetChildren()[0]
	s.activateBrowser(object)
	if page, _ := s.pages.GetFrontPage(); page != "object" {
		t.Fatal("object columns not opened")
	}
	if s.objectTable.GetCell(1, 1).Text != "ID" || s.objectTable.GetCell(1, 3).Text != "No" || s.objectTable.GetCell(1, 4).Text != "식별자 / Identifier" {
		t.Fatal("column details incomplete")
	}
	s.key(tcell.NewEventKey(tcell.KeyF4, 0, tcell.ModNone))
	s.key(tcell.NewEventKey(tcell.KeyF5, 0, tcell.ModNone))
	if len(s.jobs) != 0 {
		t.Fatal("detail allowed SQL/commit behind overlay")
	}
	s.key(tcell.NewEventKey(tcell.KeyF1, 0, tcell.ModNone))
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	if page, _ := s.pages.GetFrontPage(); page != "object" || !s.objectTable.HasFocus() {
		t.Fatal("help failed to restore object detail")
	}
	s.key(tcell.NewEventKey(tcell.KeyCtrlR, 0, tcell.ModCtrl))
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	if page, _ := s.pages.GetFrontPage(); page != "main" || !s.browser.HasFocus() {
		t.Fatal("Esc did not restore browser")
	}
	if s.editor.GetText() != "SELECT 1 FROM dual;" || s.resultRows[0][0] != "keep" {
		t.Fatal("browsing changed SQL/results")
	}
	if len(s.jobs) != 0 {
		t.Fatal("demo browser submitted a database job")
	}
}

func TestBrowserLazyLoadEmptySchemasAndLocalization(t *testing.T) {
	s := demoBrowserScreen(t)
	s.demo, s.connected = false, true
	s.browserOwner = ""
	s.populateSchemas(oracle.Catalog{Current: "Case'[red]", Schemas: []string{"OTHER", "Case'[red]"}})
	j := <-s.jobs
	if j.action != "objects" || j.object.Owner != "Case'[red]" {
		t.Fatal("initial load did not preserve actual schema name")
	}
	s.cancel()
	s.cancel, s.busy = nil, false
	s.populateObjects(j.node, nil)
	if got := j.node.GetChildren()[0].GetText(); got != "(empty)" {
		t.Fatalf("empty schema not represented: %s", got)
	}
	s.prefs.Language = "ko"
	s.refreshLabels()
	_, typeLabel := s.browserTypes.GetCurrentOption()
	if j.node.GetText() != tview.Escape("Case'[red]") || typeLabel != "테이블" {
		t.Fatal("translation changed schema identity or missed labels")
	}
	if j.node.GetChildren()[0].GetText() != "(없음)" {
		t.Fatal("empty state was not translated")
	}
	s.app.SetFocus(s.browser)
	s.key(tcell.NewEventKey(tcell.KeyCtrlR, 0, tcell.ModCtrl))
	if j := <-s.jobs; j.action != "schemas" {
		t.Fatal("refresh did not request catalog")
	}
}

func TestExpandedObjectKindsAndDetails(t *testing.T) {
	s := demoBrowserScreen(t)
	want := []string{"TABLE", "VIEW", "MATERIALIZED VIEW", "FUNCTION", "PROCEDURE", "PACKAGE", "PACKAGE BODY", "SEQUENCE", "SYNONYM", "INDEX", "TRIGGER", "TYPE", "TYPE BODY"}
	for i, kind := range want {
		s.browserTypes.SetCurrentOption(i)
		node := s.browser.GetCurrentNode()
		object := node.GetReference().(*browserNode).object
		if object.Kind != kind {
			t.Fatalf("type %d: got %s want %s", i, object.Kind, kind)
		}
		s.activateBrowser(node)
		if object.HasColumns() {
			if s.objectTable.GetCell(1, 1).Text != "ID" {
				t.Fatal("column details lost")
			}
		} else if s.objectTable.GetCell(3, 1).Text != kind {
			t.Fatal("object metadata lost")
		}
		s.closeObject()
	}
	s.prefs.Language = "ko"
	s.refreshLabels()
	s.browserTypes.SetCurrentOption(3)
	_, functionLabel := s.browserTypes.GetCurrentOption()
	s.browserTypes.SetCurrentOption(6)
	_, packageLabel := s.browserTypes.GetCurrentOption()
	if functionLabel != "함수" || packageLabel != "패키지 본문" {
		t.Fatal("type translations missing")
	}
	s.browserTypes.SetCurrentOption(6)
	s.demo, s.connected = false, true
	s.activateBrowser(s.browser.GetCurrentNode())
	job := <-s.jobs
	if job.action != "object_info" || job.object.Kind != "PACKAGE BODY" {
		t.Fatal("wrong metadata job")
	}
}

func TestColumnCommentRendering(t *testing.T) {
	s := demoBrowserScreen(t)
	s.object = oracle.Object{Owner: "APP", Name: "V", Kind: "VIEW"}
	s.openObject(s.object)
	s.renderObject([]oracle.Column{{Name: "C", DataType: "VARCHAR2(30 BYTE)", Comment: "한글 [red] 주석\n둘째 줄"}, {Name: "D", DataType: "NUMBER"}})
	if s.objectTable.GetCell(0, 4).Text != "Comment" || s.objectTable.GetCell(1, 4).Text != tview.Escape("한글 [red] 주석\n둘째 줄") || s.objectTable.GetCell(2, 4).Text != "" {
		t.Fatal("comment header/text or empty comment lost")
	}
	s.prefs.Language = "ko"
	s.renderObject(s.objectColumns)
	if s.objectTable.GetCell(0, 4).Text != "코멘트" {
		t.Fatal("missing Korean comment header")
	}
}
