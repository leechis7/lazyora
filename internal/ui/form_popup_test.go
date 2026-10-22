package ui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"testing"
)

func TestPopupCardsEraseUnderlyingBordersAndPadding(t *testing.T) {
	for _, page := range []string{"connect", "settings", "about"} {
		for _, theme := range []string{"dark", "light"} {
			for _, limited := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/limited=%v", page, theme, limited), func(t *testing.T) {
					s, terminal := resultScreen(t)
					s.prefs.Theme, s.limitedColors = theme, limited
					switch page {
					case "connect":
						s.connectionForm()
					case "settings":
						s.settingsForm()
					case "about":
						s.showAbout()
					}
					_, primitive := s.pages.GetFrontPage()
					popup := primitive.(*formPopupLayer)
					for _, size := range [][2]int{{120, 36}, {80, 24}, {40, 16}} {
						terminal.SetSize(size[0], size[1])
						for y := 0; y < size[1]; y++ {
							for x := 0; x < size[0]; x++ {
								terminal.SetContent(x, y, '╳', nil, tcell.StyleDefault.Background(tcell.ColorRed))
							}
						}
						popup.SetRect(0, 0, size[0], size[1])
						popup.Draw(terminal)
						x, y, w, h := popup.content.GetRect()
						for py := y; py < y+h; py++ {
							for px := x; px < x+w; px++ {
								r, _, style, _ := terminal.GetContent(px, py)
								_, bg, attrs := style.Decompose()
								if r == '╳' || bg == tcell.ColorRed || attrs&tcell.AttrDim != 0 {
									t.Fatalf("background leaked inside %v popup at %d,%d", size, px, py)
								}
							}
						}
					}
				})
			}
		}
	}
}

