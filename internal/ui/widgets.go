package ui

import (
	"github.com/leechis7/tuivory/go"
)

type textRole uint8

const (
	textBody textRole = iota
	textHint
	textAccent
	textWarning
)

func (s *screen) textColor(role textRole) tuivory.Color {
	return s.tuivoryStyles().TextColor(tuivory.TextRole(role))
}

// Constructors initialize both the widget box and the text renderer. Styling
// only Box leaves tview's default black text background inside a light panel.
func (s *screen) newTextView(roles ...textRole) *tuivory.WidgetTextView {
	role := textBody
	if len(roles) > 0 {
		role = roles[0]
	}
	view := tuivory.NewWidgetTextView().SetTextColor(s.textColor(role))
	s.styleBox(view.Box)
	view.SetBackgroundColor(s.colors().background)
	if s.textRoles == nil {
		s.textRoles = make(map[*tuivory.WidgetTextView]textRole)
	}
	s.textRoles[view] = role
	s.textSurface(view, surfaceCanvas)
	return view
}

func (s *screen) newBox() *tuivory.Box {
	box := tuivory.NewBox()
	s.styleBox(box)
	return box
}

func (s *screen) newFlex() *tuivory.Flex {
	flex := tuivory.NewFlex()
	s.styleBox(flex.Box)
	return flex
}

func (s *screen) newInputField() *tuivory.InputField {
	field := tuivory.NewInputField().SetFieldStyle(s.fieldStyle()).SetLabelStyle(tuivory.StyleDefault.Foreground(s.colors().text).Background(s.colors().background))
	s.setSurface(field.Box, surfaceCanvas)
	return field
}

func (s *screen) newTreeNode(text string, roles ...textRole) *tuivory.WidgetTreeNode {
	role := textBody
	if len(roles) > 0 {
		role = roles[0]
	}
	node := tuivory.NewWidgetTreeNode(text).
		SetTextStyle(tuivory.StyleDefault.Foreground(s.textColor(role)).Background(s.colors().background)).
		SetSelectedTextStyle(s.focusStyle())
	if s.treeRoles == nil {
		s.treeRoles = map[*tuivory.WidgetTreeNode]textRole{}
	}
	s.treeRoles[node] = role
	return node
}

func (s *screen) styleList(list *tuivory.WidgetList) {
	s.setSurface(list.Box, surfacePanel)
	s.tuivoryWidgets().List(list)
}

func (s *screen) styleCheckbox(box *tuivory.Checkbox) {
	s.setSurface(box.Box, surfacePanel)
	s.tuivoryWidgets().Checkbox(box)
}
