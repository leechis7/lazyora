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
	if s.limitedColors {
		c.background, c.foreground = tcell.ColorNavy, tcell.ColorWhite
	}
	normal := tcell.StyleDefault.Foreground(c.foreground).Background(c.background)
	c.SetFieldStyle(normal).SetFocusedStyle(s.focusStyle()).SetListStyles(normal, s.focusStyle()).SetPrefixStyle(normal)
}
