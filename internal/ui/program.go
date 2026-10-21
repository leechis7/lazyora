package ui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func (s *screen) sourceDirty() bool {
	return s.sourceEditor != nil && s.sourceLoaded && s.sourceEditor.GetText() != s.sourceBase
}

func (s *screen) showProgramConfirm(message string, action func()) {
	if s.programModal != nil || s.busy {
		return
	}
	s.programFocus = s.app.GetFocus()
	s.programModal = tview.NewModal().SetText(message).AddButtons([]string{s.tr("proceed"), s.tr("cancel")}).
		SetBackgroundColor(palette.background).SetTextColor(palette.text).
		SetButtonStyle(buttonStyle()).SetButtonActivatedStyle(buttonFocusStyle()).SetFocus(1)
	styleBox(s.programModal.Box)
	s.programModal.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftDoubleClick {
			action = tview.MouseLeftClick
		}
		return action, event
	})
	s.programModal.SetDoneFunc(func(index int, _ string) {
		s.pages.RemovePage("program_confirm")
		s.programModal = nil
		if s.programFocus != nil {
			s.app.SetFocus(s.programFocus)
		}
		if index == 0 {
			action()
		}
	})
	s.pages.AddPage("program_confirm", s.programModal, true, true)
	s.app.SetFocus(s.programModal)
}

func (s *screen) guardSourceDiscard(action func()) {
	if s.sourceDirty() {
		s.showProgramConfirm(s.tr("discard_source"), action)
	} else {
		action()
	}
}

func (s *screen) showSource() {
	if s.busy || !s.object.HasSource() {
		return
	}
	s.sourceBase, s.sourceLoaded = "", false
	s.sourceInfo = tview.NewTextView().SetWrap(true)
	s.sourceInfo.SetBackgroundColor(palette.background)
	s.sourceInfo.SetTextColor(palette.text)
	s.sourceInfo.SetText(s.tr("browser_loading"))
	s.sourceEditor = tview.NewTextArea()
	s.sourceEditor.SetBorder(true)
	styleBox(s.sourceEditor.Box)
	s.sourceEditor.SetBackgroundColor(palette.field).SetTitleColor(palette.background)
	s.sourceEditor.SetTextStyle(fieldStyle()).SetSelectedStyle(focusStyle())
	s.colorEditor(s.sourceEditor, s.refreshSourceTitle)
	s.sourceEditor.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if !s.sourceLoaded || s.busy {
			return nil
		}
		return event
	})
	s.sourceDiagnostics = tview.NewTable().SetFixed(1, 0).SetSelectable(true, false).SetSelectedStyle(focusStyle())
	s.sourceDiagnostics.SetBorder(true)
	styleBox(s.sourceDiagnostics.Box)
	s.renderDiagnostics(nil)
	buttons := tview.NewFlex()
	for _, item := range []struct {
		label  string
		action func()
	}{
		{"compile_button", s.requestCompile}, {"format_code", func() { s.app.SetFocus(s.sourceEditor); s.formatCode() }}, {"reload_button", s.reloadSource}, {"close", s.closeSource},
	} {
		buttons.AddItem(clickable(tview.NewButton(s.tr(item.label)).SetSelectedFunc(item.action)), 0, 1, false)
	}
	root := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(s.sourceInfo, 2, 0, false).
		AddItem(s.sourceEditor, 0, 3, true).AddItem(s.sourceDiagnostics, 0, 1, false).AddItem(buttons, 1, 0, false)
	s.pages.AddPage("source", root, true, true)
	s.app.SetFocus(s.sourceEditor)
	s.refreshSourceTitle()
	s.loadSource()
}

func (s *screen) refreshSourceTitle() {
	if s.sourceEditor == nil {
		return
	}
	mark := ""
	if s.sourceDirty() {
		mark = " *"
	}
	s.sourceEditor.SetTitle(" " + s.tr("source") + " " + tview.Escape(s.object.Owner+"."+s.object.Name) + mark + " ")
}

func (s *screen) loadSource() {
	if s.demo && !s.connected {
		s.acceptSource("CREATE OR REPLACE " + s.object.Kind + " \"LAZYORA\".\"" + s.object.Name + "\"\nAS\nBEGIN\n    NULL;\nEND;")
		return
	}
	s.submit(job{action: "source", object: s.object})
}

func (s *screen) acceptSource(source string) {
	s.sourceBase, s.sourceLoaded = source, true
	s.sourceEditor.SetText(source, false)
	s.sourceInfo.SetText(s.tr("source_ready"))
	s.refreshSourceTitle()
}

func (s *screen) reloadSource() {
	if !s.busy {
		s.guardSourceDiscard(s.loadSource)
	}
}

