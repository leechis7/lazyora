package ui

import (
	"fmt"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
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
		s.recordSQLHistory(text)
		s.submit(job{action: "script", text: text, bindings: values})
	})
}

func (s *screen) showScriptReport() {
	if len(s.scriptReport.Steps) == 0 {
		return
	}
	s.scriptFocus = s.app.GetFocus()
	list := tview.NewList().ShowSecondaryText(false).SetSelectedStyle(focusStyle())
	list.SetBorder(true).SetTitle(s.tr("script_report"))
	styleBox(list.Box)
	table := tview.NewTable().SetFixed(1, 0).SetSelectable(true, true).SetSelectedStyle(focusStyle())
	table.SetBorder(true).SetTitle(s.tr("results_title"))
	styleBox(table.Box)
	preview := tview.NewTextView().SetWrap(false).SetScrollable(true)
	preview.SetBorder(true)
	styleBox(preview.Box)
	preview.SetTextColor(palette.text)
	s.scriptList, s.scriptResult, s.scriptText = list, table, preview
	selectResult := func(index int) {
		step := s.scriptReport.Steps[index]
		preview.SetText(step.Statement.Text + "\n\n" + step.Error)
		table.Clear()
		for col, name := range step.Result.Columns {
			table.SetCell(0, col, tview.NewTableCell(tview.Escape(name)).SetSelectable(false).SetTextColor(palette.accent))
		}
		for row, values := range step.Result.Rows {
			for col, value := range values {
				table.SetCell(row+1, col, tview.NewTableCell(tview.Escape(value)).SetTextColor(palette.text))
			}
		}
	}
	for i, step := range s.scriptReport.Steps {
		status := s.tr("script_ok")
		if step.Error != "" {
			status = s.tr("script_error")
		}
		rows := strconv.Itoa(len(step.Result.Rows))
		if step.Result.Truncated {
			rows += "+"
		}
		list.AddItem(fmt.Sprintf(s.tr("script_step"), i+1, step.Statement.Line, status, rows, step.Result.Affected), "", 0, nil)
	}
	list.SetChangedFunc(func(index int, _, _ string, _ rune) { selectResult(index) })
	selectResult(0)
	use := clickable(tview.NewButton(s.tr("script_use_result")).SetSelectedFunc(func() {
		step := s.scriptReport.Steps[list.GetCurrentItem()]
		if len(step.Result.Columns) > 0 {
			s.render(step.Result)
			s.closeScriptReport()
			s.focusPanel("results")
		}
	}))
	close := clickable(tview.NewButton(s.tr("close")).SetSelectedFunc(s.closeScriptReport))
	buttons := tview.NewFlex().AddItem(use, 0, 1, false).AddItem(close, 0, 1, false)
	root := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(list, 0, 1, true).AddItem(preview, 0, 1, false).AddItem(table, 0, 2, false).AddItem(buttons, 1, 0, false)
	s.pages.AddPage("script", root, true, true)
	s.app.SetFocus(list)
}

func (s *screen) closeScriptReport() {
	s.pages.RemovePage("script")
	s.scriptList = nil
	s.scriptResult = nil
	s.scriptText = nil
	if s.scriptFocus != nil {
		s.app.SetFocus(s.scriptFocus)
	}
}

func (s *screen) scriptKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		s.closeScriptReport()
		return nil
	case tcell.KeyF1:
		s.showHelp()
		return nil
	case tcell.KeyTab, tcell.KeyBacktab:
		s.dialogTab(event, s.scriptList, s.scriptText, s.scriptResult)
		return nil
	case tcell.KeyCtrlC:
		s.sendClipboard(s.scriptReport.Summary())
		return nil
	}
	return event
}
