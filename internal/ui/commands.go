package ui

import (
	"strings"

	"github.com/leechis7/tuivory/go"
)

type binding struct {
	label string
	key   tuivory.Key
	mod   tuivory.ModMask
	char  rune
}
type command struct {
	id, label string
	group     int
	bindings  []binding
}

var commands = []command{
	{"file_new", "file_new", menuTabs, []binding{{"Ctrl+N", tuivory.KeyCtrlN, 0, 0}}},
	{"file_open", "file_open", menuFile, []binding{{"Ctrl+O", tuivory.KeyCtrlO, 0, 0}}},
	{"file_save", "file_save", menuFile, []binding{{"Ctrl+S", tuivory.KeyCtrlS, 0, 0}}},
	{"file_save_as", "file_save_as", menuFile, []binding{{"Ctrl+Shift+S", tuivory.KeyCtrlS, tuivory.ModShift, 0}}},
	{"file_close", "file_close", menuTabs, []binding{{"Ctrl+W", tuivory.KeyCtrlW, 0, 0}}},
	{"tab_rename", "tab_rename", menuTabs, nil},
	{"tab_next", "tab_next", menuTabs, []binding{{"Ctrl+Tab", tuivory.KeyTab, tuivory.ModCtrl, 0}, {"Alt+PgDn", tuivory.KeyPgDn, tuivory.ModAlt, 0}}},
	{"tab_previous", "tab_previous", menuTabs, []binding{{"Ctrl+Shift+Tab", tuivory.KeyBacktab, tuivory.ModCtrl, 0}, {"Alt+PgUp", tuivory.KeyPgUp, tuivory.ModAlt, 0}}},
	{"quit", "quit", menuFile, []binding{{"Ctrl+Q", tuivory.KeyCtrlQ, 0, 0}}},
	{"undo", "undo", menuEdit, []binding{{"Ctrl+Z", tuivory.KeyCtrlZ, 0, 0}}},
	{"redo", "redo", menuEdit, []binding{{"Ctrl+Y", tuivory.KeyCtrlY, 0, 0}}},
	{"cut", "cut", menuEdit, []binding{{"Ctrl+X", tuivory.KeyCtrlX, 0, 0}}},
	{"paste", "paste", menuEdit, []binding{{"Ctrl+V", tuivory.KeyCtrlV, 0, 0}}},
	{"select_all", "select_all", menuEdit, []binding{{"Ctrl+L", tuivory.KeyCtrlL, 0, 0}}},
	{"copy", "copy_selection", menuEdit, []binding{{"Ctrl+C", tuivory.KeyCtrlC, 0, 0}}},
	{"copy_row", "copy_row", menuEdit, []binding{{"Alt+Y", tuivory.KeyRune, tuivory.ModAlt, 'y'}}},
	{"export", "export", menuEdit, []binding{{"Ctrl+Shift+E", tuivory.KeyCtrlE, tuivory.ModShift, 0}, {"Alt+W", tuivory.KeyRune, tuivory.ModAlt, 'w'}}},
	{"copy_preview", "copy_preview", menuEdit, nil},
	{"table_edit_load", "table_edit_load", menuEdit, nil},
	{"grid_edit", "grid_edit", menuEdit, []binding{{"Alt+D", tuivory.KeyRune, tuivory.ModAlt, 'd'}}},
	{"grid_insert", "grid_insert", menuEdit, []binding{{"Alt+Insert", tuivory.KeyInsert, tuivory.ModAlt, 0}}},
	{"grid_delete", "grid_delete", menuEdit, []binding{{"Alt+Delete", tuivory.KeyDelete, tuivory.ModAlt, 0}}},
	{"grid_inline", "grid_inline", menuEdit, nil},
	{"find", "find", menuEdit, []binding{{"Ctrl+F", tuivory.KeyCtrlF, 0, 0}}},
	{"find_next", "find_next", menuEdit, []binding{{"F3", tuivory.KeyF3, 0, 0}}},
	{"find_previous", "find_previous", menuEdit, []binding{{"Shift+F3", tuivory.KeyF3, tuivory.ModShift, 0}}},
	{"replace", "replace", menuEdit, []binding{{"Ctrl+R", tuivory.KeyCtrlR, 0, 0}}},
	{"clear_sql", "clear_sql", menuEdit, []binding{{"F7", tuivory.KeyF7, 0, 0}}},
	{"connect", "connect", menuSession, []binding{{"Alt+L", tuivory.KeyRune, tuivory.ModAlt, 'l'}}},
	{"commit", "commit", menuSession, []binding{{"Alt+C", tuivory.KeyRune, tuivory.ModAlt, 'c'}}},
	{"rollback", "rollback", menuSession, []binding{{"Alt+R", tuivory.KeyRune, tuivory.ModAlt, 'r'}}},
	{"session_browser", "session_browser", menuSession, nil},
	{"pending_work", "pending_work", menuSession, nil},
	{"output_toggle", "output_toggle", menuSession, nil},
	{"execute", "execute", menuExecute, []binding{{"F9", tuivory.KeyF9, 0, 0}, {"Ctrl+Enter", tuivory.KeyEnter, tuivory.ModCtrl, 0}, {"Alt+Enter", tuivory.KeyEnter, tuivory.ModAlt, 0}, {"Shift+F9", tuivory.KeyF9, tuivory.ModShift, 0}}},
	{"script", "script", menuExecute, []binding{{"F5", tuivory.KeyF5, 0, 0}}},
	{"plan", "plan", menuExecute, []binding{{"Ctrl+E", tuivory.KeyCtrlE, 0, 0}}},
	{"actual_plan_execute", "actual_plan_execute", menuExecute, nil},
	{"actual_plan_last", "actual_plan_last", menuExecute, nil},
	{"script_report", "script_report", menuView, nil},
	{"result_area_grid", "result_area_grid", menuView, nil},
	{"parse", "parse_unavailable", menuExecute, []binding{{"Ctrl+F9", tuivory.KeyF9, tuivory.ModCtrl, 0}}},
	{"debug_start", "debug_start", menuExecute, nil},
	{"cancel", "cancel_execution", menuExecute, []binding{{"Esc", tuivory.KeyEscape, 0, 0}}},
	{"history", "history", menuView, []binding{{"F8", tuivory.KeyF8, 0, 0}}},
	{"favorites", "favorites", menuView, nil},
	{"favorite_save", "favorite_save", menuEdit, nil},
	{"output_view", "output_view", menuView, []binding{{"Alt+P", tuivory.KeyRune, tuivory.ModAlt, 'p'}}},
	{"fullscreen_sql", "fullscreen_sql", menuView, []binding{{"F2", tuivory.KeyF2, 0, 0}}},
	{"fullscreen_results", "fullscreen_results", menuView, []binding{{"Shift+F2", tuivory.KeyF2, tuivory.ModShift, 0}, {"F12", tuivory.KeyF12, 0, 0}}},
	{"focus_toggle", "focus_toggle", menuView, []binding{{"F6", tuivory.KeyF6, 0, 0}}},
	{"context_menu", "context_menu", menuView, []binding{{"Shift+F10", tuivory.KeyF10, tuivory.ModShift, 0}}},
	{"maximize", "maximize", menuView, nil},
	{"resize_panels", "resize_panels", menuView, []binding{{"Alt+F8", tuivory.KeyF8, tuivory.ModAlt, 0}}},
	{"collapse_schema", "collapse_schema", menuView, []binding{{"Alt+F7", tuivory.KeyF7, tuivory.ModAlt, 0}}},
	{"collapse_results", "collapse_results", menuView, []binding{{"Alt+F9", tuivory.KeyF9, tuivory.ModAlt, 0}}},
	{"browser", "schemas", menuView, []binding{{"Alt+B", tuivory.KeyRune, tuivory.ModAlt, 'b'}}},
	{"browser_search", "browser_search", menuView, []binding{{"Alt+I", tuivory.KeyRune, tuivory.ModAlt, 'i'}}},
	{"browser_user", "browser_user", menuView, []binding{{"Alt+U", tuivory.KeyRune, tuivory.ModAlt, 'u'}}},
	{"browser_type", "browser_type", menuView, []binding{{"Alt+O", tuivory.KeyRune, tuivory.ModAlt, 'o'}}},
	{"browser_refresh", "browser_refresh", menuView, nil},
	{"describe", "describe", menuView, []binding{{"F4", tuivory.KeyF4, 0, 0}}},
	{"quick_describe", "quick_describe", menuView, []binding{{"Ctrl+D", tuivory.KeyCtrlD, 0, 0}}},
	{"workspaces", "workspaces", menuSession, []binding{{"Alt+J", tuivory.KeyRune, tuivory.ModAlt, 'j'}}},
	{"workspace_new", "workspace_new", menuSession, nil},
	{"workspace_close", "workspace_close", menuSession, nil},
	{"settings", "settings", menuTools, nil},
	{"schema_compare", "schema_compare", menuTools, nil},
	{"data_compare", "data_compare", menuTools, nil},
	{"passwords", "passwords", menuTools, []binding{{"Alt+K", tuivory.KeyRune, tuivory.ModAlt, 'k'}}},
	{"routine_execute", "routine_execute", menuExecute, nil},
	{"routine_results", "routine_results", menuView, nil},
	{"grid_widen", "grid_widen", menuView, nil},
	{"grid_narrow", "grid_narrow", menuView, nil},
	{"grid_auto", "grid_auto", menuView, nil},
	{"fetch_more", "fetch_more", menuView, nil},
	{"result_sort_asc", "result_sort_asc", menuView, nil},
	{"result_sort_desc", "result_sort_desc", menuView, nil},
	{"result_filter", "result_filter", menuView, nil},
	{"result_view_reset", "result_view_reset", menuView, nil},
	{"cell_detail", "cell_detail", menuView, nil},
	{"complete", "completion", menuEdit, []binding{{"Ctrl+T", tuivory.KeyCtrlT, 0, 0}}},
	{"complete_refresh", "completion_refresh", menuEdit, nil},
	{"duplicate_line", "duplicate_line", menuEdit, nil},
	{"copy_line", "copy_line", menuEdit, nil},
	{"cut_line", "cut_line", menuEdit, nil},
	{"delete_line", "delete_line", menuEdit, nil},
	{"line_up", "line_up", menuEdit, nil},
	{"line_down", "line_down", menuEdit, nil},
	{"transpose_line", "transpose_line", menuEdit, nil},
	{"format_code", "format_code", menuEdit, []binding{{"Ctrl+Shift+F", tuivory.KeyCtrlF, tuivory.ModShift, 0}}},
	{"help", "keyboard_help", menuHelp, []binding{{"F1", tuivory.KeyF1, 0, 0}}},
	{"guide", "guide", menuHelp, nil},
	{"release_history", "release_history", menuHelp, nil},
	{"diagnostic_info", "diagnostic_info", menuHelp, nil},
	{"issue_report", "issue_report", menuHelp, nil},
	{"keymap_compare", "keymap_compare", menuHelp, nil},
	{"about", "about", menuHelp, nil},
}

