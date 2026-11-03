package ui

import (
	"strings"
	"unicode"

	"github.com/leechis7/tuivory/go"
)

type editorPresetBinding struct {
	action string
	binding
}

func ideKeymap(keymap string) bool           { return keymap == "vscode" || keymap == "intellij" }
func lineClipboardKeymap(keymap string) bool { return keymap == "notepad++" || ideKeymap(keymap) }

// Windows/Linux editing subsets. App/session keys retain their existing role.
// macOS command keys are not assumed to arrive in terminal key events.
func editorPresetBindings(keymap string) []editorPresetBinding {
	if !ideKeymap(keymap) {
		return nil
	}
	keys := []editorPresetBinding{
		{"editor_document_start", binding{"Ctrl+Home", tuivory.KeyHome, tuivory.ModCtrl, 0}},
		{"editor_document_end", binding{"Ctrl+End", tuivory.KeyEnd, tuivory.ModCtrl, 0}},
		{"select_all", binding{"Ctrl+A", tuivory.KeyCtrlA, 0, 0}},
		{"complete", binding{"Ctrl+Space", tuivory.KeyCtrlSpace, 0, 0}},
		{"copy", binding{"Ctrl+C", tuivory.KeyCtrlC, 0, 0}},
		{"cut", binding{"Ctrl+X", tuivory.KeyCtrlX, 0, 0}},
		{"paste", binding{"Ctrl+V", tuivory.KeyCtrlV, 0, 0}},
		{"undo", binding{"Ctrl+Z", tuivory.KeyCtrlZ, 0, 0}},
		{"redo", binding{"Ctrl+Shift+Z", tuivory.KeyCtrlZ, tuivory.ModShift, 0}},
		{"find", binding{"Ctrl+F", tuivory.KeyCtrlF, 0, 0}},
		{"replace", binding{"Ctrl+R", tuivory.KeyCtrlR, 0, 0}},
	}
	if keymap == "vscode" {
		return append(keys,
			editorPresetBinding{"redo", binding{"Ctrl+Y", tuivory.KeyCtrlY, 0, 0}},
			editorPresetBinding{"duplicate_line", binding{"Alt+Shift+Down", tuivory.KeyDown, tuivory.ModAlt | tuivory.ModShift, 0}},
			editorPresetBinding{"duplicate_above", binding{"Alt+Shift+Up", tuivory.KeyUp, tuivory.ModAlt | tuivory.ModShift, 0}},
			editorPresetBinding{"duplicate_line", binding{"Ctrl+Alt+Shift+Down", tuivory.KeyDown, tuivory.ModCtrl | tuivory.ModAlt | tuivory.ModShift, 0}},
			editorPresetBinding{"duplicate_above", binding{"Ctrl+Alt+Shift+Up", tuivory.KeyUp, tuivory.ModCtrl | tuivory.ModAlt | tuivory.ModShift, 0}},
			editorPresetBinding{"line_up", binding{"Alt+Up", tuivory.KeyUp, tuivory.ModAlt, 0}},
			editorPresetBinding{"line_down", binding{"Alt+Down", tuivory.KeyDown, tuivory.ModAlt, 0}},
			editorPresetBinding{"delete_line", binding{"Ctrl+Shift+K", tuivory.KeyCtrlK, tuivory.ModShift, 0}},
			editorPresetBinding{"format_code", binding{"Ctrl+Shift+I", tuivory.KeyCtrlI, tuivory.ModShift, 0}},
			editorPresetBinding{"format_code", binding{"Alt+Shift+F", tuivory.KeyRune, tuivory.ModAlt | tuivory.ModShift, 'f'}},
			editorPresetBinding{"unsupported_multicaret", binding{"", tuivory.KeyCtrlD, 0, 0}},
		)
	}
	return append(keys,
		editorPresetBinding{"duplicate_line", binding{"Ctrl+D", tuivory.KeyCtrlD, 0, 0}},
		editorPresetBinding{"delete_line", binding{"Ctrl+Y", tuivory.KeyCtrlY, 0, 0}},
		editorPresetBinding{"format_code", binding{"Ctrl+Alt+L", tuivory.KeyCtrlL, tuivory.ModAlt, 0}},
		editorPresetBinding{"line_up", binding{"Ctrl+Alt+Shift+Up", tuivory.KeyUp, tuivory.ModCtrl | tuivory.ModAlt | tuivory.ModShift, 0}},
		editorPresetBinding{"line_down", binding{"Ctrl+Alt+Shift+Down", tuivory.KeyDown, tuivory.ModCtrl | tuivory.ModAlt | tuivory.ModShift, 0}},
	)
}
func normalizedEditorMod(e *tuivory.EventKey) tuivory.ModMask {
	mod := e.Modifiers()
	if e.Key() >= tuivory.KeyCtrlA && e.Key() <= tuivory.KeyCtrlZ || e.Key() == tuivory.KeyCtrlSpace {
		mod &^= tuivory.ModCtrl
	}
	return mod
}
func (s *screen) editorPresetKey(e *tuivory.TextArea, event *tuivory.EventKey) bool {
	if e == nil {
		return false
	}
	// Extended terminal protocols keep Ctrl+Shift/Alt letters as runes, while
	// legacy C0 inputs use KeyCtrl*. Match both without changing other widgets.
	if event.Key() == tuivory.KeyRune && event.Modifiers()&tuivory.ModCtrl != 0 {
		r := unicode.ToLower(event.Rune())
		if r >= 'a' && r <= 'z' {
			event = tuivory.NewEventKey(tuivory.KeyCtrlA+tuivory.Key(r-'a'), 0, event.Modifiers())
		}
	}
	for _, key := range editorPresetBindings(s.prefs.EditorKeymap) {
		if key.key == tuivory.KeyCtrlSpace && event.Key() == tuivory.KeyRune && event.Rune() == ' ' && event.Modifiers() == tuivory.ModCtrl {
			if !s.busy {
				s.editorPresetAction(e, key.action)
			}
			return true
		}
		if event.Key() != key.key || normalizedEditorMod(event) != key.mod || key.key == tuivory.KeyRune && unicode.ToLower(event.Rune()) != key.char {
			continue
		}
		if !s.busy {
			s.editorPresetAction(e, key.action)
		}
		return true
	}
	return false
}
func (s *screen) editorPresetAction(e *tuivory.TextArea, action string) {
	switch action {
	case "editor_document_start":
		e.Select(0, 0)
	case "editor_document_end":
		e.Select(len(e.GetText()), len(e.GetText()))
	case "select_all":
		e.Select(0, len(e.GetText()))
	case "complete":
		s.showCompletion(false)
	case "find", "replace":
		s.showSearch(action == "replace")
	case "undo":
		s.nativeEditorKey(e, tuivory.KeyCtrlZ)
	case "redo":
		s.nativeEditorKey(e, tuivory.KeyCtrlY)
	case "copy", "cut":
		if !e.HasSelection() {
			s.editorLineAction(e, action+"_line")
		} else if action == "copy" {
			selected, _, _ := e.GetSelection()
			s.sendClipboard(selected)
		} else {
			s.nativeEditorKey(e, tuivory.KeyCtrlX)
		}
	case "paste":
		s.nativeEditorKey(e, tuivory.KeyCtrlV)
	case "format_code":
		s.formatCode()
	case "duplicate_above":
		s.copyEditorLines(e, true)
	case "unsupported_multicaret":
		s.setStatus("keymap_no_multicaret")
	default:
		s.editorLineAction(e, action)
	}
}
func (s *screen) copyEditorLines(e *tuivory.TextArea, above bool) {
	text := e.GetText()
	_, a, b := e.GetSelection()
	start, end := selectedLines(text, a, b)
	chunk := text[start:end]
	if above {
		if !strings.HasSuffix(chunk, "\n") {
			chunk += "\n"
		}
		e.Replace(start, start, chunk).Select(a, b)
	} else {
		if end == len(text) && !strings.HasSuffix(chunk, "\n") {
			chunk = "\n" + chunk
		}
		e.Replace(end, end, chunk).Select(a+len(chunk), b+len(chunk))
	}
}
func presetBindingLabels(keymap, id string, limit int) string {
	var labels []string
	for _, key := range editorPresetBindings(keymap) {
		if key.action == id && key.label != "" {
			labels = append(labels, key.label)
			if limit > 0 && len(labels) == limit {
				break
			}
		}
	}
	return strings.Join(labels, " / ")
}
func (s *screen) presetKeymapHelp(keymap string) string {
	var out strings.Builder
	out.WriteString(s.tr("ide_keymap_scope") + "\n")
	seen := map[string]bool{}
	for _, key := range editorPresetBindings(keymap) {
		if seen[key.action] || key.label == "" {
			continue
		}
		seen[key.action] = true
		label := s.tr(commandFor(key.action).label)
		if strings.HasPrefix(key.action, "editor_document_") {
			label = s.tr(key.action)
		}
		if key.action == "duplicate_above" {
			label = s.tr("duplicate_above")
		}
		out.WriteString(presetBindingLabels(keymap, key.action, 0) + "\t" + label + "\n")
	}
	return out.String()
}
