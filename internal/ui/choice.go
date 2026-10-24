package ui

import "github.com/leechis7/tuivory/go"

// Tuivory owns selection-field colors and wide-glyph overlay compatibility.
type choiceField = tuivory.Choice

func (s *screen) newChoice(label string, options []string, selected int, changed func(string, int)) *choiceField {
	choice := tuivory.NewChoice(label, options, selected, changed)
	s.styleChoice(choice)
	return choice
}
func (s *screen) styleChoice(choice *choiceField) { s.tuivoryWidgets().Choice(choice) }
