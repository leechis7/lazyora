package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leechis7/tuivory/go"
	"github.com/rivo/uniseg"
	"lazyora/internal/oracle"
	"lazyora/internal/settings"
)

// An opt-in diagnostic audit uses the real constructor and application draw
// hooks. It intentionally reports unresolved layout failures without turning
// the ordinary test suite into an always-failing audit gate.
func TestRuntimeUIAudit(t *testing.T) {
	if os.Getenv("LAZYORA_UI_AUDIT") != "1" {
		t.Skip("set LAZYORA_UI_AUDIT=1 for the exhaustive runtime audit")
	}
	for _, key := range []string{"LAZYORA_HOST", "LAZYORA_USER", "LAZYORA_PASSWORD", "LAZYORA_NAME"} {
		t.Setenv(key, "")
	}
	scenarios := []struct {
		name string
		open func(*screen)
	}{
		{"main", func(s *screen) {}},
		{"connect", func(s *screen) { s.connectionForm() }},
		{"settings", func(s *screen) { s.settingsForm() }},
		{"about", func(s *screen) { s.showAbout() }},
		{"help", func(s *screen) { s.showHelp() }},
		{"guide", func(s *screen) { s.showGuide() }},
		{"release_history", func(s *screen) { s.showReleaseHistory() }},
		{"diagnostics", func(s *screen) { s.showDiagnostics() }},
		{"issue_report", func(s *screen) { s.showIssueReport() }},
		{"keymaps", func(s *screen) { s.showKeymapComparison() }},
		{"workspaces", func(s *screen) { s.showWorkspaces() }},
		{"search", func(s *screen) { s.showSearch(true) }},
		{"completion", func(s *screen) { s.editor.SetText("se", true); s.showCompletion(false) }},
		{"rename", func(s *screen) { s.showTabRename() }},
		{"file", func(s *screen) { s.showFileDialog("open", nil) }},
		{"export", func(s *screen) { s.showExport() }},
		{"output", func(s *screen) { s.showOutput() }},
		{"bind", func(s *screen) {
			s.withBinds("SELECT :id FROM dual", func(map[string]oracle.BindValue) { t.Error("audit executed bind callback") })
		}},
		{"sql_confirm", func(s *screen) {
			s.confirmExecution("DELETE FROM audit_table", "unfiltered_dml", func(string) { t.Error("audit submitted SQL") })
		}},
		{"exit", func(s *screen) { s.showExitConfirmation() }},
		{"source", func(s *screen) {
			s.object = oracle.Object{Owner: "AUDIT", Name: "P", Kind: "PROCEDURE"}
			s.showSource()
			s.acceptSource("PROCEDURE P AS BEGIN NULL; END;")
		}},
		{"source_actions", func(s *screen) {
			s.object = oracle.Object{Owner: "AUDIT", Name: "P", Kind: "PROCEDURE"}
			s.showSource()
			s.acceptSource("PROCEDURE P AS BEGIN NULL; END;")
			s.showSourceEdit()
		}},
		{"context", func(s *screen) { s.showContextMenu() }},
	}

	extras := []struct {
		name string
		open func(*screen)
	}{
		{"history", func(s *screen) { s.showHistory() }},
		{"describe", func(s *screen) { s.describe(true) }},
		{"object", func(s *screen) { s.openObject(oracle.Object{Owner: "AUDIT", Name: "SAMPLE_ROWS", Kind: "TABLE"}) }},
		{"arguments", func(s *screen) {
			s.object = oracle.Object{Owner: "AUDIT", Name: "P", Kind: "PROCEDURE"}
			s.showArguments()
		}},
		{"definition", func(s *screen) {
			s.openObject(oracle.Object{Owner: "AUDIT", Name: "SAMPLE_ROWS", Kind: "TABLE"})
			s.showDefinition("ddl")
		}},
		{"plan", func(s *screen) {
			s.showPlan("SELECT * FROM audit_table", []oracle.PlanStep{{ID: 0, Operation: "SELECT STATEMENT"}})
		}},
		{"script", func(s *screen) {
			s.scriptReport = oracle.ScriptReport{Total: 1, Steps: []oracle.ScriptStep{{Statement: oracle.Statement{Text: "SELECT 1 FROM dual", Line: 1}, Result: oracle.Result{Columns: []string{"N"}, Rows: [][]string{{"1"}}}}}}
			s.showScriptReport()
		}},
		{"copy", func(s *screen) { s.app.SetFocus(s.table); s.clipboardText = "audit\t한글"; s.copyPreview() }},
		{"cell", func(s *screen) { auditGridData(s); s.showCellDetail() }},
		{"grid_edit", func(s *screen) { auditGridData(s); s.showGridEdit() }},
		{"grid_inline", func(s *screen) { auditGridData(s); s.startInlineEdit(nil) }},
		{"result_filter", func(s *screen) { s.showResultFilter() }},
		{"pending", func(s *screen) { s.pendingChanges = true; s.showPendingWork() }},
		{"program_confirm", func(s *screen) { s.showProgramConfirm("Audit confirmation", func() {}) }},
		{"file_unsaved", func(s *screen) { s.guardDocument(func() {}) }},
		{"reconnect", func(s *screen) { s.pendingChanges = true; s.requestReconnect(oracle.Config{}) }},
		{"master", func(s *screen) { s.masterDialog("create", func() {}) }},
		{"vault", func(s *screen) { s.passwordManager() }},
		{"routines", func(s *screen) {
			s.connected = true
			s.showRoutine(oracle.Object{Owner: "AUDIT", Name: "P", Kind: "PROCEDURE"}, "")
			s.acceptRoutines([]oracle.Routine{{Object: oracle.Object{Owner: "AUDIT", Name: "P", Kind: "PROCEDURE"}}})
		}},
		{"routine_input", func(s *screen) {
			s.showRoutineInputs(oracle.Routine{Object: oracle.Object{Owner: "AUDIT", Name: "P", Kind: "PROCEDURE"}, Arguments: []oracle.RoutineArgument{{Argument: oracle.Argument{Name: "P_ID", DataType: "NUMBER", Direction: "IN", Position: 1}}}})
		}},
		{"routine_results", func(s *screen) {
			s.showRoutineResults(oracle.RoutineResult{Routine: "AUDIT.P", Scalars: oracle.Result{Columns: []string{"VALUE"}, Rows: [][]string{{"1"}}}})
		}},
		{"routine_grid", func(s *screen) {
			s.showRoutineResults(oracle.RoutineResult{Routine: "AUDIT.P", Scalars: oracle.Result{Columns: []string{"VALUE"}, Rows: [][]string{{"1"}}}})
			s.routineResultList.InputHandler()(tuivory.NewEventKey(tuivory.KeyEnter, 0, 0), func(p tuivory.Primitive) { s.app.SetFocus(p) })
		}},
	}
	for _, extra := range extras {
		scenarios = append(scenarios, extra)
	}
	for index := range menuGroups {
		i := index
		scenarios = append(scenarios, struct {
			name string
			open func(*screen)
		}{fmt.Sprintf("menu%d", i), func(s *screen) { s.showMenu(i) }})
	}
	for _, lang := range []string{"ko", "en"} {
		for _, theme := range []string{"dark", "light", "blackwhite"} {
			for _, colors := range []int{8, 16, 256, 1 << 24} {
				for _, scenario := range scenarios {
					t.Run(fmt.Sprintf("%s/%s/%d/%s", lang, theme, colors, scenario.name), func(t *testing.T) {
						raw := colorTestTerminal(t, 120, 40)
						terminal := auditTerminal{raw, colors}
						app := tuivory.NewApplication().SetScreen(terminal)
						prefs := settings.Config{Language: lang, Theme: theme, Mouse: true, DisplayMode: "unicode"}
						s := newWorkspaceScreen(app, terminal, prefs, filepath.Join(t.TempDir(), "settings.json"), true, nil)
						spaces := &workspaceSet{app: app, active: s}
						spaces.adopt(s)
						t.Cleanup(spaces.shutdown)
						s.editor.SetText("SELECT NVL(:id, '한글') AS value, 12 FROM dual; -- comment", false)
						app.ForceDraw()
						before := s.editor.GetText()
						defer func() {
							if r := recover(); r != nil {
								t.Errorf("runtime panic: %v", r)
							}
						}()
						func() { s.workerClosed = true; defer func() { s.workerClosed = false }(); scenario.open(s) }()
						expected := scenario.name
						switch expected {
						case "source_actions":
							expected = "editor_actions"
						case "keymaps":
							expected = "keymap_compare"
						case "rename":
							expected = "tab_rename"
						case "guide", "release_history", "diagnostics", "issue_report":
							expected = "reference"
						case "context":
							expected = "menu"
						}
						if strings.HasPrefix(expected, "menu") {
							expected = "menu"
						}
						if page, _ := s.pages.GetFrontPage(); page != expected {
							t.Errorf("did not open requested page: wanted %s, got %s", expected, page)
						}
						for _, size := range [][2]int{{120, 40}, {80, 24}, {40, 16}, {120, 40}} {
							raw.SetSize(size[0], size[1])
							app.ForceDraw()
							focus := app.GetFocus()
							if focus == nil {
								t.Errorf("%dx%d: missing focus", size[0], size[1])
								continue
							}
							x, y, w, h := focus.GetRect()
							if x < 0 || y < 0 || w <= 0 || h <= 0 || x+w > size[0] || y+h > size[1] {
								t.Errorf("%dx%d: focus %T outside viewport (%d,%d %dx%d)", size[0], size[1], focus, x, y, w, h)
							}
							if s.editor.GetText() != before && scenario.name != "completion" {
								t.Error("drawing changed original SQL")
							}
							if size == ([2]int{80, 24}) && lang == "ko" {
								switch scenario.name {
								case "main", "connect", "settings", "help", "about", "keymaps", "sql_confirm", "menu2", "source_actions":
									auditDump(t, raw, fmt.Sprintf("%s-%d-%s", theme, colors, scenario.name))
								}
							}
						}
					})
				}
			}
		}
	}
}

