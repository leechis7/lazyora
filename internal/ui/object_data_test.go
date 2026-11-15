package ui

import (
	"lazyora/internal/oracle"
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestObjectWorkspaceUsesLazyDataTabAndPreservesMainResult(t *testing.T) {
	s := demoBrowserScreen(t)
	s.render(oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"keep"}}})
	editor := s.editor.GetText()
	s.app.SetFocus(s.browser)
	s.openObjectDetails(oracle.Object{Owner: "LAZYORA", Name: "SAMPLE_ROWS", Kind: "TABLE"}, true)
	w := s.objectWorkspace
	if w == nil || w.split == nil || w.dataGrid != nil {
		t.Fatal("workspace did not defer data loading")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyRune, 'v', tuivory.ModAlt))
	if w.section != "data" || w.dataGrid == nil || len(w.data.Values) != 2 {
		t.Fatal("data preview missing")
	}
	connectionKey(s, tuivory.KeyRight)
	connectionKey(s, tuivory.KeyEnter)
	if w.section != "ddl" || !strings.Contains(w.ddl.GetText(false), "CREATE TABLE") {
		t.Fatal("DDL did not open in the same workspace")
	}
	s.key(tuivory.NewEventKey(tuivory.KeyEscape, 0, 0))
	if page, _ := s.pages.GetFrontPage(); page != "main" || s.editor.GetText() != editor || s.resultRows[0][0] != "keep" {
		t.Fatal("object workspace changed the SQL editor or result")
	}
}

func TestObjectWorkspaceTabsAcceptMouseClicks(t *testing.T) {
	s, _ := inlineEntryScreen(t, 8, "ko")
	s.connected = false
	s.demo = true
	s.app.SetFocus(s.browser)
	s.openObjectDetails(oracle.Object{Owner: "LAZYORA", Name: "SAMPLE_ROWS", Kind: "TABLE"}, true)
	s.app.ForceDraw()
	w := s.objectWorkspace
	s.mainLayout.SetMouseCapture(func(action tuivory.MouseAction, event *tuivory.EventMouse) (tuivory.MouseAction, *tuivory.EventMouse) {
		t.Error("mouse event reached the covered main page")
		return action, event
	})
	var captured tuivory.Primitive
	for _, section := range []string{"data", "ddl", "columns"} {
		x, y, width, height := w.tabs.GetInnerRect()
		if height != 1 {
			t.Fatalf("expected a one-row tab strip on the full-size detail page; height=%d", height)
		}
		for _, cell := range s.objectTabs().Layout(w.tabState, width).Cells {
			if cell.ID == section {
				x += cell.Start + cell.Width/2
				break
			}
		}
		for _, action := range []tuivory.MouseAction{tuivory.MouseLeftDown, tuivory.MouseLeftUp, tuivory.MouseLeftClick} {
			buttons := tuivory.ButtonNone
			if action == tuivory.MouseLeftDown {
				buttons = tuivory.Button1
			}
			event := tuivory.NewEventMouse(x, y, buttons, 0)
			if event, action = s.app.GetMouseCapture()(event, action); event != nil {
				target := captured
				if target == nil {
					target = s.rootLayout
				}
				_, captured = target.MouseHandler()(action, event, func(p tuivory.Primitive) { s.app.SetFocus(p) })
			}
			s.app.ForceDraw()
		}
		if w.section != section || !w.tabs.HasFocus() {
			t.Fatalf("click %s at %d,%d left section %s and focus %T", section, x, y, w.section, s.app.GetFocus())
		}
	}
}
