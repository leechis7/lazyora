package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/leechis7/tuivory/go"
	"github.com/rivo/tview"
)

// A contents list keeps Up/Down for selection. Right enters its detail pane;
// Left returns from the detail pane without interfering with text inputs.
func (s *screen) listDetailArrow(event *tcell.EventKey, list *tview.List, detail tview.Primitive) bool {
	if event.Modifiers() != 0 || list == nil || detail == nil {
		return false
	}
	if event.Key() == tcell.KeyRight && list.HasFocus() {
		s.app.SetFocus(detail)
		return true
	}
	if event.Key() == tcell.KeyLeft && detail.HasFocus() {
		s.app.SetFocus(list)
		return true
	}
	return false
}

// Each form or horizontal button row keeps its own left/right focus cycle.
// Looking up the actual focused button leaves text cursors and lists untouched.
func buttonRow(root tview.Primitive, focused *tview.Button) ([]*tview.Button, *tview.Form) {
	switch container := root.(type) {
	case *formPopupLayer:
		return buttonRow(container.Flex, focused)
	case *tview.Form:
		var row []*tview.Button
		found := false
		for i := 0; i < container.GetButtonCount(); i++ {
			button := container.GetButton(i)
			row = append(row, button)
			found = found || button == focused
		}
		if found {
			return row, container
		}
	case *tview.Flex:
		var row []*tview.Button
		found := false
		for i := 0; i < container.GetItemCount(); i++ {
			item := container.GetItem(i)
			if button, ok := item.(*tview.Button); ok {
				row = append(row, button)
				found = found || button == focused
			} else if buttons, form := buttonRow(item, focused); len(buttons) > 0 {
				return buttons, form
			}
		}
		if found {
			return row, nil
		}
	case tuivory.StyleChildren:
		for _, child := range container.StyleChildren() {
			if buttons, form := buttonRow(child, focused); len(buttons) > 0 {
				return buttons, form
			}
		}

	}
	return nil, nil
}

func (s *screen) buttonArrow(event *tcell.EventKey) bool {
	if event.Modifiers() != 0 || (event.Key() != tcell.KeyLeft && event.Key() != tcell.KeyRight) {
		return false
	}
	focused, ok := s.app.GetFocus().(*tview.Button)
	if !ok {
		return false
	}
	page, root := s.pages.GetFrontPage()
	row, form := buttonRow(root, focused)
	if len(row) == 0 && page == "main" && s.footerBar != nil {
		row, form = buttonRow(s.footerBar, focused)
	}
	for current, button := range row {
		if button != focused {
			continue
		}
		step := 1
		if event.Key() == tcell.KeyLeft {
			step = -1
		}
		for offset := 1; offset <= len(row); offset++ {
			next := (current + step*offset + len(row)) % len(row)
			if row[next].IsDisabled() {
				continue
			}
			if form != nil {
				focusForm(s, form, form.GetFormItemCount()+next)
			} else {
				s.app.SetFocus(row[next])
			}
			return true
		}
	}
	return false
}

// Standalone detail pages include their action buttons in Tab navigation.
func (s *screen) dialogTab(event *tcell.EventKey, content ...tview.Primitive) {
	_, root := s.pages.GetFrontPage()
	if popup, ok := root.(*formPopupLayer); ok {
		root = popup.content
	}
	if flex, ok := root.(*tview.Flex); ok {
		last := flex.GetItem(flex.GetItemCount() - 1)
		if button, ok := last.(*tview.Button); ok {
			content = append(content, button)
		} else if row, ok := last.(*tview.Flex); ok {
			for i := 0; i < row.GetItemCount(); i++ {
				if button, ok := row.GetItem(i).(*tview.Button); ok && !button.IsDisabled() {
					content = append(content, button)
				}
			}
		}
	}
	step := 1
	if event.Key() == tcell.KeyBacktab {
		step = len(content) - 1
	}
	for i, item := range content {
		if item.HasFocus() {
			s.app.SetFocus(content[(i+step)%len(content)])
			return
		}
	}
}
