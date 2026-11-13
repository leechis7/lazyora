package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"lazyora/internal/oracle"

	"github.com/leechis7/tuivory/go"
)

type debuggerView struct {
	form    *tuivory.WidgetForm
	info    *tuivory.WidgetTextView
	source  *tuivory.WidgetTextView
	focus   tuivory.Primitive
	actions chan oracle.DebugAction
	paused  bool
}

func (s *screen) showDebugger() {
	v := &debuggerView{form: tuivory.NewWidgetForm(), info: s.newTextView().SetDynamicColors(false).SetWrap(true).SetScrollable(true), focus: s.app.GetFocus()}
	s.debugger = v
	v.source = s.newTextView().SetDynamicColors(false).SetWrap(false).SetScrollable(true)
	v.source.SetBorder(true).SetTitle(s.tr("source"))
	s.styleBox(v.source.Box)
	v.info.SetBorder(true).SetTitle(s.tr("debug_start"))
	s.styleBox(v.info.Box)
	v.info.SetText(s.connectionBadge() + "\n" + s.tr("debug_intro"))
	v.form.AddInputField(s.tr("debug_procedure"), "", 36, nil, nil).
		AddInputField(s.tr("debug_line"), "", 8, nil, nil).
		AddInputField(s.tr("debug_watch"), "", 30, nil, nil)
	v.form.AddButton(s.tr("debug_start"), func() {
		if s.busy {
			return
		}
		name := v.input(0)
		if _, _, err := oracle.ParseObjectReference(name); err != nil {
			v.info.SetText(s.tr("debug_bad_input"))
			return
		}
		line, err := strconv.Atoi(v.input(1))
		if err != nil || line < 1 || line > 1000000 {
			v.info.SetText(s.tr("debug_bad_input"))
			return
		}
		v.actions = make(chan oracle.DebugAction, 1)
		v.info.SetText(s.tr("debug_running"))
		s.submit(job{action: "debug_start", text: name, debugLine: line, debugActions: v.actions})
		v.updateButtons(s.busy)
		if s.busy {
			focusForm(s, v.form, 1)
		}
	})
	for _, item := range []struct{ label, kind string }{
		{"debug_resume", "continue"}, {"debug_over", "over"}, {"debug_into", "into"}, {"debug_out", "out"}, {"debug_watch", "watch"}, {"debug_add_break", "breakpoint"},
	} {
		v.form.AddButton(s.tr(item.label), func() { s.debugAction(item.kind) })
	}
	v.form.AddButton(s.tr("debug_stop"), s.closeDebugger)
	s.styleForm(v.form)
	for i := 0; i < v.form.GetButtonCount(); i++ {
		s.clickable(v.form.GetButton(i))
	}
	v.updateButtons(false)
	top := s.newFlex().AddItem(v.source, 0, 3, false).AddItem(v.info, 0, 2, false)
	root := s.newFlex().SetDirection(tuivory.FlexRow).AddItem(top, 0, 1, false).AddItem(v.form, 11, 0, true)
	s.pages.AddPage("debugger", s.formPopup(root, 106, 35, nil), true, true)
	focusForm(s, v.form, 0)
}

func (v *debuggerView) input(i int) string {
	return v.form.GetFormItem(i).(*tuivory.InputField).GetText()
}

func (v *debuggerView) updateButtons(running bool) {
	v.form.GetButton(0).SetDisabled(running)
	for i := 1; i < v.form.GetButtonCount()-1; i++ {
		v.form.GetButton(i).SetDisabled(!running || !v.paused)
	}
}

func (s *screen) debugAction(kind string) {
	v := s.debugger
	if v == nil || !s.busy || !v.paused {
		return
	}
	line, _ := strconv.Atoi(v.input(1))
	v.paused = false
	v.updateButtons(true)
	v.actions <- oracle.DebugAction{Kind: kind, Line: line, Watch: v.input(2)}
}

func (s *screen) acceptDebugUpdate(state oracle.DebugState, err error) {
	v := s.debugger
	if v == nil {
		return
	}
	v.paused = true
	v.updateButtons(true)
	text := fmt.Sprintf(s.tr("debug_paused"), state.Owner, state.Unit, state.Line, state.Depth)
	if state.Stack != "" {
		text += "\n" + state.Stack
	}
	if state.Watch != "" && err == nil {
		value := state.Value
		if state.Null {
			value = "NULL"
		}
		text += "\n" + state.Watch + ": " + value
	}
	if err != nil {
		text += "\n" + err.Error()
	}
	v.info.SetText(text).ScrollToBeginning()
	var source strings.Builder
	if state.Source == "" {
		source.WriteString(s.tr("debug_source_unavailable"))
	} else {
		for i, line := range strings.Split(strings.TrimSuffix(state.Source, "\n"), "\n") {
			mark := " "
			if i+1 == state.Line {
				mark = ">"
			}
			fmt.Fprintf(&source, "%s %4d  %s\n", mark, i+1, line)
		}
	}
	v.source.SetText(source.String()).ScrollTo(max(0, state.Line-3), 0)
}

func (s *screen) acceptDebugFinish(err error) {
	v := s.debugger
	if v == nil {
		return
	}
	v.paused = false
	v.updateButtons(false)
	key := "debug_finished"
	if err == context.Canceled {
		key = "debug_cancelled"
	} else if err != nil {
		v.info.SetText(fmt.Sprintf(s.tr("error"), err))
		s.setStatus("error", err)
		return
	}
	v.info.SetText(s.tr(key))
	s.setStatus(key)
}

func (s *screen) closeDebugger() {
	v := s.debugger
	if s.busy {
		if v.paused {
			s.debugAction("stop")
		} else if s.cancel != nil {
			s.cancel()
		}
		v.info.SetText(s.tr("debug_stopping"))
		return
	}
	s.pages.RemovePage("debugger")
	s.debugger = nil
	s.app.SetFocus(v.focus)
}

func (s *screen) debuggerKey(e *tuivory.EventKey) *tuivory.EventKey {
	v := s.debugger
	if e.Key() == tuivory.KeyTab && (v.source.HasFocus() || v.info.HasFocus()) {
		if v.source.HasFocus() {
			s.app.SetFocus(v.info)
		} else {
			focusForm(s, v.form, 0)
		}
		return nil
	}
	if e.Key() == tuivory.KeyBacktab && !v.form.HasFocus() {
		if v.info.HasFocus() {
			s.app.SetFocus(v.source)
		} else {
			focusForm(s, v.form, 0)
		}
		return nil
	}
	if e.Key() == tuivory.KeyRune && e.Modifiers() == tuivory.ModAlt {
		switch strings.ToLower(string(e.Rune())) {
		case "v":
			s.app.SetFocus(v.source)
		case "w":
			s.debugAction("watch")
		case "b":
			s.debugAction("breakpoint")
		case "o":
			s.debugAction("out")
		default:
			return e
		}
		return nil
	}
	if e.Modifiers() != 0 {
		return e
	}
	switch e.Key() {
	case tuivory.KeyEscape:
		s.closeDebugger()
	case tuivory.KeyF11:
		s.debugAction("continue")
	case tuivory.KeyF8:
		s.debugAction("over")
	case tuivory.KeyF7:
		s.debugAction("into")
	case tuivory.KeyF1:
		s.showHelp()
	default:
		return e
	}
	return nil
}
