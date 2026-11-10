package ui

import (
	"fmt"
	"lazyora/internal/settings"
	"slices"
	"strings"

	"github.com/leechis7/tuivory/go"
)

func (s *screen) favoriteEditorSQL() string {
	if selected, _, _ := s.editor.GetSelection(); selected != "" {
		return selected
	}
	return s.editor.GetText()
}

func (s *screen) saveFavoriteSQL(sql string) {
	if strings.TrimSpace(sql) == "" {
		return
	}
	path := settings.FavoritesPath(s.prefsPath)
	items, err := settings.LoadFavorites(path)
	if err != nil {
		s.setStatus("error", err)
		return
	}
	s.editFavorite(settings.Favorite{SQL: sql}, func(item settings.Favorite) error {
		return settings.SaveFavorites(path, items, append(slices.Clone(items), item))
	})
}

func (s *screen) editFavorite(item settings.Favorite, save func(settings.Favorite) error) {
	focus := s.app.GetFocus()
	form := tuivory.NewWidgetForm().AddInputField(s.tr("favorite_name"), item.Name, 40, nil, nil).
		AddInputField(s.tr("favorite_tags"), item.Tags, 40, nil, nil)
	close := func() {
		s.pages.RemovePage("favorite_edit")
		s.favoriteEditKey = nil
		s.app.SetFocus(focus)
	}
	apply := func() {
		item.Name = strings.TrimSpace(form.GetFormItem(0).(*tuivory.InputField).GetText())
		item.Tags = strings.TrimSpace(form.GetFormItem(1).(*tuivory.InputField).GetText())
		if !validTabName(item.Name) {
			form.SetTitle(s.tr("favorite_invalid_name"))
			return
		}
		if err := save(item); err != nil {
			form.SetTitle(fmt.Sprintf(s.tr("error"), err))
			return
		}
		close()
		s.setStatus("favorite_saved")
	}
	form.AddButton(s.tr("save"), apply).AddButton(s.tr("cancel"), close)
	form.SetTitle(s.tr("favorites"))
	s.styleForm(form)
	for i := 0; i < form.GetButtonCount(); i++ {
		s.clickable(form.GetButton(i))
	}
	s.favoriteEditKey = func(event *tuivory.EventKey) *tuivory.EventKey {
		switch event.Key() {
		case tuivory.KeyEscape:
			close()
			return nil
		case tuivory.KeyCtrlS:
			apply()
			return nil
		}
		return event
	}
	hint := s.newTextView(textHint).SetWrap(true).SetText(s.tr("favorite_local_hint"))
	s.pages.AddPage("favorite_edit", s.formDialog(form, hint, 68), true, true)
	focusForm(s, form, 0)
}

