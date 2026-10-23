package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
	"unicode"
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
		{"editor_document_start", binding{"Ctrl+Home", tcell.KeyHome, tcell.ModCtrl, 0}},
		{"editor_document_end", binding{"Ctrl+End", tcell.KeyEnd, tcell.ModCtrl, 0}},
		{"select_all", binding{"Ctrl+A", tcell.KeyCtrlA, 0, 0}},
		{"complete", binding{"Ctrl+Space", tcell.KeyCtrlSpace, 0, 0}},
		{"copy", binding{"Ctrl+C", tcell.KeyCtrlC, 0, 0}},
		{"cut", binding{"Ctrl+X", tcell.KeyCtrlX, 0, 0}},
		{"paste", binding{"Ctrl+V", tcell.KeyCtrlV, 0, 0}},
		{"undo", binding{"Ctrl+Z", tcell.KeyCtrlZ, 0, 0}},
		{"redo", binding{"Ctrl+Shift+Z", tcell.KeyCtrlZ, tcell.ModShift, 0}},
		{"find", binding{"Ctrl+F", tcell.KeyCtrlF, 0, 0}},
		{"replace", binding{"Ctrl+R", tcell.KeyCtrlR, 0, 0}},
	}
	if keymap == "vscode" {
		return append(keys,
			editorPresetBinding{"redo", binding{"Ctrl+Y", tcell.KeyCtrlY, 0, 0}},
			editorPresetBinding{"duplicate_line", binding{"Alt+Shift+Down", tcell.KeyDown, tcell.ModAlt | tcell.ModShift, 0}},
			editorPresetBinding{"duplicate_above", binding{"Alt+Shift+Up", tcell.KeyUp, tcell.ModAlt | tcell.ModShift, 0}},
			editorPresetBinding{"duplicate_line", binding{"Ctrl+Alt+Shift+Down", tcell.KeyDown, tcell.ModCtrl | tcell.ModAlt | tcell.ModShift, 0}},
			editorPresetBinding{"duplicate_above", binding{"Ctrl+Alt+Shift+Up", tcell.KeyUp, tcell.ModCtrl | tcell.ModAlt | tcell.ModShift, 0}},
			editorPresetBinding{"line_up", binding{"Alt+Up", tcell.KeyUp, tcell.ModAlt, 0}},
			editorPresetBinding{"line_down", binding{"Alt+Down", tcell.KeyDown, tcell.ModAlt, 0}},
			editorPresetBinding{"delete_line", binding{"Ctrl+Shift+K", tcell.KeyCtrlK, tcell.ModShift, 0}},
			editorPresetBinding{"format_code", binding{"Ctrl+Shift+I", tcell.KeyCtrlI, tcell.ModShift, 0}},
			editorPresetBinding{"format_code", binding{"Alt+Shift+F", tcell.KeyRune, tcell.ModAlt | tcell.ModShift, 'f'}},
			editorPresetBinding{"unsupported_multicaret", binding{"", tcell.KeyCtrlD, 0, 0}},
		)
	}
	return append(keys,
		editorPresetBinding{"duplicate_line", binding{"Ctrl+D", tcell.KeyCtrlD, 0, 0}},
		editorPresetBinding{"delete_line", binding{"Ctrl+Y", tcell.KeyCtrlY, 0, 0}},
		editorPresetBinding{"format_code", binding{"Ctrl+Alt+L", tcell.KeyCtrlL, tcell.ModAlt, 0}},
		editorPresetBinding{"line_up", binding{"Ctrl+Alt+Shift+Up", tcell.KeyUp, tcell.ModCtrl | tcell.ModAlt | tcell.ModShift, 0}},
		editorPresetBinding{"line_down", binding{"Ctrl+Alt+Shift+Down", tcell.KeyDown, tcell.ModCtrl | tcell.ModAlt | tcell.ModShift, 0}},
	)
}
func normalizedEditorMod(e *tcell.EventKey) tcell.ModMask {
	mod := e.Modifiers()
	if e.Key() >= tcell.KeyCtrlA && e.Key() <= tcell.KeyCtrlZ || e.Key() == tcell.KeyCtrlSpace {
		mod &^= tcell.ModCtrl
	}
	return mod
}
func (s *screen) editorPresetKey(e *tview.TextArea, event *tcell.EventKey) bool {
	if e == nil {
		return false
	}
	// Extended terminal protocols keep Ctrl+Shift/Alt letters as runes, while
	// legacy C0 inputs use KeyCtrl*. Match both without changing other widgets.
	if event.Key() == tcell.KeyRune && event.Modifiers()&tcell.ModCtrl != 0 {
		r := unicode.ToLower(event.Rune())
		if r >= 'a' && r <= 'z' {
			event = tcell.NewEventKey(tcell.KeyCtrlA+tcell.Key(r-'a'), 0, event.Modifiers())
		}
	}
	for _, key := range editorPresetBindings(s.prefs.EditorKeymap) {
		if key.key == tcell.KeyCtrlSpace && event.Key() == tcell.KeyRune && event.Rune() == ' ' && event.Modifiers() == tcell.ModCtrl {
			if !s.busy {
				s.editorPresetAction(e, key.action)
			}
			return true
		}
		if event.Key() != key.key || normalizedEditorMod(event) != key.mod || key.key == tcell.KeyRune && unicode.ToLower(event.Rune()) != key.char {
			continue
		}
		if !s.busy {
			s.editorPresetAction(e, key.action)
		}
		return true
	}
	return false
}
func (s *screen) editorPresetAction(e *tview.TextArea, action string) {
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
		s.nativeEditorKey(e, tcell.KeyCtrlZ)
	case "redo":
		s.nativeEditorKey(e, tcell.KeyCtrlY)
	case "copy", "cut":
		if !e.HasSelection() {
			s.editorLineAction(e, action+"_line")
		} else if action == "copy" {
			selected, _, _ := e.GetSelection()
			s.sendClipboard(selected)
		} else {
			s.nativeEditorKey(e, tcell.KeyCtrlX)
		}
	case "paste":
		s.nativeEditorKey(e, tcell.KeyCtrlV)
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
func (s *screen) copyEditorLines(e *tview.TextArea, above bool) {
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
