package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Only the focused SQL/source editor may consume editor-map keys.
func (s *screen) activeEditor() *tview.TextArea {
	page, _ := s.pages.GetFrontPage()
	if page == "main" && s.editor != nil && s.editor.HasFocus() {
		return s.editor
	}
	if page == "source" && s.sourceLoaded && s.sourceEditor != nil && s.sourceEditor.HasFocus() {
		return s.sourceEditor
	}
	return nil
}

func (s *screen) nativeEditorKey(e *tview.TextArea, key tcell.Key) {
	e.InputHandler()(tcell.NewEventKey(key, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
}

func (s *screen) editorKey(event *tcell.EventKey) bool {
	e := s.activeEditor()
	if e == nil || event.Modifiers()&tcell.ModAlt != 0 {
		return false
	}
	if s.busy && s.prefs.EditorKeymap != "" {
		switch event.Key() {
		case tcell.KeyRune, tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyDelete, tcell.KeyCtrlD, tcell.KeyCtrlL, tcell.KeyCtrlR, tcell.KeyCtrlV, tcell.KeyCtrlX, tcell.KeyCtrlZ, tcell.KeyCtrlY, tcell.KeyCtrlT, tcell.KeyCtrlSpace:
			return true
		}
		return false
	}
	mod := event.Modifiers() &^ tcell.ModCtrl
	if s.prefs.EditorKeymap == "notepad++" {
		switch {
		case event.Key() == tcell.KeyCtrlA && mod == 0:
			e.Select(0, len(e.GetText()))
			return true
		case event.Key() == tcell.KeyCtrlD && mod == 0:
			s.editorLineAction(e, "duplicate_line")
			return true
		case event.Key() == tcell.KeyCtrlL && mod == 0:
			s.editorLineAction(e, "cut_line")
			return true
		case event.Key() == tcell.KeyCtrlL && mod == tcell.ModShift:
			s.editorLineAction(e, "delete_line")
			return true
		case event.Key() == tcell.KeyCtrlT && mod == 0:
			s.editorLineAction(e, "transpose_line")
			return true
		case (event.Key() == tcell.KeyUp || event.Key() == tcell.KeyDown) && event.Modifiers() == tcell.ModCtrl|tcell.ModShift:
			action := "line_up"
			if event.Key() == tcell.KeyDown {
				action = "line_down"
			}
			s.editorLineAction(e, action)
			return true
		case (event.Key() == tcell.KeyCtrlC || event.Key() == tcell.KeyCtrlX) && mod == 0 && !e.HasSelection():
			action := "copy_line"
			if event.Key() == tcell.KeyCtrlX {
				action = "cut_line"
			}
			s.editorLineAction(e, action)
			return true
		case event.Key() == tcell.KeyCtrlSpace || event.Key() == tcell.KeyRune && event.Rune() == ' ' && event.Modifiers() == tcell.ModCtrl:
			s.showCompletion(false)
			return true
		}
	}
	if s.prefs.EditorKeymap == "vim" && s.vimKey(e, event) {
		return true
	}
	// Source overlays do not go through the main command dispatcher.
	page, _ := s.pages.GetFrontPage()
	if page == "source" {
		switch {
		case event.Key() == tcell.KeyCtrlF && mod == 0:
			s.showSearch(false)
			return true
		case event.Key() == tcell.KeyCtrlR && mod == 0 && s.prefs.EditorKeymap == "notepad++":
			s.showSearch(true)
			return true
		case event.Key() == tcell.KeyF3:
			s.findEditor(e, event.Modifiers()&tcell.ModShift != 0)
			return true
		}
	}
	return false
}

func lineStart(text string, pos int) int { return strings.LastIndex(text[:pos], "\n") + 1 }
func lineEnd(text string, pos int) int {
	if at := strings.IndexByte(text[pos:], '\n'); at >= 0 {
		return pos + at
	}
	return len(text)
}
func selectedLines(text string, a, b int) (int, int) {
	start := lineStart(text, a)
	if b > a && text[b-1] == '\n' {
		b--
	}
	end := lineEnd(text, b)
	if end < len(text) {
		end++
	}
	return start, end
}

// One Replace per operation retains the editor's undo history and original text.
func (s *screen) editorLineAction(e *tview.TextArea, action string) {
	text := e.GetText()
	selected, a, b := e.GetSelection()
	start, end := selectedLines(text, a, b)
	if action == "duplicate_line" && selected != "" {
		e.Replace(b, b, selected).Select(b, b+len(selected))
		return
	}
	switch action {
	case "duplicate_line":
		chunk := text[start:end]
		if end == len(text) && !strings.HasSuffix(chunk, "\n") {
			chunk = "\n" + chunk
		}
		e.Replace(end, end, chunk).Select(end, end)
		return
	case "copy_line", "cut_line", "delete_line":
		if action != "delete_line" {
			s.sendClipboard(text[start:end])
		}
		if action != "copy_line" {
			// Removing the last line must also remove the preceding separator.
			if end == len(text) && start > 0 && !strings.HasSuffix(text[start:end], "\n") {
				start--
			}
			e.Replace(start, end, "")
		}
	case "transpose_line", "line_up", "line_down":
		// Keep line endings and a selected block's relative byte offsets intact.
		if action == "transpose_line" {
			start, end = selectedLines(text, a, a)
		}
		block := strings.TrimSuffix(text[start:end], "\n")
		if action == "line_down" {
			if end >= len(text) {
				return
			}
			nextEnd := lineEnd(text, end)
			tail := ""
			if nextEnd < len(text) {
				nextEnd++
				tail = "\n"
			}
			next := strings.TrimSuffix(text[end:nextEnd], "\n")
			e.Replace(start, nextEnd, next+"\n"+block+tail)
			newStart := start + len(next) + 1
			e.Select(newStart+a-start, newStart+min(b-start, len(block)+len(tail)))
		} else {
			if start == 0 {
				return
			}
			prevStart := lineStart(text, start-1)
			previous := text[prevStart : start-1]
			tail := ""
			if strings.HasSuffix(text[start:end], "\n") {
				tail = "\n"
			}
			e.Replace(prevStart, end, block+"\n"+previous+tail)
			e.Select(prevStart+a-start, prevStart+min(b-start, len(block)+1))
		}
	}
}

func (s *screen) keymapHint() string {
	switch s.prefs.EditorKeymap {
	case "vim":
		return "Vim " + s.vimMode() + " | " + s.tr("vim_hint")
	case "notepad++":
		return "Notepad++ | " + s.tr("npp_hint")
	}
	return s.tr("default_keymap_hint")
}

func (s *screen) editorBindingLabels(id string, limit int) string {
	if s.prefs.EditorKeymap == "notepad++" {
		switch id {
		case "select_all":
			return "Ctrl+A"
		case "complete":
			return "Ctrl+Space"
		case "quick_describe":
			return ""
		case "duplicate_line":
			return "Ctrl+D"
		case "cut_line":
			return "Ctrl+L"
		case "delete_line":
			return "Ctrl+Shift+L"
		case "transpose_line":
			return "Ctrl+T"
		case "line_up":
			return "Ctrl+Shift+Up"
		case "line_down":
			return "Ctrl+Shift+Down"
		}
	}
	if s.prefs.EditorKeymap == "vim" && id == "redo" {
		return "Ctrl+R / Ctrl+Y"
	}
	if s.prefs.EditorKeymap == "vim" && id == "replace" {
		return ""
	}
	return bindingLabels(id, limit)
}

func (s *screen) sourceKeymapFooter() string {
	if s.prefs.EditorKeymap == "" {
		return s.tr("footer_source") + " | Alt+R / Alt+X"
	}
	return s.keymapHint() + " | F9 " + s.tr("compile") + " | Alt+R " + s.tr("reload") + " | Alt+X " + s.tr("close")
}
func (s *screen) keymapHelpSections() []struct{ title, text string } {
	return []struct{ title, text string }{
		{s.tr("help_keymap_common"), s.tr("keymap_scope")},
		{s.tr("keymap_default"), s.tr("default_commands")},
		{"Vim", s.tr("vim_commands")},
		{"Notepad++", s.tr("npp_commands")},
	}
}
func (s *screen) keymapHelp() string {
	var out strings.Builder
	out.WriteString(s.keymapHint())
	for _, section := range s.keymapHelpSections() {
		out.WriteString("\n" + section.title + "\n" + strings.ReplaceAll(section.text, "\t", "  "))
	}
	return out.String()
}

func (s *screen) closeSourceEdit() {
	s.pages.RemovePage("editor_actions")
	if s.sourceEditor != nil {
		s.app.SetFocus(s.sourceEditor)
	}
}
func (s *screen) showSourceEdit() {
	page, _ := s.pages.GetFrontPage()
	if page != "source" || s.busy || !s.sourceLoaded {
		return
	}
	e := s.sourceEditor
	list := tview.NewList().SetMainTextColor(s.colors().text).SetSecondaryTextColor(s.colors().muted).SetShortcutColor(s.colors().accent).ShowSecondaryText(false).SetSelectedStyle(s.focusStyle())
	list.SetBorder(true).SetTitle(s.tr("source_edit"))
	s.styleBox(list.Box)
	for _, id := range []string{"undo", "redo", "cut", "copy", "paste", "select_all", "find", "replace", "complete", "complete_refresh", "duplicate_line", "copy_line", "cut_line", "delete_line", "line_up", "line_down", "transpose_line", "format_code"} {
		label := s.tr(commandFor(id).label)
		if key := s.editorBindingLabels(id, 1); key != "" {
			label += "  " + key
		}
		list.AddItem(label, "", 0, func() {
			s.closeSourceEdit()
			switch id {
			case "undo", "redo", "cut", "paste", "select_all":
				if id == "cut" && s.prefs.EditorKeymap == "notepad++" && !e.HasSelection() {
					s.editorLineAction(e, "cut_line")
					break
				}
				s.nativeEditorKey(e, commandFor(id).bindings[0].key)
			case "copy":
				selected, _, _ := e.GetSelection()
				if selected == "" && s.prefs.EditorKeymap == "notepad++" {
					s.editorLineAction(e, "copy_line")
				} else {
					s.sendClipboard(selected)
				}
			case "find", "replace":
				s.showSearch(id == "replace")
			case "complete", "complete_refresh":
				s.showCompletion(id == "complete_refresh")
			case "format_code":
				s.formatCode()
			default:
				s.editorLineAction(e, id)
			}
		})
	}
	s.pages.AddPage("editor_actions", list, true, true)
	s.app.SetFocus(list)
}

func (s *screen) vimMode() string {
	if s.vim.editor == s.activeEditor() && s.vim.mode != "" {
		return s.vim.mode
	}
	return "NORMAL"
}
