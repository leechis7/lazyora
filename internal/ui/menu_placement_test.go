package ui

import (
	"fmt"
	"testing"
)

func TestTopMenuAnchorsToItsVisibleLabelAfterResize(t *testing.T) {
	for _, lang := range []string{"ko", "en"} {
		t.Run(lang, func(t *testing.T) {
			s, terminal := resultScreen(t)
			s.prefs.Language = lang
			bar := s.menuBar()
			for _, size := range [][2]int{{160, 40}, {80, 24}, {40, 12}, {20, 8}, {160, 40}} {
				for group := range menuGroups {
					t.Run(fmt.Sprintf("%dx%d/%d", size[0], size[1], group), func(t *testing.T) {
						terminal.SetSize(size[0]+14, size[1]+6)
						s.showMenu(group)
						// The menu page starts below the menu bar, at a nonzero parent origin.
						bar.SetRect(7, 3, size[0], 1)
						bar.Draw(terminal)
						_, page := s.pages.GetFrontPage()
						page.SetRect(7, 4, size[0], size[1]-1)
						page.Draw(terminal)
						x, y, w, h := s.menuCard.GetRect()
						if y != 4 || x < 7 || x+w > 7+size[0] || h < 0 || y+h > 3+size[1] {
							t.Fatalf("popup escaped menu viewport: %d,%d %dx%d", x, y, w, h)
						}
						found := false
						for _, cell := range s.mainMenu.BarLayout(size[0]).Cells {
							if cell.ID != menuRoot(group) {
								continue
							}
							found = true
							labelX := 7 + cell.Start
							if labelX+w <= 7+size[0] {
								if x != labelX {
									t.Fatalf("menu opened at x=%d instead of its label x=%d", x, labelX)
								}
							} else if x+w != 7+size[0] {
								t.Fatalf("clipped menu must align to right edge: %d+%d", x, w)
							}
						}
						if !found {
							t.Fatal("active menu label not visible")
						}
						s.closeMenu()
					})
				}
			}
		})
	}
}