func TestFormPopupsResizeSpacingAndButtonAccess(t *testing.T) {
	for _, language := range []string{"ko", "en"} {
		for _, page := range []string{"connect", "settings"} {
			t.Run(language+"/"+page, func(t *testing.T) {
				s, terminal := resultScreen(t)
				s.prefs.Language = language
				var form *tview.Form
				if page == "connect" {
					s.connectionForm()
					form = s.connectForm
				} else {
					s.settingsForm()
					form = s.prefsForm
				}
				_, root := s.pages.GetFrontPage()
				for _, size := range [][2]int{{120, 36}, {80, 24}, {60, 20}, {40, 16}} {
					terminal.SetSize(size[0], size[1])
					root.SetRect(0, 0, size[0], size[1])
					root.Draw(terminal)
					x, y, w, h := form.GetRect()
					if x < 0 || y < 0 || x+w > size[0] || y+h > size[1] {
						t.Fatalf("form clipped at %v: %d,%d %dx%d", size, x, y, w, h)
					}
					if size[0] == 120 && (x <= 0 || w >= size[0]) {
						t.Fatal("form is still fullscreen")
					}
					if size[1] >= 30 {
						_, y0, _, _ := form.GetFormItem(0).GetRect()
						_, y1, _, _ := form.GetFormItem(1).GetRect()
						if y1-y0 < 2 {
							t.Fatal("normal-height form lacks row spacing")
						}
					}
					for i := 0; i < form.GetFormItemCount(); i++ {
						focusForm(s, form, i)
						root.Draw(terminal)
						fx, fy := formFieldCell(form, i)
						if fx >= x+w-1 || fy < y || fy >= y+h-1 {
							t.Fatalf("field %d inaccessible at %v: %d,%d", i, size, fx, fy)
						}
					}
					for i := 0; i < form.GetButtonCount(); i++ {
						focusForm(s, form, form.GetFormItemCount()+i)
						root.Draw(terminal)
						bx, by, bw, bh := form.GetButton(i).GetRect()
						if bw < 4 || by < y || by+bh > y+h || bx+bw > x+w {
							t.Fatalf("button %d inaccessible at %v: %d,%d %dx%d", i, size, bx, by, bw, bh)
						}
					}
				}
			})
		}
	}
}
func TestPopupBackdropClickDoesNotActivateUnderlyingUI(t *testing.T) {
	s, terminal := resultScreen(t)
	s.connectionForm()
	_, root := s.pages.GetFrontPage()
	root.SetRect(0, 0, 120, 36)
	root.Draw(terminal)
	focus := s.app.GetFocus()
	consumed, _ := root.MouseHandler()(tview.MouseLeftClick, tcell.NewEventMouse(0, 0, tcell.Button1, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	if !consumed || s.app.GetFocus() != focus || len(s.jobs) != 0 {
		t.Fatal("backdrop click escaped popup")
	}
	focusForm(s, s.connectProfiles, 0)
	dropdown := s.connectProfiles.GetFormItem(0).(*choiceField)
	dropdown.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	root.Draw(terminal)
	if !dropdown.IsOpen() {
		t.Fatal("profile picker did not open")
	}
	root.MouseHandler()(tview.MouseLeftDown, tcell.NewEventMouse(0, 0, tcell.Button1, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
	if dropdown.IsOpen() || !s.pages.HasPage("connect") {
		t.Fatal("outside click did not close only the picker")
	}
	s.key(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if s.pages.HasPage("connect") {
		t.Fatal("popup escape failed")
	}
}
func TestInputFocusHighlightIsReversible(t *testing.T) {
	for _, limited := range []bool{false, true} {
		t.Run(fmt.Sprint(limited), func(t *testing.T) {
			s, terminal := resultScreen(t)
			s.limitedColors = limited
			s.connectionForm()
			_, root := s.pages.GetFrontPage()
			root.SetRect(0, 0, 120, 36)
			s.app.SetFocus(s.editor)
			root.Draw(terminal)
			x, y := formFieldCell(s.connectForm, connectionHost)
			_, _, normal, _ := terminal.GetContent(x, y)
			focusForm(s, s.connectForm, connectionHost)
			root.Draw(terminal)
			_, _, focused, _ := terminal.GetContent(x, y)
			if normal == focused {
				t.Fatal("focused input looks unchanged")
			}
			s.app.SetFocus(s.editor)
			root.Draw(terminal)
			_, _, restored, _ := terminal.GetContent(x, y)
			if restored != normal {
				t.Fatal("input focus style leaked")
			}
		})
	}
}

func TestSharedFormDialogLongLabelsButtonsAndScrolling(t *testing.T) {
	for _, count := range []int{1, 9, 30} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			s, terminal := resultScreen(t)
			form := tview.NewForm()
			for i := 0; i < count; i++ {
				form.AddInputField("매우 긴 입력 항목 이름 / Long parameter name", "preserved", 40, nil, nil)
			}
			for i := 0; i < 5; i++ {
				form.AddButton("긴 작업 버튼 / Long action", func() {})
			}
			styleForm(form)
			info := tview.NewTextView().SetText("설명 / validation error\nSecond line")
			root := s.formDialog(form, info, 200)
			s.pages.AddPage("test_form", root, true, true)
			for _, size := range [][2]int{{200, 36}, {80, 24}, {40, 16}, {200, 36}} {
				terminal.SetSize(size[0], size[1])
				root.SetRect(0, 0, size[0], size[1])
				for i := 0; i < count+5; i++ {
					focusForm(s, form, i)
					root.Draw(terminal)
					x, y, w, h := form.GetRect()
					if i < count {
						fx, fy := formFieldCell(form, i)
						if fx >= x+w-1 || fy < y || fy >= y+h-1 {
							t.Fatalf("field %d clipped at %v", i, size)
						}
						if form.GetFormItem(i).(*tview.InputField).GetText() != "preserved" {
							t.Fatal("resize changed data")
						}
					} else {
						bx, by, bw, bh := form.GetButton(i - count).GetRect()
						if bw < 5 || bh != 1 || bx < x || bx+bw > x+w || by < y || by >= y+h {
							t.Fatalf("button %d clipped at %v: %d,%d %dx%d form %d,%d %dx%d", i-count, size, bx, by, bw, bh, x, y, w, h)
						}
					}
				}
				focusForm(s, form, count)
				if !s.buttonArrow(tcell.NewEventKey(tcell.KeyRight, 0, 0)) || !form.GetButton(1).HasFocus() {
					t.Fatal("popup button navigation failed")
				}
			}
			if form.GetButton(0).GetLabel() != "긴 작업 버튼 / Long action" {
				t.Fatal("wide layout failed to restore button labels")
			}
		})
	}
}
