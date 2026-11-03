package ui

import (
	"fmt"
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestTuivoryShadowPreservesCardInputAndNarrowLayout(t *testing.T) {
	for _, count := range []int{8, 16, 256, 1 << 24} {
		for _, theme := range []string{"dark", "light", "blackwhite"} {
			t.Run(fmt.Sprintf("%s/%d", theme, count), func(t *testing.T) {
				s, terminal := resultScreen(t)
				s.terminal = auditTerminal{terminal, count}
				s.prefs.Theme = theme
				s.limitedColors = count <= 16
				field := s.newInputField().SetText("before")
				form := tuivory.NewWidgetForm().AddFormItem(field)
				form.AddButton("Apply", func() { t.Error("shadow click activated a button") })
				popup := s.formPopup(form, 40, 12, nil)
				s.app.SetFocus(field)
				sql := s.editor.GetText()
				for _, size := range [][2]int{{80, 24}, {40, 12}, {10, 4}, {80, 24}} {
					terminal.SetSize(size[0], size[1])
					popup.SetRect(0, 0, size[0], size[1])
					popup.Draw(terminal)
					layout := popup.shadow.Layout()
					x, y, w, h := popup.content.GetRect()
					if layout.Card != (tuivory.CellRect{X: x, Y: y, Width: w, Height: h}) {
						t.Fatal("host content differs from public card geometry")
					}
					if x < 0 || y < 0 || x+w > size[0] || y+h > size[1] {
						t.Fatal("popup escaped viewport")
					}
					for _, region := range layout.ShadowRegions() {
						if region.Width <= 0 || region.Height <= 0 {
							continue
						}
						if popup.shadow.InRect(region.X, region.Y) {
							t.Fatal("shadow exposed as an interactive card")
						}
						popup.MouseHandler()(tuivory.MouseLeftClick, tuivory.NewEventMouse(region.X, region.Y, 0, 0), func(p tuivory.Primitive) { s.app.SetFocus(p) })
						if s.app.GetFocus() != field {
							t.Fatal("shadow click stole field focus")
						}
					}
				}
				popup.shadow.PasteHandler()("한글", func(p tuivory.Primitive) { s.app.SetFocus(p) })
				if field.GetText() != "before한글" || s.editor.GetText() != sql || len(s.jobs) != 0 {
					t.Fatal("shared wrapper changed paste/data binding", field.GetText())
				}
			})
		}
	}
}

func TestPopupBottomShadowUsesSharedHalfRow(t *testing.T) {
	for _, theme := range []string{"dark", "light", "blackwhite"} {
		for _, count := range []int{8, 16, 256, 1 << 24} {
			for _, mode := range []string{"unicode", "ascii"} {
				t.Run(fmt.Sprintf("%s/%d/%s", theme, count, mode), func(t *testing.T) {
					s, terminal := resultScreen(t)
					s.terminal = auditTerminal{terminal, count}
					s.prefs.Theme, s.prefs.DisplayMode = theme, mode
					form := tuivory.NewWidgetForm().AddFormItem(s.newInputField().SetText("한글"))
					popup := s.formPopup(form, 40, 12, nil)
					checked := 0
					for _, size := range [][2]int{{80, 24}, {40, 12}, {80, 24}} {
						terminal.SetSize(size[0], size[1])
						terminal.Clear()
						popup.SetRect(0, 0, size[0], size[1])
						popup.Draw(terminal)
						terminal.Show()
						l := popup.shadow.Layout()
						if l.Shadow.Height <= l.Card.Height {
							continue
						}
						x, y := l.Shadow.X, l.Shadow.Y+l.Shadow.Height-1
						cells, w, _ := terminal.GetContents()
						cell := cells[y*w+x]
						want := "▀"
						if mode == "ascii" {
							want = "."
						}
						if string(cell.Runes) != want {
							t.Fatalf("bottom shadow is %q, want %q", string(cell.Runes), want)
						}
						fg, bg, _ := cell.Style.Decompose()
						if fg == bg {
							t.Fatal("bottom shadow has no contrast")
						}
						if popup.shadow.InRect(x, y) {
							t.Fatal("visual shadow became clickable card")
						}
						checked++
					}
					if checked < 2 {
						t.Fatalf("did not recheck shadow after resize: %d", checked)
					}
				})
			}
		}
	}
}
