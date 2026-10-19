package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func TestF5ReviewsSnapshotAndSubmitsScript(t *testing.T) {
	s := shortcutScreen(t)
	text := "SELECT 1 FROM dual;\nDELETE FROM sample_rows;"
	s.editor.SetText(text, false)
	s.key(tcell.NewEventKey(tcell.KeyF5, 0, 0))
	if s.sqlConfirmText == nil || len(s.jobs) != 0 {
		t.Fatal("F5 did not review script")
	}
	s.editor.SetText("DROP TABLE other;", false)
	s.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	button := s.app.GetFocus().(*tview.Button)
	button.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	j := <-s.jobs
	if j.action != "script" || j.text != text {
		t.Fatal("script review did not preserve snapshot")
	}
}

func TestScriptReportResultSelectionPreservesSQLAndDB(t *testing.T) {
	s := fileTestScreen(t)
	s.scriptReport = oracle.ScriptReport{Total: 2, Steps: []oracle.ScriptStep{
		{Statement: oracle.Statement{Text: "SELECT 1 FROM dual;", Line: 1}, Result: oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"1"}}}},
		{Statement: oracle.Statement{Text: "SELECT 2 FROM dual;", Line: 2}, Result: oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"2"}}}},
	}}
	s.showScriptReport()
	s.scriptList.SetCurrentItem(1)
	if s.scriptResult.GetCell(1, 0).Text != "2" {
		t.Fatal("report did not switch SELECT results")
	}
	s.key(tcell.NewEventKey(tcell.KeyF1, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "script" {
		t.Fatal("help lost report")
	}
	_, root := s.pages.GetFrontPage()
	flex := root.(*tview.Flex)
	button := flex.GetItem(flex.GetItemCount() - 1).(*tview.Flex).GetItem(0).(*tview.Button)
	button.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	if s.resultRows[0][0] != "2" || !s.pendingChanges || len(s.jobs) != 0 || s.editor.GetText() != "SELECT 1 FROM dual;" {
		t.Fatal("report adoption changed SQL/DB or lost result")
	}
}
