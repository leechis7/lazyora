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
	background, foreground := palette.footer, palette.text
	if s.limitedColors {
		background, foreground = tcell.ColorNavy, tcell.ColorWhite
	}
	normal := tcell.StyleDefault.Foreground(foreground).Background(background)
	dropdown := tview.NewDropDown().SetLabel(label).SetOptions(options, changed).SetCurrentOption(selected).
		SetFieldWidth(40).SetTextOptions("", "", " ", " ▾", " ▾").
		SetFieldStyle(normal).SetFocusedStyle(focusStyle()).SetListStyles(normal, focusStyle()).
		SetPrefixStyle(normal)
	return &choiceField{DropDown: dropdown, background: background, foreground: foreground}
}

func (c *choiceField) SetFormAttributes(labelWidth int, labelColor, bgColor, _, _ tcell.Color) tview.FormItem {
	c.DropDown.SetFormAttributes(labelWidth, labelColor, bgColor, c.foreground, c.background)
	return c
}
