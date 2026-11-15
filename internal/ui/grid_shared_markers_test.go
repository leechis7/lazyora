package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
)

// Exercise the actual consumer metadata against the published grid renderer:
// a dirty row marker must not color its unrelated cells, and clearing the
// transaction state must remove both cell and gutter cues.
func TestSharedGridDirtyMarkersStayCellSpecific(t *testing.T) {
	for _, theme := range []string{"dark", "light", "blackwhite"} {
		for _, colors := range []int{8, 16, 256, 1 << 24} {
			for _, ascii := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%d/ascii=%v", theme, colors, ascii), func(t *testing.T) {
					s, terminal := inlineFixture(t)
					s.prefs.Theme = theme
					s.terminal = auditTerminal{terminal, colors}
					s.limitedColors = colors <= 16
					if ascii {
						s.prefs.DisplayMode = "ascii"
					} else {
						s.prefs.DisplayMode = "unicode"
					}
					s.table.SetRect(0, 0, 120, 25)
					key := s.editedCellIdentity(s.resultBrowser.source, 0, 1)
					s.markEditedCell(key)
					s.resultGrid().Draw(terminal)
					model := s.grid.presentation.model
					assertMarkers := func(want bool) {
						t.Helper()
						gutter, cell, unrelated := false, false, false
						for _, line := range model.Layout(118, 23) {
							for _, span := range line.Spans {
								if span.Role == "grid.gutter" && strings.Contains(span.Text, "*") {
									gutter = true
								}
								if strings.Contains(span.Text, "first") && strings.Contains(span.Text, "*") {
									cell = true
								}
								if strings.Contains(span.Text, "rid1") && (span.Role == "grid.edited" || strings.Contains(span.Text, "*")) {
									unrelated = true
								}
							}
						}
						if gutter != want || cell != want || unrelated {
							t.Fatalf("want=%v gutter=%v cell=%v unrelated=%v", want, gutter, cell, unrelated)
						}
					}
					assertMarkers(true)
					// Focusing another column must not lose the dirty cues.
					s.table.Select(1, 0)
					s.resultGrid().Draw(terminal)
					assertMarkers(true)
					s.clearGridDrafts()
					s.resultGrid().Draw(terminal)
					assertMarkers(false)
				})
			}
		}
	}
}

func TestSharedGridHeaderUsesOneBackground(t *testing.T) {
	for _, theme := range []string{"dark", "light", "blackwhite"} {
		for _, colors := range []int{8, 16, 256, 1 << 24} {
			t.Run(fmt.Sprintf("%s/%d", theme, colors), func(t *testing.T) {
				s, terminal := inlineFixture(t)
				s.prefs.Theme = theme
				s.terminal = auditTerminal{terminal, colors}
				s.limitedColors = colors <= 16
				s.table.SetRect(0, 0, 80, 16)
				s.resultGrid().Draw(terminal)
				x, y, w, _ := s.table.GetInnerRect()
				_, _, header, _ := terminal.GetContent(x+w-1, y)
				_, _, body, _ := terminal.GetContent(x+w-1, y+1)
				_, bg, _ := header.Decompose()
				_, bodyBG, _ := body.Decompose()
				if bg == bodyBG {
					t.Fatal("header background is indistinguishable from body")
				}
				bold := false
				for col := x; col < x+w; col++ {
					r, _, style, _ := terminal.GetContent(col, y)
					fg, cellBG, attrs := style.Decompose()
					if cellBG != bg || (r != ' ' && fg == cellBG) {
						t.Fatalf("header cell %d is unreadable or breaks the band: %v", col, style)
					}
					bold = bold || r == 'C' && attrs&tuivory.AttrBold != 0
				}
				if !bold {
					t.Fatal("focused column heading is no longer bold")
				}
			})
		}
	}
}
