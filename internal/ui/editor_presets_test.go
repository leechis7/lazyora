package ui

import (
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestIDEKeymapsDuplicateMoveDeleteUndoAndIsolation(t *testing.T) {
	for _, keymap := range []string{"vscode", "intellij"} {
		t.Run(keymap, func(t *testing.T) {
			original := "첫째\n둘째\n셋째"
			s := mappedScreen(t, keymap, original)
			s.editor.Select(len(original), len(original))
			editorEvent(s, tuivory.NewEventKey(tuivory.KeyHome, 0, tuivory.ModCtrl))
			_, at, _ := s.editor.GetSelection()
			if at != 0 {
				t.Fatal("Ctrl+Home did not reach document start")
			}
			editorEvent(s, tuivory.NewEventKey(tuivory.KeyEnd, 0, tuivory.ModCtrl))
			_, at, _ = s.editor.GetSelection()
			if at != len(original) {
				t.Fatal("Ctrl+End did not reach document end")
			}
			s.editor.Select(len("첫째\n"), len("첫째\n"))
			duplicate := tuivory.NewEventKey(tuivory.KeyCtrlD, 0, 0)
			move := tuivory.NewEventKey(tuivory.KeyDown, 0, tuivory.ModCtrl|tuivory.ModAlt|tuivory.ModShift)
			del := tuivory.NewEventKey(tuivory.KeyCtrlY, 0, 0)
			if keymap == "vscode" {
				duplicate = tuivory.NewEventKey(tuivory.KeyDown, 0, tuivory.ModAlt|tuivory.ModShift)
				move = tuivory.NewEventKey(tuivory.KeyDown, 0, tuivory.ModAlt)
				del = tuivory.NewEventKey(tuivory.KeyCtrlK, 0, tuivory.ModShift)
			}
			editorEvent(s, duplicate)
			if s.editor.GetText() != "첫째\n둘째\n둘째\n셋째" {
				t.Fatalf("duplicate: %q", s.editor.GetText())
			}
			editorEvent(s, tuivory.NewEventKey(tuivory.KeyCtrlZ, 0, 0))
			if s.editor.GetText() != original {
				t.Fatal("undo")
			}
			editorEvent(s, tuivory.NewEventKey(tuivory.KeyRune, 'z', tuivory.ModCtrl|tuivory.ModShift))
			if s.editor.GetText() == original {
				t.Fatal("redo")
			}
			editorEvent(s, tuivory.NewEventKey(tuivory.KeyCtrlZ, 0, 0))
			s.editor.Select(len("첫째\n"), len("첫째\n"))
			editorEvent(s, move)
			if s.editor.GetText() != "첫째\n셋째\n둘째" {
				t.Fatalf("move: %q", s.editor.GetText())
			}
			editorEvent(s, del)
			if s.editor.GetText() != "첫째\n셋째" {
				t.Fatalf("delete: %q", s.editor.GetText())
			}
			before := s.editor.GetText()
			s.busy = true
			editorEvent(s, duplicate)
			s.busy = false
			if s.editor.GetText() != before {
				t.Fatal("busy edit changed text")
			}
			s.app.SetFocus(s.table)
			editorEvent(s, duplicate)
			if s.editor.GetText() != before {
				t.Fatal("map leaked to grid")
			}
			s.connectionForm()
			field := s.connectForm.GetFormItem(connectionAlias).(*tuivory.InputField)
			field.SetText("alias")
			s.app.SetFocus(field)
			if s.editorPresetKey(s.activeEditor(), duplicate) {
				t.Fatal("map captured connection form")
			}
		})
	}
}
func TestIDEKeymapsFormatBeforeMenuMnemonicsAndSourceIsolation(t *testing.T) {
	for _, keymap := range []string{"vscode", "intellij"} {
		s := mappedScreen(t, keymap, "select 1 from dual;")
		format := tuivory.NewEventKey(tuivory.KeyRune, 'f', tuivory.ModAlt|tuivory.ModShift)
		if keymap == "intellij" {
			format = tuivory.NewEventKey(tuivory.KeyCtrlL, 0, tuivory.ModAlt|tuivory.ModCtrl)
		}
		editorEvent(s, format)
		if s.menuList != nil || s.statusKey != "format_done" || !strings.Contains(s.editor.GetText(), "\n") {
			t.Fatalf("format opened menu or failed: %s %s %q", keymap, s.statusKey, s.editor.GetText())
		}
		editorEvent(s, tuivory.NewEventKey(tuivory.KeyCtrlA, 0, tuivory.ModCtrl))
		if !s.editor.HasSelection() {
			t.Fatal("select all")
		}
		expectedCopy := s.editor.GetText()
		editorEvent(s, tuivory.NewEventKey(tuivory.KeyCtrlC, 0, 0))
		if s.clipboardText != expectedCopy {
			t.Fatal("selected text copy failed")
		}
		s = programScreen(t)
		s.prefs.EditorKeymap = keymap
		s.app.SetFocus(s.sourceEditor)
		sql := s.editor.GetText()
		s.sourceEditor.SetText("source line", false).Select(0, 0)
		duplicate := tuivory.NewEventKey(tuivory.KeyCtrlD, 0, 0)
		if keymap == "vscode" {
			duplicate = tuivory.NewEventKey(tuivory.KeyDown, 0, tuivory.ModAlt|tuivory.ModShift)
		}
		editorEvent(s, duplicate)
		if s.editor.GetText() != sql || s.sourceEditor.GetText() != "source line\nsource line" {
			t.Fatal("source map targeted SQL or failed")
		}
	}
}
func TestToadRemainsDefaultAndIDELabelsUseRuntimeBindings(t *testing.T) {
	s := mappedScreen(t, "", "old text")
	if !strings.Contains(s.tr("keymap_default"), "Toad") || !strings.Contains(s.keymapHint(), "Toad") {
		t.Fatal("Toad default label missing")
	}
	editorEvent(s, tuivory.NewEventKey(tuivory.KeyCtrlL, 0, 0))
	if !s.editor.HasSelection() {
		t.Fatal("Toad default changed")
	}
	s.prefs.EditorKeymap = "intellij"
	if s.editorBindingLabels("delete_line", 1) != "Ctrl+Y" || s.editorBindingLabels("redo", 1) != "Ctrl+Shift+Z" {
		t.Fatal("IDEA labels differ from dispatch")
	}
	s.prefs.EditorKeymap = "vscode"
	s.editor.Select(0, 0)
	before := s.editor.GetText()
	editorEvent(s, tuivory.NewEventKey(tuivory.KeyCtrlD, 0, 0))
	if s.editor.GetText() != before || len(s.jobs) != 0 || s.statusKey != "keymap_no_multicaret" {
		t.Fatal("unsupported multi-caret executed another command")
	}
}
