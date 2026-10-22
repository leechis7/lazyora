package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

const outputHistoryBytes = 1024 * 1024
const outputHistoryLines = 8000

func (s *screen) addOutput(out oracle.Output) {
	for _, line := range out.Lines {
		s.outputLines = append(s.outputLines, line)
		s.outputSize += len(line) + 1
	}
	drop := 0
	for len(s.outputLines)-drop > outputHistoryLines || s.outputSize > outputHistoryBytes {
		s.outputSize -= len(s.outputLines[drop]) + 1
		drop++
	}
	if drop > 0 {
		s.outputLines = append([]string(nil), s.outputLines[drop:]...)
	}
	s.outputTruncated = s.outputTruncated || out.Truncated || drop > 0
	if out.Error != "" {
		s.outputError = out.Error
	}
	s.refreshOutput()
}

func (s *screen) clearOutput() {
	s.outputLines, s.outputSize = nil, 0
	s.outputTruncated, s.outputError = false, ""
	s.refreshOutput()
}

func (s *screen) outputContents() string {
	if len(s.outputLines) == 0 {
		return ""
	}
	return strings.Join(s.outputLines, "\n") + "\n"
}

func (s *screen) requestOutputToggle() {
	if !s.connected || s.busy || s.transactionUncertain {
		return
	}
	s.submit(job{action: "output_toggle"})
	s.refreshOutput()
}

func (s *screen) refreshOutput() {
	if s.outputText == nil {
		return
	}
	state, action := "output_disabled", "output_enable"
	if s.outputEnabled {
		state, action = "output_enabled", "output_disable"
	}
	info := s.tr(state) + " · " + fmt.Sprintf(s.tr("output_count"), len(s.outputLines)) + "\n" + s.tr("output_note")
	if s.outputTruncated {
		info += "\n" + s.tr("output_truncated")
	}
	if s.outputError != "" {
		info += "\n" + s.tr("output_error") + ": " + s.outputError
	}
	s.outputInfo.SetText(info)
	s.outputToggle.SetLabel(s.tr(action) + " (Alt+E)").SetDisabled(!s.connected || s.busy || s.transactionUncertain)
	s.outputText.SetText(s.outputContents())
}

func (s *screen) showOutput() {
	if s.outputText != nil {
		return
	}
	s.outputFocus = s.app.GetFocus()
	s.outputText = s.newTextView().SetWrap(false).SetScrollable(true)
	s.outputText.SetBorder(true).SetTitle("DBMS_OUTPUT")
	s.styleBox(s.outputText.Box)
	s.outputText.SetTextColor(s.colors().text)
	s.outputInfo = s.newTextView(textHint).SetWrap(true).SetWordWrap(true)
	s.outputInfo.SetTextColor(s.colors().muted).SetBackgroundColor(s.colors().background)
	s.outputToggle = s.clickable(tview.NewButton("").SetSelectedFunc(s.requestOutputToggle))
	copy := s.clickable(tview.NewButton(s.tr("output_copy") + " (Ctrl+C)").SetSelectedFunc(func() { s.sendClipboard(s.outputContents()) }))
	clear := s.clickable(tview.NewButton(s.tr("output_clear") + " (Alt+D)").SetSelectedFunc(s.clearOutput))
	close := s.clickable(tview.NewButton(s.tr("close") + " (Esc)").SetSelectedFunc(s.closeOutput))
	buttons := s.newFlex().AddItem(s.outputToggle, 0, 1, false).AddItem(copy, 0, 1, false).AddItem(clear, 0, 1, false).AddItem(close, 0, 1, false)
	root := s.newFlex().SetDirection(tview.FlexRow).AddItem(s.outputInfo, 6, 0, false).AddItem(s.outputText, 0, 1, true).AddItem(buttons, 1, 0, false)
	s.pages.AddPage("output", root, true, true)
	s.refreshOutput()
	s.app.SetFocus(s.outputText)
}

func (s *screen) closeOutput() {
	s.pages.RemovePage("output")
	s.outputText, s.outputInfo, s.outputToggle = nil, nil, nil
	if s.outputFocus != nil {
		s.app.SetFocus(s.outputFocus)
	}
	s.outputFocus = nil
}

func (s *screen) outputKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		s.closeOutput()
		return nil
	case tcell.KeyF1:
		s.showHelp()
		return nil
	case tcell.KeyCtrlC:
		s.sendClipboard(s.outputContents())
		return nil
	case tcell.KeyTab, tcell.KeyBacktab:
		s.dialogTab(event, s.outputText)
		return nil
	case tcell.KeyRune:
		if event.Modifiers()&tcell.ModAlt != 0 {
			switch event.Rune() {
			case 'e', 'E':
				s.requestOutputToggle()
				return nil
			case 'd', 'D':
				s.clearOutput()
				return nil
			}
		}
	}
	return event
}
