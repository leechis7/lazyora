package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/leechis7/tuivory/go"
	"lazyora/internal/oracle"
	"lazyora/internal/settings"
)

type displayTestTerminal struct {
	tcell.Screen
	charset, missing string
	colors           int
}

func (d displayTestTerminal) CharacterSet() string { return d.charset }
func (d displayTestTerminal) CanDisplay(r rune, _ bool) bool {
	return !strings.ContainsRune(d.missing, r)
}
func (d displayTestTerminal) Colors() int { return d.colors }

func TestDisplayDetectionAndLanguageAreIndependentOfColorsAndTERM(t *testing.T) {
	for _, term := range []string{"", "ansi", "linux", "xterm"} {
		t.Setenv("TERM", term)
		for _, colors := range []int{8, 16, 256, 1 << 24} {
			terminal := displayTestTerminal{charset: "UTF-8", colors: colors}
			if got := detectDisplay(terminal, "auto"); got.mode != "unicode" || !got.korean {
				t.Fatalf("TERM=%s colors=%d changed character support: %+v", term, colors, got)
			}
		}
	}
	for _, tc := range []struct {
		charset, missing, requested, want string
		korean                            bool
	}{
		{"ASCII", "", "auto", "ascii", true},
		{"US-ASCII", "한", "", "ascii", false},
		{"UTF-8", "▦", "auto", "ascii", true},
		{"UTF-8", "한", "auto", "unicode", false},
		{"UTF-8", "", "ascii", "ascii", true},
		{"ASCII", "한▦", "unicode", "unicode", false},
	} {
		got := detectDisplay(displayTestTerminal{charset: tc.charset, missing: tc.missing}, tc.requested)
		if got.mode != tc.want || got.korean != tc.korean {
			t.Fatalf("%+v => %+v", tc, got)
		}
	}
	s := &screen{prefs: settings.Config{Language: "ko"}, terminal: displayTestTerminal{charset: "UTF-8", missing: "한"}}
	if s.uiLanguage() != "en" || s.prefs.Language != "ko" {
		t.Fatal("effective language did not preserve Korean preference")
	}
	s.terminal = displayTestTerminal{charset: "UTF-8"}
	if s.uiLanguage() != "ko" {
		t.Fatal("normal terminal did not restore Korean UI")
	}
}

func assertASCIIFrame(t *testing.T, terminal tcell.Screen) string {
	t.Helper()
	var out strings.Builder
	w, h := terminal.Size()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, comb, _, _ := terminal.GetContent(x, y)
			if r > 127 || len(comb) > 0 {
				t.Fatalf("non-ASCII cell %q + %v at %d,%d", r, comb, x, y)
			}
			if r == 0 {
				r = ' '
			}
			out.WriteRune(r)
		}
		out.WriteByte('\n')
	}
	return out.String()
}

