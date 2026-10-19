package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"lazyora/internal/oracle"
)

func TestExplainKeyUsesCursorSQLAndPreservesResult(t *testing.T) {
	s := fileTestScreen(t)
	s.editor.SetText("SELECT 1 FROM dual;\nDELETE FROM sample_rows;", false).Select(0, 0)
	s.key(tcell.NewEventKey(tcell.KeyCtrlE, 0, tcell.ModCtrl))
	j := <-s.jobs
	if j.action != "plan" || j.text != "SELECT 1 FROM dual;" {
		t.Fatal("explain did not use cursor SQL")
	}
	s.busy = false
	s.showPlan(j.text, []oracle.PlanStep{{ID: 0, Operation: "SELECT STATEMENT", Cost: "1", Rows: "1"}, {ID: 1, Parent: 0, HasParent: true, Operation: "FAST DUAL", Filter: "DUMMY='X'"}})
	root := s.planTree.GetRoot()
	if len(root.GetChildren()) != 1 || len(root.GetChildren()[0].GetChildren()) != 1 {
		t.Fatal("plan hierarchy incorrect")
	}
	s.planTree.SetCurrentNode(root.GetChildren()[0].GetChildren()[0])
	s.key(tcell.NewEventKey(tcell.KeyF1, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.planTree != nil || !s.pendingChanges || s.table.GetCell(0, 0).Text != "keep result" || len(s.jobs) != 0 {
		t.Fatal("plan display modified results or DB")
	}
}
