package ui

import (
	"strings"

	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func (s *screen) clearSQL() {
	if s.editor.GetText() == "" {
		return
	}
	s.showProgramConfirm(s.tr("clear_sql_confirm"), func() {
		s.editor.Replace(0, len(s.editor.GetText()), "")
		s.focusPanel("sql")
	})
}

func (s *screen) executeSQL() {
	text := s.editor.GetText()
	if selected, _, _ := s.editor.GetSelection(); selected != "" {
		text = selected
	} else {
		_, offset, _ := s.editor.GetSelection()
		var err error
		text, err = oracle.AtStatement(text, offset)
		if err != nil {
			s.setStatus("error", err)
			return
		}
	}
	if s.sqlConfirmText != nil || s.busy {
		return
	}
	if s.connected {
		if risk := oracle.ExecutionRisk(text); risk != oracle.RiskNone {
			s.confirmSQL(text, risk)
			return
		}
	}
	s.submitSQL(text)
}

func (s *screen) submitSQL(text string) {
	s.withBinds(text, func(values map[string]oracle.BindValue) {
		s.recordSQLHistory(text)
		s.submit(job{action: "execute", text: text, bindings: values})
	})
}

func (s *screen) recordSQLHistory(text string) {
	if !s.busy && s.connected && strings.TrimSpace(text) != "" {
		if len(s.sqlHistory) == 0 || s.sqlHistory[len(s.sqlHistory)-1] != text {
			s.sqlHistory = append(s.sqlHistory, text)
		}
		if len(s.sqlHistory) > 100 {
			s.sqlHistory = s.sqlHistory[len(s.sqlHistory)-100:]
		}
	}
}

func (s *screen) showHistory() {
	s.historyFocus = s.app.GetFocus()
	list := tview.NewList().ShowSecondaryText(false).SetSelectedStyle(focusStyle())
	list.SetBorder(true).SetTitle(s.tr("history"))
	styleBox(list.Box)
	s.historyList = list
	for i := len(s.sqlHistory) - 1; i >= 0; i-- {
		sql := s.sqlHistory[i]
		summary := strings.Join(strings.Fields(sql), " ")
		list.AddItem(tview.Escape(summary), "", 0, func() {
			s.closeHistory()
			apply := func() { s.editor.SetText(sql, false); s.focusPanel("sql") }
			if s.editor.GetText() != sql && s.editor.GetText() != "" {
				s.showProgramConfirm(s.tr("history_replace_confirm"), apply)
			} else {
				apply()
			}
		})
	}
	if len(s.sqlHistory) == 0 {
		list.AddItem(s.tr("history_empty"), "", 0, nil)
	}
	s.pages.AddPage("history", list, true, true)
	s.app.SetFocus(list)
}

func (s *screen) closeHistory() {
	s.pages.RemovePage("history")
	s.historyList = nil
	if s.historyFocus != nil {
		s.app.SetFocus(s.historyFocus)
	}
}

func (s *screen) showSearch(replace bool) {
	s.searchFocus = s.app.GetFocus()
	form := tview.NewForm().AddInputField(s.tr("find_text"), s.findText, 40, nil, nil)
	if selected, _, _ := s.editor.GetSelection(); selected != "" && !strings.Contains(selected, "\n") {
		form.GetFormItem(0).(*tview.InputField).SetText(selected)
	}
	s.searchForm = form
	if replace {
		form.AddInputField(s.tr("replace_text"), s.replaceText, 40, nil, nil)
	}
	s.searchAction = func(action string) {
		s.findText = form.GetFormItem(0).(*tview.InputField).GetText()
		if s.findText == "" {
			form.SetTitle(s.tr("find_empty"))
			return
		}
		if replace {
			s.replaceText = form.GetFormItem(1).(*tview.InputField).GetText()
		}
		s.closeSearch()
		switch action {
		case "replace":
			selected, start, end := s.editor.GetSelection()
			if selected == s.findText {
				s.editor.Replace(start, end, s.replaceText).Select(start+len(s.replaceText), start+len(s.replaceText))
			}
			s.findSQL(false)
		case "replace_all":
			text := s.editor.GetText()
			count := strings.Count(text, s.findText)
			if count > 0 {
				s.editor.Replace(0, len(text), strings.ReplaceAll(text, s.findText, s.replaceText))
			}
			s.setStatus("replaced", count)
		default:
			s.findSQL(action == "previous")
		}
	}
	for _, entry := range []struct{ label, action string }{{"find_next", "next"}, {"find_previous", "previous"}} {
		action := entry.action
		form.AddButton(s.tr(entry.label), func() { s.searchAction(action) })
	}
	if replace {
		form.AddButton(s.tr("replace_one"), func() { s.searchAction("replace") }).AddButton(s.tr("replace_all"), func() { s.searchAction("replace_all") })
	}
	form.AddButton(s.tr("close"), s.closeSearch)
	form.SetBorder(true).SetTitle(s.tr("find_hint"))
	styleForm(form)
	for i := 0; i < form.GetButtonCount(); i++ {
		clickable(form.GetButton(i))
	}
	s.pages.AddPage("search", form, true, true)
	s.app.SetFocus(form)
}

func (s *screen) closeSearch() {
	s.pages.RemovePage("search")
	s.searchForm = nil
	s.searchAction = nil
	if s.searchFocus != nil {
		s.app.SetFocus(s.searchFocus)
	}
}

func (s *screen) findSQL(previous bool) {
	if s.findText == "" {
		s.showSearch(false)
		return
	}
	text := s.editor.GetText()
	_, start, end := s.editor.GetSelection()
	found := -1
	if previous {
		found = strings.LastIndex(text[:start], s.findText)
		if found < 0 {
			found = strings.LastIndex(text, s.findText)
		}
	} else {
		if at := strings.Index(text[end:], s.findText); at >= 0 {
			found = end + at
		} else {
			found = strings.Index(text, s.findText)
		}
	}
	if found < 0 {
		s.setStatus("find_not_found")
		return
	}
	s.editor.Select(found, found+len(s.findText))
	s.focusPanel("sql")
	s.setStatus("find_match", found+1)
}