func TestASCIIWholeLayoutPreservesDataAndCopyAcrossThemesAndPopups(t *testing.T) {
	for _, theme := range []string{"dark", "light", "blackwhite"} {
		for _, colors := range []int{8, 16, 256, 1 << 24} {
			t.Run(fmt.Sprintf("%s/%d", theme, colors), func(t *testing.T) {
				spaces, base := workspaceTestSet(t)
				s := spaces.active
				s.terminal = displayTestTerminal{Screen: base, charset: "UTF-8", colors: colors}
				s.prefs.Language, s.prefs.Theme, s.prefs.DisplayMode = "ko", theme, "ascii"
				s.applyTheme("dark")
				s.refreshLabels()
				query := "SELECT '한글👩‍💻é' FROM dual;"
				value := "한글👩‍💻é[red]"
				s.editor.SetText(query, false)
				s.demo = false // Real result values must not be relabeled as demo strings.
				s.render(oracle.Result{Columns: []string{"표시"}, Rows: [][]string{{value}}})
				s.pendingChanges = true
				s.app.SetFocus(s.browser)
				for _, page := range []string{"main", "settings", "help", "guide", "connect", "menu", "about", "output", "plan", "copy", "confirm", "resize"} {
					s.pages.SwitchToPage("main")
					switch page {
					case "settings":
						s.settingsForm()
					case "help":
						s.showHelp()
					case "guide":
						s.showGuide()
					case "connect":
						s.connectionForm()
					case "menu":
						s.showMenu(menuView)
					case "about":
						s.showAbout()
					case "output":
						s.showOutput()
					case "plan":
						s.showPlan("SELECT '한글' FROM dual", []oracle.PlanStep{{ID: 0, Operation: "SELECT STATEMENT"}})
					case "copy":
						s.clipboardText = value
						s.copyPreview()
					case "confirm":
						s.showProgramConfirm("Continue?", func() {})
					case "resize":
						s.resizePanelsCommand()
					}
					base.SetSize(120, 40)
					s.refreshBorders()
					s.rootLayout.SetRect(0, 0, 120, 40)
					s.rootLayout.Draw(base)
					assertASCIIFrame(t, base)
					switch page {
					case "settings":
						s.closeSettings()
					case "help":
						s.closeHelp()
					case "guide":
						s.closeReference()
					case "connect":
						s.closeConnection()
					case "menu":
						s.closeMenu()
					case "about":
						s.closeAbout()
					case "output":
						s.closeOutput()
					case "plan":
						s.closePlan()
					case "copy":
						s.closeCopyPreview()
					case "confirm":
						s.pages.RemovePage("program_confirm")
						s.programModal = nil
					case "resize":
						s.finishPanelSizing()
					}
				}
				s.app.SetFocus(s.table)
				s.table.Select(1, 0)
				s.copyResults("cell")
				if s.clipboardText != value || s.editor.GetText() != query || s.resultRows[0][0] != value || !s.pendingChanges || s.prefs.Language != "ko" {
					t.Fatal("display mode modified work, copy or saved language")
				}
			})
		}
	}
}

func TestDisplaySettingsSaveOverrideCancelAndWorkspaceReuse(t *testing.T) {
	spaces, terminal := workspaceTestSet(t)
	first := spaces.active
	second := secondWorkspace(spaces)
	second.prefs.Language = "ko"
	second.settingsForm()
	second.key(tcell.NewEventKey(tcell.KeyRune, 'd', tcell.ModAlt))
	if !second.prefsForm.GetFormItem(4).HasFocus() {
		t.Fatal("Alt+D did not focus display choice")
	}
	second.prefsForm.GetFormItem(4).(*choiceField).SetCurrentOption(2)
	second.savePrefs()
	saved, err := settings.Load(second.prefsPath)
	if err != nil || saved.DisplayMode != "ascii" || saved.Language != "ko" || second.uiLanguage() != "en" || first.prefs.DisplayMode != "ascii" {
		t.Fatal("display mode did not persist or sync without changing language", err, saved)
	}
	if !spaces.activate(first) || first.rootLayout.owner != first {
		t.Fatal("workspace lost rendering policy")
	}
	first.rootLayout.SetRect(0, 0, 120, 40)
	first.rootLayout.Draw(terminal)
	assertASCIIFrame(t, terminal)
	first.displayOverride = "unicode"
	first.settingsForm()
	first.prefsForm.GetFormItem(4).(*choiceField).SetCurrentOption(0)
	first.savePrefs()
	saved, _ = settings.Load(first.prefsPath)
	if saved.DisplayMode != "auto" || saved.Language != "ko" || first.displayCapability().mode != "unicode" {
		t.Fatal("one-run override was persisted over saved selection")
	}
	first.settingsForm()
	first.prefsForm.GetFormItem(4).(*choiceField).SetCurrentOption(2)
	first.closeSettings()
	if first.prefs.DisplayMode != "auto" {
		t.Fatal("cancel applied unsaved mode")
	}
	first.displayOverride = ""
	if first.uiLanguage() != "ko" {
		t.Fatal("clearing runtime override lost original language")
	}
	// A failed save must not change the active mode.
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	first.prefsPath = filepath.Join(blocker, "settings.json")
	first.settingsForm()
	first.prefsForm.GetFormItem(4).(*choiceField).SetCurrentOption(2)
	first.savePrefs()
	if first.prefs.DisplayMode != "auto" {
		t.Fatal("failed save applied display mode")
	}
	first.closeSettings()
}

