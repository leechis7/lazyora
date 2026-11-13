package ui

import (
	"fmt"
	"lazyora/internal/appinfo"
	"strings"

	"github.com/leechis7/tuivory/go"
)

// tview may report a rapid click on a different button as a double click.
// Buttons should activate on either kind of click; text selection stays unchanged.
func (s *screen) clickable(button *tuivory.WidgetButton) *tuivory.WidgetButton {
	s.tuivoryWidgets().Button(button)
	button.SetMouseCapture(func(action tuivory.MouseAction, event *tuivory.EventMouse) (tuivory.MouseAction, *tuivory.EventMouse) {
		if action == tuivory.MouseLeftDoubleClick {
			action = tuivory.MouseLeftClick
		}
		return action, event
	})
	return button
}

type footerKey struct {
	label, action string
	key           tuivory.Key
	mod           tuivory.ModMask
}

func (s *screen) toolbar() *tuivory.Flex {
	bar := s.newFlex()
	s.footerBar = bar
	for index := range 6 {
		button := s.clickable(tuivory.NewWidgetButton("").SetSelectedFunc(func() {
			if index >= len(s.footerKeys) {
				return
			}
			key := s.footerKeys[index]
			if strings.HasPrefix(key.action, "resize_") {
				if page, _ := s.pages.GetFrontPage(); page == "main" {
					s.resizePanelKey(tuivory.NewEventKey(key.key, 0, key.mod))
				}
				return
			}
			if key.action == "connection_probe" && s.testConnection != nil {
				s.testConnection()
				return
			}
			if key.action == "passwords" {
				s.passwordManager()
				return
			}
			if page, _ := s.pages.GetFrontPage(); page == "exit" && key.action == "pending_work" {
				s.showPendingWork()
				return
			}
			if page, _ := s.pages.GetFrontPage(); page == "exit" && (key.action == "exit_commit" || key.action == "exit_rollback") {
				s.confirmExit(key.action)
				return
			}
			if page, _ := s.pages.GetFrontPage(); page == "main" && commandFor(key.action).id != "" {
				s.invoke(key.action)
				return
			}
			if page, _ := s.pages.GetFrontPage(); page == "source" && key.action == "reload" {
				s.reloadSource()
				return
			}
			if page, _ := s.pages.GetFrontPage(); page == "source" && key.action == "close" {
				s.closeSource()
				return
			}
			if key.action == "source" {
				s.showSource()
				return
			}
			if key.action == "arguments" {
				s.showArguments()
				return
			}
			if page, _ := s.pages.GetFrontPage(); page == "help" {
				s.app.SetFocus(s.helpText)
			} else if page == "object" {
				s.app.SetFocus(s.objectTable)
			} else if page == "arguments" {
				s.app.SetFocus(s.argumentTable)
			}
			s.app.QueueEvent(tuivory.NewEventKey(key.key, 0, key.mod))
		}))
		s.tuivoryWidgets().FooterButton(button)
		s.buttons = append(s.buttons, button)
	}
	version := s.newTextView().SetText("v" + appinfo.Version).SetTextAlign(tuivory.AlignRight).SetWrap(false)
	s.textRoles[version] = textBody
	s.textSurface(version, surfaceFooter)
	s.footerVersion = version
	return bar
}

