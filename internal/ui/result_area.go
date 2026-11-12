package ui

import (
	"fmt"
	"lazyora/internal/oracle"
	"time"

	"github.com/leechis7/tuivory/go"
)

type executionNote struct {
	key  string
	args []any
}

type resultAreaState struct {
	section           string
	resultNew, msgNew bool
	msgError          bool
	note              executionNote
}

type resultArea struct {
	root, body  *tuivory.Flex
	tab         *tuivory.TabView
	state       tuivory.TabState
	titles      [3]string
	controls    []tuivory.Primitive
	message     *tuivory.WidgetTextView
	returnFocus tuivory.Primitive
}

var resultSections = [3]string{"grid", "messages", "output"}

func (s *screen) resultPane() *tuivory.Flex {
	if s.resultArea == nil {
		w := &resultArea{body: s.newFlex().SetDirection(tuivory.FlexRow), state: tuivory.TabState{Active: "grid", Focused: "grid"}}
		s.resultArea = w
		model := s.resultAreaTabs()
		w.tab, _ = tuivory.NewTabView(model, &w.state, s.componentPalette())
		w.tab.ASCII = s.asciiDisplay()
		w.tab.OnEvent = func(event tuivory.TabEvent) {
			if event.Kind == "activate" {
				s.selectResultSection(event.ID, false)
			}
		}
		w.root = s.newFlex().SetDirection(tuivory.FlexRow).AddItem(w.tab, 1, 0, false).AddItem(w.body, 0, 1, true)
		s.rebuildResultBody()
	}
	w := s.resultArea
	if s.fullscreen {
		w.root.ResizeItem(w.tab, 0, 0)
		w.tab.SetRect(0, 0, 0, 0)
	} else {
		w.root.ResizeItem(w.tab, 1, 0)
	}
	return w.root
}

func (s *screen) resultSectionVisible(section string) bool {
	return s.resultSection() == section && (s.maximized == "results" || s.maximized == "" && !s.resultCollapsed())
}

func (s *screen) resultSection() string {
	if s.resultState.section == "" {
		return "grid"
	}
	return s.resultState.section
}

func (s *screen) resultAreaTabs() *tuivory.Tabs {
	w := s.resultArea
	var items []tuivory.TabItem
	titles := s.resultAreaTitles()
	for i, id := range resultSections {
		items = append(items, tuivory.TabItem{ID: id, Title: titles[i]})
		w.titles[i] = items[i].Title
	}
	model, _ := tuivory.NewTabs(items)
	return model
}

func (s *screen) resultAreaTitles() [3]string {
	titles := [3]string{s.tr("result_area_grid"), s.tr("result_area_messages"), "DBMS_OUTPUT"}
	if s.resultState.resultNew {
		titles[0] += " *"
	}
	if s.resultState.msgError {
		titles[1] += " !"
	} else if s.resultState.msgNew {
		titles[1] += " *"
	}
	if s.outputError != "" {
		titles[2] += " !"
	} else if s.outputNew {
		titles[2] += " *"
	}
	return titles
}

func (s *screen) refreshResultAreaTabs() {
	if w := s.resultArea; w != nil && w.titles != s.resultAreaTitles() {
		_ = w.tab.SetTabs(s.resultAreaTabs())
	}
}

func (s *screen) resetResultPaneForTab() {
	if w := s.resultArea; w != nil {
		w.returnFocus = nil
		s.rebuildResultBody()
	}
}

func (s *screen) rebuildResultBody() {
	w := s.resultArea
	if w == nil {
		return
	}
	w.body.Clear()
	w.message = nil
	s.outputText, s.outputInfo, s.outputToggle = nil, nil, nil
	s.scriptList, s.scriptResult, s.scriptText = nil, nil, nil
	switch s.resultSection() {
	case "messages":
		w.controls = s.buildExecutionMessages(w.body)
	case "output":
		w.controls = s.buildOutput(w.body)
	default:
		w.body.AddItem(s.resultGrid(), 0, 1, true)
		w.controls = []tuivory.Primitive{s.table}
	}
	w.state.Active, w.state.Focused = s.resultSection(), s.resultSection()
	s.refreshResultAreaTabs()
}

func (s *screen) selectResultSection(section string, enter bool) {
	s.resultPane()
	w := s.resultArea
	if !s.resultAreaHasFocus() {
		w.returnFocus = s.app.GetFocus()
	}
	if section != s.resultSection() {
		s.resultState.section = section
		s.rebuildResultBody()
	}
	switch section {
	case "grid":
		s.resultState.resultNew = false
	case "messages":
		s.resultState.msgNew = false
	case "output":
		s.outputNew = false
	}
	s.refreshResultAreaTabs()
	if s.maximized == "sql" || s.maximized == "" && s.resultCollapsed() {
		s.focusPanel("results")
	}
	if enter || s.fullscreen {
		s.app.SetFocus(s.resultAreaContent())
	} else {
		s.app.SetFocus(w.tab)
	}
	s.refreshFooter()
}

func (s *screen) resultAreaContent() tuivory.Primitive {
	if w := s.resultArea; w != nil && len(w.controls) > 0 {
		return w.controls[0]
	}
	return s.table
}

