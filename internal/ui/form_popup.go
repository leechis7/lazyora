package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/leechis7/tuivory/go"
	"github.com/rivo/tview"
)

// formDialog gives all input dialogs the same responsive card layout. Keep the
// Form itself intact so its validation, focus order and shortcuts stay local.
func (s *screen) formDialog(form *tview.Form, info *tview.TextView, width int) *formPopupLayer {
	form.SetBorder(true)
	fieldsHeight := 0
	for i := 0; i < form.GetFormItemCount(); i++ {
		fieldsHeight += max(1, form.GetFormItem(i).GetFieldHeight())
	}
	fit := formLayout(form)
	infoHeight := 0
	content := s.newFlex().SetDirection(tview.FlexRow)
	if info != nil {
		infoHeight = 5
		s.textSurface(info, surfacePanel)
		info.SetBorder(true).SetBorderPadding(0, 0, 2, 2)
		content.AddItem(info, infoHeight, 0, false)
	}
	content.AddItem(form, 0, 1, true)
	preferredHeight := infoHeight + fieldsHeight + form.GetFormItemCount() + 6
	return s.formPopup(content, width, preferredHeight, func(w, h int) {
		if info != nil {
			infoHeight = min(5, max(2, h/4))
			content.ResizeItem(info, infoHeight, 0)
		}
		fit(w, h-infoHeight)
	})
}

// Keep the underlying work visible, but route every mouse event through the
// modal layer. The returned Flex retains normal Form/DropDown event routing.
func (s *screen) formPopup(content tview.Primitive, preferredWidth, preferredHeight int, layout func(width, height int)) *formPopupLayer {
	// The content and its layout remain host-owned; shared surfaces/shadows
	// and wide-glyph protection are delegated to published Tuivory components.
	card := s.newFlex().AddItem(content, 0, 1, true)
	card.Box = s.newBox() // Opaque, borderless host container; see SDK gap record.
	s.setSurface(card.Box, surfacePanel)
	shadow, err := tuivory.NewShadowView(card, s.componentPalette(), tuivory.ShadowOptions{Mode: tuivory.ShadowAuto, OffsetX: 1, OffsetY: 1})
	if err != nil {
		panic(err)
	}
	root := s.newFlex().AddItem(shadow, 0, 1, true)
	root.SetDrawFunc(func(terminal tcell.Screen, x, y, width, height int) (int, int, int, int) {
		tuivory.DimRegion(terminal, tuivory.CellRect{X: x, Y: y, Width: width, Height: height})
		insetX, insetY := 2, 1
		if width < 44 {
			insetX = 0
		}
		if height < 12 {
			insetY = 0
		}
		w, h := max(0, min(preferredWidth, width-2*insetX)), max(0, min(preferredHeight, height-2*insetY))
		left, top := x+(width-w)/2, y+(height-h)/2
		dx, dy := min(1, max(0, x+width-left-w)), min(1, max(0, y+height-top-h))
		options := tuivory.ShadowOptions{Mode: tuivory.ShadowAuto, OffsetX: dx, OffsetY: dy}
		if dx == 0 && dy == 0 {
			options.Mode = tuivory.ShadowOff
		}
		_ = shadow.SetShadow(options)
		_ = shadow.SetPalette(s.componentPalette())
		shadow.ASCII = s.asciiDisplay()
		s.styleBox(card.Box)
		if layout != nil {
			layout(w, h)
		}
		return left, top, w + dx, h + dy
	})
	return &formPopupLayer{Flex: root, content: content, shadow: shadow}
}

type formPopupLayer struct {
	*tview.Flex
	content tview.Primitive
	shadow  *tuivory.ShadowView
}

func (p *formPopupLayer) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	handler := p.Flex.MouseHandler()
	return func(action tview.MouseAction, event *tcell.EventMouse, focus func(tview.Primitive)) (bool, tview.Primitive) {
		if dropdown := openFormChoice(p.content); dropdown != nil {
			_, capture := dropdown.MouseHandler()(action, event, focus)
			return true, capture
		}
		_, capture := handler(action, event, focus)
		// Preserve text-field drag capture, and never pass blank/backdrop clicks
		// through to SQL, results or another dialog underneath.
		return true, capture
	}
}

func openFormChoice(root tview.Primitive) *choiceField {
	switch item := root.(type) {
	case *tview.Form:
		for i := 0; i < item.GetFormItemCount(); i++ {
			if choice, ok := item.GetFormItem(i).(*choiceField); ok && choice.IsOpen() {
				return choice
			}
		}
	case *tview.Flex:
		for i := 0; i < item.GetItemCount(); i++ {
			if child := item.GetItem(i); child != nil {
				if choice := openFormChoice(child); choice != nil {
					return choice
				}
			}
		}
	case tuivory.StyleChildren:
		for _, child := range item.StyleChildren() {
			if choice := openFormChoice(child); choice != nil {
				return choice
			}
		}

	}
	return nil
}

// Use Tuivory's common literal-text clipping, including its ASCII loss marker.
func popupLabel(text string, width int) string {
	return tuivory.CellText(text, width)
}

func formLayout(form *tview.Form) func(int, int) {
	labels := make([]string, form.GetFormItemCount())
	buttons := make([]string, form.GetButtonCount())
	fieldsHeight := 0
	for i := range labels {
		labels[i] = form.GetFormItem(i).GetLabel()
		fieldsHeight += max(1, form.GetFormItem(i).GetFieldHeight())
	}
	for i := range buttons {
		buttons[i] = form.GetButton(i).GetLabel()
	}
	return func(w, h int) {
		spacing := 1
		if h < fieldsHeight+len(labels)+5 {
			spacing = 0
		}
		form.SetItemPadding(spacing)
		for i, label := range labels {
			short := popupLabel(label, max(4, (w-6)/2))
			switch item := form.GetFormItem(i).(type) {
			case *tview.InputField:
				item.SetLabel(short)
			case *choiceField:
				item.SetLabel(short)
			case *tview.Checkbox:
				item.SetLabel(short)
			}
		}
		total := 0
		for _, label := range buttons {
			total += tview.TaggedStringWidth(label) + 5
		}
		for i, label := range buttons {
			if total > w-5 {
				label = popupLabel(label, max(1, (w-6)/len(buttons)-5))
			}
			form.GetButton(i).SetLabel(label)
		}
	}
}
