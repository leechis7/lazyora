package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type panelSizing struct {
	sql, schema          int
	horizontal, vertical *tview.Box
	result               tview.Primitive
	drag                 string
	keyboard             bool
	keyboardFocus        tview.Primitive
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
	divider := func(horizontal bool) *tview.Box {
		box := s.newBox()
		box.SetDrawFunc(func(screen tcell.Screen, x, y, w, h int) (int, int, int, int) {
			style := s.tuivoryStyles().SplitterStyle()
			char := '│'
			if horizontal {
				char = '─'
			}
			for py := y; py < y+h; py++ {
				for px := x; px < x+w; px++ {
					screen.SetContent(px, py, char, nil, style)
				}
			}
			return x, y, w, h
		})
		return box
	}
	if s.sizing.horizontal == nil {
		s.sizing.horizontal, s.sizing.vertical = divider(true), divider(false)
		s.queryLayout.SetDrawFunc(func(_ tcell.Screen, x, y, w, h int) (int, int, int, int) {
			if s.maximized == "" {
				reserved := 3
				if s.tabBar != nil {
					reserved++
				}
				s.queryLayout.ResizeItem(s.editor, panelSize(max(0, h-reserved), s.sizing.sql, 4), 0)
			}
			return x, y, w, h
		})
		s.mainLayout.SetDrawFunc(func(_ tcell.Screen, x, y, w, h int) (int, int, int, int) {
			if s.maximized == "" && !s.browserHidden {
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
func (s *screen) resizePanelKey(e *tcell.EventKey) bool {
	if !s.sizing.keyboard {
		return false
	}
	switch e.Key() {
	case tcell.KeyEscape, tcell.KeyEnter:
		s.finishPanelSizing()
	case tcell.KeyUp:
		s.sizing.sql = max(10, s.sizing.sql-5)
	case tcell.KeyDown:
		s.sizing.sql = min(90, s.sizing.sql+5)
	case tcell.KeyLeft:
		s.sizing.schema = max(10, s.sizing.schema-5)
	case tcell.KeyRight:
		s.sizing.schema = min(70, s.sizing.schema+5)
	case tcell.KeyHome:
		s.sizing.sql, s.sizing.schema = 40, 25
	}
	return true
}
func (s *screen) resizePanelMouse(event *tcell.EventMouse, action tview.MouseAction) bool {
	page, _ := s.pages.GetFrontPage()
	if page != "main" || s.maximized != "" {
		s.sizing.drag = ""
		return false
	}
	x, y := event.Position()
	if action == tview.MouseLeftDown {
		if s.sizing.horizontal != nil && s.sizing.horizontal.InRect(x, y) {
			s.sizing.drag = "sql"
		}
		if !s.browserHidden && s.sizing.vertical != nil && s.sizing.vertical.InRect(x, y) {
			s.sizing.drag = "schema"
		}
	}
	if s.sizing.drag == "" {
		return false
	}
	if action == tview.MouseMove || action == tview.MouseLeftDown {
		if s.sizing.drag == "sql" {
			_, top, _, height := s.queryLayout.GetInnerRect()
			reserved := 3
			if s.tabBar != nil {
				top++
				reserved++
			}
			s.sizing.sql = max(10, min(90, 100*(y-top)/max(1, height-reserved)))
		} else {
			left, _, width, _ := s.mainLayout.GetInnerRect()
			s.sizing.schema = max(10, min(70, 100*(x-left)/max(1, width-1)))
		}
	}
	if action == tview.MouseLeftUp || event.Buttons()&tcell.Button1 == 0 && action == tview.MouseMove {
		s.sizing.drag = ""
	}
	return true
}
