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
	page, _ := s.pages.GetFrontPage()
	ctrlEnter := event.Key() == tcell.KeyEnter && event.Modifiers()&tcell.ModCtrl != 0
	alt := event.Modifiers()&tcell.ModAlt != 0
	letter := unicode.ToLower(event.Rune())
	if event.Key() == tcell.KeyCtrlQ || event.Key() == tcell.KeyCtrlC {
		s.action("quit")
		return nil
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
		if event.Key() == tcell.KeyCtrlS || (alt && letter == 's') {
			s.saveConnection()
			return nil
		}
		if event.Key() == tcell.KeyEscape || (alt && letter == 'x') {
			s.closeConnection()
			return nil
		}
		if ctrlEnter || (alt && letter == 'c') {
			s.connectAction()
			return nil
		}
		if alt {
			for i, key := range []rune{'h', 'p', 'u', 'w', 't', 'n', 'r', 'a'} {
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
	if ctrlEnter {
		s.action("execute")
		return nil
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
	switch event.Key() {
	case tcell.KeyF2:
		s.action("connect")
		return nil
	case tcell.KeyF4:
		s.action("execute")
		return nil
	case tcell.KeyF5:
		s.action("commit")
		return nil
	case tcell.KeyF6:
		s.action("rollback")
		return nil
	case tcell.KeyF7:
		s.toggleBrowser()
		return nil
	case tcell.KeyF9:
		s.app.SetFocus(s.editor)
		return nil
	case tcell.KeyF10:
		s.app.SetFocus(s.table)
		return nil
	case tcell.KeyTab, tcell.KeyBacktab:
		panels := []tview.Primitive{s.editor, s.table}
		if s.browser != nil && !s.browserHidden {
			panels = append(panels, s.browser)
		}
		step := 1
		if event.Key() == tcell.KeyBacktab {
			step = len(panels) - 1
		}
		for i, panel := range panels {
			if panel.HasFocus() {
				s.app.SetFocus(panels[(i+step)%len(panels)])
				return nil
			}
		}
		s.app.SetFocus(s.editor)
		return nil
	}
	return event
}
