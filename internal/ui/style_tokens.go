package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Profiles describe output capability, independently of the user's theme.
type colorProfile uint8

const (
	profileRGB colorProfile = iota
	profileANSI8
	profileANSI16
	profile256
)

func (s *screen) colorProfile() colorProfile {
	n := 0
	if s.terminal != nil {
		n = s.terminal.Colors()
	}
	if s.limitedColors {
		if n > 0 && n <= 8 {
			return profileANSI8
		}
		return profileANSI16
	}
	if n > 0 && n <= 8 {
		return profileANSI8
	}
	if n > 0 && n <= 16 {
		return profileANSI16
	}
	if n > 0 && n <= 256 {
		return profile256
	}
	return profileRGB
}
func (s *screen) ansiColors() bool {
	p := s.colorProfile()
	return p == profileANSI8 || p == profileANSI16
}
func indexed(i int) tcell.Color { return tcell.PaletteColor(i) }

// Extended tokens keep content, input, selection and status roles independent.
type styleTokens struct {
	themeColors
	selection, selectionFG, edited, editedFG, disabled, focus tcell.Color
}

func (s *screen) tokens() styleTokens {
	p := s.colorProfile()
	if s.monochrome() {
		c := blackWhitePalette
		c.text, c.editorText, c.muted, c.accent, c.warning = tcell.ColorSilver, tcell.ColorSilver, tcell.ColorSilver, tcell.ColorSilver, tcell.ColorSilver
		c.field, c.button, c.placeholder = tcell.ColorSilver, tcell.ColorSilver, tcell.ColorBlack
		return styleTokens{c, tcell.ColorSilver, tcell.ColorBlack, tcell.ColorSilver, tcell.ColorBlack, tcell.ColorSilver, tcell.ColorSilver}
	}
	light := s.prefs.Theme == "light"
	c := palette
	if light {
		c = lightPalette
	}
	t := styleTokens{themeColors: c, selection: tcell.GetColor("#005f87"), selectionFG: tcell.ColorWhite,
		edited: tcell.GetColor("#d7af87"), editedFG: tcell.GetColor("#202a36"), disabled: c.muted, focus: c.accent}
	if light {
		t.focus = tcell.GetColor("#005fcc")
		t.accent = t.focus
		t.warning = tcell.GetColor("#7a4300")
	}
	t.selectionText = t.selectionFG
	if p == profileANSI8 || p == profileANSI16 {
		c = darkANSI
		c.text, c.editorText, c.muted, c.placeholder = indexed(7), indexed(7), indexed(7), indexed(0)
		if p == profileANSI8 {
			c.accent, c.warning = indexed(3), indexed(3)
		} else {
			c.accent = indexed(14)
		}
		if light {
			c = lightANSI
			c.background, c.surface, c.editor, c.field, c.footer = indexed(7), indexed(7), indexed(7), indexed(7), indexed(7)
			c.text, c.muted, c.editorText, c.placeholder = indexed(0), indexed(0), indexed(0), indexed(0)
			if p == profileANSI16 {
				c.background, c.editor, c.field, c.footer = indexed(15), indexed(15), indexed(15), indexed(15)
			}
		}
		t = styleTokens{c, indexed(4), indexed(7), indexed(3), indexed(0), c.muted, indexed(7)}
		if light {
			t.focus = indexed(4)
		}
		if p == profileANSI16 {
			t.selectionFG = indexed(15)
		}
		t.selectionText = t.selectionFG
		return t
	}
	if p == profile256 {
		c.background, c.surface, c.footer, c.editor, c.text, c.editorText, c.muted = indexed(236), indexed(237), indexed(237), indexed(236), indexed(252), indexed(252), indexed(250)
		c.field, c.fieldText, c.button, c.placeholder, c.accent, c.warning = indexed(254), indexed(235), indexed(153), indexed(239), indexed(117), indexed(222)
		if light {
			c.background, c.surface, c.footer, c.editor = indexed(255), indexed(254), indexed(254), indexed(231)
			c.text, c.editorText, c.muted, c.field, c.accent, c.warning = indexed(235), indexed(235), indexed(239), indexed(231), indexed(25), indexed(94)
		}
		t = styleTokens{c, indexed(24), indexed(255), indexed(180), indexed(0), c.muted, c.accent}
		if light {
			t.selectionFG = indexed(231)
		}
		t.selectionText = t.selectionFG
	}
	return t
}

