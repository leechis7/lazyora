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
	// tcell reports legacy ESC CR as Ctrl+Alt+m rather than KeyEnter+Alt.
	if event.Key() == tcell.KeyRune && event.Rune() == 'm' && event.Modifiers()&(tcell.ModAlt|tcell.ModCtrl) == tcell.ModAlt|tcell.ModCtrl {
		event = tcell.NewEventKey(tcell.KeyEnter, 0, event.Modifiers()&^tcell.ModCtrl)
	}

	page, _ := s.pages.GetFrontPage()
	executeEnter := event.Key() == tcell.KeyEnter && event.Modifiers()&(tcell.ModCtrl|tcell.ModAlt) != 0
	alt := event.Modifiers()&tcell.ModAlt != 0
	letter := unicode.ToLower(event.Rune())
	if event.Key() == tcell.KeyCtrlQ {
		s.action("quit")
		return nil
	}
	if page == "main" || page == "menu" {
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
	if event.Key() == tcell.KeyF8 {
		s.action("cancel")
		return nil
	}
	if page == "exit" {
		return s.exitKey(event)
	}
	if page == "program_confirm" {
		return event
	}
	if event.Key() == tcell.KeyF1 {
		if page == "help" {
			s.closeHelp()
		} else {
			s.showHelp()
		}
		return nil
	}
	if page == "help" {
		if event.Key() == tcell.KeyEscape || (alt && letter == 'x') {
			s.closeHelp()
			return nil
		}
		return event
	}
	if page == "object" {
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
		if event.Key() == tcell.KeyEscape {
			s.closeSource()
			return nil
		}
		if event.Key() == tcell.KeyCtrlR {
			s.reloadSource()
			return nil
		}
		if event.Key() == tcell.KeyF9 && event.Modifiers()&tcell.ModCtrl != 0 {
			s.requestCompile()
			return nil
		}
		if alt && letter == 'a' {
			s.showArguments()
			return nil
		}
		if event.Key() == tcell.KeyTab || event.Key() == tcell.KeyBacktab {
			if s.sourceEditor.HasFocus() {
				s.app.SetFocus(s.sourceDiagnostics)
			} else {
				s.app.SetFocus(s.sourceEditor)
			}
			return nil
		}
		return event
	}
	if page == "arguments" {
		if event.Key() == tcell.KeyEscape {
			s.closeArguments()
			return nil
		}
		return event
	}
	if event.Key() == tcell.KeyF3 && page != "settings" {
		s.settingsForm()
		return nil
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
			}
		}
		return event
	}
	if page == "connect" {
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
	if s.browser != nil && s.browser.HasFocus() {
		switch event.Key() {
		case tcell.KeyLeft, tcell.KeyRight:
			s.browserArrow(event.Key())
			return nil
		case tcell.KeyCtrlR:
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
