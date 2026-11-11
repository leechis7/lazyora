package ui

import (
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestPanelResizeMouseKeyboardAndFullscreenPreserveWork(t *testing.T) {
	s, terminal := resultScreen(t)
	s.rebuildPanels()
	draw := func(w, h int) { terminal.SetSize(w, h); s.mainLayout.SetRect(0, 0, w, h); s.mainLayout.Draw(terminal) }
	draw(120, 35)
	_, _, _, oldHeight := s.editor.GetRect()
	s.key(tuivory.NewEventKey(tuivory.KeyF8, 0, tuivory.ModAlt))
	s.key(tuivory.NewEventKey(tuivory.KeyDown, 0, 0))
	s.key(tuivory.NewEventKey(tuivory.KeyRight, 0, 0))
	s.key(tuivory.NewEventKey(tuivory.KeyEnter, 0, 0))
	draw(120, 35)
	_, _, _, height := s.editor.GetRect()
	if height <= oldHeight || s.sizing.schema != 30 {
		t.Fatalf("keyboard sizing not applied: old=%d new=%d sql=%d schema=%d mode=%v", oldHeight, height, s.sizing.sql, s.sizing.schema, s.sizing.keyboard)
	}
	x, y, _, _ := s.sizing.vertical.GetRect()
	if !s.resizePanelMouse(tuivory.NewEventMouse(x, y, tuivory.Button1, 0), tuivory.MouseLeftDown) {
		t.Fatal("divider not draggable")
	}
	s.resizePanelMouse(tuivory.NewEventMouse(60, y, tuivory.Button1, 0), tuivory.MouseMove)
	s.resizePanelMouse(tuivory.NewEventMouse(60, y, 0, 0), tuivory.MouseLeftUp)
	draw(120, 35)
	if s.sizing.schema < 45 || s.sizing.drag != "" {
		t.Fatal("drag did not update width/release")
	}
	s.editor.SetText("select 'keep';", false)
	s.table.Select(2, 1)
	sql, schema := s.sizing.sql, s.sizing.schema
	s.togglePanelFullscreen("results")
	draw(120, 35)
	s.key(tuivory.NewEventKey(tuivory.KeyEscape, 0, 0))
	draw(120, 35)
	row, col := s.table.GetSelection()
	if s.sizing.sql != sql || s.sizing.schema != schema || row != 2 || col != 1 || s.editor.GetText() != "select 'keep';" {
		t.Fatal("resize/fullscreen altered work")
	}
	for _, size := range [][2]int{{20, 10}, {40, 15}, {200, 60}} {
		draw(size[0], size[1])
		ex, ey, ew, eh := s.editor.GetRect()
		rx, ry, rw, rh := s.table.GetRect()
		if ex < 0 || ey < 0 || rx < 0 || ry < 0 || ew < 0 || eh < 0 || rw < 0 || rh < 0 || ex+ew > size[0] || rx+rw > size[0] || ry+rh > size[1] {
			t.Fatal("small terminal layout out of bounds")
		}
	}
}

func TestPanelResizeFooterShowsOneHintAndWorkingModeButtons(t *testing.T) {
	for _, finish := range []int{1, 2} {
		s, terminal := resultScreen(t)
		s.app.SetFocus(s.table)
		s.setStatus("disconnected")
		before := s.status.GetText(false)
		s.refreshFooter()
		normal := append([]footerKey(nil), s.footerKeys...)
		s.key(tuivory.NewEventKey(tuivory.KeyF8, 0, tuivory.ModAlt))
		s.refreshFooter()
		if s.status.GetText(false) != before || strings.Count(s.status.GetText(false)+s.footer.GetText(false), s.tr("resize_panels_hint")) != 1 {
			t.Fatal("resize hint duplicated or replaced the result status")
		}
		if len(s.footerKeys) != 3 || s.footerKeys[0].key != tuivory.KeyHome || s.footerKeys[1].key != tuivory.KeyEnter || s.footerKeys[2].key != tuivory.KeyEscape {
			t.Fatal("missing resize mode buttons")
		}
		s.footerBar.SetRect(0, 0, 120, 1)
		s.footerBar.Draw(terminal)
		s.sizing.sql, s.sizing.schema = 65, 45
		click := func(index int) {
			button := s.buttons[index]
			x, y, _, _ := button.GetRect()
			button.MouseHandler()(tuivory.MouseLeftClick, tuivory.NewEventMouse(x+1, y, 0, 0), func(p tuivory.Primitive) { s.app.SetFocus(p) })
		}
		click(0)
		if !s.sizing.keyboard || s.sizing.sql != 40 || s.sizing.schema != 25 || len(s.jobs) != 0 {
			t.Fatal("reset button did not reset only the panel sizes")
		}
		click(finish)
		s.refreshFooter()
		if s.sizing.keyboard || !s.table.HasFocus() || len(s.footerKeys) != len(normal) {
			t.Fatal("finish did not restore normal toolbar and panel focus")
		}
		for i, key := range normal {
			if s.footerKeys[i].action != key.action {
				t.Fatal("normal toolbar actions changed")
			}
		}
	}
}

func TestPanelCollapseClickKeyboardAndFullscreenPreserveWork(t *testing.T) {
	s, terminal := resultScreen(t)
	s.rebuildPanels()
	draw := func() {
		terminal.SetSize(120, 35)
		s.mainLayout.SetRect(0, 0, 120, 35)
		s.mainLayout.Draw(terminal)
	}
	draw()
	_, _, schemaWidth, _ := s.browserPanel.GetRect()
	_, _, _, resultHeight := s.table.GetRect()
	s.table.Select(2, 1)
	s.app.SetFocus(s.table)
	s.key(tuivory.NewEventKey(tuivory.KeyF9, 0, tuivory.ModAlt))
	draw()
	if !s.resultCollapsed() || !s.editor.HasFocus() || len(s.jobs) != 0 {
		t.Fatal("result toggle failed or executed SQL")
	}
	s.togglePanelFullscreen("results")
	draw()
	s.key(tuivory.NewEventKey(tuivory.KeyEscape, 0, 0))
	draw()
	if !s.resultCollapsed() || !s.editor.HasFocus() {
		t.Fatal("fullscreen did not restore the collapsed result state")
	}
	click := func(divider *tuivory.Box) {
		x, y, w, h := divider.GetRect()
		x, y = x+w/2, y+h/2
		if !s.resizePanelMouse(tuivory.NewEventMouse(x, y, tuivory.Button1, 0), tuivory.MouseLeftDown) ||
			!s.resizePanelMouse(tuivory.NewEventMouse(x, y, 0, 0), tuivory.MouseLeftUp) ||
			!s.resizePanelMouse(tuivory.NewEventMouse(x, y, 0, 0), tuivory.MouseLeftClick) {
			t.Fatal("divider click was not consumed")
		}
		draw()
	}
	click(s.sizing.horizontal)
	if s.resultCollapsed() {
		t.Fatal("result divider did not restore the pane")
	}
	s.app.SetFocus(s.browser)
	click(s.sizing.vertical)
	if s.browserVisible() || !s.editor.HasFocus() {
		t.Fatal("schema divider did not collapse or transfer focus")
	}
	click(s.sizing.vertical)
	_, _, gotSchema, _ := s.browserPanel.GetRect()
	_, _, _, gotResult := s.table.GetRect()
	row, col := s.table.GetSelection()
	if !s.browserVisible() || gotSchema != schemaWidth || gotResult != resultHeight || row != 2 || col != 1 {
		t.Fatal("restoration changed size or selected cell")
	}
}
