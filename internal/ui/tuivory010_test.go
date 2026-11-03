package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
	"lazyora/internal/oracle"
	"lazyora/internal/settings"
)

// Inspect actual application output: cell metadata alone missed disappearing
// wide glyphs and offscreen confirmation buttons during the previous migration.
func TestTuivoryRenderedUpgradeRegressions(t *testing.T) {
	for _, key := range []string{"LAZYORA_HOST", "LAZYORA_USER", "LAZYORA_PASSWORD", "LAZYORA_NAME"} {
		t.Setenv(key, "")
	}
	for _, theme := range []string{"dark", "light", "blackwhite"} {
		for _, colors := range []int{8, 16, 256, 1 << 24} {
			for _, lang := range []string{"ko", "en"} {
				t.Run(fmt.Sprintf("%s/%d/%s", theme, colors, lang), func(t *testing.T) {
					raw := colorTestTerminal(t, 80, 24)
					terminal := auditTerminal{raw, colors}
					app := tuivory.NewApplication().SetScreen(terminal)
					s := newWorkspaceScreen(app, terminal, settings.Config{Language: lang, Theme: theme, Mouse: true, DisplayMode: "unicode"}, filepath.Join(t.TempDir(), "settings.json"), true, nil)
					spaces := &workspaceSet{app: app, active: s}
					spaces.adopt(s)
					t.Cleanup(spaces.shutdown)
					s.editor.SetText(" SELECT NVL(:id, CAST('text' AS NUMBER)), 12 FROM dual; -- comment", true)
					original := s.editor.GetText()
					app.ForceDraw()
					for _, divider := range []*tuivory.Box{s.sizing.horizontal, s.sizing.vertical} {
						x, y, w, h := divider.GetRect()
						if w < 1 || h < 1 {
							t.Fatal("missing splitter")
						}
						r, _, st, _ := raw.GetContent(x, y)
						fg, bg, _ := st.Decompose()
						if (r != '│' && r != '─') || fg == bg {
							t.Fatalf("invisible splitter %q: %v/%v", r, fg, bg)
						}
					}
					if theme == "dark" && colors == 8 {
						roles := map[sqlColorKind]bool{}
						styles := map[tuivory.Style]bool{}
						x, y, _, _ := s.editor.GetInnerRect()
						for _, span := range colorSQL(original, nil) {
							_, _, style, _ := raw.GetContent(x+span.start, y)
							fg, bg, _ := style.Decompose()
							if fg == bg {
								t.Fatal("invisible SQL token")
							}
							roles[span.kind] = true
							styles[style] = true
						}
						if len(roles) != 7 || len(styles) != 7 {
							t.Fatalf("8-color SQL roles not distinct: %d roles, %d styles", len(roles), len(styles))
						}
					}
					s.showKeymapComparison()
					for _, size := range [][2]int{{80, 24}, {40, 16}, {120, 40}, {80, 24}} {
						raw.SetSize(size[0], size[1])
						app.ForceDraw()
						// Include changing current cells: Table restyles every wide-cell tail.
						s.comparison.table.Select(2, 2)
						app.ForceDraw()
						raw.Sync()
						table := s.comparison.table
						x, y, _ := table.GetCell(0, 0).GetLastPosition()
						want := s.tr("comparison_action")
						if got := renderedAuditRow(raw, x, y, tuivory.TaggedStringWidth(want)); got != want {
							t.Fatalf("header text lost at %v: %q != %q", size, got, want)
						}
						x, y, _ = table.GetCell(2, 0).GetLastPosition()
						want = table.GetCell(2, 0).Text
						if size[0] >= 80 {
							if got := renderedAuditRow(raw, x, y, tuivory.TaggedStringWidth(want)); got != want {
								t.Fatalf("body text lost: %q != %q", got, want)
							}
						}
						cells, _, _ := raw.GetContents()
						for _, cell := range cells {
							if string(cell.Runes) == "│" {
								fg, bg, _ := cell.Style.Decompose()
								if fg == bg {
									t.Fatal("comparison separator invisible")
								}
							}
						}
					}
					s.closeKeymapComparison()
					s.pendingChanges = true
					s.workerClosed = true
					s.requestReconnect(oracle.Config{})
					s.workerClosed = false
					for _, size := range [][2]int{{40, 16}, {80, 24}, {40, 16}} {
						raw.SetSize(size[0], size[1])
						app.ForceDraw()
						button := app.GetFocus()
						x, y, w, h := button.GetRect()
						if x < 0 || y < 0 || w < 1 || h < 1 || x+w > size[0] || y+h > size[1] {
							t.Fatalf("default Cancel outside %v: %d,%d %dx%d", size, x, y, w, h)
						}
						if got := renderedAuditRow(raw, x, y, w); !strings.Contains(got, s.tr("cancel")) {
							t.Fatalf("default Cancel not rendered: %q", got)
						}
					}
					app.GetFocus().InputHandler()(tuivory.NewEventKey(tuivory.KeyEnter, 0, 0), func(p tuivory.Primitive) { app.SetFocus(p) })
					if s.reconnectDialog != nil || !s.pendingChanges || s.busy || s.editor.GetText() != original {
						t.Fatal("default cancel changed transaction or SQL")
					}
				})
			}
		}
	}
}

func renderedAuditRow(screen tuivory.SimulationScreen, x, y, width int) string {
	cells, cols, rows := screen.GetContents()
	if y < 0 || y >= rows || x < 0 || x >= cols {
		return ""
	}
	var out strings.Builder
	for col := x; col < min(cols, x+width); {
		glyph := string(cells[y*cols+col].Runes)
		out.WriteString(glyph)
		col += max(1, tuivory.TaggedStringWidth(glyph))
	}
	return out.String()
}
