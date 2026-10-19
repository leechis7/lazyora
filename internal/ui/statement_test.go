package ui

import (
	"strings"
	"testing"
)

func TestCursorExecutionOnlySubmitsTarget(t *testing.T) {
	s := shortcutScreen(t)
	text := "DELETE FROM secret;\nSELECT 42 FROM dual;\nDROP TABLE secret;"
	s.editor.SetText(text, false).Select(strings.Index(text, "42"), strings.Index(text, "42"))
	s.executeSQL()
	j := <-s.jobs
	if j.text != "SELECT 42 FROM dual;" || s.sqlConfirmText != nil || s.editor.GetText() != text {
		t.Fatalf("wrong target: %+v", j)
	}
	s.busy = false
	s.editor.Select(0, 0)
	s.executeSQL()
	if s.sqlConfirmText == nil || s.sqlConfirmText.GetText(false) != "DELETE FROM secret;" || len(s.jobs) != 0 {
		t.Fatal("dangerous cursor SQL not reviewed independently")
	}
}

func TestInvalidCursorSQLDoesNotSubmit(t *testing.T) {
	s := shortcutScreen(t)
	s.editor.SetText("SELECT 'unfinished", false)
	s.executeSQL()
	if len(s.jobs) != 0 || s.busy {
		t.Fatal("invalid boundary submitted")
	}
}
