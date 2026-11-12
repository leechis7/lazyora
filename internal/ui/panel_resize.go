package ui

import (
	"github.com/leechis7/tuivory/go"
)

type panelSizing struct {
	sql, schema                  int
	horizontal, vertical         *tuivory.Box
	result                       tuivory.Primitive
	resultControl, schemaControl tuivory.SplitControl
	swallowClick                 bool
	clickX, clickY               int
	drag                         string
	keyboard                     bool
	keyboardFocus                tuivory.Primitive
}

func panelSize(total, percent, minimum int) int {
	if total <= 1 {
		return 0
	}
	minimum = min(minimum, total/2)
	return max(minimum, min(total-minimum, total*percent/100))
}
func (s *screen) initPanelSizing() {
	if s.sizing.sql == 0 {
		s.sizing.sql, s.sizing.schema = 40, 25
	}
	divider := func(horizontal bool) *tuivory.Box {
		box := s.newBox()
		box.SetDrawFunc(func(screen tuivory.Screen, x, y, w, h int) (int, int, int, int) {
			style := s.tuivoryStyles().SplitterStyle()
			side, control := tuivory.SplitFirst, &s.sizing.schemaControl
			if horizontal {
				side, control = tuivory.SplitSecond, &s.sizing.resultControl
			}
			for py := y; py < y+h; py++ {
				for px := x; px < x+w; px++ {
					grip := px == x+w/2 && py == y+h/2
					char := tuivory.SplitGlyph(horizontal, s.asciiDisplay(), grip, side, control.Collapsed())
					screen.SetContent(px, py, char, nil, style)
				}
			}
			return x, y, w, h
		})
		return box
	}
	if s.sizing.horizontal == nil {
		s.sizing.horizontal, s.sizing.vertical = divider(true), divider(false)
		s.queryLayout.SetDrawFunc(func(_ tuivory.Screen, x, y, w, h int) (int, int, int, int) {
			if s.maximized == "" && !s.resultCollapsed() {
				reserved := 3
				if s.tabBar != nil {
					reserved++
				}
				s.queryLayout.ResizeItem(s.editor, panelSize(max(0, h-reserved), s.sizing.sql, 4), 0)
			}
			return x, y, w, h
		})
		s.mainLayout.SetDrawFunc(func(_ tuivory.Screen, x, y, w, h int) (int, int, int, int) {
			if s.maximized == "" && s.browserVisible() {
				s.mainLayout.ResizeItem(s.browserPanel, panelSize(max(0, w-1), s.sizing.schema, 18), 0)
			}
			return x, y, w, h
		})
	}
}
func (s *screen) resizePanelsCommand() {
	if s.maximized != "" {
		return
	}
	if s.sizing.keyboard {
		s.finishPanelSizing()
		return
	}
	s.sizing.keyboard = true
	s.sizing.keyboardFocus = s.app.GetFocus()
}

func (s *screen) finishPanelSizing() {
	s.sizing.keyboard = false
	if s.sizing.keyboardFocus != nil {
		s.app.SetFocus(s.sizing.keyboardFocus)
		s.sizing.keyboardFocus = nil
	}
	s.setStatus("resize_panels_done")
}
func (s *screen) resizePanelKey(e *tuivory.EventKey) bool {
	if !s.sizing.keyboard {
		return false
	}
	switch e.Key() {
	case tuivory.KeyEscape, tuivory.KeyEnter:
		s.finishPanelSizing()
	case tuivory.KeyUp:
		s.sizing.sql = max(10, s.sizing.sql-5)
	case tuivory.KeyDown:
		s.sizing.sql = min(90, s.sizing.sql+5)
	case tuivory.KeyLeft:
		s.sizing.schema = max(10, s.sizing.schema-5)
	case tuivory.KeyRight:
		s.sizing.schema = min(70, s.sizing.schema+5)
	case tuivory.KeyHome:
		s.sizing.sql, s.sizing.schema = 40, 25
	}
	return true
}
func (s *screen) resultCollapsed() bool {
	return s.sizing.resultControl.Collapsed() != tuivory.SplitNeither
}

func (s *screen) schemaCollapsed() bool {
	return s.sizing.schemaControl.Collapsed() != tuivory.SplitNeither
}

func (s *screen) browserVisible() bool {
	return s.browser != nil && !s.browserHidden && !s.schemaCollapsed()
}

