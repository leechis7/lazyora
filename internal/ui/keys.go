package ui

import (
	"unicode"

	"github.com/leechis7/tuivory/go"
)

func focusForm(s *screen, form *tuivory.WidgetForm, index int) {
	form.SetFocus(index)
	s.app.SetFocus(form)
}

func (s *screen) key(event *tuivory.EventKey) *tuivory.EventKey {
	// Legacy terminal mappings report Shift+F1..F12 as F13..F24. Keep those
	// aliases equivalent to explicit modifiers; preserve modified higher keys.
	if event.Key() >= tuivory.KeyF13 && event.Key() <= tuivory.KeyF24 && event.Modifiers() == 0 {
		event = tuivory.NewEventKey(tuivory.KeyF1+event.Key()-tuivory.KeyF13, 0, tuivory.ModShift)
	}
	// tview treats an unchanged Ctrl+C event as an immediate application stop.
	// Forward a copy for text widgets so dialogs cannot bypass the SQL-file
	// and DB exit confirmations; explicit exit continues to use Ctrl+Q.
	if event.Key() == tuivory.KeyCtrlC {
		event = tuivory.NewEventKey(event.Key(), event.Rune(), event.Modifiers())
	}
	page, _ := s.pages.GetFrontPage()
	if page == "keymap_compare" {
		return s.keymapComparisonKey(event)
	}
	if page == "main" && s.inlineEdit != nil {
		return s.inlineEditKey(event)
	}
	if s.vaultBusy {
		return nil
	}
	if s.workspaces != nil && event.Key() == tuivory.KeyRune && event.Modifiers()&tuivory.ModAlt != 0 && unicode.ToLower(event.Rune()) == 'j' {
		s.showWorkspaces()
		return nil
	}
	// tcell reports legacy ESC CR as Ctrl+Alt+m rather than KeyEnter+Alt.
	if event.Key() == tuivory.KeyRune && event.Rune() == 'm' && event.Modifiers()&(tuivory.ModAlt|tuivory.ModCtrl) == tuivory.ModAlt|tuivory.ModCtrl {
		event = tuivory.NewEventKey(tuivory.KeyEnter, 0, event.Modifiers()&^tuivory.ModCtrl)
	}

	page, _ = s.pages.GetFrontPage()
	if page == "main" && s.resizePanelKey(event) {
		return nil
	}
	if page == "favorite_edit" && s.favoriteEditKey != nil {
		return s.favoriteEditKey(event)
	}
	if page == "tab_rename" {
		return event
	}
	if page == "favorites" && s.favoritesKey != nil {
		return s.favoritesKey(event)
	}
	if page == "reference" {
		return s.referenceKey(event)
	}
	if page == "workspaces" {
		return s.workspaceKey(event)
	}
	if page == "about" {
		return s.aboutKey(event)
	}
	if page == "grid_edit" {
		return s.gridEditKey(event)
	}
	if page == "cell" {
		return s.cellKey(event)
	}
	if page == "main" && s.table.HasFocus() && event.Key() == tuivory.KeyEnter && event.Modifiers() == 0 {
		s.editSelectedResultCell()
		return nil
	}
	if page == "result_filter" {
		if event.Key() == tuivory.KeyEscape {
			s.pages.RemovePage("result_filter")
			s.app.SetFocus(s.table)
			return nil
		}
		if event.Key() == tuivory.KeyF1 {
			s.showHelp()
			return nil
		}
		return event
	}
	executeEnter := event.Key() == tuivory.KeyEnter && event.Modifiers()&(tuivory.ModCtrl|tuivory.ModAlt) != 0
	alt := event.Modifiers()&tuivory.ModAlt != 0
	letter := unicode.ToLower(event.Rune())
	if event.Key() == tuivory.KeyCtrlQ {
		s.action("quit")
		return nil
	}
	if page == "reconnect" {
		if event.Key() == tuivory.KeyEscape {
			s.cancelReconnect()
			return nil
		}
		// Modified Enter is the connection shortcut, never implicit approval.
		if executeEnter || event.Key() == tuivory.KeyCtrlC || event.Key() == tuivory.KeyF1 {
			return nil
		}
		return event
	}
	if page == "export" && s.busy {
		return s.exportKey(event)
	}
	if s.buttonArrow(event) {
		return nil
	}
	if page == "definition" {
		return s.definitionKey(event)
	}
	if page == "editor_actions" {
		if event.Key() == tuivory.KeyEscape || event.Key() == tuivory.KeyF10 && event.Modifiers() == tuivory.ModShift {
			s.closeSourceEdit()
			return nil
		}
		if event.Key() == tuivory.KeyF1 {
			s.showHelp()
			return nil
		}
		if event.Key() == tuivory.KeyCtrlC {
			return nil
		}
		return event
	}
	if page == "completion" {
		return s.completionKey(event)
	}
	if page == "output" {
		return s.outputKey(event)
	}
	if page == "script" {
		return s.scriptKey(event)
	}
	if page == "routines" || page == "routine_input" || page == "routine_results" || page == "routine_grid" {
		return s.routineKey(event, page)
	}
	if page == "bind" {
		return s.bindKey(event)
	}
	if page == "plan" {
		return s.planKey(event)
	}
	if page == "export" {
		return s.exportKey(event)
	}
	if page == "file" {
		return s.fileKey(event)
	}
	if page == "file_unsaved" {
		if event.Key() == tuivory.KeyCtrlC {
			return nil
		}
		if event.Key() == tuivory.KeyF1 {
			s.showHelp()
			return nil
		}
		return event
	}
	if page == "sql_confirm" {
		return s.sqlConfirmationKey(event)
	}
	if page == "master" || page == "vault" {
		return s.vaultKey(event, page)
	}
	if s.editorPresetKey(s.activeEditor(), event) {
		return nil
	}
	if event.Key() == tuivory.KeyTab || event.Key() == tuivory.KeyBacktab {
		switch page {
		case "help":
			s.dialogTab(event, s.helpTopics, s.helpText)
			return nil
		case "object":
			s.dialogTab(event, s.objectTable)
			return nil
		case "source":
			s.dialogTab(event, s.sourceEditor, s.sourceDiagnostics)
			return nil
		case "pending":
			s.dialogTab(event, s.pendingTable, s.pendingSQL)
			return nil
		case "arguments":
			s.dialogTab(event, s.argumentTable)
			return nil
		}
	}
	if page == "main" || page == "menu" {
		if page == "main" && event.Key() == tuivory.KeyF10 && event.Modifiers() == 0 {
			s.showMenu(menuFile)
			return nil
		}
		if alt {
			if letter == 'm' {
				s.showMenu(0)
				return nil
			}
			for index, group := range menuGroups {
				if letter == group.key {
					s.showMenu(index)
					return nil
				}
			}
		}
	}
	if page == "menu" {
		return s.menuKey(event)
	}
	if (page == "copy" || page == "describe") && event.Key() == tuivory.KeyF1 {
		s.showHelp()
		return nil
	}
	if page == "pending" && event.Key() != tuivory.KeyF1 {
		if event.Key() == tuivory.KeyEscape {
			s.closePendingWork()
			return nil
		}
		if event.Key() == tuivory.KeyCtrlC {
			s.sendClipboard(s.pendingSQL.GetText(false))
			return nil
		}
		return event
	}
	if page == "copy" {
		if event.Key() == tuivory.KeyEscape {
			s.closeCopyPreview()
			return nil
		}
		return event
	}
	if page == "describe" {
		if event.Key() == tuivory.KeyEnter && s.describeForm.GetFormItem(0).HasFocus() {
			s.describeAction()
			return nil
		}
		if event.Key() == tuivory.KeyEscape {
			s.closeDescribe()
			return nil
		}
		return event
	}
	if page == "search" {
		if event.Key() == tuivory.KeyEscape {
			s.closeSearch()
			return nil
		}
		if event.Key() == tuivory.KeyEnter && s.searchForm.GetFormItem(0).HasFocus() {
			s.searchAction("submit")
			return nil
		}
		if event.Key() == tuivory.KeyF3 {
			action := "next"
			if event.Modifiers()&tuivory.ModShift != 0 {
				action = "previous"
			}
			s.searchAction(action)
			return nil
		}
		return event
	}
	if page == "history" {
		if event.Key() == tuivory.KeyRune && event.Modifiers() == tuivory.ModAlt {
			switch unicode.ToLower(event.Rune()) {
			case 's':
				index := len(s.sqlHistory) - 1 - s.historyList.GetCurrentItem()
				if index >= 0 && index < len(s.sqlHistory) {
					s.saveFavoriteSQL(s.sqlHistory[index])
				}
				return nil
			case 'f':
				s.closeHistory()
				s.showFavorites()
				return nil
			}
		}
		if event.Key() == tuivory.KeyEscape || event.Key() == tuivory.KeyF8 {
			s.closeHistory()
			return nil
		}
		return event
	}
	if page == "main" && s.browserHasFocus() && (s.browserUsers.IsOpen() || s.browserTypes.IsOpen()) && event.Key() == tuivory.KeyEscape {
		return event
	}
	// Help owns its close keys even while a background operation is running.
	// Closing the overlay must never be mistaken for cancelling that operation.
	if page == "help" {
		if alt && letter == 'k' {
			s.showKeymapComparison()
			return nil
		}
		if s.listDetailArrow(event, s.helpTopics, s.helpText) {
			return nil
		}
		if event.Key() == tuivory.KeyRune && event.Modifiers() == 0 && event.Rune() >= '0' && event.Rune() <= '9' {
			s.helpJump(int(event.Rune() - '0'))
			return nil
		}
		if event.Key() == tuivory.KeyEscape || event.Key() == tuivory.KeyF1 || (alt && letter == 'x') {
			s.closeHelp()
			return nil
		}
		return event
	}
	if s.busy && event.Key() == tuivory.KeyEscape && page != "exit" && page != "program_confirm" {
		s.action("cancel")
		return nil
	}
	// Dialogs and running queries own Escape first. On the idle main page,
	// restore fullscreen before editor keymaps or result-range clearing handle it.
	if page == "main" && event.Key() == tuivory.KeyEscape && event.Modifiers() == 0 && s.fullscreen && s.maximized != "" {
		s.toggleFullscreen()
		return nil
	}
	if page == "exit" {
		return s.exitKey(event)
	}
	if page == "program_confirm" {
		if event.Key() == tuivory.KeyCtrlC {
			return nil
		}
		return event
	}
	if event.Key() == tuivory.KeyF1 {
		s.showHelp()
		return nil
	}
	if s.editorKey(event) {
		return nil
	}
	if page == "object" {
		if alt {
			switch letter {
			case 'g':
				s.showDefinition("grants")
				return nil
			case 'd':
				s.showDefinition("ddl")
				return nil
			case 'c':
				s.showDefinition("constraints")
				return nil
			case 'i':
				s.showDefinition("indexes")
				return nil
			}
		}
		if alt && letter == 'e' {
			s.showRoutine(s.object, "")
			return nil
		}
		if alt && letter == 's' {
			s.showSource()
			return nil
		}
		if alt && letter == 'a' {
			s.showArguments()
			return nil
		}
		switch event.Key() {
		case tuivory.KeyEscape:
			s.closeObject()
			return nil
		case tuivory.KeyCtrlR:
			if !s.busy {
				s.openObject(s.object)
			}
			return nil
		}
		if alt && letter == 'x' {
			s.closeObject()
			return nil
		}
		return event
	}
	if page == "source" {
		if event.Key() == tuivory.KeyF10 && event.Modifiers() == tuivory.ModShift {
			s.showSourceEdit()
			return nil
		}
		if alt && letter == 'x' {
			s.closeSource()
			return nil
		}
		if alt && letter == 'r' {
			s.reloadSource()
			return nil
		}
		if event.Key() == tuivory.KeyCtrlF && event.Modifiers()&tuivory.ModShift != 0 {
			s.formatCode()
			return nil
		}
		if event.Key() == tuivory.KeyCtrlT {
			s.showCompletion(false)
			return nil
		}
		if event.Key() == tuivory.KeyEscape {
			s.closeSource()
			return nil
		}
		if event.Key() == tuivory.KeyCtrlR {
			s.reloadSource()
			return nil
		}
		if event.Key() == tuivory.KeyF9 && event.Modifiers() == tuivory.ModNone {
			s.requestCompile()
			return nil
		}
		if alt && letter == 'a' {
			s.showArguments()
			return nil
		}
		return event
	}
	if page == "arguments" {
		if alt && letter == 'e' {
			s.showRoutine(s.object, "")
			return nil
		}
		if event.Key() == tuivory.KeyEscape {
			s.closeArguments()
			return nil
		}
		return event
	}
	if page == "settings" {
		if event.Key() == tuivory.KeyEscape || (alt && letter == 'x') {
			s.closeSettings()
			return nil
		}
		if event.Key() == tuivory.KeyCtrlS || (alt && letter == 's') {
			s.savePrefs()
			return nil
		}
		if alt {
			switch letter {
			case 'l':
				focusForm(s, s.prefsForm, 0)
				return nil
			case 'm':
				focusForm(s, s.prefsForm, 1)
				return nil
			case 'k':
				focusForm(s, s.prefsForm, 2)
				return nil
			case 't':
				focusForm(s, s.prefsForm, 3)
				return nil
			case 'd':
				focusForm(s, s.prefsForm, 4)
				return nil
			}
		}
		return event
	}
	if page == "connect" {
		if s.testingConnection {
			if event.Key() == tuivory.KeyEscape || alt && letter == 'x' {
				s.action("cancel")
			}
			return nil
		}
		if alt && letter == 'v' {
			s.testConnection()
			return nil
		}
		if alt && letter == 'k' {
			s.passwordManager()
			return nil
		}
		if s.connectionTab(event) {
			return nil
		}
		if event.Key() == tuivory.KeyCtrlS || (alt && letter == 's') {
			s.saveConnection()
			return nil
		}
		if event.Key() == tuivory.KeyEscape || (alt && letter == 'x') {
			s.closeConnection()
			return nil
		}
		if executeEnter || (alt && letter == 'c') {
			s.connectAction()
			return nil
		}
		if alt {
			if letter == 'e' {
				focusForm(s, s.connectForm, connectionEnvironment)
				return nil
			}
			if letter == 'r' {
				focusForm(s, s.connectProfiles, 0)
				return nil
			}
			for i, key := range []rune{'a', 'h', 'p', 'u', 'w', 't', 'n'} {
				if letter == key {
					focusForm(s, s.connectForm, i)
					return nil
				}
			}
		}
		return event
	}
	if page != "main" {
		return event
	}
	if event.Modifiers()&tuivory.ModCtrl != 0 && (event.Key() == tuivory.KeyTab || event.Key() == tuivory.KeyBacktab) {
		step := 1
		if event.Key() == tuivory.KeyBacktab || event.Modifiers()&tuivory.ModShift != 0 {
			step = -1
		}
		s.cycleTab(step)
		return nil
	}
	if s.browserHasFocus() {
		if (s.browserUsers.IsOpen() || s.browserTypes.IsOpen()) && event.Modifiers()&(tuivory.ModCtrl|tuivory.ModAlt) == 0 {
			return event
		}
		if event.Key() == tuivory.KeyCtrlR {
			s.refreshBrowser()
			return nil
		}
	}

	if page == "main" && s.table.HasFocus() && event.Key() == tuivory.KeyRune && event.Modifiers() & ^tuivory.ModShift == 0 {
		r := event.Rune()
		if s.startInlineEdit(&r) {
			return nil
		}
	}
	if s.table.HasFocus() {
		switch event.Key() {
		case tuivory.KeyUp, tuivory.KeyDown, tuivory.KeyLeft, tuivory.KeyRight:
			if event.Modifiers()&tuivory.ModShift != 0 {
				s.extendResultSelection(event.Key())
				return nil
			}
			s.clearResultSelection()
		case tuivory.KeyEscape:
			s.clearResultSelection()
			return nil
		}
	}
	if s.commandKey(event) {
		return nil
	}
	switch event.Key() {
	case tuivory.KeyTab, tuivory.KeyBacktab:
		s.cyclePanel(event.Key() == tuivory.KeyBacktab)
		return nil
	}
	return event
}
