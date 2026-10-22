package ui

import (
	"fmt"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

// Compare painted cells, rather than Box alone: TextView and List each keep
// separate text styles which can otherwise leave black strips on light pages.
func paintedWidget(t *testing.T, terminal tcell.SimulationScreen, widget tview.Primitive) []tcell.Style {
	t.Helper()
	terminal.Clear()
	widget.SetRect(0, 0, 60, 6)
	widget.Draw(terminal)
	styles := make([]tcell.Style, 0, 360)
	for y := 0; y < 6; y++ {
		for x := 0; x < 60; x++ {
			_, _, style, _ := terminal.GetContent(x, y)
			styles = append(styles, style)
		}
	}
	return styles
}

func TestCommonWidgetThemeMatchesFreshScreen(t *testing.T) {
	for _, limited := range []bool{false, true} {
		for _, target := range []string{"dark", "light"} {
			t.Run(fmt.Sprintf("%s/limited=%v", target, limited), func(t *testing.T) {
				fresh, terminal := resultScreen(t)
				fresh.limitedColors, fresh.prefs.Theme = limited, target
				changed, changedTerminal := resultScreen(t)
				changed.limitedColors = limited
				previous := "dark"
				if target == "dark" {
					previous = "light"
				}
				changed.prefs.Theme = previous
				makeWidgets := func(s *screen) []tview.Primitive {
					body := s.newTextView().SetText("본문 body")
					hint := s.newTextView(textHint).SetText("안내 hint")
					accent := s.newTextView(textAccent).SetText("강조 accent")
					list := tview.NewList().AddItem("첫 항목", "보조 설명", 'a', nil).AddItem("두 번째", "detail", 'b', nil).SetCurrentItem(1)
					s.styleList(list)
					input := s.newInputField().SetLabel("입력 ").SetText("secret unchanged")
					check := tview.NewCheckbox().SetLabel("선택 ").SetChecked(true)
					s.styleCheckbox(check)
					widgets := []tview.Primitive{body, hint, accent, list, input, check}
					widgets = append(widgets, s.newTextView(textWarning).SetText("경고 warning"))
					table := tview.NewTable().SetCell(0, 0, tview.NewTableCell("객체 상세").SetTextColor(s.colors().text))
					s.styleBox(table.Box)
					widgets = append(widgets, table)
					root := s.newFlex()
					for _, widget := range widgets {
						root.AddItem(widget, 0, 1, false)
					}
					s.pages.AddPage("style-test", root, true, true)
					return widgets
				}
				want, got := makeWidgets(fresh), makeWidgets(changed)
				changed.prefs.Theme = target
				changed.applyTheme(previous)
				for index := range want {
					expected := paintedWidget(t, terminal, want[index])
					actual := paintedWidget(t, changedTerminal, got[index])
					for cell := range expected {
						if expected[cell] != actual[cell] {
							t.Fatalf("widget %T cell %d: fresh %v, switched %v", want[index], cell, expected[cell], actual[cell])
						}
					}
				}
				if got[3].(*tview.List).GetCurrentItem() != 1 || got[4].(*tview.InputField).GetText() != "secret unchanged" || !got[5].(*tview.Checkbox).IsChecked() {
					t.Fatal("recoloring changed selection, input or checkbox state")
				}
				for index, role := range []textRole{textBody, textHint, textAccent} {
					styles := paintedWidget(t, changedTerminal, got[index])
					fg, bg, _ := styles[0].Decompose()
					if fg != changed.textColor(role) || bg != changed.colors().background {
						t.Fatalf("text role %v lost its semantic colors", role)
					}
				}
			})
		}
	}
}

func TestClosedTextStyleRolesAreReleased(t *testing.T) {
	s, _ := resultScreen(t)
	s.showGuide()
	guide := s.reference.text
	s.showHelp()
	s.refreshBorders()
	if _, exists := s.textRoles[guide]; !exists {
		t.Fatal("hidden guide lost its style role")
	}
	s.closeHelp()
	s.closeReference()
	s.refreshBorders()
	if _, exists := s.textRoles[guide]; exists {
		t.Fatal("closed guide is retained by style bookkeeping")
	}
}

func TestPlanTreeThemeUsesCommonTextAndSelection(t *testing.T) {
	for _, limited := range []bool{false, true} {
		s, terminal := resultScreen(t)
		s.limitedColors = limited
		s.showPlan("SELECT 1 FROM dual", []oracle.PlanStep{{ID: 0, Operation: "SELECT STATEMENT"}})
		for _, theme := range []string{"dark", "light", "dark"} {
			previous := s.prefs.Theme
			s.prefs.Theme = theme
			s.applyTheme(previous)
			s.planTree.GetRoot().Walk(func(node, parent *tview.TreeNode) bool {
				fg, bg, _ := node.GetTextStyle().Decompose()
				wantFG := s.colors().text
				if parent == nil {
					wantFG = s.colors().accent
				}
				if fg != wantFG || bg != s.colors().background || node.GetSelectedTextStyle() != s.focusStyle() {
					t.Fatalf("plan node style differs from common palette: %s limited=%v", theme, limited)
				}
				return true
			})
			paintedWidget(t, terminal, s.planTree)
		}
	}
}

func TestLimitedPaletteUsesNeutralSurfacesAndContrastingControls(t *testing.T) {
	for _, theme := range []string{"dark", "light"} {
		s := &screen{limitedColors: true}
		s.prefs.Theme = theme
		p := s.colors()
		for _, color := range []tcell.Color{p.background, p.surface, p.footer, p.button, p.field, p.text} {
			if color.IsRGB() || color == tcell.ColorTeal || color == tcell.ColorAqua {
				t.Fatalf("%s uses RGB approximation or a cyan surface", theme)
			}
		}
		for _, style := range []tcell.Style{s.fieldStyle(), s.focusStyle(), s.buttonStyle(), buttonFocusStyle()} {
			fg, bg, _ := style.Decompose()
			if fg == bg {
				t.Fatalf("%s control has no foreground/background contrast", theme)
			}
		}
	}
}
