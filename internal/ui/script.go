package ui

import (
	"fmt"
	"lazyora/internal/oracle"
	"strconv"

	"github.com/leechis7/tuivory/go"
)

func (s *screen) executeScript() {
	if s.busy || !s.connected {
		return
	}
	text := s.editor.GetText()
	if selected, _, _ := s.editor.GetSelection(); selected != "" {
		text = selected
	}
	if _, err := oracle.ScriptStatements(text); err != nil {
		s.setStatus("error", err)
		return
	}
	s.confirmExecution(text, "script", s.submitScript)
}

func (s *screen) submitScript(text string) {
	s.withBinds(text, func(values map[string]oracle.BindValue) {
		s.submit(job{action: "script", text: text, bindings: values})
	})
}

func (s *screen) showScriptReport() { s.selectResultSection("messages", true) }

func (s *screen) buildScriptReport(body *tuivory.Flex) []tuivory.Primitive {
	list := tuivory.NewWidgetList().SetMainTextColor(s.colors().text).SetSecondaryTextColor(s.colors().muted).SetShortcutColor(s.colors().accent).ShowSecondaryText(false).SetSelectedStyle(s.focusStyle())
	list.SetBorder(true).SetTitle(s.tr("script_report"))
	s.styleList(list)
	table := s.newDataTable().SetFixed(1, 0).SetSelectable(true, true).SetSelectedStyle(s.focusStyle())
	table.SetBorder(true).SetTitle(s.tr("results_title"))
	s.styleBox(table.Box)
	preview := s.newTextView().SetDynamicColors(false).SetWrap(false).SetScrollable(true)
	preview.SetBorder(true)
	s.styleBox(preview.Box)
	preview.SetTextColor(s.colors().text)
	s.scriptList, s.scriptResult, s.scriptText = list, table, preview
	selectResult := func(index int) {
		step := s.scriptReport.Steps[index]
		preview.SetText(step.Statement.Text + "\n\n" + step.Error)
		table.Clear()
		for col, name := range step.Result.Columns {
			table.SetCell(0, col, tuivory.NewTableCell(tuivory.Escape(name)).SetSelectable(false).SetTextColor(s.colors().accent))
		}
		for row, values := range step.Result.Rows {
			for col, value := range values {
				table.SetCell(row+1, col, tuivory.NewTableCell(tuivory.Escape(value)).SetTextColor(s.colors().text))
			}
		}
	}
	for i, step := range s.scriptReport.Steps {
		list.AddItem(s.scriptStepLabel(i, step), "", 0, nil)
	}
	list.SetChangedFunc(func(index int, _, _ string, _ rune) {
		if index >= 0 && index < len(s.scriptReport.Steps) {
			selectResult(index)
		}
	})
	selectResult(0)
	use := s.clickable(tuivory.NewWidgetButton(s.tr("script_use_result")).SetSelectedFunc(func() {
		if s.busy {
			return
		}
		if s.gridDraftCount() != 0 {
			s.setStatus("grid_resolve_drafts")
			return
		}
		step := s.scriptReport.Steps[list.GetCurrentItem()]
		if len(step.Result.Columns) > 0 {
			s.render(step.Result)
			s.selectResultSection("grid", true)
		}
	}))
	upper := s.newFlex().AddItem(list, 0, 2, true).AddItem(preview, 0, 3, false)
	body.AddItem(upper, 0, 1, true).AddItem(table, 0, 1, false).AddItem(use, 1, 0, false)
	return []tuivory.Primitive{list, preview, table, use}
}

func (s *screen) scriptStepLabel(index int, step oracle.ScriptStep) string {
	status := s.tr("script_ok")
	if step.Error != "" {
		status = s.tr("script_error")
	}
	rows := strconv.Itoa(len(step.Result.Rows))
	if step.Result.Truncated {
		rows += "+"
	}
	return fmt.Sprintf(s.tr("script_step"), index+1, step.Statement.Line, status, rows, step.Result.Affected)
}

func (s *screen) scriptKey(event *tuivory.EventKey) *tuivory.EventKey {
	if s.listDetailArrow(event, s.scriptList, s.scriptText) {
		return nil
	}
	switch event.Key() {
	case tuivory.KeyF1:
		s.showHelp()
		return nil
	case tuivory.KeyCtrlC:
		if len(s.scriptReport.Steps) > 0 {
			s.sendClipboard(s.scriptReport.Summary())
		} else if note := s.resultState.note; note.key != "" {
			s.sendClipboard(fmt.Sprintf(s.tr(note.key), note.args...))
		}
		return nil
	}
	return event
}
