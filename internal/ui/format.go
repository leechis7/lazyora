package ui

import (
	"lazyora/internal/oracle"
)

func (s *screen) formatCode() {
	if s.busy {
		return
	}
	page, _ := s.pages.GetFrontPage()
	editor := s.editor
	if page == "source" && s.sourceLoaded && s.sourceEditor.HasFocus() {
		editor = s.sourceEditor
	} else if page != "main" || !s.editor.HasFocus() {
		return
	}
	text := editor.GetText()
	selected, start, end := editor.GetSelection()
	cursor := start
	if selected == "" {
		start = 0
		end = len(text)
	}
	formatted, err := oracle.FormatSQLRange(text, start, end)
	if err != nil {
		s.setStatus("format_failed", err)
		return
	}
	if formatted == text[start:end] {
		s.setStatus("format_unchanged")
		return
	}
	mapped := 0
	if selected == "" {
		mapped = oracle.FormattedCursor(text, formatted, cursor)
	}
	editor.Replace(start, end, formatted)
	if selected != "" {
		editor.Select(start, start+len(formatted))
	} else {
		editor.Select(mapped, mapped)
	}
	s.app.SetFocus(editor)
	s.setStatus("format_done")
}
