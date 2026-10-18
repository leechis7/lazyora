package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func warningKey(s *screen, event *tcell.EventKey) {
	if next := s.key(event); next != nil {
		s.app.GetFocus().InputHandler()(next, func(p tview.Primitive) { s.app.SetFocus(p) })
	}
}
func TestRiskySQLDefaultsToCancelAndPreservesState(t *testing.T) {
	for _, key := range []*tcell.EventKey{tcell.NewEventKey(tcell.KeyF9, 0, 0), tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModCtrl), tcell.NewEventKey(tcell.KeyEscape, 0, 0)} {
		s := shortcutScreen(t)
		s.editor.SetText("DELETE FROM demo", false)
		s.render(oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"keep"}}})
		s.pendingChanges = true
		s.executeSQL()
		if page, _ := s.pages.GetFrontPage(); page != "sql_confirm" || s.sqlConfirmText == nil {
			t.Fatal("dangerous SQL did not prompt")
		}
		if len(s.jobs) != 0 || len(s.sqlHistory) != 0 || s.busy {
			t.Fatal("prompt submitted SQL")
		}
		// Repeating an execution shortcut must not automatically accept.
		warningKey(s, key)
		if page, _ := s.pages.GetFrontPage(); page == "sql_confirm" {
			warningKey(s, tcell.NewEventKey(tcell.KeyEnter, 0, 0))
		}
		if len(s.jobs) != 0 || len(s.sqlHistory) != 0 || s.sqlConfirmText != nil {
			t.Fatal("default confirmation executed SQL")
		}
		if s.editor.GetText() != "DELETE FROM demo" || s.table.GetCell(1, 0).Text != "keep" || !s.pendingChanges || !s.editor.HasFocus() {
			t.Fatal("cancel changed SQL, result, transaction or focus")
		}
	}
}
func TestRiskConfirmationExecutesReviewedSelectionOnly(t *testing.T) {
	s := shortcutScreen(t)
	selected := "DROP TABLE demo"
	s.editor.SetText(selected+"\nSELECT 1 FROM dual;", false).Select(0, len(selected))
	s.executeSQL()
	if s.sqlConfirmText == nil || s.sqlConfirmText.GetText(false) != selected {
		t.Fatal("preview did not use selected SQL")
	}
	// Even if the editor is changed programmatically, execute the reviewed snapshot.
	s.editor.SetText("DELETE FROM different_table", false)
	warningKey(s, tcell.NewEventKey(tcell.KeyLeft, 0, 0))
	warningKey(s, tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	select {
	case j := <-s.jobs:
		if j.action != "execute" || j.text != selected {
			t.Fatal("reviewed snapshot not submitted")
		}
	default:
		t.Fatal("explicit confirm did not execute")
	}
	if len(s.sqlHistory) != 1 || s.sqlHistory[0] != selected || s.sqlConfirmText != nil {
		t.Fatal("confirmed execution history incorrect")
	}
}
func TestFilteredDMLAndReadOnlySQLDoNotPrompt(t *testing.T) {
	for _, sql := range []string{"SELECT 1 FROM dual", "UPDATE demo SET id=2 WHERE id=1", "DELETE FROM demo WHERE id=1"} {
		s := shortcutScreen(t)
		s.editor.SetText(sql, false)
		s.executeSQL()
		if s.sqlConfirmText != nil || len(s.jobs) != 1 {
			t.Fatal("non-target SQL was blocked")
		}
	}
	s := shortcutScreen(t)
	s.connected = false
	s.editor.SetText("TRUNCATE TABLE demo", false)
	s.executeSQL()
	if s.sqlConfirmText != nil || len(s.jobs) != 0 || s.statusKey != "connect_first" {
		t.Fatal("disconnected editor prompted or submitted SQL")
	}
}
func TestWarningHelpTabAndLiteralPreview(t *testing.T) {
	s := shortcutScreen(t)
	sql := "UPDATE demo SET name='[red]WHERE[white]'"
	s.editor.SetText(sql, false)
	s.executeSQL()
	warningKey(s, tcell.NewEventKey(tcell.KeyTab, 0, 0))
	if !s.sqlConfirmText.HasFocus() {
		t.Fatal("Tab did not reach SQL preview")
	}
	if s.sqlConfirmText.GetText(false) != sql {
		t.Fatal("preview modified SQL")
	}
	warningKey(s, tcell.NewEventKey(tcell.KeyF1, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "help" {
		t.Fatal("help missing")
	}
	warningKey(s, tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "sql_confirm" {
		t.Fatal("help did not restore warning")
	}
	warningKey(s, tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if len(s.jobs) != 0 {
		t.Fatal("help/close executed SQL")
	}
	for _, language := range []string{"en", "ko"} {
		s.prefs.Language = language
		if strings.HasPrefix(s.tr("sql_risk_ddl"), "sql_risk_") {
			t.Fatal("translation missing")
		}
	}
}

func TestSQLWarningMouseCancelAndSmallTerminal(t *testing.T) {
	terminal := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.Init(); err != nil {
		t.Fatal(err)
	}
	defer terminal.Fini()
	terminal.SetSize(80, 24)
	for _, language := range []string{"en", "ko"} {
		s := shortcutScreen(t)
		s.prefs.Language = language
		s.editor.SetText("DROP TABLE demo", false)
		s.executeSQL()
		_, page := s.pages.GetFrontPage()
		root := page.(*tview.Flex)
		root.SetRect(0, 0, 80, 21)
		root.Draw(terminal)
		buttons := root.GetItem(root.GetItemCount() - 1).(*tview.Flex)
		cancel := buttons.GetItem(1).(*tview.Button)
		x, y, width, height := cancel.GetRect()
		if x < 0 || x+width > 80 || height != 1 || y >= 21 {
			t.Fatal("confirmation button off screen")
		}
		if !cancel.HasFocus() {
			t.Fatal("cancel not selected by default")
		}
		root.MouseHandler()(tview.MouseLeftClick, tcell.NewEventMouse(x+width/2, y, tcell.ButtonNone, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
		if s.sqlConfirmText != nil || len(s.jobs) != 0 {
			t.Fatal("mouse cancel did not dismiss safely")
		}
	}
}
