package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func TestRoutineInputsConfirmDefaultCancelAndValidateBeforeWorker(t *testing.T) {
	s := demoBrowserScreen(t)
	s.connected = true
	r := oracle.Routine{Object: oracle.Object{Owner: "APP", Name: "P", Kind: "PROCEDURE"}, Member: "P", Overload: "-", Arguments: []oracle.RoutineArgument{{Argument: oracle.Argument{Name: "N", DataType: "NUMBER", Direction: "IN/OUT", Position: 1}}}}
	s.showRoutine(r.Object, "")
	load := <-s.jobs
	s.cancel()
	s.cancel = nil
	s.busy = false
	if load.action != "routines" || load.object != r.Object {
		t.Fatal("wrong metadata request")
	}
	s.acceptRoutines([]oracle.Routine{r})
	s.showRoutineInputs(r)
	s.routineForm.GetFormItem(0).(*choiceField).SetCurrentOption(0)
	s.routineForm.GetFormItem(1).(*tview.InputField).SetText("bad NUMBER")
	s.key(tcell.NewEventKey(tcell.KeyF9, 0, 0))
	if len(s.jobs) != 0 || s.programModal != nil {
		t.Fatal("invalid input reached worker")
	}
	s.routineForm.GetFormItem(1).(*tview.InputField).SetText("123")
	s.key(tcell.NewEventKey(tcell.KeyF9, 0, 0))
	if s.programModal == nil || len(s.jobs) != 0 {
		t.Fatal("missing routine confirmation")
	}
	modalKey(s, tcell.KeyEnter)
	if len(s.jobs) != 0 || s.programModal != nil {
		t.Fatal("default confirmation ran routine")
	}
	s.routineSubmit()
	modalKey(s, tcell.KeyLeft)
	modalKey(s, tcell.KeyEnter)
	j := <-s.jobs
	s.cancel()
	if j.action != "routine_run" || j.routineInputs[1].Value.Text != "123" || j.routine.Object != r.Object {
		t.Fatalf("wrong job: %+v", j)
	}
}
func TestRoutineResultGridPreservesSourceAndSupportsRawCopy(t *testing.T) {
	s := programScreen(t)
	source := s.sourceEditor.GetText() + "\n-- keep edits"
	s.sourceEditor.SetText(source, false)
	result := oracle.RoutineResult{Cursors: []oracle.NamedResult{{Name: "C", Result: oracle.Result{Columns: []string{"TEXT"}, Rows: [][]string{{"한글 [red]"}}, Values: [][]oracle.Cell{{{Text: "한글 [red]", Type: "VARCHAR2"}}}}}}}
	s.showRoutineResults(result)
	s.routineResultList.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	if page, _ := s.pages.GetFrontPage(); page != "routine_grid" {
		t.Fatal(page)
	}
	s.key(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0))
	if s.clipboardText != "한글 [red]" || s.resultQuery != nil {
		t.Fatalf("unsafe copy/replay: %q", s.clipboardText)
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "routine_results" {
		t.Fatal(page)
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.sourceEditor.GetText() != source || !s.sourceDirty() || !s.sourceEditor.HasFocus() {
		t.Fatal("result selection discarded source")
	}
}
