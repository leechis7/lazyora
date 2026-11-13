package ui

import (
	"strings"
	"testing"

	"lazyora/internal/oracle"

	"github.com/leechis7/tuivory/go"
)

func TestDebuggerRoutesPausedControlsToItsOwnJob(t *testing.T) {
	s := shortcutScreen(t)
	s.invoke("debug_start")
	v := s.debugger
	v.form.GetFormItem(0).(*tuivory.InputField).SetText("app.proc")
	v.form.GetFormItem(1).(*tuivory.InputField).SetText("4")
	v.form.GetButton(0).InputHandler()(tuivory.NewEventKey(tuivory.KeyEnter, 0, 0), func(tuivory.Primitive) {})
	j := <-s.jobs
	if j.action != "debug_start" || j.text != "app.proc" || j.debugLine != 4 {
		t.Fatalf("unexpected debug target: %+v", j)
	}
	s.key(tuivory.NewEventKey(tuivory.KeyF8, 0, 0))
	if len(j.debugActions) != 0 {
		t.Fatal("step accepted before a pause")
	}
	s.acceptDebugUpdate(oracle.DebugState{Unit: "PROC", Line: 2, Source: "begin\n null;\nend;\n", Stack: "APP.PROC:2"}, nil)
	if !strings.Contains(v.source.GetText(false), ">    2") {
		t.Fatal("current source line is missing")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyF8, 0, 0))
	if len(j.debugActions) != 1 || (<-j.debugActions).Kind != "over" {
		t.Fatal("F8 did not send step over")
	}
	s.acceptDebugUpdate(oracle.DebugState{Line: 3}, nil)
	s.key(tuivory.NewEventKey(tuivory.KeyEscape, 0, 0))
	if len(j.debugActions) != 1 || (<-j.debugActions).Kind != "stop" {
		t.Fatal("Esc did not stop the paused procedure")
	}
}
