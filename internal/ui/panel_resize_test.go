package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"testing"
)

func TestPanelResizeMouseKeyboardAndFullscreenPreserveWork(t *testing.T) {
	s, terminal := resultScreen(t)
	s.rebuildPanels()
	draw := func(w, h int) { terminal.SetSize(w, h); s.mainLayout.SetRect(0, 0, w, h); s.mainLayout.Draw(terminal) }
	draw(120, 35)
	_, _, _, oldHeight := s.editor.GetRect()
	s.key(tcell.NewEventKey(tcell.KeyF8, 0, tcell.ModAlt))
	s.key(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	s.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	draw(120, 35)
	_, _, _, height := s.editor.GetRect()
	if height <= oldHeight || s.sizing.schema != 30 {
		t.Fatalf("keyboard sizing not applied: old=%d new=%d sql=%d schema=%d mode=%v", oldHeight, height, s.sizing.sql, s.sizing.schema, s.sizing.keyboard)
	}
	x, y, _, _ := s.sizing.vertical.GetRect()
	if !s.resizePanelMouse(tcell.NewEventMouse(x, y, tcell.Button1, 0), tview.MouseLeftDown) {
		t.Fatal("divider not draggable")
	}
	s.resizePanelMouse(tcell.NewEventMouse(60, y, tcell.Button1, 0), tview.MouseMove)
	s.resizePanelMouse(tcell.NewEventMouse(60, y, 0, 0), tview.MouseLeftUp)
	draw(120, 35)
	if s.sizing.schema < 45 || s.sizing.drag != "" {
		t.Fatal("drag did not update width/release")
	}
	s.editor.SetText("select 'keep';", false)
	s.table.Select(2, 1)
	sql, schema := s.sizing.sql, s.sizing.schema
	s.togglePanelFullscreen("results")
	draw(120, 35)
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
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
