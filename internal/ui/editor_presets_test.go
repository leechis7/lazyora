package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
	"testing"
)

func TestIDEKeymapsDuplicateMoveDeleteUndoAndIsolation(t *testing.T) {
	for _, keymap := range []string{"vscode", "intellij"} {
		t.Run(keymap, func(t *testing.T) {
			original := "첫째\n둘째\n셋째"
			s := mappedScreen(t, keymap, original)
			s.editor.Select(len(original), len(original))
			editorEvent(s, tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModCtrl))
			_, at, _ := s.editor.GetSelection()
			if at != 0 {
				t.Fatal("Ctrl+Home did not reach document start")
			}
			editorEvent(s, tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModCtrl))
			_, at, _ = s.editor.GetSelection()
			if at != len(original) {
				t.Fatal("Ctrl+End did not reach document end")
			}
			s.editor.Select(len("첫째\n"), len("첫째\n"))
			duplicate := tcell.NewEventKey(tcell.KeyCtrlD, 0, 0)
			move := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModCtrl|tcell.ModAlt|tcell.ModShift)
			del := tcell.NewEventKey(tcell.KeyCtrlY, 0, 0)
			if keymap == "vscode" {
				duplicate = tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModAlt|tcell.ModShift)
				move = tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModAlt)
				del = tcell.NewEventKey(tcell.KeyCtrlK, 0, tcell.ModShift)
			}
			editorEvent(s, duplicate)
			if s.editor.GetText() != "첫째\n둘째\n둘째\n셋째" {
				t.Fatalf("duplicate: %q", s.editor.GetText())
			}
			editorEvent(s, tcell.NewEventKey(tcell.KeyCtrlZ, 0, 0))
			if s.editor.GetText() != original {
				t.Fatal("undo")
			}
			editorEvent(s, tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModCtrl|tcell.ModShift))
			if s.editor.GetText() == original {
				t.Fatal("redo")
			}
			editorEvent(s, tcell.NewEventKey(tcell.KeyCtrlZ, 0, 0))
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
			field := s.connectForm.GetFormItem(connectionAlias).(*tview.InputField)
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
		format := tcell.NewEventKey(tcell.KeyRune, 'f', tcell.ModAlt|tcell.ModShift)
		if keymap == "intellij" {
			format = tcell.NewEventKey(tcell.KeyCtrlL, 0, tcell.ModAlt|tcell.ModCtrl)
		}
		editorEvent(s, format)
		if s.menuList != nil || s.statusKey != "format_done" || !strings.Contains(s.editor.GetText(), "\n") {
			t.Fatalf("format opened menu or failed: %s %s %q", keymap, s.statusKey, s.editor.GetText())
		}
		editorEvent(s, tcell.NewEventKey(tcell.KeyCtrlA, 0, tcell.ModCtrl))
		if !s.editor.HasSelection() {
			t.Fatal("select all")
		}
		expectedCopy := s.editor.GetText()
		editorEvent(s, tcell.NewEventKey(tcell.KeyCtrlC, 0, 0))
		if s.clipboardText != expectedCopy {
			t.Fatal("selected text copy failed")
		}
		s = programScreen(t)
		s.prefs.EditorKeymap = keymap
		s.app.SetFocus(s.sourceEditor)
		sql := s.editor.GetText()
		s.sourceEditor.SetText("source line", false).Select(0, 0)
		duplicate := tcell.NewEventKey(tcell.KeyCtrlD, 0, 0)
		if keymap == "vscode" {
			duplicate = tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModAlt|tcell.ModShift)
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
	editorEvent(s, tcell.NewEventKey(tcell.KeyCtrlL, 0, 0))
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
	editorEvent(s, tcell.NewEventKey(tcell.KeyCtrlD, 0, 0))
	if s.editor.GetText() != before || len(s.jobs) != 0 || s.statusKey != "keymap_no_multicaret" {
		t.Fatal("unsupported multi-caret executed another command")
	}
}