// Keep the existing percentage-based workspace sizing. The SDK owns the
// click-versus-drag gesture and the collapsed side.
func (s *screen) splitGeometry(schema bool) (control *tuivory.SplitControl, side tuivory.SplitSide, spec tuivory.SplitSpec, state tuivory.SplitState, total, origin int) {
	var percent, minimum int
	if schema {
		control, side = &s.sizing.schemaControl, tuivory.SplitFirst
		percent, minimum = s.sizing.schema, 18
		origin, _, total, _ = s.mainLayout.GetInnerRect()
	} else {
		control, side = &s.sizing.resultControl, tuivory.SplitSecond
		percent, minimum = s.sizing.sql, 4
		_, origin, _, total = s.queryLayout.GetInnerRect()
		total -= 2
		if s.tabBar != nil {
			total--
			origin++
		}
	}
	minimum = max(1, min(minimum, (total-1)/2))
	spec = tuivory.SplitSpec{FirstMin: minimum, SecondMin: minimum}
	state.Preferred = panelSize(total-1, percent, minimum)
	return
}

func (s *screen) togglePanelCollapse(schema bool) {
	if s.mainLayout == nil || s.queryLayout == nil || s.maximized != "" || s.inlineEdit != nil || (schema && s.browser == nil) {
		return
	}
	if schema && s.browserHidden {
		s.browserHidden = false
		s.rebuildPanels()
		return
	}
	control, side, spec, state, total, _ := s.splitGeometry(schema)
	if control.Toggle(spec, &state, total, side) {
		s.applyPanelCollapse()
	}
}

func (s *screen) applyPanelCollapse() {
	if (s.resultCollapsed() && s.resultAreaHasFocus()) || (s.schemaCollapsed() && s.browserHasFocus()) {
		s.app.SetFocus(s.editor)
	}
	s.rebuildPanels()
}

func (s *screen) resizePanelMouse(event *tuivory.EventMouse, action tuivory.MouseAction) bool {
	page, _ := s.pages.GetFrontPage()
	if page != "main" || s.maximized != "" || s.inlineEdit != nil {
		s.sizing.resultControl.CancelPointer()
		s.sizing.schemaControl.CancelPointer()
		s.sizing.drag = ""
		s.sizing.swallowClick = false
		return false
	}
	x, y := event.Position()
	if action == tuivory.MouseLeftClick || action == tuivory.MouseLeftDoubleClick {
		swallow := s.sizing.swallowClick && x == s.sizing.clickX && y == s.sizing.clickY
		s.sizing.swallowClick = false
		return swallow
	}
	if action == tuivory.MouseLeftDown {
		s.sizing.swallowClick = false
		s.sizing.drag = ""
		if s.sizing.horizontal != nil && s.sizing.horizontal.InRect(x, y) {
			s.sizing.drag = "sql"
		} else if s.browser != nil && !s.browserHidden && s.sizing.vertical != nil && s.sizing.vertical.InRect(x, y) {
			s.sizing.drag = "schema"
		}
	}
	if s.sizing.drag == "" {
		return false
	}
	schema := s.sizing.drag == "schema"
	control, side, spec, state, total, origin := s.splitGeometry(schema)
	position, region := y-origin, s.queryLayout
	if schema {
		position, region = x-origin, s.mainLayout
	}
	kind := ""
	switch action {
	case tuivory.MouseLeftDown:
		kind = "down"
	case tuivory.MouseLeftUp:
		kind = "up"
	case tuivory.MouseMove:
		kind = "move"
		if event.Buttons()&tuivory.Button1 == 0 {
			kind = "cancel"
		}
	}
	if kind == "" {
		return true
	}
	if kind == "down" && control.Collapsed() == tuivory.SplitNeither {
		*control = tuivory.SplitControl{}
	}
	before := control.Collapsed()
	preferred := state.Preferred
	handled := control.Pointer(spec, &state, total, position, region.InRect(x, y), kind, side)
	if state.Preferred != preferred && total > 1 {
		percent := 100 * state.Preferred / (total - 1)
		if schema {
			s.sizing.schema = max(10, min(70, percent))
		} else {
			s.sizing.sql = max(10, min(90, percent))
		}
	}
	if !control.Captured() {
		s.sizing.drag = ""
	}
	if control.Collapsed() != before {
		s.sizing.swallowClick, s.sizing.clickX, s.sizing.clickY = true, x, y
		s.applyPanelCollapse()
	}
	return handled
}
