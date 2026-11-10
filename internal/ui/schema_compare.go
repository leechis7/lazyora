package ui

import (
	"fmt"
	"lazyora/internal/oracle"
	"strings"

	"github.com/leechis7/tuivory/go"
)

type schemaCompareView struct {
	form         *tuivory.WidgetForm
	info, detail *tuivory.WidgetTextView
	list         *tuivory.WidgetList
	focus        tuivory.Primitive
	run          *schemaCompareRun
}

type schemaCompareRun struct {
	host       *screen
	targets    [2]*screen
	structures [2]oracle.SchemaStructure
	errors     [2]error
	status     [2]string
	statusArgs [2][]any
	pending    int
	cancelled  bool
}

func (s *screen) compareConnections() []*screen {
	if s.workspaces == nil {
		return nil
	}
	var available []*screen
	for _, other := range s.workspaces.screens {
		if other.connected && !other.busy && !other.workerClosed && !other.transactionUncertain {
			available = append(available, other)
		}
	}
	return available
}

func (s *screen) showSchemaCompare() {
	targets := s.compareConnections()
	if len(targets) < 2 {
		return
	}
	v := &schemaCompareView{
		form:  tuivory.NewWidgetForm(),
		info:  s.newTextView(textHint).SetWrap(true).SetText(s.tr("schema_compare_hint")),
		focus: s.app.GetFocus(),
	}
	s.schemaCompare = v
	labels := make([]string, len(targets))
	left := 0
	for i, target := range targets {
		labels[i] = fmt.Sprintf("#%d · %s", target.workspaceID, target.connectionBadge())
		if target == s {
			left = i
		}
	}
	right := (left + 1) % len(targets)
	v.form.AddFormItem(s.newChoice(s.tr("schema_compare_left"), labels, left, nil)).
		AddInputField(s.tr("schema_compare_owner"), "", 30, nil, nil).
		AddFormItem(s.newChoice(s.tr("schema_compare_right"), labels, right, nil)).
		AddInputField(s.tr("schema_compare_owner"), "", 30, nil, nil)
	start := func() {
		if v.run != nil {
			return
		}
		run := &schemaCompareRun{host: s, pending: 2}
		var owners [2]string
		for side := range owners {
			index, _ := v.form.GetFormItem(2 * side).(*choiceField).GetCurrentOption()
			if index < 0 || index >= len(targets) {
				return
			}
			run.targets[side] = targets[index]
			raw := strings.TrimSpace(v.form.GetFormItem(2*side + 1).(*tuivory.InputField).GetText())
			if raw != "" {
				prefix, name, err := oracle.ParseObjectReference(raw)
				if err != nil || prefix != "" {
					v.info.SetText(s.tr("schema_compare_invalid_owner"))
					focusForm(s, v.form, 2*side+1)
					return
				}
				owners[side] = name
			}
		}
		if run.targets[0] == run.targets[1] {
			v.info.SetText(s.tr("schema_compare_distinct"))
			return
		}
		for _, target := range run.targets {
			if !target.connected || target.busy || target.workerClosed || target.transactionUncertain {
				v.info.SetText(s.tr("workspace_busy"))
				return
			}
		}
		v.run = run
		v.info.SetText(s.tr("schema_compare_loading"))
		v.form.GetButton(0).SetDisabled(true)
		for side, target := range run.targets {
			run.status[side], run.statusArgs[side] = target.statusKey, target.statusArgs
			target.submit(job{action: "schema_compare", text: owners[side], schemaCompare: run, compareSide: side})
		}
	}
	v.form.AddButton(s.tr("schema_compare_start"), start).AddButton(s.tr("cancel"), s.closeSchemaCompare)
	v.form.SetTitle(s.tr("schema_compare"))
	s.styleForm(v.form)
	for i := 0; i < v.form.GetButtonCount(); i++ {
		s.clickable(v.form.GetButton(i))
	}
	s.pages.AddPage("schema_compare", s.formDialog(v.form, v.info, 100), true, true)
	focusForm(s, v.form, 0)
}

func (s *screen) closeSchemaCompare() {
	v := s.schemaCompare
	if v == nil {
		return
	}
	if v.run != nil {
		v.run.cancelled = true
		for _, target := range v.run.targets {
			if target.cancel != nil {
				target.cancel()
			}
		}
		return
	}
	s.pages.RemovePage("schema_compare_results")
	s.pages.RemovePage("schema_compare")
	s.schemaCompare = nil
	s.app.SetFocus(v.focus)
}

func (r *schemaCompareRun) accept(side int, structure oracle.SchemaStructure, err error) {
	target := r.targets[side]
	target.setStatus(r.status[side], r.statusArgs[side]...)
	r.structures[side], r.errors[side] = structure, err
	r.pending--
	if r.pending > 0 {
		return
	}
	host := r.host
	v := host.schemaCompare
	v.run = nil
	v.form.GetButton(0).SetDisabled(false)
	if r.cancelled {
		v.info.SetText(host.tr("schema_compare_cancelled"))
		return
	}
	for i, err := range r.errors {
		if err != nil {
			sideName := host.tr("schema_compare_left")
			if i == 1 {
				sideName = host.tr("schema_compare_right")
			}
			v.info.SetText(sideName + " · " + fmt.Sprintf(host.tr("error"), err))
			return
		}
	}
	host.renderSchemaCompare(r.structures)
}

