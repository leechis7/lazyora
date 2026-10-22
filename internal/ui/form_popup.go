package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rivo/uniseg"
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
		info.SetBackgroundColor(s.colors().surface).SetBorder(true).SetBorderPadding(0, 0, 2, 2)
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
	root := s.newFlex().AddItem(content, 0, 1, true)
	root.SetDrawFunc(func(terminal tcell.Screen, x, y, width, height int) (int, int, int, int) {
		for py := y; py < y+height; py++ {
			for px := x; px < x+width; px++ {
				r, comb, style, cellWidth := terminal.GetContent(px, py)
				backdrop := style.Foreground(s.colors().muted).Background(s.colors().background).Dim(true)
				if s.limitedColors {
					backdrop = backdrop.Foreground(tcell.ColorSilver).Background(tcell.ColorBlack)
				}
				terminal.SetContent(px, py, r, comb, backdrop)
				if cellWidth > 1 {
					px += cellWidth - 1
				}
			}
		}
		insetX, insetY := 2, 1
		if width < 44 {
			insetX = 0
		}
		if height < 12 {
			insetY = 0
		}
		w, h := max(0, min(preferredWidth, width-2*insetX)), max(0, min(preferredHeight, height-2*insetY))
		left, top := x+(width-w)/2, y+(height-h)/2
		// Flex leaves padding and spacers untouched. Paint an opaque card first
		// so the underlying SQL and result borders cannot show through it.
		card := tcell.StyleDefault.Foreground(s.colors().text).Background(s.colors().surface)
		for py := top; py < top+h; py++ {
			gridFill(terminal, left, py, w, card)
		}
		shadow := tcell.StyleDefault.Background(tcell.ColorBlack)
		if left+w < x+width {
			for py := top + 1; py < top+h+1 && py < y+height; py++ {
				terminal.SetContent(left+w, py, ' ', nil, shadow)
			}
		}
		if top+h < y+height {
			gridFill(terminal, left+1, top+h, min(w, x+width-left-1), shadow)
		}
		if layout != nil {
			layout(w, h)
		}
		return left, top, w, h
	})
	return &formPopupLayer{Flex: root, content: content}
}

type formPopupLayer struct {
	*tview.Flex
	content tview.Primitive
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
	}
	return nil
}

// Clip at grapheme boundaries and count terminal columns, including Korean.
func popupLabel(text string, width int) string {
	if uniseg.StringWidth(text) <= width {
		return text
	}
	g := uniseg.NewGraphemes(text)
	result, used := "", 0
	for g.Next() {
		if used+g.Width() > width-1 {
			break
		}
		result += g.Str()
		used += g.Width()
	}
	return result + "…"
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
