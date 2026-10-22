package ui

import (
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func focusForm(s *screen, form *tview.Form, index int) {
	form.SetFocus(index)
	s.app.SetFocus(form)
}

func (s *screen) key(event *tcell.EventKey) *tcell.EventKey {
	// Legacy terminal mappings report Shift+F1..F12 as F13..F24. Keep those
	// aliases equivalent to explicit modifiers; preserve modified higher keys.
	if event.Key() >= tcell.KeyF13 && event.Key() <= tcell.KeyF24 && event.Modifiers() == 0 {
		event = tcell.NewEventKey(tcell.KeyF1+event.Key()-tcell.KeyF13, 0, tcell.ModShift)
	}
	// tview treats an unchanged Ctrl+C event as an immediate application stop.
	// Forward a copy for text widgets so dialogs cannot bypass the SQL-file
	// and DB exit confirmations; explicit exit continues to use Ctrl+Q.
	if event.Key() == tcell.KeyCtrlC {
		event = tcell.NewEventKey(event.Key(), event.Rune(), event.Modifiers())
	}
	if s.vaultBusy {
		return nil
	}
	if s.workspaces != nil && event.Key() == tcell.KeyRune && event.Modifiers()&tcell.ModAlt != 0 && unicode.ToLower(event.Rune()) == 'j' {
		s.showWorkspaces()
		return nil
	}
	// tcell reports legacy ESC CR as Ctrl+Alt+m rather than KeyEnter+Alt.
	if event.Key() == tcell.KeyRune && event.Rune() == 'm' && event.Modifiers()&(tcell.ModAlt|tcell.ModCtrl) == tcell.ModAlt|tcell.ModCtrl {
		event = tcell.NewEventKey(tcell.KeyEnter, 0, event.Modifiers()&^tcell.ModCtrl)
	}

	page, _ := s.pages.GetFrontPage()
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
	if page == "main" && s.table.HasFocus() && event.Key() == tcell.KeyEnter && event.Modifiers() == 0 {
		s.showCellDetail()
		return nil
	}
	if page == "result_filter" {
		if event.Key() == tcell.KeyEscape {
			s.pages.RemovePage("result_filter")
			s.app.SetFocus(s.table)
			return nil
		}
		if event.Key() == tcell.KeyF1 {
			s.showHelp()
			return nil
		}
		return event
	}
	executeEnter := event.Key() == tcell.KeyEnter && event.Modifiers()&(tcell.ModCtrl|tcell.ModAlt) != 0
	alt := event.Modifiers()&tcell.ModAlt != 0
	letter := unicode.ToLower(event.Rune())
	if event.Key() == tcell.KeyCtrlQ {
		s.action("quit")
		return nil
	}
	if page == "reconnect" {
		if event.Key() == tcell.KeyEscape {
			s.cancelReconnect()
			return nil
		}
		// Modified Enter is the connection shortcut, never implicit approval.
		if executeEnter || event.Key() == tcell.KeyCtrlC || event.Key() == tcell.KeyF1 {
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
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyF10 && event.Modifiers() == tcell.ModShift {
			s.closeSourceEdit()
			return nil
		}
		if event.Key() == tcell.KeyF1 {
			s.showHelp()
			return nil
		}
		if event.Key() == tcell.KeyCtrlC {
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
		if event.Key() == tcell.KeyCtrlC {
			return nil
		}
		if event.Key() == tcell.KeyF1 {
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
	if event.Key() == tcell.KeyTab || event.Key() == tcell.KeyBacktab {
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
		if page == "main" && event.Key() == tcell.KeyF10 && event.Modifiers() == 0 {
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
	if (page == "copy" || page == "describe") && event.Key() == tcell.KeyF1 {
		s.showHelp()
		return nil
	}
	if page == "pending" && event.Key() != tcell.KeyF1 {
		if event.Key() == tcell.KeyEscape {
			s.closePendingWork()
			return nil
		}
		if event.Key() == tcell.KeyCtrlC {
			s.sendClipboard(s.pendingSQL.GetText(false))
			return nil
		}
		return event
	}
	if page == "copy" {
		if event.Key() == tcell.KeyEscape {
			s.closeCopyPreview()
			return nil
		}
		return event
	}
	if page == "describe" {
		if event.Key() == tcell.KeyEnter && s.describeForm.GetFormItem(0).HasFocus() {
			s.describeAction()
			return nil
		}
		if event.Key() == tcell.KeyEscape {
			s.closeDescribe()
			return nil
		}
		return event
	}
	if page == "search" {
		if event.Key() == tcell.KeyEscape {
			s.closeSearch()
			return nil
		}
		if event.Key() == tcell.KeyEnter && s.searchForm.GetFormItem(0).HasFocus() {
			s.searchAction("submit")
			return nil
		}
		if event.Key() == tcell.KeyF3 {
			action := "next"
			if event.Modifiers()&tcell.ModShift != 0 {
				action = "previous"
			}
			s.searchAction(action)
			return nil
		}
		return event
	}
	if page == "history" {
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyF8 {
			s.closeHistory()
			return nil
		}
		return event
	}
	if page == "main" && s.browserHasFocus() && (s.browserUsers.IsOpen() || s.browserTypes.IsOpen()) && event.Key() == tcell.KeyEscape {
		return event
	}
	// Help owns its close keys even while a background operation is running.
	// Closing the overlay must never be mistaken for cancelling that operation.
	if page == "help" {
		if s.listDetailArrow(event, s.helpTopics, s.helpText) {
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Modifiers() == 0 && event.Rune() >= '0' && event.Rune() <= '9' {
			s.helpJump(int(event.Rune() - '0'))
			return nil
		}
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyF1 || (alt && letter == 'x') {
			s.closeHelp()
			return nil
		}
		return event
	}
	if s.busy && event.Key() == tcell.KeyEscape && page != "exit" && page != "program_confirm" {
		s.action("cancel")
		return nil
	}
	// Dialogs and running queries own Escape first. On the idle main page,
	// restore fullscreen before editor keymaps or result-range clearing handle it.
	if page == "main" && event.Key() == tcell.KeyEscape && event.Modifiers() == 0 && s.fullscreen && s.maximized != "" {
		s.toggleFullscreen()
		return nil
	}
	if page == "exit" {
		return s.exitKey(event)
	}
	if page == "program_confirm" {
		if event.Key() == tcell.KeyCtrlC {
			return nil
		}
		return event
	}
	if event.Key() == tcell.KeyF1 {
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
		case tcell.KeyEscape:
			s.closeObject()
			return nil
		case tcell.KeyCtrlR:
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
		if event.Key() == tcell.KeyF10 && event.Modifiers() == tcell.ModShift {
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
		if event.Key() == tcell.KeyCtrlF && event.Modifiers()&tcell.ModShift != 0 {
			s.formatCode()
			return nil
		}
		if event.Key() == tcell.KeyCtrlT {
			s.showCompletion(false)
			return nil
		}
		if event.Key() == tcell.KeyEscape {
			s.closeSource()
			return nil
		}
		if event.Key() == tcell.KeyCtrlR {
			s.reloadSource()
			return nil
		}
		if event.Key() == tcell.KeyF9 && event.Modifiers() == tcell.ModNone {
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
		if event.Key() == tcell.KeyEscape {
			s.closeArguments()
			return nil
		}
		return event
	}
	if page == "settings" {
		if event.Key() == tcell.KeyEscape || (alt && letter == 'x') {
			s.closeSettings()
			return nil
		}
		if event.Key() == tcell.KeyCtrlS || (alt && letter == 's') {
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
			}
		}
		return event
	}
	if page == "connect" {
		if alt && letter == 'k' {
			s.passwordManager()
			return nil
		}
		if s.connectionTab(event) {
			return nil
		}
		if event.Key() == tcell.KeyCtrlS || (alt && letter == 's') {
			s.saveConnection()
			return nil
		}
		if event.Key() == tcell.KeyEscape || (alt && letter == 'x') {
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
	if event.Modifiers()&tcell.ModCtrl != 0 && (event.Key() == tcell.KeyTab || event.Key() == tcell.KeyBacktab) {
		step := 1
		if event.Key() == tcell.KeyBacktab || event.Modifiers()&tcell.ModShift != 0 {
			step = -1
		}
		s.cycleTab(step)
		return nil
	}
	if s.browserHasFocus() {
		if (s.browserUsers.IsOpen() || s.browserTypes.IsOpen()) && event.Modifiers()&(tcell.ModCtrl|tcell.ModAlt) == 0 {
			return event
		}
		if event.Key() == tcell.KeyCtrlR {
			s.refreshBrowser()
			return nil
		}
	}

	if s.table.HasFocus() {
		switch event.Key() {
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyLeft, tcell.KeyRight:
			if event.Modifiers()&tcell.ModShift != 0 {
				s.extendResultSelection(event.Key())
				return nil
			}
			s.clearResultSelection()
		case tcell.KeyEscape:
			s.clearResultSelection()
			return nil
		}
	}
	if s.commandKey(event) {
		return nil
	}
	switch event.Key() {
	case tcell.KeyTab, tcell.KeyBacktab:
		s.cyclePanel(event.Key() == tcell.KeyBacktab)
		return nil
	}
	return event
}
