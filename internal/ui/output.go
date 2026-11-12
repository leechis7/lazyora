package ui

import (
	"fmt"
	"lazyora/internal/oracle"
	"strings"

	"github.com/leechis7/tuivory/go"
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
	if (len(out.Lines) > 0 || out.Truncated || out.Error != "") && !s.resultSectionVisible("output") {
		s.outputNew = true
	}
	s.outputTruncated = s.outputTruncated || out.Truncated || drop > 0
	if out.Error != "" {
		s.outputError = out.Error
	}
	s.refreshOutput()
}

func (s *screen) clearOutput() {
	s.outputNew = false
	s.outputLines, s.outputSize = nil, 0
	s.outputTruncated, s.outputError = false, ""
	s.refreshOutput()
	if s.outputText != nil {
		s.outputText.ScrollToBeginning()
	}
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
	s.refreshResultAreaTabs()
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
	row, col := s.outputText.GetScrollOffset()
	s.outputText.SetText(s.outputContents()).ScrollTo(row, col)
}

func (s *screen) showOutput()  { s.selectResultSection("output", true) }
func (s *screen) closeOutput() { s.closeResultSection() }

func (s *screen) buildOutput(body *tuivory.Flex) []tuivory.Primitive {
	s.outputText = s.newTextView().SetDynamicColors(false).SetWrap(false).SetScrollable(true)
	s.outputText.SetBorder(true).SetTitle("DBMS_OUTPUT")
	s.styleBox(s.outputText.Box)
	s.outputInfo = s.newTextView(textHint).SetDynamicColors(false).SetWrap(true).SetWordWrap(true).SetScrollable(true)
	s.outputToggle = s.clickable(tuivory.NewWidgetButton("").SetSelectedFunc(s.requestOutputToggle))
	copy := s.clickable(tuivory.NewWidgetButton(s.tr("output_copy")).SetSelectedFunc(func() { s.sendClipboard(s.outputContents()) }))
	clear := s.clickable(tuivory.NewWidgetButton(s.tr("output_clear")).SetSelectedFunc(s.clearOutput))
	buttons := s.newFlex().AddItem(s.outputToggle, 0, 1, false).AddItem(copy, 0, 1, false).AddItem(clear, 0, 1, false)
	body.AddItem(s.outputInfo, 0, 1, false).AddItem(s.outputText, 0, 3, true).AddItem(buttons, 1, 0, false)
	s.refreshOutput()
	return []tuivory.Primitive{s.outputText, s.outputInfo, s.outputToggle, copy, clear}
}

func (s *screen) outputKey(event *tuivory.EventKey) *tuivory.EventKey {
	switch event.Key() {
	case tuivory.KeyF1:
		s.showHelp()
		return nil
	case tuivory.KeyCtrlC:
		s.sendClipboard(s.outputContents())
		return nil
	case tuivory.KeyRune:
		if event.Modifiers()&tuivory.ModAlt != 0 {
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
