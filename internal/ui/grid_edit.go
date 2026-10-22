package ui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func (s *screen) editTableTarget() oracle.Object {
	if s.table.HasFocus() && s.resultBrowser.source.EditObject.Kind == "TABLE" {
		return s.resultBrowser.source.EditObject
	}
	if s.browser != nil {
		if node := s.browser.GetCurrentNode(); node != nil {
			if data, ok := node.GetReference().(*browserNode); ok && data.kind == "object" {
				return data.object
			}
		}
	}
	return oracle.Object{}
}
func (s *screen) loadEditTable() {
	object := s.editTableTarget()
	if object.Kind != "TABLE" {
		return
	}
	s.submit(job{action: "edit_load", object: object})
}
func (s *screen) invalidateEditResults(active uint64) {
	defer s.refreshResultPosition()
	if s.resultBrowser.source.EditID != active {
		s.resultBrowser.source.EditID = 0
	}
	for _, tab := range s.tabs {
		if tab.browser.source.EditID != active {
			tab.browser.source.EditID = 0
		}
	}
}
func (s *screen) showGridEdit() {
	source := s.resultBrowser.source
	row, col := s.table.GetSelection()
	if s.busy || source.EditID == 0 || row < 1 || row > len(s.resultBrowser.visible) || col < 0 || col >= len(source.Columns) {
		return
	}
	index := s.resultBrowser.visible[row-1]
	old := source.Values[index][col]
	s.editChange = oracle.EditChange{ID: source.EditID, Row: index, Column: col, Text: old.Text, Null: old.Null}
	value := old.Text
	if old.Null {
		value = ""
	}
	s.editInfo = s.newTextView().SetWrap(true).SetText(fmt.Sprintf("%s · %s (%s)\n%s: %s\n%s", tview.Escape(oracle.QualifiedObject(source.EditObject)), tview.Escape(source.Columns[col]), old.Type, s.tr("edit_old"), tview.Escape(old.Text), s.tr("edit_hint"))).SetDynamicColors(true)
	s.styleBox(s.editInfo.Box)
	form := tview.NewForm().AddInputField(s.tr("edit_new"), value, 50, nil, nil).AddCheckbox("NULL", old.Null, nil)
	s.editForm = form
	form.AddButton(s.tr("edit_preview"), s.previewGridEdit).AddButton(s.tr("cancel"), s.closeGridEdit)
	s.styleForm(form)
	for i := 0; i < form.GetButtonCount(); i++ {
		s.clickable(form.GetButton(i))
	}
	root := s.formDialog(form, s.editInfo, 92)
	s.pages.AddPage("grid_edit", root, true, true)
	focusForm(s, form, 0)
}
func (s *screen) previewGridEdit() {
	if s.busy {
		return
	}
	change := s.editChange
	change.Text = s.editForm.GetFormItem(0).(*tview.InputField).GetText()
	change.Null = s.editForm.GetFormItem(1).(*tview.Checkbox).IsChecked()
	source := s.resultBrowser.source
	if source.EditID != change.ID {
		return
	}
	old := source.Values[change.Row][change.Column]
	next := change.Text
	if change.Null {
		next = "NULL"
	}
	message := fmt.Sprintf("%s\n%s · %s\n%s: %s\n%s: %s\n\n%s", s.tr("edit_preview"), tview.Escape(oracle.QualifiedObject(source.EditObject)), tview.Escape(source.Columns[change.Column]), s.tr("edit_old"), tview.Escape(old.Text), s.tr("edit_new"), tview.Escape(next), s.tr("edit_confirm"))
	object := source.EditObject
	s.showProgramConfirm(message, func() { s.submit(job{action: "edit_apply", edit: change, object: object}) })
}
func (s *screen) closeGridEdit() {
	if s.busy {
		s.action("cancel")
		return
	}
	s.pages.RemovePage("grid_edit")
	s.editForm = nil
	s.editInfo = nil
	s.editChange = oracle.EditChange{}
	s.app.SetFocus(s.table)
}
func (s *screen) gridEditKey(e *tcell.EventKey) *tcell.EventKey {
	if e.Key() == tcell.KeyF1 {
		s.showHelp()
		return nil
	}
	if e.Key() == tcell.KeyEscape {
		s.closeGridEdit()
		return nil
	}
	if s.busy {
		return nil
	}
	if s.buttonArrow(e) {
		return nil
	}
	if e.Key() == tcell.KeyCtrlS {
		s.previewGridEdit()
		return nil
	}
	return e
}