func commandFor(id string) command {
	for _, cmd := range commands {
		if cmd.id == id {
			return cmd
		}
	}
	return command{}
}
func bindingLabels(id string, limit int) string {
	var labels []string
	for i, key := range commandFor(id).bindings {
		if i == limit {
			break
		}
		labels = append(labels, key.label)
	}
	return strings.Join(labels, " / ")
}

func shortcut(id string) string {
	cmd := commandFor(id)
	if len(cmd.bindings) > 0 {
		return cmd.bindings[0].label
	}
	return ""
}

func (s *screen) commandEnabled(id string) bool {
	page, _ := s.pages.GetFrontPage()
	if id == "workspaces" || id == "workspace_new" || id == "workspace_close" {
		return s.canSwitchWorkspace()
	}
	if page != "main" {
		return false
	}
	switch id {
	case "session_browser", "debug_start":
		return s.connected && !s.busy && !s.workerClosed && !s.transactionUncertain
	case "schema_compare", "data_compare":
		return !s.busy && len(s.compareConnections()) >= 2
	case "workspaces", "workspace_new", "workspace_close":
		return s.workspaces != nil && !s.busy && !s.vaultBusy
	case "table_edit_load":
		return !s.busy && s.connected && !s.transactionUncertain && s.editTableTarget().Kind == "TABLE"
	case "tab_rename":
		return !s.busy && s.document != nil && s.document.Path == "" && len(s.tabs) > 0
	case "resize_panels":
		return s.maximized == ""
	case "collapse_schema", "collapse_results":
		return s.maximized == "" && s.inlineEdit == nil && (id != "collapse_schema" || s.browser != nil)
	case "grid_insert":
		return s.canChangeGridRows() && s.resultBrowser.source.CanInsertRow()
	case "grid_delete":
		row, ok := s.selectedGridSourceRow()
		source := s.resultBrowser.source
		return s.canChangeGridRows() && ok && (row < 0 || source.EditID != 0 && source.EditCanDelete && row < len(source.EditRowIDs))
	case "grid_edit", "grid_inline":
		row, ok := s.selectedGridSourceRow()
		_, col := s.table.GetSelection()
		return !s.busy && s.connected && !s.transactionUncertain && s.table.HasFocus() && ok && (s.canEditGridCell(row, s.resultSourceColumn(col)) || id == "grid_edit" && row < 0 && s.resultBrowser.source.CanInsertRow())

	case "cell_detail":
		return !s.busy && s.table.HasFocus() && len(s.resultRows) > 0
	case "fetch_more":
		return !s.busy && s.connected && s.table.HasFocus() && s.resultBrowser.source.PageID != 0
	case "result_sort_asc", "result_sort_desc", "result_filter", "result_view_reset":
		return !s.busy && s.table.HasFocus() && len(s.resultHeaders) > 0
	case "grid_widen", "grid_narrow", "grid_auto":
		return !s.busy && s.table.HasFocus() && len(s.resultHeaders) > 0
	case "duplicate_line", "copy_line", "cut_line", "delete_line", "line_up", "line_down", "transpose_line":
		return !s.busy && s.editor.HasFocus()
	case "format_code":
		return !s.busy && s.editor.HasFocus()
	case "complete", "complete_refresh":
		return !s.busy && s.editor.HasFocus() && !s.editor.HasSelection()
	case "routine_execute":
		return s.connected && !s.busy && !s.transactionUncertain
	case "routine_results":
		return !s.busy && (len(s.routineResults.Scalars.Rows) > 0 || len(s.routineResults.Cursors) > 0)
	case "output_view":
		return true
	case "output_toggle":
		return s.connected && !s.busy && !s.transactionUncertain
	case "quit", "about", "keymap_compare", "help", "guide", "release_history", "diagnostic_info", "issue_report", "settings", "focus_toggle", "context_menu", "passwords":
		return true
	case "fullscreen_sql", "fullscreen_results":
		return true
	case "maximize":
		return s.editor.HasFocus() || s.resultAreaHasFocus()
	case "file_new", "file_open", "file_save", "file_save_as", "file_close", "tab_next", "tab_previous":
		return !s.busy
	case "find", "find_next", "find_previous", "replace", "clear_sql":
		return !s.busy && s.editor.HasFocus()
	case "undo", "redo", "paste", "select_all":
		return !s.busy && s.editor.HasFocus()
	case "cut":
		return !s.busy && s.editor.HasFocus() && (s.editor.HasSelection() || lineClipboardKeymap(s.prefs.EditorKeymap))
	case "browser_refresh":
		return !s.busy && s.browser != nil && (s.connected || s.demo)
	case "history", "favorites":
		return !s.busy
	case "favorite_save":
		return !s.busy && s.editor.HasFocus() && strings.TrimSpace(s.favoriteEditorSQL()) != ""
	case "pending_work":
		return !s.busy && s.hasPendingWork()
	case "browser", "browser_user", "browser_type", "browser_search":
		return s.browser != nil
	case "connect":
		return !s.busy
	case "execute", "script", "plan", "actual_plan_execute", "commit", "rollback":
		return s.connected && !s.busy
	case "actual_plan_last":
		return s.connected && !s.busy && s.lastActualPlan != nil
	case "script_report", "result_area_grid":
		return true
	case "cancel":
		return s.busy
	case "describe", "quick_describe":
		return !s.busy && (s.connected || s.demo)
	case "copy":
		return (s.table.HasFocus() && len(s.resultRows) > 0) || (s.editor.HasFocus() && (s.editor.HasSelection() || lineClipboardKeymap(s.prefs.EditorKeymap)))
	case "copy_row":
		return s.table.HasFocus() && len(s.resultRows) > 0
	case "export":
		return !s.busy && len(s.resultHeaders) > 0
	case "copy_preview":
		return s.clipboardText != ""
	}
	return false
}

