package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"testing"
)

func TestFormButtonArrowsWrapWithoutActivatingOrLeavingRow(t *testing.T) {
	for _, page := range []string{"connect", "settings", "search", "describe"} {
		t.Run(page, func(t *testing.T) {
			s := shortcutScreen(t)
			var form *tview.Form
			switch page {
			case "connect":
				s.connectionForm()
				form = s.connectForm
			case "settings":
				s.settingsForm()
				form = s.prefsForm
			case "search":
				s.showSearch(true)
				form = s.searchForm
			case "describe":
				s.describe(true)
				form = s.describeForm
			}
			focusForm(s, form, form.GetFormItemCount())
			s.key(tcell.NewEventKey(tcell.KeyLeft, 0, 0))
			if !form.GetButton(form.GetButtonCount() - 1).HasFocus() {
				t.Fatal("Left did not wrap to last button")
			}
			s.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
			if !form.GetButton(0).HasFocus() {
				t.Fatal("Right did not wrap to first button")
			}
			s.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
			if !form.GetButton(1).HasFocus() {
				t.Fatal("Right did not select next button")
			}
			if current, _ := s.pages.GetFrontPage(); current != page || len(s.jobs) != 0 {
				t.Fatal("arrows activated a button")
			}
			focusForm(s, form, 0)
			event := tcell.NewEventKey(tcell.KeyLeft, 0, 0)
			if s.key(event) == nil || !form.GetFormItem(0).HasFocus() {
				t.Fatal("button navigation stole input-field arrows")
			}
		})
	}
}

func TestButtonArrowsSkipDisabledButtonsAndEnterActivatesSelected(t *testing.T) {
	s := shortcutScreen(t)
	s.showSearch(false)
	form := s.searchForm
	form.GetButton(1).SetDisabled(true)
	focusForm(s, form, form.GetFormItemCount())
	s.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	if !form.GetButton(2).HasFocus() {
		t.Fatal("disabled button remained in arrow cycle")
	}
	event := s.key(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	form.GetButton(2).InputHandler()(event, func(p tview.Primitive) { s.app.SetFocus(p) })
	if s.searchForm != nil || !s.editor.HasFocus() {
		t.Fatal("Enter did not activate selected Close button")
	}
}

func TestDetailButtonsAreReachableByTabAndArrows(t *testing.T) {
	s := programScreen(t)
	s.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	if !s.sourceDiagnostics.HasFocus() {
		t.Fatal("Tab skipped compiler diagnostics")
	}
	s.key(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	first, ok := s.app.GetFocus().(*tview.Button)
	if !ok {
		t.Fatal("Tab did not reach source buttons")
	}
	s.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	if s.app.GetFocus() == first {
		t.Fatal("Right did not move between source buttons")
	}
	s.key(tcell.NewEventKey(tcell.KeyLeft, 0, 0))
	if s.app.GetFocus() != first {
		t.Fatal("Left did not return to first source button")
	}
	s.key(tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModShift))
	if !s.sourceDiagnostics.HasFocus() {
		t.Fatal("Shift+Tab did not leave buttons for diagnostics")
	}
	if len(s.jobs) != 0 {
		t.Fatal("button navigation compiled source")
	}
}

func TestSelectedButtonHasDistinctRenderedStyleAndModalArrowsWork(t *testing.T) {
	terminal := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.Init(); err != nil {
		t.Fatal(err)
	}
	defer terminal.Fini()
	terminal.SetSize(100, 30)
	s := shortcutScreen(t)
	s.settingsForm()
	focusForm(s, s.prefsForm, s.prefsForm.GetFormItemCount())
	s.prefsForm.SetRect(0, 0, 100, 30)
	s.prefsForm.Draw(terminal)
	styleAt := func(button *tview.Button) tcell.Style {
		x, y, w, _ := button.GetRect()
		_, _, style, _ := terminal.GetContent(x+w/2, y)
		return style
	}
	selected := styleAt(s.prefsForm.GetButton(0))
	normal := styleAt(s.prefsForm.GetButton(1))
	fg, bg, attrs := selected.Decompose()
	_, otherBG, _ := normal.Decompose()
	if fg != s.tokens().selectionFG || bg != s.tokens().selection || bg == otherBG || attrs&tcell.AttrUnderline == 0 {
		t.Fatal("selected button lacks the contrasting focus palette and underline")
	}
	s.key(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	s.prefsForm.Draw(terminal)
	if styleAt(s.prefsForm.GetButton(1)) != selected || styleAt(s.prefsForm.GetButton(0)) != normal {
		t.Fatal("selection styling did not follow focus")
	}
	s.closeSettings()
	s.showProgramConfirm("Confirm", func() { t.Fatal("arrow activated confirmation") })
	modalKey(s, tcell.KeyLeft)
	if focus, ok := s.app.GetFocus().(*tview.Button); !ok || focus.GetLabel() != s.tr("proceed") {
		t.Fatal("modal Left did not select Proceed")
	}
	modalKey(s, tcell.KeyRight)
	if focus, ok := s.app.GetFocus().(*tview.Button); !ok || focus.GetLabel() != s.tr("cancel") {
		t.Fatal("modal Right did not select Cancel")
	}
}