func (s *screen) refreshFooter() {
	if s.footer == nil {
		return
	}
	page, _ := s.pages.GetFrontPage()
	if page == "main" && s.inlineEdit != nil {
		page = "grid_inline"
	}
	keys := []footerKey{
		{"F1", "help", tuivory.KeyF1, tuivory.ModNone},
		{"", "connect", 0, 0},
		{"", "settings", 0, 0},
		{"Ctrl+Q", "quit", tuivory.KeyCtrlQ, tuivory.ModNone},
	}
	hint := "footer_sql"
	switch page {
	case "debugger":
		hint = "debug_keys"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Esc", "debug_stop", tuivory.KeyEscape, 0}}
	case "workspaces":
		hint = "footer_workspaces"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Esc", "close", tuivory.KeyEscape, 0}}
	case "about":
		hint = "footer_about"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Ctrl+C", "copy", tuivory.KeyCtrlC, 0}, {"Esc", "close", tuivory.KeyEscape, 0}}
	case "reference":
		hint = "footer_reference"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Ctrl+C", "copy", tuivory.KeyCtrlC, 0}, {"Esc", "close", tuivory.KeyEscape, 0}}
	case "keymap_compare":
		hint = "comparison_keys"
		keys = []footerKey{{"Ctrl+C", "copy", tuivory.KeyCtrlC, 0}, {"Esc", "close", tuivory.KeyEscape, 0}}
	case "grid_inline":
		hint = "grid_inline_hint"
		keys = []footerKey{{"Enter", "edit_stage", tuivory.KeyEnter, 0}, {"Esc", "cancel", tuivory.KeyEscape, 0}}
	case "grid_edit":
		hint = "footer_grid_edit"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Ctrl+S", "edit_stage", tuivory.KeyCtrlS, 0}, {"Esc", "cancel", tuivory.KeyEscape, 0}}
	case "cell":
		hint = "footer_cell"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Ctrl+S", "save", tuivory.KeyCtrlS, 0}, {"Ctrl+C", "copy", tuivory.KeyCtrlC, 0}, {"Esc", "close", tuivory.KeyEscape, 0}}
	case "result_filter":
		hint = "footer_result_filter"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Esc", "close", tuivory.KeyEscape, 0}}
	case "definition":
		hint = "footer_definition"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Ctrl+R", "reload", tuivory.KeyCtrlR, 0}, {"Esc", "close", tuivory.KeyEscape, 0}}
	case "editor_actions":
		hint = "menu_hint"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Esc", "close", tuivory.KeyEscape, 0}}
	case "completion":
		hint = "footer_completion"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Ctrl+R", "completion_refresh", tuivory.KeyCtrlR, 0}, {"Esc", "cancel", tuivory.KeyEscape, 0}}
	case "export":
		hint = "footer_export"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Ctrl+S", "export", tuivory.KeyCtrlS, 0}, {"Esc", "cancel", tuivory.KeyEscape, 0}}
		if s.busy {
			hint = "footer_export_running"
			keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Esc", "cancel", tuivory.KeyEscape, 0}}
		}
	case "plan":
		hint = "footer_plan"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Esc", "close", tuivory.KeyEscape, 0}}
	case "routines", "routine_input", "routine_results", "routine_grid":
		hint = "footer_routine"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Esc", "close", tuivory.KeyEscape, 0}}
	case "bind":
		hint = "footer_bind"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Ctrl+Enter", "execute", tuivory.KeyEnter, tuivory.ModCtrl}, {"Esc", "cancel", tuivory.KeyEscape, 0}}
	case "main":
		keys = nil
		for _, id := range []string{"help", "connect", "commit", "rollback", "quit"} {
			key := binding{}
			if bindings := commandFor(id).bindings; len(bindings) > 0 {
				key = bindings[0]
			}
			keys = append(keys, footerKey{key.label, id, key.key, key.mod})
		}
		if s.resultAreaHasFocus() {
			s.resultsFocused = true
		} else if s.editor.HasFocus() {
			s.resultsFocused = false
		}
		if s.resultsFocused {
			hint = "footer_results"
			if s.resultArea != nil && s.resultArea.tab.HasFocus() {
				hint = "footer_result_tabs"
			} else if s.resultSection() == "output" {
				hint = "footer_output"
			} else if s.resultSection() == "messages" {
				hint = "footer_script"
			}
		}
		if s.browserHasFocus() {
			hint = "footer_browser"
		}
	case "search":
		hint = "footer_search"
		keys = []footerKey{{"Esc", "close", tuivory.KeyEscape, 0}}
	case "pending":
		hint = "footer_pending"
		keys = []footerKey{{"Esc", "close", tuivory.KeyEscape, 0}}
	case "history":
		hint = "footer_history"
		keys = []footerKey{{"Esc", "close", tuivory.KeyEscape, 0}}
	case "menu":
		hint = "menu_hint"
		keys = []footerKey{{"Esc", "close", tuivory.KeyEscape, 0}}
	case "copy":
		hint = "copy_preview_hint"
		keys = []footerKey{{"Esc", "close", tuivory.KeyEscape, 0}}
	case "describe":
		hint = "footer_describe"
		keys = []footerKey{{"Esc", "close", tuivory.KeyEscape, 0}}
	case "file":
		hint = "footer_file"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Esc", "cancel", tuivory.KeyEscape, 0}}
	case "file_unsaved":
		hint = "footer_file_unsaved"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Esc", "cancel", tuivory.KeyEscape, 0}}
	case "sql_confirm":
		hint = "footer_sql_confirm"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Esc", "cancel", tuivory.KeyEscape, 0}}
	case "master":
		hint = "footer_master"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Esc", "cancel", tuivory.KeyEscape, 0}}
	case "vault":
		hint = "footer_vault"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, 0}, {"Esc", "close", tuivory.KeyEscape, 0}}
	case "connect":
		hint = "footer_connect"
		keys[1] = footerKey{"Ctrl+Enter", "connect", tuivory.KeyEnter, tuivory.ModCtrl}
		keys[2] = footerKey{"Ctrl+S", "save", tuivory.KeyCtrlS, tuivory.ModNone}
		keys[3] = footerKey{"Esc", "cancel", tuivory.KeyEscape, tuivory.ModNone}
		keys = append(keys, footerKey{"Alt+K", "passwords", tuivory.KeyRune, tuivory.ModAlt}, footerKey{"Alt+V", "connection_probe", tuivory.KeyRune, tuivory.ModAlt})
	case "reconnect":
		hint = "footer_reconnect"
		keys = []footerKey{{"Esc", "cancel", tuivory.KeyEscape, 0}}
	case "settings":
		hint = "footer_settings"
		keys[1] = footerKey{"Ctrl+S", "save", tuivory.KeyCtrlS, tuivory.ModNone}
		keys[2] = footerKey{"Esc", "cancel", tuivory.KeyEscape, tuivory.ModNone}
		keys = keys[:3]
	case "help":
		hint = "footer_help"
		keys[0].action = "close"
		keys[1] = footerKey{"PgUp", "previous_page", tuivory.KeyPgUp, tuivory.ModNone}
		keys[2] = footerKey{"PgDn", "next_page", tuivory.KeyPgDn, tuivory.ModNone}
		keys[3] = footerKey{"Esc", "close", tuivory.KeyEscape, tuivory.ModNone}
	case "object":
		hint = "footer_object"
		if s.objectWorkspace != nil {
			hint = "footer_object_workspace"
		}
		keys[0].action = "help"
		keys[1] = footerKey{"PgUp", "previous_page", tuivory.KeyPgUp, tuivory.ModNone}
		keys[2] = footerKey{"PgDn", "next_page", tuivory.KeyPgDn, tuivory.ModNone}
		keys[3] = footerKey{"Esc", "close", tuivory.KeyEscape, tuivory.ModNone}
		if s.object.HasSource() {
			keys[1] = footerKey{"Alt+S", "source", tuivory.KeyRune, tuivory.ModAlt}
		}
		if s.object.HasArguments() {
			keys[2] = footerKey{"Alt+A", "arguments", tuivory.KeyRune, tuivory.ModAlt}
		}
	case "source":
		hint = "footer_source"
		keys = []footerKey{{"F9", "compile", tuivory.KeyF9, 0}, {"Alt+R", "reload", tuivory.KeyRune, tuivory.ModAlt}, {"Alt+X", "close", tuivory.KeyRune, tuivory.ModAlt}}
	case "arguments":
		hint = "footer_arguments"
		keys = []footerKey{{"F1", "help", tuivory.KeyF1, tuivory.ModNone}, {"PgUp", "previous_page", tuivory.KeyPgUp, tuivory.ModNone}, {"PgDn", "next_page", tuivory.KeyPgDn, tuivory.ModNone}, {"Esc", "close", tuivory.KeyEscape, tuivory.ModNone}}
	case "program_confirm":
		hint = "footer_confirm"
		keys = []footerKey{{"Esc", "cancel", tuivory.KeyEscape, tuivory.ModNone}}
	case "exit":
		hint = "footer_exit"
		keys = []footerKey{{"Esc", "exit_cancel", tuivory.KeyEscape, tuivory.ModNone}}
		if s.hasPendingWork() {
			keys = append(keys, footerKey{"Alt+V", "pending_work", tuivory.KeyRune, tuivory.ModAlt})
			if !s.transactionUncertain {
				keys = append(keys, footerKey{shortcut("commit"), "exit_commit", tuivory.KeyRune, tuivory.ModAlt}, footerKey{shortcut("rollback"), "exit_rollback", tuivory.KeyRune, tuivory.ModAlt})
			}
		}
	}
	if page == "main" && s.sizing.keyboard {
		hint = "resize_panels_hint"
		keys = []footerKey{{"Home", "resize_reset", tuivory.KeyHome, 0}, {"Enter", "resize_finish", tuivory.KeyEnter, 0}, {"Esc", "resize_end", tuivory.KeyEscape, 0}}
	}
	text := s.tr(hint)
	if hint == "footer_source" {
		text = s.sourceKeymapFooter()
	}
	viewKeys := shortcut("fullscreen_sql") + " / " + shortcut("fullscreen_results") + " " + s.tr("fullscreen_short")
	if hint == "footer_sql" {
		text = fmt.Sprintf(text, bindingLabels("execute", 2), shortcut("describe"), shortcut("focus_toggle"), viewKeys)
	}
	if hint == "footer_sql" && s.activeEditor() != nil {
		if lineClipboardKeymap(s.prefs.EditorKeymap) {
			text = strings.ReplaceAll(text, "Ctrl+T", "Ctrl+Space")
		}
		if ideKeymap(s.prefs.EditorKeymap) {
			text = s.keymapHint() + " | " + text
		}
		if s.prefs.EditorKeymap == "vim" {
			text = "Vim " + s.vimMode() + " | " + text
		}
		if s.prefs.EditorKeymap == "notepad++" {
			text = "Notepad++ | " + text
		}
	}
	if hint == "footer_results" {
		text = s.resultNavigationHint(viewKeys)
	}
	if page == "main" {
		if s.maximized != "" {
			if s.fullscreen {
				text += " | " + s.tr("fullscreen_short")
			} else {
				text += " | " + s.tr("maximized")
			}
		}

	}
	if s.busy && page != "help" && !(page == "main" && s.resultSection() == "output" && s.resultAreaHasFocus()) && page != "reference" {
		text = s.tr("footer_busy") + " | " + text
	}
	s.footer.SetText(text)
	s.footerKeys = keys
	s.footerBar.Clear()
	for i, key := range keys {
		label := s.tr(key.action)
		if key.label != "" {
			label = key.label + " " + label
		}
		s.buttons[i].SetLabel(label)
		s.footerBar.AddItem(s.buttons[i], 0, 1, false)
	}
	s.footerBar.AddItem(s.footerVersion, len(appinfo.Version)+3, 0, false)
}