func (s *screen) renderSchemaCompare(structures [2]oracle.SchemaStructure) {
	v := s.schemaCompare
	v.list = tuivory.NewWidgetList().ShowSecondaryText(false)
	s.styleList(v.list)
	v.list.SetBorder(true).SetTitle(s.tr("schema_compare"))
	v.detail = s.newTextView().SetDynamicColors(false).SetScrollable(true).SetWrap(false)
	v.detail.SetBorder(true).SetTitle(s.tr("object_details"))
	s.styleBox(v.detail.Box)
	differences := oracle.CompareSchemas(structures[0], structures[1])
	v.list.SetChangedFunc(func(index int, _, _ string, _ rune) {
		if index >= 0 && index < len(differences) {
			v.detail.SetText(s.schemaDifferenceText(differences[index])).ScrollToBeginning()
		}
	})
	v.list.SetSelectedFunc(func(index int, _, _ string, _ rune) {
		if index >= 0 && index < len(differences) {
			s.app.SetFocus(v.detail)
		}
	})
	for _, change := range differences {
		v.list.AddItem(tuivory.Escape(change.Name+" · "+s.schemaDifferenceKind(change)), "", 0, nil)
	}
	if len(differences) > 0 {
		v.detail.SetText(s.schemaDifferenceText(differences[0]))
	} else {
		v.list.AddItem(s.tr("schema_compare_equal"), "", 0, nil)
	}
	title := fmt.Sprintf(s.tr("schema_compare_summary"), structures[0].Owner, len(structures[0].Tables), structures[1].Owner, len(structures[1].Tables), len(differences))
	info := s.newTextView(textHint).SetDynamicColors(false).SetText(title)
	close := s.clickable(tuivory.NewWidgetButton(s.tr("close")).SetSelectedFunc(s.closeSchemaCompareResults))
	panels := s.newFlex().AddItem(v.list, 0, 2, true).AddItem(v.detail, 0, 3, false)
	root := s.newFlex().SetDirection(tuivory.FlexRow).AddItem(info, 2, 0, false).AddItem(panels, 0, 1, true).AddItem(close, 1, 0, false)
	s.pages.AddPage("schema_compare_results", s.formPopup(root, 115, 28, nil), true, true)
	s.app.SetFocus(v.list)
}

func (s *screen) schemaDifferenceKind(change oracle.SchemaTableChange) string {
	switch {
	case change.LeftOnly:
		return s.tr("schema_compare_left_only")
	case change.RightOnly:
		return s.tr("schema_compare_right_only")
	default:
		return s.tr("schema_compare_changed")
	}
}

func (s *screen) schemaDifferenceText(change oracle.SchemaTableChange) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s · %s\n", change.Name, s.schemaDifferenceKind(change))
	describe := func(col *oracle.SchemaColumn) string {
		if col == nil {
			return s.tr("schema_compare_missing")
		}
		null := "NULL"
		if !col.Nullable {
			null = "NOT NULL"
		}
		text := fmt.Sprintf("#%d · %s · %s", col.Position, col.Type, null)
		if col.Default != "" {
			text += " · DEFAULT " + col.Default
		}
		return text
	}
	for _, column := range change.Columns {
		fmt.Fprintf(&b, "\n%s\n%s: %s\n%s: %s\n", column.Name, s.tr("schema_compare_left"), describe(column.Left), s.tr("schema_compare_right"), describe(column.Right))
	}
	return b.String()
}

func (s *screen) closeSchemaCompareResults() {
	v := s.schemaCompare
	s.pages.RemovePage("schema_compare_results")
	v.list, v.detail = nil, nil
	v.info.SetText(s.tr("schema_compare_hint"))
	focusForm(s, v.form, 0)
}

func (s *screen) schemaCompareKey(page string, event *tuivory.EventKey) *tuivory.EventKey {
	v := s.schemaCompare
	switch event.Key() {
	case tuivory.KeyEscape:
		if page == "schema_compare_results" {
			s.closeSchemaCompareResults()
		} else {
			for _, index := range []int{0, 2} {
				if v.form.GetFormItem(index).(*choiceField).IsOpen() {
					return event
				}
			}
			s.closeSchemaCompare()
		}
		return nil
	case tuivory.KeyTab, tuivory.KeyBacktab:
		if page == "schema_compare_results" {
			s.dialogTab(event, v.list, v.detail)
			return nil
		}
	case tuivory.KeyCtrlC:
		if page == "schema_compare_results" && v.detail.HasFocus() {
			s.sendClipboard(v.detail.GetText(false))
			return nil
		}
	}
	return event
}
