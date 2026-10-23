package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type commandMenuRow struct {
	id, label     string
	children      []command
	divider, back bool
}
type commandMenuParent struct {
	title           string
	entries         []command
	index, selected int
}

func menuSection(id string) string {
	switch id {
	case "undo", "redo":
		return "mg_undo"
	case "cut", "copy", "paste", "select_all", "copy_row", "copy_preview", "export":
		return "mg_clipboard"
	case "find", "find_next", "find_previous", "replace":
		return "mg_search"
	case "duplicate_line", "copy_line", "cut_line", "delete_line", "line_up", "line_down", "transpose_line", "clear_sql":
		return "mg_lines"
	case "complete", "complete_refresh", "format_code":
		return "mg_code"
	case "table_edit_load", "grid_edit", "grid_inline":
		return "mg_data_edit"
	case "fullscreen_sql", "fullscreen_results", "focus_toggle", "context_menu", "maximize", "resize_panels":
		return "mg_windows"
	case "browser", "browser_user", "browser_type", "browser_search", "browser_refresh", "describe", "quick_describe":
		return "mg_schema"
	case "grid_widen", "grid_narrow", "grid_auto", "fetch_more", "result_sort_asc", "result_sort_desc", "result_filter", "result_view_reset", "cell_detail":
		return "mg_results"
	case "history", "output_view", "script_report", "routine_results", "output_toggle":
		return "mg_reports"
	case "connect", "workspaces", "workspace_new", "workspace_close":
		return "mg_connections"
	case "commit", "rollback", "pending_work":
		return "mg_transaction"
	case "file_new", "file_close", "tab_rename":
		return "mg_tabs"
	case "tab_next", "tab_previous":
		return "mg_navigation"
	case "file_open", "file_save", "file_save_as":
		return "mg_files"
	case "execute", "script", "routine_execute":
		return "mg_run"
	case "plan", "parse", "debug_continue":
		return "mg_analysis"
	case "help", "guide", "release_history":
		return "mg_help"
	case "diagnostic_info", "issue_report":
		return "mg_support"
	}
	return id
}
func (s *screen) commandMenuRows(entries []command, index int) []commandMenuRow {
	var order []string
	buckets := map[string][]command{}
	for _, cmd := range entries {
		section := menuSection(cmd.id)
		if len(buckets[section]) == 0 {
			order = append(order, section)
		}
		buckets[section] = append(buckets[section], cmd)
	}
	var rows []commandMenuRow
	if s.menuParent != nil {
		rows = append(rows, commandMenuRow{back: true, label: s.tr("mg_back")}, commandMenuRow{divider: true})
	}
	nested := index >= 0 && len(entries) > 12 && s.menuParent == nil
	for group, key := range order {
		if group > 0 && !nested {
			rows = append(rows, commandMenuRow{divider: true})
		}
		if nested && len(buckets[key]) > 1 {
			rows = append(rows, commandMenuRow{label: s.tr(key) + "  ›", children: buckets[key]})
			continue
		}
		for _, cmd := range buckets[key] {
			rows = append(rows, commandMenuRow{id: cmd.id})
		}
	}
	return rows
}
func (s *screen) menuRowAction(index int) {
	if index < 0 || index >= len(s.menuRows) {
		return
	}
	row := s.menuRows[index]
	if row.divider {
		return
	}
	if row.back {
		s.backCommandMenu()
		return
	}
	if len(row.children) > 0 {
		s.menuParent = &commandMenuParent{s.menuTitle, s.menuEntries, s.menuIndex, index}
		s.renderCommandMenu(s.menuTitle+" / "+row.label, row.children, s.menuIndex)
		return
	}
	s.closeMenu()
	s.invoke(row.id)
}
func (s *screen) backCommandMenu() bool {
	parent := s.menuParent
	if parent == nil {
		return false
	}
	s.menuParent = nil
	s.renderCommandMenu(parent.title, parent.entries, parent.index)
	s.menuList.SetCurrentItem(parent.selected)
	return true
}
func (s *screen) menuListKey(e *tcell.EventKey) *tcell.EventKey {
	list := s.menuList
	index := list.GetCurrentItem()
	step := 0
	switch e.Key() {
	case tcell.KeyUp:
		step = -1
	case tcell.KeyDown:
		step = 1
	case tcell.KeyHome:
		index = 0
		step = 1
	case tcell.KeyEnd:
		index = len(s.menuRows) - 1
		step = -1
	case tcell.KeyPgUp:
		index = max(0, index-5)
		step = -1
	case tcell.KeyPgDn:
		index = min(len(s.menuRows)-1, index+5)
		step = 1
	default:
		return e
	}
	if e.Key() == tcell.KeyUp || e.Key() == tcell.KeyDown {
		index = (index + step + len(s.menuRows)) % len(s.menuRows)
	}
	for s.menuRows[index].divider {
		index = (index + step + len(s.menuRows)) % len(s.menuRows)
	}
	list.SetCurrentItem(index)
	return nil
}
func (s *screen) menuListMouse(action tview.MouseAction, e *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
	if action == tview.MouseLeftDoubleClick {
		action = tview.MouseLeftClick
	}
	if action == tview.MouseLeftClick {
		x, y := e.Position()
		left, top, w, h := s.menuList.GetInnerRect()
		offset, _ := s.menuList.GetOffset()
		index := y - top + offset
		if x >= left && x < left+w && y >= top && y < top+h && index >= 0 && index < len(s.menuRows) && s.menuRows[index].divider {
			return tview.MouseConsumed, nil
		}
	}
	return action, e
}
