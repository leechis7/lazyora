package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
)

type binding struct {
	label string
	key   tcell.Key
	mod   tcell.ModMask
	char  rune
}
type command struct {
	id, label string
	group     int
	bindings  []binding
}

var commands = []command{
	{"file_new", "file_new", menuTabs, []binding{{"Ctrl+N", tcell.KeyCtrlN, 0, 0}}},
	{"file_open", "file_open", menuFile, []binding{{"Ctrl+O", tcell.KeyCtrlO, 0, 0}}},
	{"file_save", "file_save", menuFile, []binding{{"Ctrl+S", tcell.KeyCtrlS, 0, 0}}},
	{"file_save_as", "file_save_as", menuFile, []binding{{"Ctrl+Shift+S", tcell.KeyCtrlS, tcell.ModShift, 0}}},
	{"file_close", "file_close", menuTabs, []binding{{"Ctrl+W", tcell.KeyCtrlW, 0, 0}}},
	{"tab_next", "tab_next", menuTabs, []binding{{"Ctrl+Tab", tcell.KeyTab, tcell.ModCtrl, 0}, {"Alt+PgDn", tcell.KeyPgDn, tcell.ModAlt, 0}}},
	{"tab_previous", "tab_previous", menuTabs, []binding{{"Ctrl+Shift+Tab", tcell.KeyBacktab, tcell.ModCtrl, 0}, {"Alt+PgUp", tcell.KeyPgUp, tcell.ModAlt, 0}}},
	{"quit", "quit", menuFile, []binding{{"Ctrl+Q", tcell.KeyCtrlQ, 0, 0}}},
	{"undo", "undo", menuEdit, []binding{{"Ctrl+Z", tcell.KeyCtrlZ, 0, 0}}},
	{"redo", "redo", menuEdit, []binding{{"Ctrl+Y", tcell.KeyCtrlY, 0, 0}}},
	{"cut", "cut", menuEdit, []binding{{"Ctrl+X", tcell.KeyCtrlX, 0, 0}}},
	{"paste", "paste", menuEdit, []binding{{"Ctrl+V", tcell.KeyCtrlV, 0, 0}}},
	{"select_all", "select_all", menuEdit, []binding{{"Ctrl+L", tcell.KeyCtrlL, 0, 0}}},
	{"copy", "copy_selection", menuEdit, []binding{{"Ctrl+C", tcell.KeyCtrlC, 0, 0}}},
	{"copy_row", "copy_row", menuEdit, []binding{{"Alt+Y", tcell.KeyRune, tcell.ModAlt, 'y'}}},
	{"export", "export", menuEdit, []binding{{"Ctrl+Shift+E", tcell.KeyCtrlE, tcell.ModShift, 0}, {"Alt+W", tcell.KeyRune, tcell.ModAlt, 'w'}}},
	{"copy_preview", "copy_preview", menuEdit, nil},
	{"find", "find", menuEdit, []binding{{"Ctrl+F", tcell.KeyCtrlF, 0, 0}}},
	{"find_next", "find_next", menuEdit, []binding{{"F3", tcell.KeyF3, 0, 0}}},
	{"find_previous", "find_previous", menuEdit, []binding{{"Shift+F3", tcell.KeyF3, tcell.ModShift, 0}}},
	{"replace", "replace", menuEdit, []binding{{"Ctrl+R", tcell.KeyCtrlR, 0, 0}}},
	{"clear_sql", "clear_sql", menuEdit, []binding{{"F7", tcell.KeyF7, 0, 0}}},
	{"connect", "connect", menuSession, []binding{{"Alt+L", tcell.KeyRune, tcell.ModAlt, 'l'}}},
	{"commit", "commit", menuSession, []binding{{"Alt+C", tcell.KeyRune, tcell.ModAlt, 'c'}}},
	{"rollback", "rollback", menuSession, []binding{{"Alt+R", tcell.KeyRune, tcell.ModAlt, 'r'}}},
	{"pending_work", "pending_work", menuSession, nil},
	{"output_toggle", "output_toggle", menuSession, nil},
	{"execute", "execute", menuExecute, []binding{{"F9", tcell.KeyF9, 0, 0}, {"Ctrl+Enter", tcell.KeyEnter, tcell.ModCtrl, 0}, {"Alt+Enter", tcell.KeyEnter, tcell.ModAlt, 0}, {"Shift+F9", tcell.KeyF9, tcell.ModShift, 0}}},
	{"script", "script", menuExecute, []binding{{"F5", tcell.KeyF5, 0, 0}}},
	{"plan", "plan", menuExecute, []binding{{"Ctrl+E", tcell.KeyCtrlE, 0, 0}}},
	{"script_report", "script_report", menuView, nil},
	{"parse", "parse_unavailable", menuExecute, []binding{{"Ctrl+F9", tcell.KeyF9, tcell.ModCtrl, 0}}},
	{"debug_continue", "debug_unavailable", menuExecute, []binding{{"F11", tcell.KeyF11, 0, 0}}},
	{"cancel", "cancel_execution", menuExecute, []binding{{"Esc", tcell.KeyEscape, 0, 0}}},
	{"history", "history", menuView, []binding{{"F8", tcell.KeyF8, 0, 0}}},
	{"output_view", "output_view", menuView, []binding{{"Alt+P", tcell.KeyRune, tcell.ModAlt, 'p'}}},
	{"fullscreen_sql", "fullscreen_sql", menuView, []binding{{"F2", tcell.KeyF2, 0, 0}}},
	{"fullscreen_results", "fullscreen_results", menuView, []binding{{"Shift+F2", tcell.KeyF2, tcell.ModShift, 0}}},
	{"focus_toggle", "focus_toggle", menuView, []binding{{"F6", tcell.KeyF6, 0, 0}}},
	{"context_menu", "context_menu", menuView, []binding{{"F10", tcell.KeyF10, 0, 0}}},
	{"maximize", "maximize", menuView, nil},
	{"browser", "schemas", menuView, []binding{{"Alt+B", tcell.KeyRune, tcell.ModAlt, 'b'}}},
	{"browser_user", "browser_user", menuView, []binding{{"Alt+U", tcell.KeyRune, tcell.ModAlt, 'u'}}},
	{"browser_type", "browser_type", menuView, []binding{{"Alt+O", tcell.KeyRune, tcell.ModAlt, 'o'}}},
	{"browser_refresh", "browser_refresh", menuView, nil},
	{"describe", "describe", menuView, []binding{{"F4", tcell.KeyF4, 0, 0}}},
	{"quick_describe", "quick_describe", menuView, []binding{{"Ctrl+D", tcell.KeyCtrlD, 0, 0}}},
	{"settings", "settings", menuTools, nil},
	{"passwords", "passwords", menuTools, []binding{{"Alt+K", tcell.KeyRune, tcell.ModAlt, 'k'}}},
	{"routine_execute", "routine_execute", menuExecute, nil},
	{"routine_results", "routine_results", menuView, nil},
	{"complete", "completion", menuEdit, []binding{{"Ctrl+T", tcell.KeyCtrlT, 0, 0}}},
	{"complete_refresh", "completion_refresh", menuEdit, nil},
	{"help", "help", menuHelp, []binding{{"F1", tcell.KeyF1, 0, 0}}},
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
	if page != "main" {
		return false
	}
	switch id {
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
	case "quit", "help", "settings", "focus_toggle", "context_menu", "passwords":
		return true
	case "fullscreen_sql", "fullscreen_results":
		return true
	case "maximize":
		return s.editor.HasFocus() || s.table.HasFocus()
	case "file_new", "file_open", "file_save", "file_save_as", "file_close", "tab_next", "tab_previous":
		return !s.busy
	case "find", "find_next", "find_previous", "replace", "clear_sql":
		return !s.busy && s.editor.HasFocus()
	case "undo", "redo", "paste", "select_all":
		return !s.busy && s.editor.HasFocus()
	case "cut":
		return !s.busy && s.editor.HasFocus() && s.editor.HasSelection()
	case "browser_refresh":
		return !s.busy && s.browser != nil && (s.connected || s.demo)
	case "history":
		return !s.busy
	case "pending_work":
		return !s.busy && s.pendingChanges
	case "browser", "browser_user", "browser_type":
		return s.browser != nil
	case "connect":
		return !s.busy
	case "execute", "script", "plan", "commit", "rollback":
		return s.connected && !s.busy
	case "script_report":
		return !s.busy && len(s.scriptReport.Steps) > 0
	case "cancel":
		return s.busy
	case "describe", "quick_describe":
		return !s.busy && (s.connected || s.demo)
	case "copy":
		return (s.table.HasFocus() && len(s.resultRows) > 0) || (s.editor.HasFocus() && s.editor.HasSelection())
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
		if id == "parse" || id == "debug_continue" {
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
		// Dispatch directly to the editor to retain its selection and undo history.
		// Going through the global key handler would invoke this command again.
		s.editor.InputHandler()(tcell.NewEventKey(commandFor(id).bindings[0].key, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	case "browser_refresh":
		s.refreshBrowser()
	case "export":
		s.showExport()
	case "plan":
		s.explainSQL()
	case "script":
		s.executeScript()
	case "script_report":
		s.showScriptReport()
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
			s.sendClipboard(text)
		}
	case "copy_preview":
		s.copyPreview()
	default:
		s.action(id)
	}
}

func (s *screen) commandKey(event *tcell.EventKey) bool {
	for _, cmd := range commands {
		for _, key := range cmd.bindings {
			// tcell normalizes C0 control keys to include ModCtrl.
			mod := event.Modifiers()
			if event.Key() >= tcell.KeyCtrlA && event.Key() <= tcell.KeyCtrlZ && event.Key() != tcell.KeyEnter && event.Key() != tcell.KeyTab {
				mod &^= tcell.ModCtrl
			}
			if event.Key() == key.key && mod == key.mod && (key.key != tcell.KeyRune || event.Rune() == key.char) {
				s.invoke(cmd.id)
				return true
			}
		}
	}
	return false
}

func (s *screen) menuBar() *tview.Flex {
	bar := tview.NewFlex()
	for index := range menuGroups {
		button := clickable(tview.NewButton("").SetSelectedFunc(func() { s.showMenu(index) }))
		button.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
			// Menu clicks act on the current panel; do not transfer focus to the bar.
			if action == tview.MouseLeftDown {
				return tview.MouseConsumed, nil
			}
			if action == tview.MouseLeftDoubleClick {
				action = tview.MouseLeftClick
			}
			return action, event
		})
		s.menuButtons = append(s.menuButtons, button)
		bar.AddItem(button, 0, 1, false)
	}
	s.refreshMenuLabels()
	return bar
}