func (s *screen) showFavorites() {
	path := settings.FavoritesPath(s.prefsPath)
	items, err := settings.LoadFavorites(path)
	if err != nil {
		s.setStatus("error", err)
		return
	}
	focus := s.app.GetFocus()
	close := func() {
		s.pages.RemovePage("favorites")
		s.favoritesKey = nil
		s.app.SetFocus(focus)
	}
	filter := s.newInputField().SetLabel(s.tr("favorite_filter") + " ")
	list := tuivory.NewWidgetList().ShowSecondaryText(false)
	s.styleList(list)
	list.SetBorder(true).SetTitle(s.tr("favorites"))
	preview := s.newTextView().SetDynamicColors(false).SetScrollable(true).SetWrap(false)
	preview.SetBorder(true).SetTitle(s.tr("sql_title"))
	s.styleBox(preview.Box)
	var visible []int
	selected := func() int {
		index := list.GetCurrentItem()
		if index < 0 || index >= len(visible) {
			return -1
		}
		return visible[index]
	}
	open := func() {
		index := selected()
		if index < 0 {
			return
		}
		if s.busy || len(s.gridDrafts) != 0 {
			if len(s.gridDrafts) != 0 {
				s.setStatus("grid_resolve_drafts")
			}
			return
		}
		item := items[index]
		close()
		s.newTab()
		s.tabs[s.activeTab].name = item.Name
		s.editor.SetText(item.SQL, false)
		s.refreshDocumentTitle()
	}
	var refresh func()
	buttons := s.newFlex()
	edit := func() {
		index := selected()
		if index < 0 {
			return
		}
		s.editFavorite(items[index], func(item settings.Favorite) error {
			next := slices.Clone(items)
			next[index] = item
			if err := settings.SaveFavorites(path, items, next); err != nil {
				return err
			}
			items = next
			refresh()
			return nil
		})
	}
	remove := func() {
		index := selected()
		if index < 0 {
			return
		}
		s.showProgramConfirm(s.tr("favorite_delete_confirm"), func() {
			next := slices.Delete(slices.Clone(items), index, index+1)
			if err := settings.SaveFavorites(path, items, next); err != nil {
				s.setStatus("error", err)
				return
			}
			items = next
			refresh()
		})
	}
	openButton := s.clickable(tuivory.NewWidgetButton(s.tr("favorite_open")).SetSelectedFunc(open))
	editButton := s.clickable(tuivory.NewWidgetButton(s.tr("favorite_edit")).SetSelectedFunc(edit))
	deleteButton := s.clickable(tuivory.NewWidgetButton(s.tr("favorite_delete")).SetSelectedFunc(remove))
	buttons.AddItem(openButton, 0, 1, false).AddItem(editButton, 0, 1, false).AddItem(deleteButton, 0, 1, false).
		AddItem(s.clickable(tuivory.NewWidgetButton(s.tr("close")).SetSelectedFunc(close)), 0, 1, false)
	refresh = func() {
		visible = visible[:0]
		list.Clear()
		query := strings.ToLower(strings.TrimSpace(filter.GetText()))
		for i, item := range items {
			if !strings.Contains(strings.ToLower(item.Name+" "+item.Tags), query) {
				continue
			}
			visible = append(visible, i)
			label := safeIdentity(item.Name)
			if item.Tags != "" {
				label += " · " + safeIdentity(item.Tags)
			}
			list.AddItem(tuivory.Escape(label), "", 0, nil)
		}
		empty := len(visible) == 0
		openButton.SetDisabled(empty)
		editButton.SetDisabled(empty)
		deleteButton.SetDisabled(empty)
		if empty {
			list.AddItem(s.tr("favorite_empty"), "", 0, nil)
			preview.SetText("")
		} else {
			list.SetCurrentItem(0)
			preview.SetText(items[visible[0]].SQL).ScrollToBeginning()
		}
	}
	list.SetChangedFunc(func(index int, _, _ string, _ rune) {
		if index >= 0 && index < len(visible) {
			preview.SetText(items[visible[index]].SQL).ScrollToBeginning()
		}
	})
	filter.SetChangedFunc(func(string) { refresh() })
	refresh()
	s.favoritesKey = func(event *tuivory.EventKey) *tuivory.EventKey {
		switch event.Key() {
		case tuivory.KeyEscape:
			close()
		case tuivory.KeyTab, tuivory.KeyBacktab:
			s.dialogTab(event, filter, list, preview)
		case tuivory.KeyEnter:
			if list.HasFocus() {
				open()
				return nil
			}
			if !filter.HasFocus() {
				return event
			}
			s.app.SetFocus(list)
		case tuivory.KeyDown:
			if !filter.HasFocus() {
				return event
			}
			s.app.SetFocus(list)
		default:
			return event
		}
		return nil
	}
	content := s.newFlex().SetDirection(tuivory.FlexRow).AddItem(filter, 1, 0, false).
		AddItem(list, 0, 2, false).AddItem(preview, 0, 1, false).AddItem(buttons, 1, 0, false)
	s.pages.AddPage("favorites", s.formPopup(content, 96, 26, nil), true, true)
	s.app.SetFocus(filter)
}
