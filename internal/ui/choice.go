package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Form reapplies its shared field colors on every draw. Keep selection fields
// distinct from white text inputs even after redraws and language changes.
type choiceField struct {
	*tview.DropDown
	background, foreground tcell.Color
}

// A list can begin halfway through a wide glyph drawn by a button behind it.
// Clear the old glyph's leading cell, or terminal rendering skips the first
// overlaid character even though its cell contains the correct new text.
type choiceOverlayScreen struct{ tcell.Screen }

func (s choiceOverlayScreen) SetContent(x, y int, main rune, combining []rune, style tcell.Style) {
	if x > 0 {
		_, _, previous, width := s.Screen.GetContent(x-1, y)
		if width > 1 {
			s.Screen.SetContent(x-1, y, ' ', nil, previous)
		}
	}
	s.Screen.SetContent(x, y, main, combining, style)
}

func (c *choiceField) Draw(screen tcell.Screen) {
	if c.IsOpen() {
		c.DropDown.Draw(choiceOverlayScreen{Screen: screen})
		return
	}
	c.DropDown.Draw(screen)
}

func (s *screen) newChoice(label string, options []string, selected int, changed func(string, int)) *choiceField {

	normal := tcell.StyleDefault.Foreground(s.colors().text).Background(s.colors().footer)
	dropdown := tview.NewDropDown().SetLabel(label).SetOptions(options, changed).SetCurrentOption(selected).
		SetFieldWidth(40).SetTextOptions("", "", " ", " ▾", " ▾").
		SetFieldStyle(normal).SetFocusedStyle(s.focusStyle()).SetListStyles(normal, s.focusStyle()).
		SetPrefixStyle(normal)
	choice := &choiceField{DropDown: dropdown}
	s.styleChoice(choice)
	return choice
}

func (c *choiceField) SetFormAttributes(labelWidth int, labelColor, bgColor, _, _ tcell.Color) tview.FormItem {
	c.DropDown.SetFormAttributes(labelWidth, labelColor, bgColor, c.foreground, c.background)
	return c
}

func (s *screen) styleChoice(c *choiceField) {
	c.background, c.foreground = s.colors().footer, s.colors().text
	if s.limitedColors && !s.monochrome() {
		c.background, c.foreground = tcell.ColorNavy, tcell.ColorWhite
	}
	normal := tcell.StyleDefault.Foreground(c.foreground).Background(c.background)
	c.SetFieldStyle(normal).SetFocusedStyle(s.focusStyle()).SetListStyles(normal, s.focusStyle()).SetPrefixStyle(normal)
}