func (s *screen) invoke(id string) {
	if !s.commandEnabled(id) {
		if id == "parse" {
			s.setStatus(commandFor(id).label)
		}
		if id == "execute" || id == "script" || id == "plan" || id == "commit" || id == "rollback" {
			if !s.connected {
				s.setStatus("connect_first")
			}
		}
		return
	}
	switch id {
	case "schema_compare":
		s.showSchemaCompare()
	case "data_compare":
		s.showComparison(true)
	case "session_browser":
		s.showSessionBrowser()
	case "debug_start":
		s.showDebugger()
	case "guide":
		s.showGuide()
	case "release_history":
		s.showReleaseHistory()
	case "diagnostic_info":
		s.showDiagnostics()
	case "issue_report":
		s.showIssueReport()
	case "workspaces":
		s.showWorkspaces()
	case "workspace_new":
		s.workspaces.addNew()
	case "workspace_close":
		s.workspaces.closeOne(s)
	case "keymap_compare":
		s.showKeymapComparison()
	case "about":
		s.showAbout()
	case "table_edit_load":
		s.loadEditTable()
	case "resize_panels":
		s.resizePanelsCommand()
	case "collapse_schema", "collapse_results":
		s.togglePanelCollapse(id == "collapse_schema")
	case "grid_insert":
		s.showGridInsert(0)
	case "grid_delete":
		s.toggleGridDelete()
	case "grid_inline":
		s.startInlineEdit(nil)
	case "grid_edit":
		s.showGridEdit()
	case "cell_detail":
		s.showCellDetail()
	case "fetch_more":
		s.fetchMoreResults()
	case "result_sort_asc":
		s.sortResult(false)
	case "result_sort_desc":
		s.sortResult(true)
	case "result_filter":
		s.showResultFilter()
	case "result_view_reset":
		s.resetResultView()
	case "grid_widen":
		s.resizeResultColumn(4)
	case "grid_narrow":
		s.resizeResultColumn(-4)
	case "grid_auto":
		s.resizeResultColumn(0)
	case "duplicate_line", "copy_line", "cut_line", "delete_line", "line_up", "line_down", "transpose_line":
		s.editorLineAction(s.editor, id)
	case "format_code":
		s.formatCode()
	case "complete":
		s.showCompletion(false)
	case "complete_refresh":
		s.showCompletion(true)
	case "routine_execute":
		s.requestRoutine()
	case "routine_results":
		s.showRoutineResults(s.routineResults)
	case "output_view":
		s.showOutput()
	case "output_toggle":
		s.requestOutputToggle()
	case "undo", "redo", "cut", "paste", "select_all":
		if id == "cut" && lineClipboardKeymap(s.prefs.EditorKeymap) && !s.editor.HasSelection() {
			s.editorLineAction(s.editor, "cut_line")
			return
		}
		// Dispatch directly to the editor to retain its selection and undo history.
		// Going through the global key handler would invoke this command again.
		s.editor.InputHandler()(tuivory.NewEventKey(commandFor(id).bindings[0].key, 0, 0), func(p tuivory.Primitive) { s.app.SetFocus(p) })
	case "browser_refresh":
		s.refreshBrowser()
	case "export":
		s.showExport()
	case "plan":
		s.explainSQL()
	case "actual_plan_execute":
		s.runSQL(true)
	case "actual_plan_last":
		s.showActualPlan()
	case "script":
		s.executeScript()
	case "script_report":
		s.showScriptReport()
	case "result_area_grid":
		s.selectResultSection("grid", true)
	case "tab_rename":
		s.showTabRename()
	case "file_new":
		s.newDocument()
	case "file_close":
		s.closeDocument()
	case "tab_next":
		s.cycleTab(1)
	case "tab_previous":
		s.cycleTab(-1)
	case "file_open":
		s.showFileDialog("open", nil)
	case "file_save":
		s.saveDocument(false, nil)
	case "file_save_as":
		s.saveDocument(true, nil)
	case "passwords":
		s.passwordManager()
	case "maximize":
		s.toggleMaximize()
	case "fullscreen_sql":
		s.togglePanelFullscreen("sql")
	case "fullscreen_results":
		s.togglePanelFullscreen("results")
	case "focus_toggle":
		s.toggleQueryResults()
	case "browser":
		s.toggleBrowser()
	case "browser_search":
		s.showBrowserSelector(false)
		s.app.SetFocus(s.browserSearch)
	case "browser_user", "browser_type":
		s.showBrowserSelector(id == "browser_type")
	case "context_menu":
		s.showContextMenu()
	case "find":
		s.showSearch(false)
	case "replace":
		s.showSearch(true)
	case "find_next":
		s.findSQL(false)
	case "find_previous":
		s.findSQL(true)
	case "clear_sql":
		s.clearSQL()
	case "history":
		s.showHistory()
	case "favorites":
		s.showFavorites()
	case "favorite_save":
		s.saveFavoriteSQL(s.favoriteEditorSQL())
	case "pending_work":
		s.showPendingWork()
	case "describe":
		s.describeObject()
	case "quick_describe":
		s.describe(true)
	case "copy_row":
		s.copyResults("row")
	case "copy":
		if s.table.HasFocus() {
			s.copyResults("selection")
		} else {
			text, _, _ := s.editor.GetSelection()
			if text == "" && lineClipboardKeymap(s.prefs.EditorKeymap) {
				s.editorLineAction(s.editor, "copy_line")
			} else {
				s.sendClipboard(text)
			}
		}
	case "copy_preview":
		s.copyPreview()
	default:
		s.action(id)
	}
}

