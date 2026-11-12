package ui

import (
	"errors"
	"lazyora/internal/oracle"
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestResultAreaKeepsExecutionFocusAndSeparatesTabAndConnectionState(t *testing.T) {
	s := demoBrowserScreen(t)
	s.connected = true
	s.initDocument()
	s.initTabs()
	s.app.SetFocus(s.editor)
	s.recordResultExecution("execute", oracle.Result{}, oracle.ScriptReport{}, errors.New("ORA-00942 test"))
	if !s.editor.HasFocus() || !strings.HasSuffix(s.resultArea.titles[1], " !") || s.resultSection() != "grid" {
		t.Fatal("failed execution stole focus or did not mark the messages tab")
	}
	s.addOutput(oracle.Output{Lines: []string{"workspace-only output"}})
	s.showScriptReport()
	if text := s.resultAreaContent().(*tuivory.WidgetTextView).GetText(false); !strings.Contains(text, "ORA-00942 test") {
		t.Fatalf("last SQL failure missing: %s", text)
	}
	s.newTab()
	if s.resultSection() != "grid" || s.resultState.note.key != "" || s.resultState.msgError || !s.editor.HasFocus() {
		t.Fatal("previous SQL tab state leaked into the new tab")
	}
	s.showOutput()
	if s.outputText.GetText(false) != "workspace-only output\n" || strings.HasSuffix(s.resultArea.titles[2], " *") {
		t.Fatal("workspace output was lost or remained marked unread")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyBacktab, 0, 0))
	if !s.resultArea.tab.HasFocus() {
		t.Fatal("keyboard could not enter tab strip")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyBacktab, 0, tuivory.ModCtrl))
	if s.activeTab != 0 || s.resultSection() != "messages" || !s.resultState.msgError || !s.editor.HasFocus() {
		t.Fatal("SQL tab switching from the result tabs did not restore its own state")
	}
	other := demoBrowserScreen(t)
	other.showOutput()
	if other.outputText.GetText(false) != "" || other.outputNew {
		t.Fatal("DBMS_OUTPUT leaked into another connection")
	}
}
