package ui

import (
	"github.com/leechis7/tuivory/go"
	"strings"
)

// Test accessors use the public menu model; production has no parallel list or
// navigation state. Old behavioral tests continue to exercise real key events.
func (v *commandMenuView) testItems() []tuivory.MenuItem {
	parent := ""
	path := v.model.Path()
	if len(path) > 0 {
		parent = path[len(path)-1]
	}
	var out []tuivory.MenuItem
	if len(path) > 1 {
		out = append(out, tuivory.MenuItem{ID: "@back", Label: v.model.BackLabel}, tuivory.MenuItem{Separator: true})
	}
	for _, item := range v.model.Items() {
		if item.Parent == parent {
			out = append(out, item)
		}
	}
	return out
}
func (v *commandMenuView) GetItemCount() int { return len(v.testItems()) }
func (v *commandMenuView) GetItemText(n int) (string, string) {
	i := v.testItems()[n]
	label := i.Label
	if i.Shortcut != "" {
		label += "  " + i.Shortcut
	}
	return label, ""
}
func (v *commandMenuView) SetCurrentItem(n int) {
	items := v.testItems()
	if n >= 0 && n < len(items) {
		v.model.Focus(items[n].ID)
	}
}
func (v *commandMenuView) GetCurrentItem() int {
	for n, i := range v.testItems() {
		if i.ID == v.model.FocusedID() {
			return n
		}
	}
	return 0
}
func (v *commandMenuView) GetOffset() (int, int) {
	_, _, w, h := v.GetInnerRect()
	lines := v.model.Layout(w, h, v.ASCII)
	if len(lines) > 0 {
		for n, i := range v.testItems() {
			if i.ID == lines[0].ID {
				return n, 0
			}
		}
	}
	return 0, 0
}
func menuTestRows(s *screen) []commandMenuRow {
	var rows []commandMenuRow
	for _, item := range s.menuList.testItems() {
		row := commandMenuRow{label: item.Label, divider: item.Separator}
		for _, child := range s.menuModel.Items() {
			if child.Parent == item.ID {
				for _, cmd := range commands {
					if strings.HasSuffix(child.ID, "/"+cmd.id) {
						row.children = append(row.children, cmd)
					}
				}
			}
		}
		if len(row.children) == 0 {
			for _, cmd := range commands {
				if strings.HasSuffix(item.ID, "/"+cmd.id) {
					row.id = cmd.id
					break
				}
			}
		}
		rows = append(rows, row)
	}
	return rows
}
func menuNested(s *screen) bool { return s.menuModel != nil && len(s.menuModel.Path()) > 1 }
func (s *screen) menuRowAction(n int) {
	items := s.menuList.testItems()
	if n >= 0 && n < len(items) {
		s.menuEvent(s.menuModel.Click(items[n].ID))
	}
}
