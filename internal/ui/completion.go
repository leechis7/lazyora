package ui

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func (s *screen) showCompletion(refresh bool) {
	if s.busy || s.completionEditor != nil {
		return
	}
	page, _ := s.pages.GetFrontPage()
	editor := s.editor
	if page == "source" && s.sourceLoaded && s.sourceEditor.HasFocus() {
		editor = s.sourceEditor
	} else if page != "main" || !s.editor.HasFocus() {
		return
	}
	if editor == nil || editor.HasSelection() {
		return
	}
	text := editor.GetText()
	_, cursor, _ := editor.GetSelection()
	context, err := oracle.SQLCompletionContext(text, cursor)
	if err != nil {
		s.setStatus("error", err)
		return
	}
	if context.Blocked {
		s.setStatus("completion_blocked")
		return
	}
	s.completionEditor = editor
	s.completionOriginal = text
	s.completionCursor = cursor
	s.completionContext = context
	s.completionCandidates = nil
	s.completionRows = nil
	s.completionResult = oracle.Completions{}
	s.completionList = tview.NewList().SetMainTextColor(s.colors().text).SetSecondaryTextColor(s.colors().muted).SetShortcutColor(s.colors().accent).ShowSecondaryText(true).SetSelectedStyle(s.focusStyle())
	s.completionList.SetBorder(true).SetTitle(s.tr("completion"))
	s.styleBox(s.completionList.Box)
	s.completionList.SetSelectedFunc(func(int, string, string, rune) { s.applyCompletion() })
	s.completionFilter = tview.NewInputField().SetLabel(s.tr("completion_filter")).SetText(context.Prefix).SetFieldWidth(40)
	s.completionFilter.SetFieldStyle(s.fieldStyle()).SetLabelStyle(tcell.StyleDefault.Foreground(s.colors().text).Background(s.colors().background))
	s.completionFilter.SetChangedFunc(func(string) { s.filterCompletions() })
	s.completionFilter.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			s.applyCompletion()
		}
	})
	s.completionInfo = tview.NewTextView().SetWrap(true).SetText(s.tr("browser_loading"))
	s.completionInfo.SetTextColor(s.colors().text).SetBackgroundColor(s.colors().background)
	buttons := tview.NewFlex().AddItem(s.clickable(tview.NewButton(s.tr("completion_insert")).SetSelectedFunc(s.applyCompletion)), 0, 1, false).
		AddItem(s.clickable(tview.NewButton(s.tr("completion_refresh")).SetSelectedFunc(func() { s.loadCompletions(true) })), 0, 1, false).
		AddItem(s.clickable(tview.NewButton(s.tr("cancel")).SetSelectedFunc(s.cancelCompletion)), 0, 1, false)
	root := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(s.completionInfo, 4, 0, false).AddItem(s.completionFilter, 1, 0, false).AddItem(s.completionList, 0, 1, true).AddItem(buttons, 1, 0, false)
	s.pages.AddPage("completion", root, true, true)
	s.app.SetFocus(s.completionList)
	s.loadCompletions(refresh)
}
func (s *screen) loadCompletions(refresh bool) {
	if s.busy || s.completionEditor == nil {
		return
	}
	context := s.completionContext
	context.Prefix = s.completionFilter.GetText()
	if s.connected {
		s.completionInfo.SetText(s.tr("browser_loading"))
		s.submit(job{action: "complete", completion: context, refresh: refresh})
	} else {
		s.acceptCompletions(oracle.Completions{}, nil)
	}
}
func (s *screen) acceptCompletions(result oracle.Completions, err error) {
	if s.completionEditor == nil {
		return
	}
	s.completionResult = result
	s.completionCandidates = append([]oracle.CompletionCandidate(nil), result.Candidates...)
	if len(s.completionContext.Qualifier) == 0 && !s.completionContext.ObjectsOnly && !s.completionContext.Quoted {
		var names []string
		for name := range sqlWords {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			insert := name
			prefix := s.completionFilter.GetText()
			if prefix != "" && prefix == strings.ToLower(prefix) {
				insert = strings.ToLower(name)
			}
			s.completionCandidates = append(s.completionCandidates, oracle.CompletionCandidate{Text: insert, Name: name, Kind: "SQL", Detail: "SQL / PL/SQL"})
		}
	}
	s.filterCompletions()
	message := s.tr("completion_hint")
	if !s.connected {
		message += "\n" + s.tr("completion_connect")
	}
	if result.Schema != "" {
		message = result.Schema + " · " + message
	}
	if result.Truncated {
		message += "\n" + s.tr("completion_limited")
	}
	if len(result.Warnings) > 0 {
		message += "\n" + fmt.Sprintf(s.tr("completion_partial"), len(result.Warnings))
	}
	if err != nil {
		message += "\n" + err.Error()
	}
	s.completionInfo.SetText(message)
	if err == nil {
		s.setStatus("completion_loaded", len(s.completionRows))
	}
}
func (s *screen) filterCompletions() {
	if s.completionList == nil {
		return
	}
	s.completionList.Clear()
	s.completionRows = nil
	prefix := s.completionFilter.GetText()
	if !s.completionContext.Quoted {
		prefix = strings.ToUpper(prefix)
	}
	for _, candidate := range s.completionCandidates {
		name := candidate.Name
		if !s.completionContext.Quoted {
			name = strings.ToUpper(name)
		}
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if len(s.completionRows) == oracle.MaxCompletionCandidates {
			break
		}
		s.completionRows = append(s.completionRows, candidate)
		s.completionList.AddItem(tview.Escape(candidate.Text), tview.Escape(candidate.Detail), 0, nil)
	}
	s.completionList.SetTitle(fmt.Sprintf("%s · %d", s.tr("completion"), len(s.completionRows)))
}
func (s *screen) applyCompletion() {
	if s.busy || s.completionEditor == nil || len(s.completionRows) == 0 {
		return
	}
	editor := s.completionEditor
	context := s.completionContext
	_, cursor, end := editor.GetSelection()
	// Completion reads asynchronously; never replace another tab or subsequently
	// edited text, and do not reset the editor/undo stack with SetText.
	if editor.GetText() != s.completionOriginal || cursor != end || cursor != s.completionCursor || cursor < context.Start || cursor > context.End {
		s.completionInfo.SetText(s.tr("completion_stale"))
		return
	}
	index := s.completionList.GetCurrentItem()
	if index < 0 || index >= len(s.completionRows) {
		return
	}
	editor.Replace(context.Start, context.End, s.completionRows[index].Text)
	s.closeCompletion()
}
func (s *screen) closeCompletion() {
	if s.busy {
		return
	}
	editor := s.completionEditor
	s.pages.RemovePage("completion")
	s.completionEditor = nil
	s.completionOriginal = ""
	s.completionList = nil
	s.completionFilter = nil
	s.completionInfo = nil
	s.completionCandidates = nil
	s.completionRows = nil
	s.completionResult = oracle.Completions{}
	if editor != nil {
		s.app.SetFocus(editor)
	}
}
func (s *screen) cancelCompletion() {
	if s.busy {
		if s.cancel != nil {
			s.cancel()
		}
		return
	}
	s.closeCompletion()
}
func (s *screen) completionKey(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyF1 {
		s.showHelp()
		return nil
	}
	if s.busy {
		if event.Key() == tcell.KeyEscape {
			s.cancelCompletion()
		}
		return nil
	}
	switch event.Key() {
	case tcell.KeyEscape:
		s.closeCompletion()
		return nil
	case tcell.KeyCtrlR:
		s.loadCompletions(true)
		return nil
	case tcell.KeyTab, tcell.KeyBacktab:
		s.dialogTab(event, s.completionFilter, s.completionList)
		return nil
	case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn:
		if s.completionFilter.HasFocus() {
			s.completionList.InputHandler()(event, func(tview.Primitive) {})
			return nil
		}
	case tcell.KeyRune:
		if s.completionList.HasFocus() && event.Modifiers()&(tcell.ModCtrl|tcell.ModAlt) == 0 {
			s.completionFilter.SetText(s.completionFilter.GetText() + string(event.Rune()))
			s.app.SetFocus(s.completionFilter)
			return nil
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if s.completionList.HasFocus() {
			text := s.completionFilter.GetText()
			_, size := utf8.DecodeLastRuneInString(text)
			s.completionFilter.SetText(text[:len(text)-size])
			s.app.SetFocus(s.completionFilter)
			return nil
		}
	}
	return event
}
