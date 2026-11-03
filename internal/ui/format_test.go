package ui

import (
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestFormatWholeCodePreservesCursorAndOneStepUndo(t *testing.T) {
	for _, lang := range []string{"en", "ko"} {
		s := demoBrowserScreen(t)
		s.prefs.Language = lang
		original := "SELECT a,b FROM t WHERE id=:id;"
		s.editor.SetText(original, false)
		cursor := len("SELECT a")
		s.editor.Select(cursor, cursor)
		s.app.SetFocus(s.editor)
		s.key(tuivory.NewEventKey(tuivory.KeyCtrlF, 0, tuivory.ModCtrl|tuivory.ModShift))
		formatted := s.editor.GetText()
		_, offset, _ := s.editor.GetSelection()
		if !strings.Contains(formatted, "\n  a,\n  b") || formatted[offset-1] != 'a' || !s.editor.HasFocus() || len(s.jobs) != 0 {
			t.Fatalf("%q offset %d", formatted, offset)
		}
		s.editor.InputHandler()(tuivory.NewEventKey(tuivory.KeyCtrlZ, 0, 0), func(p tuivory.Primitive) { s.app.SetFocus(p) })
		if s.editor.GetText() != original {
			t.Fatal("undo did not restore whole edit")
		}
	}
}
func TestFormatSelectionKeepsOtherStatementsAndRefusesLiteralCuts(t *testing.T) {
	s := demoBrowserScreen(t)
	s.app.SetFocus(s.editor)
	original := "SELECT 1 FROM dual;\nSELECT a,b FROM t;\nSELECT 3 FROM dual;"
	s.editor.SetText(original, false)
	start := strings.Index(original, "SELECT a")
	end := strings.Index(original, ";\nSELECT 3") + 1
	s.editor.Select(start, end)
	s.formatCode()
	if !strings.HasPrefix(s.editor.GetText(), original[:start]) || !strings.HasSuffix(s.editor.GetText(), original[end:]) {
		t.Fatal("changed unselected text")
	}
	selected, from, to := s.editor.GetSelection()
	if from != start || to != start+len(selected) || !strings.Contains(selected, "\n") {
		t.Fatal("lost selected replacement")
	}
	s.editor.SetText("SELECT q'[a,b]' FROM dual", false)
	start = strings.Index(s.editor.GetText(), "a,b")
	s.editor.Select(start, start+3)
	s.formatCode()
	if s.editor.GetText() != "SELECT q'[a,b]' FROM dual" || s.statusKey != "format_failed" {
		t.Fatal("edited literal contents")
	}
}
func TestSourceFormatDoesNotCompileOrChangeOtherEditor(t *testing.T) {
	s := programScreen(t)
	original := s.sourceEditor.GetText()
	sql := s.editor.GetText()
	s.key(tuivory.NewEventKey(tuivory.KeyCtrlF, 0, tuivory.ModShift))
	if s.sourceEditor.GetText() == original || !s.sourceDirty() || s.programModal != nil || len(s.jobs) != 0 || s.editor.GetText() != sql {
		t.Fatal("format compiled/lost editor isolation")
	}
	s.sourceEditor.InputHandler()(tuivory.NewEventKey(tuivory.KeyCtrlZ, 0, 0), func(p tuivory.Primitive) { s.app.SetFocus(p) })
	if s.sourceEditor.GetText() != original || s.sourceDirty() {
		t.Fatal("source undo did not restore base")
	}
	s.app.SetFocus(s.sourceDiagnostics)
	s.key(tuivory.NewEventKey(tuivory.KeyCtrlF, 0, tuivory.ModShift))
	if s.sourceEditor.GetText() != original {
		t.Fatal("formatted without editor focus")
	}
}
