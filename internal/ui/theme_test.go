package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"lazyora/internal/settings"
)

func TestThemeSwitchPreservesWorkspaceAndCredentials(t *testing.T) {
	for _, limited := range []bool{false, true} {
		t.Run(map[bool]string{false: "rich", true: "ansi"}[limited], func(t *testing.T) {
			s, terminal := resultScreen(t)
			s.limitedColors = limited
			s.prefsPath = filepath.Join(t.TempDir(), "settings.json")
			text := "SELECT '한글' FROM dual;"
			s.editor.SetText(text, false).Select(7, 15)
			s.render(oracle.Result{Columns: []string{"TEXT"}, Rows: [][]string{{"original value"}}})
			s.pendingChanges = true
			s.connectionForm()
			input := s.connectForm.GetFormItem(connectionPassword).(*tview.InputField)
			input.SetText("not-for-config")
			profile := s.connectProfiles.GetFormItem(0).(*choiceField)
			for _, theme := range []string{"light", "dark", "light"} {
				s.settingsForm()
				s.key(tcell.NewEventKey(tcell.KeyRune, 't', tcell.ModAlt))
				if !s.prefsForm.GetFormItem(3).HasFocus() {
					t.Fatal("Alt+T did not focus theme")
				}
				index := 0
				if theme == "light" {
					index = 1
				}
				s.prefsForm.GetFormItem(3).(*choiceField).SetCurrentOption(index)
				s.savePrefs()
				if s.editor.GetBackgroundColor() != s.colors().editor {
					t.Fatal("editor did not follow theme")
				}
				if s.prefs.Theme != theme || s.connectForm.GetBackgroundColor() != s.colors().surface {
					t.Fatal("theme not applied to open form")
				}
				if profile.background != s.colors().footer && !limited {
					t.Fatal("existing choice retained old palette")
				}
				if input.GetText() != "not-for-config" || s.editor.GetText() != text || s.resultRows[0][0] != "original value" || !s.pendingChanges || len(s.jobs) != 0 {
					t.Fatal("theme switch changed work or submitted a job")
				}
				if _, start, end := s.editor.GetSelection(); start != 7 || end != 15 {
					t.Fatal("selection lost")
				}
				if !s.connectForm.HasFocus() {
					t.Fatal("focus was not restored")
				}
				_, root := s.pages.GetFrontPage()
				root.SetRect(0, 0, 100, 32)
				root.Draw(terminal)
				config, err := settings.Load(s.prefsPath)
				if err != nil || config.Theme != theme {
					t.Fatal("theme not persisted", err)
				}
				data, _ := os.ReadFile(s.prefsPath)
				if string(data) == "" || strings.Contains(string(data), "not-for-config") {
					t.Fatal("credential entered settings")
				}
			}
		})
	}
}

func TestLightGridAndSyntaxContrast(t *testing.T) {
	for _, limited := range []bool{false, true} {
		s, terminal := resultScreen(t)
		s.limitedColors = limited
		s.prefs.Theme = "light"
		s.editor.SetText("SELECT '한글' FROM dual;", false)
		s.editor.SetTextStyle(s.fieldStyle()).SetSelectedStyle(s.focusStyle()).SetBorder(false)
		s.editor.SetRect(0, 0, 80, 5)
		s.colorEditor(s.editor, nil)
		s.editor.Draw(terminal)
		_, _, st, _ := terminal.GetContent(0, 0)
		fg, bg, _ := st.Decompose()
		if fg == bg {
			t.Fatal("SQL keyword invisible")
		}
		s.render(oracle.Result{Columns: []string{"N", "VALUE"}, Rows: [][]string{{"1", "한글"}, {"2", "other"}}})
		s.table.SetRect(0, 0, 80, 15)
		s.app.SetFocus(s.table)
		s.table.Select(1, 0)
		s.resultGrid().Draw(terminal)
		for row := 0; row < 3; row++ {
			fg, bg, _ := s.table.GetCell(row, 0).Style.Decompose()
			if fg == bg {
				t.Fatal("grid cell invisible", row)
			}
		}
		if !limited && s.table.GetBackgroundColor() != lightGridPalette.background {
			t.Fatal("grid still dark")
		}
		s.table.Select(1, 1)
		s.copyResults("cell")
		if s.clipboardText != "한글" {
			t.Fatal("theme changed raw copy")
		}
	}
}

func TestScreenThemesAreIndependent(t *testing.T) {
	dark, _ := resultScreen(t)
	light, _ := resultScreen(t)
	light.prefs.Theme = "light"
	light.connectionForm()
	dark.connectionForm()
	if dark.connectForm.GetBackgroundColor() == light.connectForm.GetBackgroundColor() {
		t.Fatal("themes leaked across screens")
	}
}

func TestHiddenTabFollowsThemeWithoutSwitchingItsData(t *testing.T) {
	s, _ := resultScreen(t)
	s.limitedColors = true
	s.initDocument()
	s.initTabs()
	s.render(oracle.Result{Columns: []string{"VALUE"}, Rows: [][]string{{"first"}}})
	first := s.table
	s.newTab()
	s.render(oracle.Result{Columns: []string{"VALUE"}, Rows: [][]string{{"second"}}})
	s.prefs.Theme = "light"
	s.applyTheme("dark")
	fg, bg, _ := first.GetCell(1, 0).Style.Decompose()
	if fg != tcell.ColorBlack || bg == tcell.ColorBlack {
		t.Fatal("hidden ANSI result kept dark colors")
	}
	if first.GetBackgroundColor() != tcell.ColorWhite {
		t.Fatal("hidden grid background kept old theme")
	}
	s.switchTab(0)
	if s.resultRows[0][0] != "first" {
		t.Fatal("theme changed inactive tab data")
	}
}

func TestLightToDarkKeepsCopyPreviewReadable(t *testing.T) {
	s, terminal := resultScreen(t)
	s.prefs.Theme = "light"
	s.table.Select(1, 0)
	s.copyResults("cell")
	s.copyPreview()
	s.prefs.Theme = "dark"
	s.applyTheme("light")
	if s.copyText.GetBackgroundColor() != s.colors().field {
		t.Fatal("read-only field was mistaken for SQL editor")
	}
	s.copyText.SetRect(0, 0, 80, 10)
	s.copyText.Draw(terminal)
}
