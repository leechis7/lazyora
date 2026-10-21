package ui

import (
	"fmt"

	"github.com/rivo/tview"
	"lazyora/internal/settings"
)

func (s *screen) closeSettings() {
	s.pages.RemovePage("settings")
	s.prefsForm, s.savePrefs = nil, nil
	if s.prefsFocus != nil {
		s.app.SetFocus(s.prefsFocus)
	} else {
		s.app.SetFocus(s.editor)
	}
	s.prefsFocus = nil
}

func (s *screen) settingsForm() {
	s.prefsFocus = s.app.GetFocus()
	form := tview.NewForm()
	s.prefsForm = form
	selected := 0
	if s.prefs.Language == "ko" {
		selected = 1
	}
	form.AddFormItem(s.newChoice(s.tr("language"), []string{"English", "한국어"}, selected, nil)).
		AddCheckbox(s.tr("mouse")+" — "+s.tr("mouse_label"), s.prefs.Mouse, nil)
	keymaps := []string{"", "vim", "notepad++"}
	keymapIndex := 0
	for i, keymap := range keymaps {
		if keymap == s.prefs.EditorKeymap {
			keymapIndex = i
		}
	}
	form.AddFormItem(s.newChoice(s.tr("editor_keymap"), []string{s.tr("keymap_default"), "Vim", "Notepad++"}, keymapIndex, nil))
	s.savePrefs = func() {
		index, _ := form.GetFormItem(0).(*choiceField).GetCurrentOption()
		language := "en"
		if index == 1 {
			language = "ko"
		}
		prefs := settings.Config{Language: language, Mouse: form.GetFormItem(1).(*tview.Checkbox).IsChecked()}
		keymapIndex, _ := form.GetFormItem(2).(*choiceField).GetCurrentOption()
		prefs.EditorKeymap = keymaps[keymapIndex]
		if err := settings.Save(s.prefsPath, prefs); err != nil {
			form.SetTitle(" " + fmt.Sprintf(s.tr("settings_error"), err) + " ")
			return
		}
		s.prefs = prefs
		s.vim = vimState{}
		s.app.EnableMouse(prefs.Mouse)
		s.refreshLabels()
		s.closeSettings()
		s.setStatus("settings_saved")
	}
	form.AddButton(s.tr("save")+" (Ctrl+S)", s.savePrefs).
		AddButton(s.tr("cancel")+" (Esc)", s.closeSettings)
	clickable(form.GetButton(0))
	clickable(form.GetButton(1))
	form.SetBorder(true).SetTitle(s.tr("settings_title"))
	styleForm(form)
	root := s.formPopup(form, 74, 17, func(width, height int) {
		spacing := 2
		if height < 16 {
			spacing = 1
		}
		if height < 12 {
			spacing = 0
		}
		form.SetItemPadding(spacing)
		if width < 60 {
			form.GetFormItem(1).(*tview.Checkbox).SetLabel(s.tr("mouse"))
			form.GetButton(0).SetLabel(s.tr("save"))
			form.GetButton(1).SetLabel(s.tr("cancel"))
		} else {
			form.GetFormItem(1).(*tview.Checkbox).SetLabel(s.tr("mouse") + " — " + s.tr("mouse_label"))
			form.GetButton(0).SetLabel(s.tr("save") + " (Ctrl+S)")
			form.GetButton(1).SetLabel(s.tr("cancel") + " (Esc)")
		}
	})
	s.pages.AddPage("settings", root, true, true)
	s.app.SetFocus(form)
}
