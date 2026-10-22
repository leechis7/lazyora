package ui

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"lazyora/internal/settings"
)

func TestBlackWhiteThemeRendersWithoutChromaticColors(t *testing.T) {
	for _, limited := range []bool{false, true} {
		for _, page := range []string{"main", "results", "source", "help", "guide", "about", "connect", "dropdown", "settings", "output", "plan", "menu", "confirm"} {
			t.Run(fmt.Sprintf("%s/limited=%v", page, limited), func(t *testing.T) {
				spaces, terminal := workspaceTestSet(t)
				s := spaces.active
				s.prefs.Theme = "blackwhite"
				s.applyTheme("dark")
				s.limitedColors = limited
				s.editor.SetText("SELECT '한글', :value FROM dual; -- comment", false)
				switch page {
				case "results":
					s.app.SetFocus(s.table)
					s.resultSelection = resultSelection{anchorRow: 1, anchorCol: 0, endRow: 1, endCol: 1, active: true}
					s.highlightResults()
				case "source":
					s.browserTypes.SetCurrentOption(3)
					s.activateBrowser(s.browser.GetCurrentNode())
					s.key(tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModAlt))
				case "confirm":
					s.showProgramConfirm("Review changes?", func() {})
				case "help":
					s.showHelp()
				case "guide":
					s.showGuide()
				case "about":
					s.showAbout()
				case "connect":
					s.connectionForm()
				case "dropdown":
					s.connectionForm()
					focusForm(s, s.connectForm, connectionMode)
					s.connectForm.GetFormItem(connectionMode).(*choiceField).InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), func(p tview.Primitive) { s.app.SetFocus(p) })
				case "settings":
					s.settingsForm()
				case "output":
					s.showOutput()
				case "plan":
					s.showPlan("SELECT 1 FROM dual", []oracle.PlanStep{{ID: 0, Operation: "SELECT STATEMENT"}})
				case "menu":
					s.showMenu(menuView)
				}
				s.refreshBorders()
				s.rootLayout.SetRect(0, 0, 120, 40)
				s.rootLayout.Draw(terminal)
				for y := 0; y < 40; y++ {
					for x := 0; x < 120; x++ {
						_, _, style, _ := terminal.GetContent(x, y)
						fg, bg, _ := style.Decompose()
						for _, color := range []tcell.Color{fg, bg} {
							r, g, b := color.RGB()
							if r != g || g != b {
								t.Fatalf("chromatic color %v at %d,%d", color, x, y)
							}
						}
					}
				}
			})
		}
	}
}

func TestLightEditorUsesExplicitWhite(t *testing.T) {
	s, terminal := resultScreen(t)
	s.prefs.Theme = "light"
	s.applyTheme("dark")
	s.editor.SetText("SELECT 'white' FROM dual;", false).SetRect(0, 0, 80, 10)
	s.editor.Draw(terminal)
	x, y, _, _ := s.editor.GetInnerRect()
	_, _, style, _ := terminal.GetContent(x, y)
	_, bg, _ := style.Decompose()
	r, g, b := bg.RGB()
	if !bg.IsRGB() || r != 255 || g != 255 || b != 255 {
		t.Fatal("rich-color light editor relies on the terminal's gray/white palette")
	}
}

func TestBlackWhiteSettingsPersistAndRestoreEditorTheme(t *testing.T) {
	s, _ := resultScreen(t)
	s.prefsPath = filepath.Join(t.TempDir(), "settings.json")
	s.editor.SetText("SELECT 'keep' FROM dual;", false).Select(7, 13)
	s.render(oracle.Result{Columns: []string{"VALUE"}, Rows: [][]string{{"unchanged"}}})
	for _, index := range []int{2, 1, 0, 2} {
		s.settingsForm()
		s.prefsForm.GetFormItem(3).(*choiceField).SetCurrentOption(index)
		s.savePrefs()
		config, err := settings.Load(s.prefsPath)
		if err != nil || config.Theme != s.prefs.Theme {
			t.Fatal("theme was not persisted", err)
		}
		if s.editor.GetBackgroundColor() != s.colors().editor || s.editor.GetText() != "SELECT 'keep' FROM dual;" || s.resultRows[0][0] != "unchanged" {
			t.Fatal("theme switch changed work or left the old editor background")
		}
		_, start, end := s.editor.GetSelection()
		if start != 7 || end != 13 || len(s.jobs) != 0 {
			t.Fatal("theme switch changed selection or submitted DB work")
		}
	}
}
