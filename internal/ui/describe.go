package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"strings"
	"unicode"
	"unicode/utf8"
)

func objectAtCursor(editor *tview.TextArea) string {
	selected, start, _ := editor.GetSelection()
	if selected != "" {
		return strings.TrimSpace(selected)
	}
	text := editor.GetText()
	start = min(start, len(text))
	from, to := start, start
	valid := func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("_$#.\"", r)
	}
	for from > 0 {
		r, size := utf8.DecodeLastRuneInString(text[:from])
		if !valid(r) {
			break
		}
		from -= size
	}
	for to < len(text) {
		r, size := utf8.DecodeRuneInString(text[to:])
		if !valid(r) {
			break
		}
		to += size
	}
	return text[from:to]
}

func (s *screen) describeObject() { s.describe(false) }

func (s *screen) describe(prompt bool) {
	reference := objectAtCursor(s.editor)
	if s.browser != nil && s.browser.HasFocus() {
		if node := s.browser.GetCurrentNode(); node != nil {
			if data, ok := node.GetReference().(*browserNode); ok && data.kind == "object" {
				if !prompt {
					s.openObject(data.object)
					return
				}
				reference = oracle.QualifiedObject(data.object)
			}
		}
	}
	s.describeFocus = s.app.GetFocus()
	form := tview.NewForm().AddInputField(s.tr("object_reference"), reference, 40, nil, nil)
	s.describeForm = form
	load := func() {
		reference := strings.TrimSpace(form.GetFormItem(0).(*tview.InputField).GetText())
		if _, _, err := oracle.ParseObjectReference(reference); err != nil {
			form.SetTitle(s.tr("object_reference_error"))
			return
		}
		if s.demo && !s.connected {
			_, name, _ := oracle.ParseObjectReference(reference)
			for _, object := range demoObjects("LAZYORA") {
				if object.Name == name {
					s.closeDescribe()
					s.openObject(object)
					return
				}
			}
			form.SetTitle(s.tr("object_not_found"))
			return
		}
		s.closeDescribe()
		s.submit(job{action: "describe", text: reference})
	}
	s.describeAction = load
	form.AddButton(s.tr("describe"), load).AddButton(s.tr("cancel"), s.closeDescribe)
	form.SetBorder(true).SetTitle(s.tr("describe"))
	styleForm(form)
	clickable(form.GetButton(0))
	clickable(form.GetButton(1))
	form.GetFormItem(0).SetFinishedFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			load()
		}
	})
	s.pages.AddPage("describe", form, true, true)
	s.app.SetFocus(form)
	// F4 on a selected/under-cursor object resolves immediately.
	if !prompt && reference != "" {
		if _, _, err := oracle.ParseObjectReference(reference); err == nil && !strings.EqualFold(reference, "SELECT") {
			load()
		}
	}
}

func (s *screen) closeDescribe() {
	s.pages.RemovePage("describe")
	s.describeForm = nil
	s.describeAction = nil
	if s.describeFocus != nil {
		s.app.SetFocus(s.describeFocus)
	}
}
