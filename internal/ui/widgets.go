package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/leechis7/tuivory/go"
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
	return s.tuivoryStyles().TextColor(tuivory.TextRole(role))
}

// Constructors initialize both the widget box and the text renderer. Styling
// only Box leaves tview's default black text background inside a light panel.
func (s *screen) newTextView(roles ...textRole) *tview.TextView {
	role := textBody
	if len(roles) > 0 {
		role = roles[0]
	}
	view := tview.NewTextView().SetTextColor(s.textColor(role))
	s.styleBox(view.Box)
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
	s.setSurface(field.Box, surfaceCanvas)
	return field
}

func (s *screen) newTreeNode(text string, roles ...textRole) *tview.TreeNode {
	role := textBody
	if len(roles) > 0 {
		role = roles[0]
	}
	node := tview.NewTreeNode(text).
		SetTextStyle(tcell.StyleDefault.Foreground(s.textColor(role)).Background(s.colors().background)).
		SetSelectedTextStyle(s.focusStyle())
	if s.treeRoles == nil {
		s.treeRoles = map[*tview.TreeNode]textRole{}
	}
	s.treeRoles[node] = role
	return node
}

func (s *screen) styleList(list *tview.List) {
	s.setSurface(list.Box, surfacePanel)
	s.tuivoryWidgets().List(list)
}

func (s *screen) styleCheckbox(box *tview.Checkbox) {
	s.setSurface(box.Box, surfacePanel)
	s.tuivoryWidgets().Checkbox(box)
}
