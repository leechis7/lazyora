package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/leechis7/tuivory/go"
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
	switch s.tuivoryProfile() {
	case "8":
		return profileANSI8
	case "16":
		return profileANSI16
	case "256":
		return profile256
	default:
		return profileRGB
	}
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

// Tuivory is the source of common style values. Legacy field names below are
// an adapter for product-specific rendering, not a second palette.
func (s *screen) tuivoryProfile() string {
	count := 0
	if s.terminal != nil {
		count = s.terminal.Colors()
	}
	return tuivory.ProfileForColors(count, s.limitedColors)
}
func (s *screen) tuivoryStyles() tuivory.Styles {
	theme := s.prefs.Theme
	if theme != "light" && theme != "blackwhite" {
		theme = "dark"
	}
	p, err := tuivory.NewStyles(theme, s.tuivoryProfile())
	if err != nil {
		panic(err)
	} // Both arguments are normalized above.
	return p
}
func (s *screen) tuivoryWidgets() tuivory.Styler { return tuivory.Styler{Styles: s.tuivoryStyles()} }
func legacyColors(c tuivory.Colors, selectionFG tcell.Color) themeColors {
	return themeColors{c.Background, c.Surface, c.Footer, c.Field, c.Text, c.Muted, c.Accent, c.Button,
		c.Editor, c.EditorText, c.FieldText, selectionFG, c.Placeholder, c.Warning}
}
func (s *screen) tokens() styleTokens {
	p := s.tuivoryStyles()
	return styleTokens{legacyColors(p.Colors, p.SelectionFG), p.Selection, p.SelectionFG, p.Edited, p.EditedFG, p.Disabled, p.Focus}
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
	return s.tuivoryStyles().SurfaceColor(tuivory.Surface(role))
}
func (s *screen) surfaceText(role surfaceRole) tcell.Color {
	return s.tuivoryStyles().SurfaceText(tuivory.Surface(role))
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
	palette := s.componentPalette()
	palette["surface"] = tuivory.Pair{FG: s.surfaceText(role), BG: s.surfaceColor(role)}
	textRole := map[textRole]string{textBody: "body", textHint: "hint", textAccent: "info", textWarning: "warning"}[s.textRoles[view]]
	if textRole == "" {
		textRole = "body"
	}
	style, err := tuivory.TextOnSurface(palette, textRole, "surface")
	if err != nil {
		panic(err)
	}
	view.SetBackgroundColor(s.surfaceColor(role))
	view.SetTextStyle(style)
}
func (s *screen) disabledStyle() tcell.Style {
	return s.tuivoryStyles().DisabledStyle()
}

func widgetBox(p tview.Primitive) *tview.Box {
	switch w := p.(type) {
	case *displayLayout:
		return w.Box
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
	case *browserTree:
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
// Main query grids and the keymap comparison own their richer state tuivory.
func (s *screen) styleDataTable(table *tview.Table) {
	s.tuivoryWidgets().TableOn(table, tuivory.Surface(s.surfaceRoles[table.Box]))
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
