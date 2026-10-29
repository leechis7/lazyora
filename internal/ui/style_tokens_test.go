package ui

import (
	"fmt"
	"math"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func referenceContrast(fg, bg string) float64 {
	lum := func(hex string) float64 {
		r, g, b := tcell.GetColor(hex).RGB()
		linear := func(v int32) float64 {
			f := float64(v) / 255
			if f <= 0.04045 {
				return f / 12.92
			}
			return math.Pow((f+0.055)/1.055, 2.4)
		}
		return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
	}
	a, b := lum(fg), lum(bg)
	if a < b {
		a, b = b, a
	}
	return (a + 0.05) / (b + 0.05)
}

// This catches invisible/low-contrast input, selection and SQL under both
// common interpretations of ANSI bold, rather than comparing token constants.
func TestStyleProfilesKeepContentReadable(t *testing.T) {
	for _, count := range []int{8, 16, 256, 1 << 24} {
		for _, theme := range []string{"dark", "light", "blackwhite"} {
			t.Run(fmt.Sprintf("%d/%s", count, theme), func(t *testing.T) {
				s, base := resultScreen(t)
				s.terminal = auditTerminal{base, count}
				s.limitedColors = count <= 16
				s.prefs.Theme = theme
				styles := map[string]tcell.Style{
					"accent":  tcell.StyleDefault.Foreground(s.colors().accent).Background(s.colors().surface),
					"version": tcell.StyleDefault.Foreground(s.colors().text).Background(s.colors().footer),
					"warning": tcell.StyleDefault.Foreground(s.colors().warning).Background(s.colors().surface),
					"body":    tcell.StyleDefault.Foreground(s.colors().text).Background(s.colors().surface),
					"hint":    tcell.StyleDefault.Foreground(s.colors().muted).Background(s.colors().surface),
					"input":   s.fieldStyle(), "selection": s.focusStyle(), "button": s.buttonStyle(),
					"focused button": s.buttonFocusStyle(), "disabled": s.disabledStyle(), "edited": s.editedCellStyle(),
					"grid": s.gridBaseStyle(1, 1), "grid null": s.gridBaseStyle(1, 2), "active": s.gridActiveStyle(true),
				}
				for kind := sqlKeyword; kind <= sqlBind; kind++ {
					styles[fmt.Sprintf("SQL %d", kind)] = s.profileSQLStyle(kind, s.editorStyle())
				}
				for name, style := range styles {
					if count <= 16 {
						fg, bg, _ := style.Decompose()
						for _, color := range []tcell.Color{fg, bg} {
							if color.IsRGB() || int(color&^tcell.ColorValid) >= count {
								t.Errorf("%s exceeds %d-color profile: %v", name, count, color)
							}
						}
					}
					mapped := auditStyleOf(style, count)
					for _, colors := range [][2]string{{mapped.FGHex, mapped.BGHex}, {mapped.BrightFGHex, mapped.BrightBGHex}} {
						minimum := 4.5
						// Tuivory v0.4.0 deliberately distinguishes inactive controls with gray/Dim.
						// Its published disabled text target is 2:1; actionable text remains 4.5:1.
						if name == "disabled" {
							minimum = 2.0
						}
						if ratio := referenceContrast(colors[0], colors[1]); ratio < minimum {
							t.Errorf("%s contrast %.3f: %v", name, ratio, colors)
						}
					}
				}
			})
		}
	}
}

func TestReadOnlyFormDescriptionsAndCopyTitleRemainVisible(t *testing.T) {
	for _, count := range []int{8, 16, 256, 1 << 24} {
		for _, theme := range []string{"dark", "light", "blackwhite"} {
			t.Run(fmt.Sprintf("%d/%s", count, theme), func(t *testing.T) {
				spaces, base := workspaceTestSet(t)
				s := spaces.active
				s.terminal = auditTerminal{base, count}
				s.limitedColors = count <= 16
				s.prefs.Theme = theme
				s.applyTheme("dark")
				// Reproduce the original Form.AddTextView input-color leak at render time.
				form := tview.NewForm().AddTextView("", "Read-only 안내문", 0, 2, true, false).AddInputField("Value", "keep", 20, nil, nil)
				s.styleForm(form)
				s.pages.AddPage("read-only-regression", form, true, true)
				form.SetRect(0, 0, 90, 12)
				s.refreshBorders()
				form.Draw(base)
				found := 0
				for y := 0; y < 12; y++ {
					for x := 0; x < 90; x++ {
						r, _, st, _ := base.GetContent(x, y)
						if r == 'R' || r == '안' {
							a := auditStyleOf(st, count)
							if referenceContrast(a.FGHex, a.BGHex) < 4.5 {
								t.Fatal("form description invisible")
							}
							found++
						}
					}
				}
				if found < 2 {
					t.Fatal("description fixture was not drawn")
				}
				s.pages.RemovePage("read-only-regression")
				s.clipboardText = "copied 한글"
				s.copyPreview()
				focus := s.app.GetFocus()
				for _, next := range []string{theme, "light", "blackwhite", "dark", theme} {
					previous := s.prefs.Theme
					s.prefs.Theme = next
					s.applyTheme(previous)
					s.copyText.SetRect(0, 0, 90, 12)
					s.refreshBorders()
					s.copyText.Draw(base)
					var titleStyle tcell.Style
					foundTitle := false
					for x := 0; x < 90; x++ {
						r, _, st, _ := base.GetContent(x, 0)
						if r == 'C' {
							titleStyle = st
							foundTitle = true
							break
						}
					}
					if !foundTitle {
						t.Fatal("copy title not drawn")
					}
					st := auditStyleOf(titleStyle, count)
					if referenceContrast(st.FGHex, st.BGHex) < 4.5 {
						t.Fatal("copy title invisible", next)
					}
					if s.app.GetFocus() != focus || s.clipboardText != "copied 한글" {
						t.Fatal("theme changed focus or clipboard")
					}
				}
			})
		}
	}
}

func TestThemeRoundTripKeepsActiveTabFocusAndStyles(t *testing.T) {
	s, base := resultScreen(t)
	s.initDocument()
	s.initTabs()
	s.newTab()
	var active *tview.Button
	for i := 0; i < s.tabBar.GetItemCount(); i++ {
		if b, ok := s.tabBar.GetItem(i).(*tview.Button); ok {
			active = b
		}
	}
	if active == nil {
		t.Fatal("tab button missing")
	}
	s.app.SetFocus(active)
	s.editor.SetText("SELECT 'unchanged' FROM dual", false)
	paint := func() tcell.Style {
		base.Clear()
		active.SetRect(0, 0, 50, 1)
		active.Draw(base)
		_, _, st, _ := base.GetContent(25, 0)
		return st
	}
	original := paint()
	for _, theme := range []string{"light", "blackwhite", "dark"} {
		previous := s.prefs.Theme
		s.prefs.Theme = theme
		s.applyTheme(previous)
	}
	if s.app.GetFocus() != active || paint() != original || s.editor.GetText() != "SELECT 'unchanged' FROM dual" {
		t.Fatal("theme round trip changed active tab, its style, focus or SQL")
	}
}
