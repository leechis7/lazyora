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
	form.AddDropDown(s.tr("language"), []string{"English", "한국어"}, selected, nil).
		AddCheckbox(s.tr("mouse")+" — "+s.tr("mouse_label"), s.prefs.Mouse, nil)
	s.savePrefs = func() {
		index, _ := form.GetFormItem(0).(*tview.DropDown).GetCurrentOption()
		language := "en"
		if index == 1 {
			language = "ko"
		}
		prefs := settings.Config{Language: language, Mouse: form.GetFormItem(1).(*tview.Checkbox).IsChecked()}
		if err := settings.Save(s.prefsPath, prefs); err != nil {
			form.SetTitle(" " + fmt.Sprintf(s.tr("settings_error"), err) + " ")
			return
		}
		s.prefs = prefs
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
	s.pages.AddPage("settings", form, true, true)
	s.app.SetFocus(form)
}
