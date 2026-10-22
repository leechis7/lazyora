package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type textRole uint8

const (
	textBody textRole = iota
	textHint
	textAccent
	textWarning
)

func (s *screen) textColor(role textRole) tcell.Color {
	switch role {
	case textHint:
		return s.colors().muted
	case textAccent:
		return s.colors().accent
	case textWarning:
		return s.colors().warning
	}
	return s.colors().text
}

// Constructors initialize both the widget box and the text renderer. Styling
// only Box leaves tview's default black text background inside a light panel.
func (s *screen) newTextView(roles ...textRole) *tview.TextView {
	role := textBody
	if len(roles) > 0 {
		role = roles[0]
	}
	view := tview.NewTextView().SetTextColor(s.textColor(role))
	view.SetBackgroundColor(s.colors().background)
	if s.textRoles == nil {
		s.textRoles = make(map[*tview.TextView]textRole)
	}
	s.textRoles[view] = role
	return view
}

func (s *screen) newBox() *tview.Box {
	box := tview.NewBox()
	s.styleBox(box)
	return box
}

func (s *screen) newFlex() *tview.Flex {
	flex := tview.NewFlex()
	s.styleBox(flex.Box)
	return flex
}

func (s *screen) newInputField() *tview.InputField {
	field := tview.NewInputField().SetFieldStyle(s.fieldStyle()).SetLabelStyle(tcell.StyleDefault.Foreground(s.colors().text).Background(s.colors().background))
	field.SetBackgroundColor(s.colors().background)
	return field
}

func (s *screen) newTreeNode(text string, roles ...textRole) *tview.TreeNode {
	role := textBody
	if len(roles) > 0 {
		role = roles[0]
	}
	return tview.NewTreeNode(text).
		SetTextStyle(tcell.StyleDefault.Foreground(s.textColor(role)).Background(s.colors().background)).
		SetSelectedTextStyle(s.focusStyle())
}

func (s *screen) styleList(list *tview.List) {
	p := s.colors()
	s.styleBox(list.Box)
	list.SetBackgroundColor(p.surface)
	list.SetMainTextStyle(tcell.StyleDefault.Foreground(p.text).Background(p.surface)).
		SetSecondaryTextStyle(tcell.StyleDefault.Foreground(p.muted).Background(p.surface)).
		SetShortcutStyle(tcell.StyleDefault.Foreground(p.accent).Background(p.surface)).
		SetSelectedStyle(s.focusStyle()).SetHighlightFullLine(true)
}

func (s *screen) styleCheckbox(box *tview.Checkbox) {
	box.SetBackgroundColor(s.colors().surface)
	box.SetLabelStyle(tcell.StyleDefault.Foreground(s.colors().text).Background(s.colors().surface)).
		SetCheckedStyle(s.fieldStyle()).SetUncheckedStyle(s.fieldStyle()).SetActivatedStyle(s.focusStyle()).
		SetCheckedString(tview.Escape("[x]")).SetUncheckedString(tview.Escape("[ ]"))
}