func (s *screen) resultAreaHasFocus() bool {
	if s.table.HasFocus() {
		return true
	}
	if w := s.resultArea; w != nil {
		if w.tab.HasFocus() {
			return true
		}
		for _, c := range w.controls {
			if c.HasFocus() {
				return true
			}
		}
	}
	return false
}

func (s *screen) closeResultSection() {
	w := s.resultArea
	if w == nil {
		return
	}
	focus := w.returnFocus
	s.selectResultSection("grid", true)
	w.returnFocus = nil
	if focus != nil && s.maximized == "" {
		s.app.SetFocus(focus)
	}
}

func (s *screen) resultAreaKey(event *tuivory.EventKey) bool {
	w := s.resultArea
	if w == nil || !s.resultAreaHasFocus() {
		return false
	}
	section := s.resultSection()
	if event.Key() == tuivory.KeyEscape && !s.fullscreen && section != "grid" {
		s.closeResultSection()
		return true
	}
	if event.Modifiers() & ^tuivory.ModShift != 0 && (event.Key() == tuivory.KeyTab || event.Key() == tuivory.KeyBacktab) {
		return false
	}
	if event.Key() == tuivory.KeyBacktab && !s.fullscreen && !w.tab.HasFocus() {
		s.app.SetFocus(w.tab)
		return true
	}
	if event.Key() == tuivory.KeyTab {
		if w.tab.HasFocus() {
			s.app.SetFocus(s.resultAreaContent())
			return true
		}
		for i, c := range w.controls {
			if !c.HasFocus() {
				continue
			}
			for _, next := range w.controls[i+1:] {
				if button, ok := next.(*tuivory.WidgetButton); !ok || !button.IsDisabled() {
					s.app.SetFocus(next)
					return true
				}
			}
			break // let normal panel navigation leave the result area
		}
	}
	if w.tab.HasFocus() {
		return false
	}
	switch section {
	case "output":
		return s.outputKey(event) == nil
	case "messages":
		return s.scriptKey(event) == nil
	}
	return false
}

func (s *screen) recordResultExecution(action string, result oracle.Result, report oracle.ScriptReport, err error) {
	if action != "execute" && action != "script" {
		return
	}
	state := &s.resultState
	state.msgError = err != nil
	state.msgNew = !s.resultSectionVisible("messages")
	hasRows := len(result.Columns) > 0
	switch {
	case err != nil:
		state.note = executionNote{"error", []any{err}}
	case action == "script":
		state.note = executionNote{"script_done", []any{len(report.Steps), report.Total}}
	case hasRows:
		key := "rows"
		if result.Truncated {
			key = "rows_limited"
		}
		state.note = executionNote{key, []any{len(result.Rows), result.Elapsed.Round(time.Millisecond)}}
	default:
		state.note = executionNote{"executed", []any{result.Affected, result.Elapsed.Round(time.Millisecond)}}
	}
	for _, step := range report.Steps {
		hasRows = hasRows || len(step.Result.Columns) > 0
	}
	if hasRows {
		state.resultNew = !s.resultSectionVisible("grid")
	}
	if s.resultSection() == "messages" && s.resultArea != nil {
		if action == "script" {
			focus := s.resultAreaHasFocus() && !s.resultArea.tab.HasFocus()
			s.rebuildResultBody()
			if focus {
				s.app.SetFocus(s.resultAreaContent())
			}
		} else if message := s.resultArea.message; message != nil {
			message.SetText(s.resultExecutionText()).ScrollToBeginning()
		}
	}
	s.refreshResultAreaTabs()
}

func (s *screen) resultExecutionText() string {
	note := s.resultState.note
	if note.key != "" {
		return fmt.Sprintf(s.tr(note.key), note.args...)
	}
	return s.tr("result_area_no_messages")
}

func (s *screen) localizeResultArea() {
	s.refreshResultAreaTabs()
	w := s.resultArea
	if w == nil {
		return
	}
	if s.outputText != nil {
		s.refreshOutput()
		w.controls[3].(*tuivory.WidgetButton).SetLabel(s.tr("output_copy"))
		w.controls[4].(*tuivory.WidgetButton).SetLabel(s.tr("output_clear"))
	}
	if w.message != nil {
		row, col := w.message.GetScrollOffset()
		w.message.SetTitle(s.tr("result_area_messages"))
		w.message.SetText(s.resultExecutionText()).ScrollTo(row, col)
	}
	if s.scriptList != nil {
		s.scriptList.SetTitle(s.tr("script_report"))
		s.scriptResult.SetTitle(s.tr("results_title"))
		for i, step := range s.scriptReport.Steps {
			s.scriptList.SetItemText(i, s.scriptStepLabel(i, step), "")
		}
		w.controls[3].(*tuivory.WidgetButton).SetLabel(s.tr("script_use_result"))
	}
}

func (s *screen) buildExecutionMessages(body *tuivory.Flex) []tuivory.Primitive {
	status := s.newTextView().SetDynamicColors(false).SetScrollable(true).SetWrap(true).SetText(s.resultExecutionText())
	s.resultArea.message = status
	status.SetBorder(true).SetTitle(s.tr("result_area_messages"))
	s.styleBox(status.Box)
	if len(s.scriptReport.Steps) == 0 {
		body.AddItem(status, 0, 1, true)
		return []tuivory.Primitive{status}
	}
	body.AddItem(status, 3, 0, false)
	return append(s.buildScriptReport(body), status)
}
