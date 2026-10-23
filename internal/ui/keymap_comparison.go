package ui

import (
	"encoding/csv"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
)

type keymapComparison struct {
	table *tview.Table
	focus tview.Primitive
	rows  [][]string
}

var comparisonMaps = []struct{ id, name string }{{"", "Toad"}, {"vim", "Vim"}, {"notepad++", "Notepad++"}, {"vscode", "VS Code"}, {"intellij", "IntelliJ IDEA"}}
var comparisonGroups = []struct {
	label   string
	actions []string
}{
	{"mg_clipboard", []string{"select_all", "copy", "cut", "paste"}},
	{"mg_lines", []string{"undo", "redo", "duplicate_line", "delete_line", "line_up", "line_down"}},
	{"mg_code", []string{"find", "find_next", "find_previous", "replace", "complete", "format_code"}},
	{"mg_navigation", []string{"editor_document_start", "editor_document_end"}},
	{"comparison_shared", []string{"execute", "script", "commit", "rollback", "fullscreen_sql", "fullscreen_results", "context_menu", "workspaces"}},
}

func (s *screen) comparisonBinding(keymap, action string) string {
	view := screen{prefs: s.prefs}
	view.prefs.EditorKeymap = keymap
	if strings.HasPrefix(action, "editor_document_") {
		if label := presetBindingLabels(keymap, action, 0); label != "" {
			return label
		}
		return "—"
	}
	if label := view.editorBindingLabels(action, -1); label != "" {
		return label
	}
	if commandFor(action).id != "" {
		return s.tr("comparison_menu")
	}
	return "—"
}
func (s *screen) showKeymapComparison() {
	if s.comparison != nil {
		return
	}
	c := &keymapComparison{focus: s.app.GetFocus()}
	s.comparison = c
	table := tview.NewTable().SetFixed(1, 1).SetSelectable(true, true).SetSeparator('│').SetSelectedStyle(s.focusStyle())
	c.table = table
	s.styleBox(table.Box)
	table.SetBorder(true).SetTitle(s.tr("keymap_compare"))
	selected := 1
	headers := []string{s.tr("comparison_action")}
	for i, preset := range comparisonMaps {
		label := preset.name
		if preset.id == s.prefs.EditorKeymap {
			label += " *"
			selected = i + 1
		}
		headers = append(headers, label)
	}
	c.rows = append(c.rows, headers)
	for col, name := range headers {
		style := s.gridHeaderStyle(false)
		if col == selected {
			style = s.focusStyle()
		}
		table.SetCell(0, col, tview.NewTableCell(tview.Escape(name)).SetStyle(style).SetSelectable(false).SetMaxWidth(22))
	}
	row := 1
	for _, group := range comparisonGroups {
		c.rows = append(c.rows, []string{s.tr(group.label)})
		for col := 0; col <= len(comparisonMaps); col++ {
			label := ""
			if col == 0 {
				label = s.tr(group.label)
			}
			table.SetCell(row, col, tview.NewTableCell(tview.Escape(label)).SetStyle(s.gridHeaderStyle(false)).SetSelectable(false).SetMaxWidth(22))
		}
		row++
		for _, action := range group.actions {
			label := s.tr(commandFor(action).label)
			if strings.HasPrefix(action, "editor_document_") {
				label = s.tr(action)
			}
			values := []string{label}
			for _, preset := range comparisonMaps {
				values = append(values, s.comparisonBinding(preset.id, action))
			}
			c.rows = append(c.rows, values)
			for col, value := range values {
				table.SetCell(row, col, tview.NewTableCell(tview.Escape(" "+value+" ")).SetStyle(tcell.StyleDefault.Foreground(s.colors().text).Background(s.colors().surface)).SetMaxWidth(22))
			}
			row++
		}
	}
	table.Select(2, selected)
	info := s.newTextView(textHint).SetText(s.tr("comparison_note")).SetWrap(true)
	close := s.clickable(tview.NewButton(s.tr("close") + " (Esc)").SetSelectedFunc(s.closeKeymapComparison))
	content := s.newFlex().SetDirection(tview.FlexRow).AddItem(table, 0, 1, true).AddItem(info, 4, 0, false).AddItem(close, 1, 0, false)
	root := s.formPopup(content, 140, 34, func(w, h int) { content.ResizeItem(info, min(4, max(1, h/4)), 0) })
	s.pages.AddPage("keymap_compare", root, true, true)
	s.app.SetFocus(table)
}
func (s *screen) closeKeymapComparison() {
	if s.comparison == nil {
		return
	}
	focus := s.comparison.focus
	s.pages.RemovePage("keymap_compare")
	s.comparison = nil
	if focus != nil {
		s.app.SetFocus(focus)
	}
}
func (s *screen) keymapComparisonKey(e *tcell.EventKey) *tcell.EventKey {
	switch e.Key() {
	case tcell.KeyEscape:
		s.closeKeymapComparison()
		return nil
	case tcell.KeyCtrlC:
		var out strings.Builder
		writer := csv.NewWriter(&out)
		writer.Comma = '\t'
		for _, row := range s.comparison.rows {
			_ = writer.Write(row)
		}
		writer.Flush()
		s.sendClipboard(out.String())
		return nil
	case tcell.KeyTab, tcell.KeyBacktab:
		s.dialogTab(e, s.comparison.table)
		return nil
	}
	if s.buttonArrow(e) {
		return nil
	}
	return e
}