func (s *screen) closeSource() {
	if s.busy {
		return
	}
	s.guardSourceDiscard(func() {
		s.pages.RemovePage("source")
		s.sourceEditor, s.sourceLoaded = nil, false
		s.app.SetFocus(s.objectTable)
	})
}

func (s *screen) requestCompile() {
	if s.busy || !s.sourceLoaded {
		return
	}
	if s.demo && !s.connected {
		s.sourceInfo.SetText(s.tr("connect_first"))
		return
	}
	text := ""
	if s.sourceDirty() {
		text = s.sourceEditor.GetText()
		if _, err := oracle.SourceDDL(s.object, text); err != nil {
			s.sourceInfo.SetText(err.Error())
			return
		}
	}
	s.showProgramConfirm(s.tr("compile_confirm"), func() {
		s.sourceInfo.SetText(s.tr("compiling"))
		s.submit(job{action: "compile_source", object: s.object, text: text})
	})
}

func (s *screen) renderDiagnostics(diagnostics []oracle.Diagnostic) {
	s.sourceDiagnostics.Clear().SetTitle(s.tr("diagnostics"))
	for i, key := range []string{"line", "position", "object_type", "message"} {
		s.sourceDiagnostics.SetCell(0, i, tview.NewTableCell(s.tr(key)).SetTextColor(palette.accent).SetSelectable(false))
	}
	for row, diagnostic := range diagnostics {
		for col, text := range []string{fmt.Sprint(diagnostic.Line), fmt.Sprint(diagnostic.Position), diagnostic.Attribute, diagnostic.Text} {
			s.sourceDiagnostics.SetCell(row+1, col, tview.NewTableCell(tview.Escape(text)).SetTextColor(palette.text))
		}
	}
}

func (s *screen) showArguments() {
	if s.busy || !s.object.HasArguments() {
		return
	}
	s.argumentFocus = s.app.GetFocus()
	s.argumentInfo = tview.NewTextView().SetText(s.object.Owner + "." + s.object.Name + " — " + s.tr("browser_loading")).SetWrap(true)
	s.argumentInfo.SetBackgroundColor(palette.background)
	s.argumentInfo.SetTextColor(palette.text)
	s.argumentTable = tview.NewTable().SetFixed(1, 0).SetSelectable(true, true).SetSelectedStyle(focusStyle())
	s.argumentTable.SetBorder(true)
	styleBox(s.argumentTable.Box)
	s.argumentTable.SetTitle(s.tr("arguments"))
	close := clickable(tview.NewButton(s.tr("close") + " (Esc)").SetSelectedFunc(s.closeArguments))
	root := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(s.argumentInfo, 2, 0, false).
		AddItem(s.argumentTable, 0, 1, true).AddItem(tview.NewFlex().AddItem(clickable(tview.NewButton(s.tr("routine_button")).SetSelectedFunc(func() { s.showRoutine(s.object, "") })), 0, 1, false).AddItem(close, 0, 1, false), 1, 0, false)
	s.pages.AddPage("arguments", root, true, true)
	s.app.SetFocus(s.argumentTable)
	if s.demo && !s.connected {
		s.renderArguments([]oracle.Argument{{Member: s.object.Name, Name: "P_VALUE", DataType: "NUMBER", Direction: "IN", Position: 1}})
	} else {
		s.submit(job{action: "arguments", object: s.object})
	}
}

func (s *screen) renderArguments(arguments []oracle.Argument) {
	s.argumentTable.Clear()
	for i, key := range []string{"member", "overload", "position", "argument", "direction", "data_type", "depth"} {
		s.argumentTable.SetCell(0, i, tview.NewTableCell(s.tr(key)).SetTextColor(palette.accent).SetSelectable(false))
	}
	for row, argument := range arguments {
		name := argument.Name
		if argument.Position == 0 && argument.Depth == 0 {
			name = s.tr("return_value")
		}
		for col, text := range []string{argument.Member, argument.Overload, fmt.Sprint(argument.Position), name, argument.Direction, argument.DataType, fmt.Sprint(argument.Depth)} {
			s.argumentTable.SetCell(row+1, col, tview.NewTableCell(tview.Escape(text)).SetTextColor(palette.text))
		}
	}
	s.argumentInfo.SetText(s.object.Owner + "." + s.object.Name)
	if len(arguments) == 0 {
		s.argumentInfo.SetText(s.argumentInfo.GetText(false) + "\n" + s.tr("arguments_empty"))
	}
}

func (s *screen) closeArguments() {
	s.pages.RemovePage("arguments")
	if s.argumentFocus != nil {
		s.app.SetFocus(s.argumentFocus)
	}
}
