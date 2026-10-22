package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func (s *screen) showRoutine(object oracle.Object, reference string) {
	if s.busy || !s.connected || s.transactionUncertain {
		return
	}
	s.routineFocus = s.app.GetFocus()
	s.routines = nil
	s.routineList = tview.NewList().SetMainTextColor(s.colors().text).SetSecondaryTextColor(s.colors().muted).SetShortcutColor(s.colors().accent).ShowSecondaryText(true).SetSelectedStyle(s.focusStyle())
	s.routineList.SetBorder(true).SetTitle(s.tr("routine_execute"))
	s.styleList(s.routineList)
	s.routineInfo = s.newTextView().SetWrap(true).SetText(s.tr("browser_loading"))
	s.routineInfo.SetBackgroundColor(s.colors().background)
	s.routineInfo.SetTextColor(s.colors().text)
	s.routineList.SetSelectedFunc(func(i int, _, _ string, _ rune) {
		if !s.busy && i < len(s.routines) {
			s.showRoutineInputs(s.routines[i])
		}
	})
	close := s.clickable(tview.NewButton(s.tr("close")).SetSelectedFunc(s.closeRoutines))
	root := s.newFlex().SetDirection(tview.FlexRow).AddItem(s.routineInfo, 4, 0, false).AddItem(s.routineList, 0, 1, true).AddItem(close, 1, 0, false)
	s.pages.AddPage("routines", root, true, true)
	s.app.SetFocus(s.routineList)
	s.submit(job{action: "routines", object: object, text: reference})
}
func (s *screen) requestRoutine() {
	if s.browserHasFocus() {
		if node := s.browser.GetCurrentNode(); node != nil {
			if data, ok := node.GetReference().(*browserNode); ok && data.kind == "object" {
				s.showRoutine(data.object, "")
				return
			}
		}
	}
	reference := objectAtCursor(s.editor)
	if strings.TrimSpace(reference) == "" {
		s.setStatus("routine_select")
		return
	}
	s.showRoutine(oracle.Object{}, reference)
}
func (s *screen) acceptRoutines(routines []oracle.Routine) {
	s.routines = routines
	s.routineList.Clear()
	s.routineInfo.SetText(s.tr("routine_hint"))
	if len(routines) == 0 {
		s.routineInfo.SetText(s.tr("arguments_empty"))
		return
	}
	for _, r := range routines {
		var signature []string
		for _, a := range r.Arguments {
			name := a.Name
			if a.Position == 0 {
				name = s.tr("return_value")
			}
			signature = append(signature, fmt.Sprintf("%s %s %s", name, a.Direction, a.DataType))
		}
		label := r.Object.Owner + "." + r.Object.Name
		if r.Object.Kind == "PACKAGE" || r.Object.Kind == "PACKAGE BODY" {
			label += "." + r.Member
		}
		if r.Overload != "-" && r.Overload != "" {
			label += " #" + r.Overload
		}
		s.routineList.AddItem(tview.Escape(label), tview.Escape(strings.Join(signature, " · ")), 0, nil)
	}
}
func (s *screen) closeRoutines() {
	if s.busy {
		return
	}
	s.pages.RemovePage("routines")
	s.routineList = nil
	s.routines = nil
	if s.routineFocus != nil {
		s.app.SetFocus(s.routineFocus)
	}
}
func (s *screen) closeRoutineInputs() {
	if s.busy {
		return
	}
	s.pages.RemovePage("routine_input")
	s.routineForm = nil
	s.routineSubmit = nil
	s.app.SetFocus(s.routineList)
}
func (s *screen) showRoutineInputs(r oracle.Routine) {
	form := tview.NewForm().SetItemPadding(0)
	type field struct {
		a     oracle.RoutineArgument
		mode  *choiceField
		value *tview.InputField
	}
	var fields []field
	var signatures []string
	for _, a := range r.Arguments {
		name := a.Name
		if a.Position == 0 {
			name = s.tr("return_value")
		}
		signatures = append(signatures, name+" · "+a.Direction+" · "+a.DataType)
		if _, err := a.InputKind(); err != nil {
			s.routineInfo.SetText(err.Error())
			return
		}
		if a.Position == 0 || a.Direction == "OUT" {
			continue
		}
		modes := []string{s.tr("bind_value"), "NULL"}
		selected := 1
		if a.Defaulted && a.Direction == "IN" && (r.Overload == "-" || r.Overload == "") {
			modes = append(modes, s.tr("routine_default"))
			selected = 2
		}
		mode := s.newChoice(name+" · "+a.DataType, modes, selected, nil)
		value := s.newInputField().SetLabel(s.tr("bind_value")).SetFieldWidth(40)
		form.AddFormItem(mode).AddFormItem(value)
		fields = append(fields, field{a, mode, value})
	}
	close := s.closeRoutineInputs
	s.routineSubmit = func() {
		if s.busy {
			return
		}
		values := map[int]oracle.RoutineInput{}
		for _, f := range fields {
			mode, _ := f.mode.GetCurrentOption()
			values[f.a.Position] = oracle.RoutineInput{Value: oracle.BindValue{Text: f.value.GetText(), Null: mode == 1}, Default: mode == 2}
		}
		if err := oracle.ValidateRoutineInputs(r, values); err != nil {
			form.SetTitle(tview.Escape(err.Error()))
			return
		}
		s.showProgramConfirm(s.tr("routine_confirm"), func() {
			s.submit(job{action: "routine_run", routine: r, routineInputs: values})
		})
	}
	form.AddButton(s.tr("execute")+" (F9)", s.routineSubmit).AddButton(s.tr("cancel"), close)
	form.SetBorder(true).SetTitle(s.tr("routine_execute") + " · " + tview.Escape(r.Member))
	s.styleForm(form)
	for i := 0; i < form.GetButtonCount(); i++ {
		s.clickable(form.GetButton(i))
	}
	info := s.newTextView().SetWrap(true).SetText(strings.Join(signatures, "\n") + "\n\n" + s.tr("routine_hint"))
	info.SetTextColor(s.colors().text).SetBackgroundColor(s.colors().background)
	root := s.formDialog(form, info, 96)
	s.routineForm = form
	s.pages.AddPage("routine_input", root, true, true)
	s.app.SetFocus(form)
}
func (s *screen) showRoutineResults(result oracle.RoutineResult) {
	s.routineResultsFocus = s.app.GetFocus()
	s.routineResults = result
	s.routineResultList = tview.NewList().SetMainTextColor(s.colors().text).SetSecondaryTextColor(s.colors().muted).SetShortcutColor(s.colors().accent).ShowSecondaryText(true).SetSelectedStyle(s.focusStyle())
	list := s.routineResultList
	list.SetBorder(true).SetTitle(s.tr("routine_results") + " · " + tview.Escape(result.Routine))
	s.styleList(list)
	if len(result.Scalars.Rows) > 0 {
		list.AddItem(s.tr("routine_scalars"), fmt.Sprintf("%d", len(result.Scalars.Rows)), 0, nil)
	}
	for _, cursor := range result.Cursors {
		label := cursor.Name
		if cursor.Null {
			label += " · NULL"
		}
		count := fmt.Sprint(len(cursor.Result.Rows))
		if cursor.Result.Truncated {
			count += "+"
		}
		list.AddItem(tview.Escape(label), count+" · "+s.tr("results_title"), 0, nil)
	}
	use := func() {
		if list.GetItemCount() == 0 {
			return
		}
		index := list.GetCurrentItem()
		selected := result.Scalars
		if len(result.Scalars.Rows) > 0 {
			index--
		}
		if index >= 0 {
			selected = result.Cursors[index].Result
		}
		// Use the existing grid, clipboard and loaded-row export paths. A routine
		// result never has replay SQL: exporting must not call the routine again.
		s.render(selected)
		closeGrid := func() { s.pages.RemovePage("routine_grid"); s.app.SetFocus(list) }
		buttons := s.newFlex().AddItem(s.clickable(tview.NewButton(s.tr("export")).SetSelectedFunc(s.showExport)), 0, 1, false).
			AddItem(s.clickable(tview.NewButton(s.tr("copy_row")).SetSelectedFunc(func() { s.copyResults("row") })), 0, 1, false).
			AddItem(s.clickable(tview.NewButton(s.tr("close")).SetSelectedFunc(closeGrid)), 0, 1, false)
		grid := s.newFlex().SetDirection(tview.FlexRow).AddItem(s.resultGrid(), 0, 1, true).AddItem(buttons, 1, 0, false)
		s.pages.AddPage("routine_grid", grid, true, true)
		s.app.SetFocus(s.table)
	}
	list.SetSelectedFunc(func(int, string, string, rune) { use() })
	buttons := s.newFlex().AddItem(s.clickable(tview.NewButton(s.tr("script_use_result")).SetSelectedFunc(use)), 0, 1, false).
		AddItem(s.clickable(tview.NewButton(s.tr("close")).SetSelectedFunc(func() { s.closeRoutineResults() })), 0, 1, false)
	hint := s.tr("routine_result_hint")
	if list.GetItemCount() == 0 {
		hint = s.tr("routine_no_outputs") + "\n" + hint
	}
	info := s.newTextView().SetText(hint).SetWrap(true)
	info.SetTextColor(s.colors().text).SetBackgroundColor(s.colors().background)
	root := s.newFlex().SetDirection(tview.FlexRow).AddItem(info, 4, 0, false).AddItem(list, 0, 1, true).AddItem(buttons, 1, 0, false)
	s.pages.AddPage("routine_results", root, true, true)
	s.app.SetFocus(list)
}
func (s *screen) closeRoutineResults() {
	s.pages.RemovePage("routine_results")
	s.routineResultList = nil
	if s.routineResultsFocus != nil {
		s.app.SetFocus(s.routineResultsFocus)
	} else {
		s.focusPanel("results")
	}
}
func (s *screen) routineKey(event *tcell.EventKey, page string) *tcell.EventKey {
	if event.Key() == tcell.KeyF1 {
		s.showHelp()
		return nil
	}
	if s.busy {
		if event.Key() == tcell.KeyEscape {
			s.action("cancel")
		}
		return nil
	}
	if page == "routine_grid" {
		if s.table.HasFocus() {
			switch event.Key() {
			case tcell.KeyEnter:
				s.showCellDetail()
				return nil
			case tcell.KeyUp, tcell.KeyDown, tcell.KeyLeft, tcell.KeyRight:
				if event.Modifiers()&tcell.ModShift != 0 {
					s.extendResultSelection(event.Key())
					return nil
				}
				s.clearResultSelection()
			case tcell.KeyEscape:
				if s.resultSelection.active {
					s.clearResultSelection()
					return nil
				}
			}
		}
		if event.Key() == tcell.KeyCtrlC {
			s.copyResults("selection")
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Modifiers()&tcell.ModAlt != 0 {
			if event.Rune() == 'w' {
				s.showExport()
				return nil
			}
			if event.Rune() == 'y' {
				s.copyResults("row")
				return nil
			}
		}
	}
	switch event.Key() {
	case tcell.KeyEscape:
		switch page {
		case "routines":
			s.closeRoutines()
		case "routine_input":
			s.closeRoutineInputs()
		case "routine_results":
			s.closeRoutineResults()
		case "routine_grid":
			s.pages.RemovePage("routine_grid")
			s.app.SetFocus(s.routineResultList)
		}
		return nil
	case tcell.KeyTab, tcell.KeyBacktab:
		if page == "routine_grid" {
			s.dialogTab(event, s.table)
			return nil
		}
		if page == "routines" {
			s.dialogTab(event, s.routineList)
			return nil
		}
		if page == "routine_results" {
			s.dialogTab(event, s.routineResultList)
			return nil
		}
	case tcell.KeyF9:
		if page == "routine_input" && s.routineSubmit != nil {
			s.routineSubmit()
			return nil
		}
	}
	return event
}