func (s *screen) resultNavigationHint(viewKeys string) string {
	text := fmt.Sprintf(s.tr("footer_results"), shortcut("copy"), shortcut("copy_row"), shortcut("focus_toggle"), viewKeys)
	if s.resultBrowser.source.EditID != 0 {
		text = shortcut("grid_edit") + " " + s.tr("grid_edit_short") + " | " + text
	}
	if s.resultBrowser.source.CanInsertRow() {
		text = shortcut("grid_insert") + " + | " + text
	}
	if s.resultBrowser.source.EditCanDelete && s.resultBrowser.source.EditID != 0 || len(s.gridInserts) != 0 {
		text = shortcut("grid_delete") + " - | " + text
	}
	if !s.fullscreen {
		text += " | " + s.tr("result_area_tab_hint")
	}
	return text
}

func (s *screen) action(action string) {
	switch action {
	case "help":
		s.showHelp()
	case "connect":
		if !s.busy {
			s.connectionForm()
		}
	case "settings":
		s.settingsForm()
	case "execute":
		s.executeSQL()
	case "commit", "rollback":
		s.submit(job{action: action})
	case "cancel":
		if s.cancel != nil {
			s.cancel()
		}
	case "quit":
		s.requestQuit()
	}
}

func clickable(button *tuivory.WidgetButton) *tuivory.WidgetButton {
	return (&screen{}).clickable(button)
}
