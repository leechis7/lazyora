package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

var bindKinds = []string{"text", "number", "date", "timestamp"}

func (s *screen) clearBindValues() {
	s.resultQuery = nil
	s.resultBrowser.source.Query = nil
	for i := range s.scriptReport.Steps {
		s.scriptReport.Steps[i].Result.Query = nil
	}
	s.bindSQL = ""
	s.bindValues = nil
	for _, tab := range s.tabs {
		tab.query = nil
		tab.browser.source.Query = nil
		for i := range tab.script.Steps {
			tab.script.Steps[i].Result.Query = nil
		}
		tab.bindSQL = ""
		tab.bindValues = nil
	}
}

func (s *screen) withBinds(text string, next func(map[string]oracle.BindValue)) {
	if s.busy || s.bindForm != nil {
		return
	}
	names, err := oracle.BindNames(text)
	if err != nil {
		s.setStatus("error", err)
		return
	}
	if len(names) == 0 {
		next(nil)
		return
	}
	focus := s.app.GetFocus()
	form := tview.NewForm().SetItemPadding(0)
	for _, name := range names {
		value := oracle.BindValue{Kind: "text"}
		if s.bindSQL == text {
			if old, ok := s.bindValues[name]; ok {
				value = old
			}
		}
		kind := 0
		for i, k := range bindKinds {
			if k == value.Kind {
				kind = i
			}
		}
		form.AddFormItem(s.newChoice(":"+name+" · "+s.tr("bind_type"), []string{"Text", "NUMBER", "DATE", "TIMESTAMP"}, kind, nil))
		form.AddInputField(s.tr("bind_value"), value.Text, 40, nil, nil)
		form.AddCheckbox("NULL", value.Null, nil)
	}
	close := func() {
		s.pages.RemovePage("bind")
		s.bindForm = nil
		s.bindSubmit = nil
		s.bindCancel = nil
		s.app.SetFocus(focus)
	}
	s.bindSubmit = func() {
		values := map[string]oracle.BindValue{}
		for i, name := range names {
			kind, _ := form.GetFormItem(i * 3).(*choiceField).GetCurrentOption()
			value := oracle.BindValue{Kind: bindKinds[kind], Text: form.GetFormItem(i*3 + 1).(*tview.InputField).GetText(), Null: form.GetFormItem(i*3 + 2).(*tview.Checkbox).IsChecked()}
			if _, err := value.Value(); err != nil {
				form.SetTitle(s.tr("bind_title") + " · :" + name + " · " + tview.Escape(err.Error()))
				return
			}
			values[name] = value
		}
		s.bindSQL, s.bindValues = text, values
		close()
		next(values)
	}
	s.bindCancel = close
	form.AddButton(s.tr("execute"), s.bindSubmit).AddButton(s.tr("bind_clear"), func() {
		s.clearBindValues()
		for i := range names {
			form.GetFormItem(i*3 + 1).(*tview.InputField).SetText("")
			form.GetFormItem(i*3 + 2).(*tview.Checkbox).SetChecked(false)
		}
	}).AddButton(s.tr("cancel"), close)
	form.SetBorder(true).SetTitle(s.tr("bind_title"))
	styleForm(form)
	for i := 0; i < form.GetButtonCount(); i++ {
		clickable(form.GetButton(i))
	}
	info := tview.NewTextView().SetWrap(true).SetText(s.tr("bind_hint"))
	info.SetTextColor(palette.muted).SetBackgroundColor(palette.background)
	root := s.formDialog(form, info, 88)
	s.bindForm = form
	s.pages.AddPage("bind", root, true, true)
	s.app.SetFocus(form)
}

func (s *screen) bindKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		s.bindCancel()
		return nil
	case tcell.KeyF1:
		s.showHelp()
		return nil
	case tcell.KeyEnter:
		if event.Modifiers()&(tcell.ModCtrl|tcell.ModAlt) != 0 {
			s.bindSubmit()
			return nil
		}
	}
	return event
}
