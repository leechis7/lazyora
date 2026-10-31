package ui

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// GetContent alone reports an intact border even when the physical renderer
// skips that cell because a surviving wide glyph starts immediately before it.
func assertPopupPhysicalCells(t *testing.T, terminal tcell.SimulationScreen, x, y, w, h int) {
	t.Helper()
	terminal.Show()
	cells, sw, sh := terminal.GetContents()
	for py := y; py < y+h && py < sh; py++ {
		for px := x; px < x+w && px < sw; {
			r, comb, style, cw := terminal.GetContent(px, py)
			if r == 0 {
				r = ' '
			}
			physical := cells[py*sw+px]
			if !reflect.DeepEqual(physical.Runes, append([]rune{r}, comb...)) || physical.Style != style {
				t.Fatalf("physical output differs at %d,%d: logical %q physical %q", px, py, string(append([]rune{r}, comb...)), string(physical.Runes))
			}
			px += max(1, cw)
		}
	}
}

func TestPopupWideGlyphsDoNotHidePhysicalBorders(t *testing.T) {
	for _, page := range []string{"connect", "settings", "about"} {
		for _, language := range []string{"ko", "en"} {
			for _, theme := range []string{"dark", "light", "blackwhite"} {
				for _, colors := range []int{8, 16, 256, 1 << 24} {
					t.Run(fmt.Sprintf("%s/%s/%s/%d", page, language, theme, colors), func(t *testing.T) {
						s, terminal := resultScreen(t)
						s.terminal = auditTerminal{terminal, colors}
						s.prefs.Language, s.prefs.Theme = language, theme
						s.limitedColors = colors <= 16
						sql := s.editor.GetText()
						rows := fmt.Sprint(s.resultRows)
						switch page {
						case "connect":
							s.connectionForm()
						case "settings":
							s.settingsForm()
						case "about":
							s.showAbout()
						}
						_, root := s.pages.GetFrontPage()
						popup := root.(*formPopupLayer)
						focus := s.app.GetFocus()
						for _, size := range [][2]int{{120, 36}, {80, 24}, {41, 16}, {120, 36}} {
							terminal.SetSize(size[0], size[1])
							popup.SetRect(0, 0, size[0], size[1])
							popup.Draw(terminal)
							x, y, w, h := popup.content.GetRect()
							for _, glyph := range []rune{'한', '界', '🧪'} {
								terminal.Clear()
								for py := y; py < y+h; py++ {
									if x > 0 {
										terminal.SetContent(x-1, py, glyph, []rune{'\u0301'}, tcell.StyleDefault)
									}
									terminal.SetContent(x+w-1, py, glyph, nil, tcell.StyleDefault)
								}
								// The bottom shadow starts one column to the right of the card.
								if y+h < size[1] {
									terminal.SetContent(x, y+h, glyph, nil, tcell.StyleDefault)
								}
								terminal.Show()
								for frame := 0; frame < 2; frame++ {
									popup.Draw(terminal)
									assertPopupPhysicalCells(t, terminal, x, y, min(w+1, size[0]-x), min(h+1, size[1]-y))
								}
							}
						}
						if s.editor.GetText() != sql || fmt.Sprint(s.resultRows) != rows || s.app.GetFocus() != focus {
							t.Fatal("popup drawing changed SQL, result values or focus")
						}
					})
				}
			}
		}
	}
}

func TestPopupWideBoundaryPreservesOutsideContentAndRestoresOnClose(t *testing.T) {
	s, terminal := resultScreen(t)
	terminal.SetSize(80, 24)
	background := func() {
		terminal.Clear()
		for y := 0; y < 24; y++ {
			for x := 0; x < 80; x += 2 {
				terminal.SetContent(x, y, '한', nil, tcell.StyleDefault.Foreground(tcell.ColorGreen))
			}
		}
	}
	background()
	terminal.Show()
	original, _, _ := terminal.GetContents()
	baseline := append([]tcell.SimCell(nil), original...)
	popup := s.formPopup(tview.NewBox().SetBorder(true), 40, 10, nil)
	popup.SetRect(0, 0, 80, 24)
	popup.Draw(terminal)
	x, y, w, h := popup.content.GetRect()
	// Both ordinary and shadow boundaries cut through a wide glyph.
	assertPopupPhysicalCells(t, terminal, x, y, w+1, h+1)
	r, _, style, cw := terminal.GetContent(x-2, y+2)
	wantStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen).Dim(true)
	if r != '한' || cw != 2 || style != wantStyle {
		t.Fatal("intact wide glyph outside popup was erased")
	}
	// Repaint the underlying page after closing; drawing substitutions must not
	// alter the source and leave permanent holes in its next physical frame.
	background()
	terminal.Show()
	restored, _, _ := terminal.GetContents()
	// A wide glyph's continuation slot is not independently emitted by tcell.
	// Compare the visible leading cells, including their combining marks/style.
	for py := 0; py < 24; py++ {
		for px := 0; px < 80; px += 2 {
			if !reflect.DeepEqual(baseline[py*80+px], restored[py*80+px]) {
				t.Fatalf("background did not restore at %d,%d", px, py)
			}
		}
	}
}
