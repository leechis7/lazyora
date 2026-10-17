package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"testing"
)

func programScreen(t *testing.T) *screen {
	s := demoBrowserScreen(t)
	node := s.browser.GetCurrentNode().GetChildren()[3].GetChildren()[0]
	s.activateBrowser(node)
	s.key(tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModAlt))
	return s
}

func modalKey(s *screen, key tcell.Key) {
	s.programModal.InputHandler()(tcell.NewEventKey(key, 0, tcell.ModNone), func(p tview.Primitive) { s.app.SetFocus(p) })
}

func TestSourceDiscardDefaultsToCancelAndPreservesOtherEditors(t *testing.T) {
	s := programScreen(t)
	original := s.sourceEditor.GetText()
	s.sourceEditor.SetText(original+"\n-- edit", false)
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	if s.programModal == nil {
		t.Fatal("dirty close did not ask")
	}
	modalKey(s, tcell.KeyEnter)
	if !s.sourceDirty() || s.sourceEditor.GetText() != original+"\n-- edit" {
		t.Fatal("default confirmation lost edits")
	}
	s.requestQuit()
	if s.programModal == nil || s.exitDialog != nil {
		t.Fatal("quit skipped unapplied source guard")
	}
	modalKey(s, tcell.KeyEscape)
	s.closeSource()
	modalKey(s, tcell.KeyLeft)
	modalKey(s, tcell.KeyEnter)
	if s.sourceEditor != nil {
		t.Fatal("confirmed close retained editor")
	}
	if s.editor.GetText() != "SELECT 1 FROM dual;" {
		t.Fatal("source editing changed SQL editor")
	}
}

func TestCompileRequiresConfirmationAndPinsSelectedObject(t *testing.T) {
	s := programScreen(t)
	s.demo, s.connected = false, true
	source := `CREATE OR REPLACE FUNCTION "LAZYORA"."SAMPLE_FUNCTION" RETURN NUMBER IS BEGIN RETURN 1; END;`
	s.acceptSource(source)
	s.sourceEditor.SetText(source+"\n", false)
	s.key(tcell.NewEventKey(tcell.KeyF9, 0, tcell.ModCtrl))
	if s.programModal == nil || len(s.jobs) != 0 {
		t.Fatal("compile bypassed confirmation")
	}
	modalKey(s, tcell.KeyEnter)
	if len(s.jobs) != 0 {
		t.Fatal("default confirm compiled source")
	}
	s.requestCompile()
	modalKey(s, tcell.KeyLeft)
	modalKey(s, tcell.KeyEnter)
	j := <-s.jobs
	if j.action != "compile_source" || j.object != s.object || j.text != source+"\n" {
		t.Fatalf("wrong target/source: %+v", j)
	}
}

func TestArgumentsRestoreSourceAndShowReturnValue(t *testing.T) {
	s := programScreen(t)
	s.sourceEditor.SetText(s.sourceEditor.GetText()+"\n-- keep", false)
	s.key(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModAlt))
	s.renderArguments([]oracle.Argument{{Member: "F", Name: "-", Position: 0, DataType: "NUMBER", Direction: "OUT"}, {Member: "F", Name: "P", Position: 1, DataType: "NUMBER", Direction: "IN"}})
	if s.argumentTable.GetCell(1, 3).Text != s.tr("return_value") || s.argumentTable.GetCell(2, 3).Text != "P" {
		t.Fatal("return/argument labels incorrect")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	if !s.sourceEditor.HasFocus() || !s.sourceDirty() {
		t.Fatal("arguments lost source edits/focus")
	}
	s.renderDiagnostics([]oracle.Diagnostic{{Line: 3, Position: 7, Attribute: "ERROR", Text: "unknown identifier"}})
	if s.sourceDiagnostics.GetCell(1, 0).Text != "3" || s.sourceDiagnostics.GetCell(1, 1).Text != "7" {
		t.Fatal("missing compiler location")
	}
}