func (s *screen) commandKey(event *tuivory.EventKey) bool {
	for _, cmd := range commands {
		for _, key := range cmd.bindings {
			// tcell normalizes C0 control keys to include ModCtrl.
			mod := event.Modifiers()
			if event.Key() >= tuivory.KeyCtrlA && event.Key() <= tuivory.KeyCtrlZ && event.Key() != tuivory.KeyEnter && event.Key() != tuivory.KeyTab {
				mod &^= tuivory.ModCtrl
			}
			if event.Key() == key.key && mod == key.mod && (key.key != tuivory.KeyRune || event.Rune() == key.char) {
				s.invoke(cmd.id)
				return true
			}
		}
	}
	return false
}

func (s *screen) menuBar() *tuivory.Flex {
	s.refreshMenuLabels()
	view, err := tuivory.NewMenuBarView(s.mainMenu, s.componentPalette(), func(e tuivory.MenuEvent) {
		if e.Kind != "open_request" {
			return
		}
		for i := range menuGroups {
			if e.ID == menuRoot(i) {
				s.showMenu(i)
				return
			}
		}
	})
	if err != nil {
		panic(err)
	}
	// Component hit testing/navigation is retained; menu-bar clicks act on
	// the existing panel rather than transferring application focus to the bar.
	mouse := view.Mouse
	view.Mouse = func(a tuivory.MouseAction, e *tuivory.EventMouse, _ func(tuivory.Primitive)) (bool, tuivory.Primitive) {
		return mouse(a, e, func(tuivory.Primitive) {})
	}

	s.menuBarView = view
	return s.newFlex().AddItem(view, 0, 1, false)
}