type surfaceRole uint8

const (
	surfaceCanvas surfaceRole = iota
	surfacePanel
	surfaceField
	surfaceFooter
	surfaceEditor
)

func (s *screen) surfaceColor(role surfaceRole) tcell.Color {
	p := s.colors()
	switch role {
	case surfacePanel:
		return p.surface
	case surfaceField:
		return p.field
	case surfaceFooter:
		return p.footer
	case surfaceEditor:
		return p.editor
	default:
		return p.background
	}
}
func (s *screen) surfaceText(role surfaceRole) tcell.Color {
	switch role {
	case surfaceField:
		return s.colors().fieldText
	case surfaceEditor:
		return s.colors().editorText
	default:
		return s.colors().text
	}
}
func (s *screen) setSurface(box *tview.Box, role surfaceRole) {
	if s.surfaceRoles == nil {
		s.surfaceRoles = map[*tview.Box]surfaceRole{}
	}
	s.surfaceRoles[box] = role
	box.SetBackgroundColor(s.surfaceColor(role))
}
func (s *screen) textSurface(view *tview.TextView, role surfaceRole) {
	s.setSurface(view.Box, role)
	fg := s.textColor(s.textRoles[view])
	if role == surfaceField {
		fg = s.colors().fieldText
	}
	view.SetBackgroundColor(s.surfaceColor(role))
	view.SetTextColor(fg)
}
func (s *screen) disabledStyle() tcell.Style {
	return tcell.StyleDefault.Foreground(s.tokens().disabled).Background(s.colors().surface)
}

func widgetBox(p tview.Primitive) *tview.Box {
	switch w := p.(type) {
	case *formPopupLayer:
		return w.Flex.Box
	case *resultGrid:
		return w.Table.Box
	case *inlineCellEditor:
		return w.InputField.Box
	case *tview.Box:
		return w
	case *tview.Flex:
		return w.Box
	case *tview.Pages:
		return w.Box
	case *tview.TextView:
		return w.Box
	case *tview.TextArea:
		return w.Box
	case *tview.Form:
		return w.Box
	case *choiceField:
		return w.Box
	case *tview.InputField:
		return w.Box
	case *tview.Checkbox:
		return w.Box
	case *tview.Table:
		return w.Box
	case *tview.TreeView:
		return w.Box
	case *tview.List:
		return w.Box
	case *tview.Button:
		return w.Box
	case *tview.Modal:
		return w.Box
	}
	return nil
}

// Data-detail tables use semantic row roles from their first draw onward.
// Main query grids and the keymap comparison own their richer state styles.
func (s *screen) styleDataTable(table *tview.Table) {
	bg := s.surfaceColor(s.surfaceRoles[table.Box])
	table.SetBackgroundColor(bg)
	table.SetBordersColor(s.colors().muted).SetSelectedStyle(s.focusStyle())
	for row := 0; row < table.GetRowCount(); row++ {
		for col := 0; col < table.GetColumnCount(); col++ {
			cell := table.GetCell(row, col)
			st := tcell.StyleDefault.Foreground(s.colors().text).Background(bg)
			if row == 0 && cell.NotSelectable {
				st = st.Foreground(s.colors().accent).Underline(true)
			}
			cell.SetStyle(st).SetTransparency(false)
		}
	}
}
func (s *screen) newDataTable() *tview.Table {
	table := tview.NewTable()
	s.styleBox(table.Box)
	table.SetDrawFunc(func(_ tcell.Screen, x, y, w, h int) (int, int, int, int) {
		s.styleDataTable(table)
		return x, y, w, h
	})
	return table
}