func auditDump(t *testing.T, terminal tuivory.SimulationScreen, name string) {
	folder := os.Getenv("LAZYORA_UI_AUDIT_DIR")
	if folder == "" {
		return
	}
	if err := os.MkdirAll(folder, 0755); err != nil {
		t.Fatal(err)
	}
	width, height := terminal.Size()
	rows := make([]string, height)
	type cell struct {
		X, Y, W         int
		Text, FG, BG    string
		Bold, Underline bool
	}
	var cells []cell
	color := func(c tuivory.Color, fallback string) string {
		r, g, b := c.RGB()
		if r < 0 {
			return fallback
		}
		return fmt.Sprintf("#%02x%02x%02x", r, g, b)
	}
	for y := 0; y < height; y++ {
		var row strings.Builder
		for x := 0; x < width; {
			r, comb, st, w := terminal.GetContent(x, y)
			if r == 0 {
				r = ' '
			}
			text := string(append([]rune{r}, comb...))
			w = max(1, w)
			fg, bg, attrs := st.Decompose()
			if attrs&tuivory.AttrReverse != 0 {
				fg, bg = bg, fg
			}
			cells = append(cells, cell{x, y, max(w, uniseg.StringWidth(text)), text, color(fg, "#e5edf5"), color(bg, "#242d3a"), attrs&tuivory.AttrBold != 0, attrs&tuivory.AttrUnderline != 0})
			row.WriteString(text)
			x += w
		}
		rows[y] = row.String()
	}
	data, err := json.Marshal(map[string]any{"width": width, "height": height, "cells": cells})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(folder, name+".json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(folder, name+".txt"), []byte(strings.Join(rows, "\n")), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeUIFocusAudit(t *testing.T) {
	if os.Getenv("LAZYORA_UI_AUDIT") != "1" {
		t.Skip("set LAZYORA_UI_AUDIT=1 for the exhaustive runtime audit")
	}
	for _, key := range []string{"LAZYORA_HOST", "LAZYORA_USER", "LAZYORA_PASSWORD", "LAZYORA_NAME"} {
		t.Setenv(key, "")
	}
	scenarios := []struct {
		name string
		open func(*screen)
	}{
		{"connect", func(s *screen) { s.connectionForm() }}, {"settings", func(s *screen) { s.settingsForm() }},
		{"about", func(s *screen) { s.showAbout() }}, {"help", func(s *screen) { s.showHelp() }},
		{"guide", func(s *screen) { s.showGuide() }}, {"keymaps", func(s *screen) { s.showKeymapComparison() }},
		{"workspaces", func(s *screen) { s.showWorkspaces() }}, {"search", func(s *screen) { s.showSearch(true) }},
		{"completion", func(s *screen) { s.editor.SetText("se", true); s.showCompletion(false) }},
		{"rename", func(s *screen) { s.showTabRename() }}, {"file", func(s *screen) { s.showFileDialog("open", nil) }},
		{"export", func(s *screen) { s.showExport() }}, {"output", func(s *screen) { s.showOutput() }},
		{"bind", func(s *screen) {
			s.withBinds("SELECT :id FROM dual", func(map[string]oracle.BindValue) { t.Error("unexpected execution") })
		}},
		{"sql_confirm", func(s *screen) {
			s.confirmExecution("DELETE FROM audit_table", "unfiltered_dml", func(string) { t.Error("unexpected execution") })
		}},
		{"exit", func(s *screen) { s.showExitConfirmation() }},
	}
	for _, lang := range []string{"ko", "en"} {
		for _, theme := range []string{"dark", "light", "blackwhite"} {
			for _, colors := range []int{8, 1 << 24} {
				for _, scenario := range scenarios {
					t.Run(fmt.Sprintf("%s/%s/%d/%s", lang, theme, colors, scenario.name), func(t *testing.T) {
						raw := colorTestTerminal(t, 120, 40)
						terminal := auditTerminal{raw, colors}
						app := tuivory.NewApplication().SetScreen(terminal)
						s := newWorkspaceScreen(app, terminal, settings.Config{Language: lang, Theme: theme, Mouse: true, DisplayMode: "unicode"}, filepath.Join(t.TempDir(), "settings.json"), true, nil)
						spaces := &workspaceSet{app: app, active: s}
						spaces.adopt(s)
						t.Cleanup(spaces.shutdown)
						scenario.open(s)
						for _, size := range [][2]int{{80, 24}, {40, 16}} {
							raw.SetSize(size[0], size[1])
							app.ForceDraw()
							seen := map[tuivory.Primitive]bool{}
							for step := 0; step < 36; step++ {
								focus := app.GetFocus()
								if focus == nil {
									t.Error("missing focus")
									break
								}
								if !seen[focus] {
									seen[focus] = true
									x, y, w, h := focus.GetRect()
									if x < 0 || y < 0 || w <= 0 || h <= 0 || x+w > size[0] || y+h > size[1] {
										t.Errorf("%dx%d tab=%d: %T outside viewport (%d,%d %dx%d)", size[0], size[1], step, focus, x, y, w, h)
									}
								}
								key := tuivory.KeyTab
								if step >= 18 {
									key = tuivory.KeyBacktab
								}
								event := tuivory.NewEventKey(key, 0, 0)
								if capture := app.GetInputCapture(); capture != nil {
									event = capture(event)
								}
								if event != nil {
									if handler := app.GetFocus().InputHandler(); handler != nil {
										handler(event, func(p tuivory.Primitive) { app.SetFocus(p) })
									}
								}
								app.ForceDraw()
							}
						}
					})
				}
			}
		}
	}
}

func auditGridData(s *screen) {
	s.connected = true
	s.render(oracle.Result{EditID: 5, EditObject: oracle.Object{Owner: "AUDIT", Name: "ITEMS", Kind: "TABLE"}, EditColumns: []bool{true, true}, EditColumnNames: []string{"ID", "TITLE"}, EditRowIDs: []string{"rid1"}, Columns: []string{"ID", "TITLE"}, Rows: [][]string{{"1", "한글 value"}}, Values: [][]oracle.Cell{{{Text: "1", Type: "NUMBER"}, {Text: "한글 value", Type: "VARCHAR2"}}}})
	s.table.Select(1, 1)
	s.app.SetFocus(s.table)
}

func TestRuntimeUIChoiceAudit(t *testing.T) {
	if os.Getenv("LAZYORA_UI_AUDIT") != "1" {
		t.Skip("set LAZYORA_UI_AUDIT=1 for the exhaustive runtime audit")
	}
	for _, key := range []string{"LAZYORA_HOST", "LAZYORA_USER", "LAZYORA_PASSWORD", "LAZYORA_NAME"} {
		t.Setenv(key, "")
	}
	for _, lang := range []string{"ko", "en"} {
		for _, theme := range []string{"dark", "light", "blackwhite"} {
			for _, colors := range []int{8, 16, 256, 1 << 24} {
				for _, page := range []string{"connect", "settings", "browser"} {
					t.Run(fmt.Sprintf("%s/%s/%d/%s", lang, theme, colors, page), func(t *testing.T) {
						raw := colorTestTerminal(t, 80, 24)
						terminal := auditTerminal{raw, colors}
						app := tuivory.NewApplication().SetScreen(terminal)
						s := newWorkspaceScreen(app, terminal, settings.Config{Language: lang, Theme: theme, Mouse: true, DisplayMode: "unicode"}, filepath.Join(t.TempDir(), "settings.json"), true, nil)
						spaces := &workspaceSet{app: app, active: s}
						spaces.adopt(s)
						t.Cleanup(spaces.shutdown)
						var forms []*tuivory.WidgetForm
						switch page {
						case "connect":
							s.connectionForm()
							forms = []*tuivory.WidgetForm{s.connectForm, s.connectProfiles}
						case "settings":
							s.settingsForm()
							forms = []*tuivory.WidgetForm{s.prefsForm}
						case "browser":
							forms = []*tuivory.WidgetForm{s.browserForm}
						}
						original := s.editor.GetText()
						for _, size := range [][2]int{{80, 24}, {40, 16}} {
							raw.SetSize(size[0], size[1])
							app.ForceDraw()
							for _, form := range forms {
								for index := 0; index < form.GetFormItemCount(); index++ {
									choice, ok := form.GetFormItem(index).(*choiceField)
									if !ok {
										continue
									}
									focusForm(s, form, index)
									app.ForceDraw()
									choice.InputHandler()(tuivory.NewEventKey(tuivory.KeyEnter, 0, 0), func(p tuivory.Primitive) { app.SetFocus(p) })
									app.ForceDraw()
									if !choice.IsOpen() {
										t.Errorf("%dx%d item %d did not open", size[0], size[1], index)
									}
									if size == ([2]int{80, 24}) && lang == "ko" && page == "settings" && index == 3 {
										auditDump(t, raw, fmt.Sprintf("%s-%d-theme-open", theme, colors))
									}
									choice.InputHandler()(tuivory.NewEventKey(tuivory.KeyEscape, 0, 0), func(p tuivory.Primitive) { app.SetFocus(p) })
									app.ForceDraw()
									if choice.IsOpen() {
										t.Error("Escape failed to close dropdown")
									}
									if original != s.editor.GetText() {
										t.Error("dropdown changed SQL")
									}
								}
							}
						}
					})
				}
			}
		}
	}
}
