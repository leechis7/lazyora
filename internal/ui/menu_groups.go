package ui

type commandMenuRow struct {
	id, label string
	children  []command
	divider   bool
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
	nested := index >= 0 && len(entries) > 12
	for group, key := range order {
		if group > 0 && !nested {
			rows = append(rows, commandMenuRow{divider: true})
		}
		if nested && len(buckets[key]) > 1 {
			rows = append(rows, commandMenuRow{label: s.tr(key), children: buckets[key]})
			continue
		}
		for _, cmd := range buckets[key] {
			rows = append(rows, commandMenuRow{id: cmd.id})
		}
	}
	return rows
}
