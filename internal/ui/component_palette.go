package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/leechis7/tuivory/go"
)

func stylePair(st tcell.Style) tuivory.Pair {
	fg, bg, attrs := st.Decompose()
	return tuivory.Pair{FG: fg, BG: bg, Bold: attrs&tcell.AttrBold != 0}
}

// Adapt published semantic styles to component roles without a second palette.
func (s *screen) componentPalette() tuivory.Palette {
	p := s.tuivoryStyles()
	c := p.Colors
	return tuivory.Palette{
		"surface": {FG: c.Text, BG: c.Surface}, "body": {FG: c.Text, BG: c.Surface},
		"hint": {FG: c.Muted, BG: c.Surface}, "field": stylePair(p.TextStyle(tuivory.Field, tuivory.Body)),
		"button": stylePair(p.ButtonStyle()), "selection": stylePair(p.SelectionStyle()),
		"disabled": stylePair(p.DisabledStyle()), "edited": stylePair(p.EditedStyle()),
		"header": stylePair(p.GridHeaderStyle(false)), "info": {FG: c.Accent, BG: c.Surface},
		"success": {FG: c.Accent, BG: c.Surface}, "warning": {FG: c.Warning, BG: c.Surface},
		"error": {FG: c.Warning, BG: c.Surface}, "focus.border": {FG: s.borderColor(true, c.Surface), BG: c.Surface},
	}
}
func (s *screen) resultPalette() tuivory.Palette {
	p := s.tuivoryStyles()
	out := s.componentPalette()
	out["surface"] = stylePair(p.GridCellStyle(tuivory.CellState{Row: 1}))
	for role, style := range map[string]tcell.Style{
		"grid.base":        p.GridCellStyle(tuivory.CellState{Row: 1}),
		"grid.stripe":      p.GridCellStyle(tuivory.CellState{Row: 2}),
		"grid.current-row": p.GridCellStyle(tuivory.CellState{Row: 1, CurrentRow: true}),
		"grid.null":        p.GridCellStyle(tuivory.CellState{Row: 1, Null: true}),
		"grid.active":      p.GridActiveStyle(s.table != nil && s.table.HasFocus(), false),
		"grid.range":       p.GridCellStyle(tuivory.CellState{Row: 1, InRange: true}),
		"grid.edited":      p.EditedStyle(), "grid.header-active": p.GridHeaderStyle(true),
		"grid.gutter": p.GridGutterStyle(), "grid.status": p.GridStatusStyle(),
		"grid.divider": p.GridGutterStyle().Foreground(p.GridDividerColor()),
	} {
		out[role] = stylePair(style)
	}
	return out
}