func TestASCIIWideFallbackKeepsFollowingCellAndCombiningMarksOutOfFrame(t *testing.T) {
	base := colorTestTerminal(t, 10, 2)
	a := asciiScreen{base}
	st := tcell.StyleDefault
	a.SetContent(0, 0, '한', nil, st)
	a.SetContent(2, 0, 'X', nil, st)
	a.SetContent(3, 0, 'e', []rune{'\u0301'}, st)
	a.SetContent(4, 0, '═', nil, st)
	text := assertASCIIFrame(t, base)
	if !strings.HasPrefix(text, "??Xe=") {
		t.Fatalf("wide fallback shifted data: %q", text)
	}
}

func TestInvalidDisplayOverrideDoesNotOpenTerminal(t *testing.T) {
	if err := RunWithDisplayMode(false, "bad-mode"); err == nil {
		t.Fatal("invalid CLI mode accepted")
	}
}

func TestDisplaySavePreservesRealResultAfterDemoStartup(t *testing.T) {
	spaces, _ := workspaceTestSet(t)
	s := spaces.active
	s.connected = true
	s.pendingChanges = true
	s.editor.SetText("SELECT '한글' FROM dual", false)
	s.render(oracle.Result{Columns: []string{"ID", "VALUE"}, Rows: [][]string{{"1", "실제 결과 원문"}}})
	s.settingsForm()
	s.prefsForm.GetFormItem(4).(*choiceField).SetCurrentOption(2)
	s.savePrefs()
	if s.resultRows[0][1] != "실제 결과 원문" || !s.pendingChanges || s.editor.GetText() != "SELECT '한글' FROM dual" {
		t.Fatal("display save replaced real data with demo labels")
	}
	s.table.Select(1, 1)
	s.copyResults("cell")
	if s.clipboardText != "실제 결과 원문" {
		t.Fatal("copy lost real result")
	}
}

func TestASCIIWritesThroughTuivorySafeScreenPreserveWidths(t *testing.T) {
	for _, order := range []string{"host-first", "library-first"} {
		t.Run(order, func(t *testing.T) {
			base := tcell.NewSimulationScreen("UTF-8")
			if err := base.Init(); err != nil {
				t.Fatal(err)
			}
			defer base.Fini()
			base.SetSize(40, 4)
			var writer tcell.Screen = asciiScreen{tuivory.WideSafeScreen(base)}
			if order == "library-first" {
				writer = tuivory.WideSafeScreen(asciiScreen{base})
			}
			style := tcell.StyleDefault.Foreground(tcell.ColorGreen)
			writer.SetContent(0, 0, '한', nil, style)
			rest, width := writer.Put(3, 0, "界Z", style)
			if rest != "Z" || width != 2 {
				t.Fatalf("Put lost original grapheme boundaries: %q/%d", rest, width)
			}
			writer.SetCell(6, 0, style, []rune("👩‍💻")...)
			writer.PutStrStyled(10, 0, "A한é", style)
			writer.PutStr(18, 0, "─한")
			if rest, width = writer.Put(-1, 0, "한", style); rest != "한" || width != 0 {
				t.Fatal("out-of-bounds Put consumed input")
			}
			assertASCIIFrame(t, base)
			for x, want := range map[int]rune{0: '?', 1: '?', 3: '?', 4: '?', 6: '?', 7: '?', 10: 'A', 11: '?', 12: '?', 13: 'e', 18: '-', 19: '?', 20: '?'} {
				got, comb, _, _ := base.GetContent(x, 0)
				if got != want || len(comb) != 0 {
					t.Fatalf("column %d: got %q/%v want %q", x, got, comb, want)
				}
			}
		})
	}
}
