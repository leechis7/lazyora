package ui

// Opt-in inspection harness: it draws production widgets but executes no DB
// jobs, changes no production palette, and is skipped by normal test runs.
import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"lazyora/internal/settings"
)

type auditTerminal struct {
	tcell.SimulationScreen
	count int
}

func (a auditTerminal) Colors() int { return a.count }

type auditStyle struct {
	FG, BG                                                           int64
	Bold, Dim, Reverse, Underline                                    bool
	FGHex, BGHex, BrightFGHex, BrightBGHex, SourceFGHex, SourceBGHex string
}
type auditRun struct {
	X     int
	Text  string
	Style int
}
type auditShot struct {
	Scene, Theme, Page, Variant, Language string
	Colors, W, H                          int
	Styles                                []auditStyle
	Rows                                  [][]auditRun
}
type auditRole struct {
	Theme, Role string
	Colors      int
	Style       auditStyle
}
type auditOutput struct {
	Shots []auditShot
	Roles []auditRole
}

var auditPalettes = map[int][]tcell.Color{}
var auditMapped = map[struct {
	Color tcell.Color
	Count int
}]tcell.Color{}

func auditMapColor(c tcell.Color, count int) tcell.Color {
	if count > 256 && c.IsRGB() {
		return c
	}
	n := min(count, 256)
	if !c.IsRGB() && int(c&^tcell.ColorValid) < n {
		return c
	}
	key := struct {
		Color tcell.Color
		Count int
	}{c, count}
	if mapped, ok := auditMapped[key]; ok {
		return mapped
	}
	palette := auditPalettes[n]
	if palette == nil {
		for i := 0; i < n; i++ {
			palette = append(palette, tcell.PaletteColor(i))
		}
		auditPalettes[n] = palette
	}
	mapped := tcell.FindColor(c, palette)
	auditMapped[key] = mapped
	return mapped
}
func auditRGB(c tcell.Color) string { r, g, b := c.RGB(); return fmt.Sprintf("#%02x%02x%02x", r, g, b) }
func auditStyleOf(style tcell.Style, count int) auditStyle {
	fg, bg, a := style.Decompose()
	if !fg.Valid() {
		fg = tcell.ColorSilver
	}
	if !bg.Valid() {
		bg = tcell.ColorBlack
	}
	mappedFG, mappedBG := auditMapColor(fg, count), auditMapColor(bg, count)
	brightFG := mappedFG
	if a&tcell.AttrBold != 0 && !mappedFG.IsRGB() && int(mappedFG&^tcell.ColorValid) < 8 {
		brightFG = tcell.PaletteColor(int(mappedFG&^tcell.ColorValid) + 8)
	}
	fgHex, bgHex := auditRGB(mappedFG), auditRGB(mappedBG)
	brightFGHex, brightBGHex := auditRGB(brightFG), bgHex
	if a&tcell.AttrReverse != 0 {
		fgHex, bgHex = bgHex, fgHex
		brightFGHex, brightBGHex = brightBGHex, brightFGHex
	}
	return auditStyle{FG: int64(fg), BG: int64(bg), Bold: a&tcell.AttrBold != 0, Dim: a&tcell.AttrDim != 0, Reverse: a&tcell.AttrReverse != 0, Underline: a&tcell.AttrUnderline != 0, FGHex: fgHex, BGHex: bgHex, BrightFGHex: brightFGHex, BrightBGHex: brightBGHex, SourceFGHex: auditRGB(fg), SourceBGHex: auditRGB(bg)}
}

func TestThemeAudit(t *testing.T) {
	output := os.Getenv("LAZYORA_THEME_AUDIT_DIR")
	if output == "" {
		t.Skip("set LAZYORA_THEME_AUDIT_DIR to collect the review matrix")
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	t.Chdir(fixture)
	if err := os.WriteFile(filepath.Join(fixture, "example.sql"), []byte("SELECT 1 FROM dual;"), 0600); err != nil {
		t.Fatal(err)
	}
	scenes := []string{"main", "production", "results", "range", "edited", "schema", "resize", "busy", "error", "menu", "submenu", "context", "connect", "connect_input", "connect_choice", "settings", "theme_choice", "help", "help_body", "guide", "release_history", "diagnostics", "issue_report", "about", "comparison", "comparison_detail", "workspaces", "master_create", "master_unlock", "vault", "bind", "export", "search", "history", "completion", "object", "definition_ddl", "definition_constraints", "definition_relations", "source", "source_errors", "arguments", "editor_actions", "routines", "routine_input", "routine_results", "routine_grid", "copy", "describe", "result_filter", "cell", "grid_edit", "grid_inline", "grid_inline_null", "confirm", "sql_confirm", "exit_clean", "exit_pending", "exit_uncertain", "reconnect", "pending", "plan", "output", "script", "file", "file_unsaved", "tab_rename"}
	var result auditOutput
	for _, count := range []int{8, 16, 256, 1 << 24} {
		for _, theme := range []string{"dark", "light", "blackwhite"} {
			for _, scene := range scenes {
				for _, language := range []string{"ko", "en"} {
					t.Run(fmt.Sprintf("%d/%s/%s/%s", count, theme, scene, language), func(t *testing.T) {
						base := tcell.NewSimulationScreen("UTF-8")
						terminal := auditTerminal{base, count}
						app := tview.NewApplication().SetScreen(terminal)
						prefs := settings.Config{Language: language, Mouse: true, Theme: theme}
						s := newWorkspaceScreen(app, terminal, prefs, filepath.Join(t.TempDir(), "settings.json"), true, nil)
						s.stopWorker()
						<-s.done
						s.jobs = make(chan job, 32)
						s.workerClosed = false // capture requests; never execute them
						spaces := &workspaceSet{app: app, active: s}
						spaces.adopt(s)
						defer spaces.shutdown()
						defer terminal.Fini()
						s.connectionIdentity = settings.Connection{Alias: "개발 DB", Host: "oracle.example.invalid", User: "LAZYORA", Name: "FREEPDB1", Mode: "service", Environment: "dev"}
						s.editor.SetText("SELECT id, title, NVL(amount, 0) AS amount\nFROM sample_rows\nWHERE id > :id AND title <> 'sample';\n-- 한글 주석과 Oracle SQL 문법 강조", false)
						auditPrepare(t, s, scene, fixture)
						// Fixture construction may request read-only metadata; no worker exists.
						for len(s.jobs) > 0 {
							j := <-s.jobs
							if j.action == "execute" || j.action == "edit_apply" || j.action == "commit" || j.action == "rollback" {
								t.Fatalf("audit submitted a mutating job: %s", j.action)
							}
						}
						if scene != "busy" {
							if s.cancel != nil {
								s.cancel()
								s.cancel = nil
							}
							s.busy = false
						}
						terminal.SetSize(128, 40)
						for _, variant := range []string{"fresh", "roundtrip"} {
							if variant == "roundtrip" {
								for _, next := range []string{"dark", "light", "blackwhite", theme} {
									if next != s.prefs.Theme {
										previous := s.prefs.Theme
										s.prefs.Theme = next
										s.applyTheme(previous)
									}
								}
							}
							s.syncChrome()
							s.refreshFooter()
							s.refreshBorders()
							s.rootLayout.SetRect(0, 0, 128, 40)
							s.rootLayout.Draw(terminal)
							terminal.Show()
							page, _ := s.pages.GetFrontPage()
							if page == "main" && !strings.Contains(" main production results range edited schema resize busy error ", " "+scene+" ") {
								t.Fatalf("scene %s failed to open its page", scene)
							}
							shot := auditShot{Scene: scene, Theme: theme, Colors: count, Page: page, Variant: variant, Language: language, W: 128, H: 40}
							styles := map[auditStyle]int{}
							for y := 0; y < 40; y++ {
								var row []auditRun
								for x := 0; x < 128; {
									r, comb, style, width := terminal.GetContent(x, y)
									width = max(1, width)
									if r == 0 {
										r = ' '
									}
									key := auditStyleOf(style, count)
									index, ok := styles[key]
									if !ok {
										index = len(shot.Styles)
										styles[key] = index
										shot.Styles = append(shot.Styles, key)
									}
									text := string(append([]rune{r}, comb...))
									if len(row) > 0 && row[len(row)-1].Style == index {
										row[len(row)-1].Text += text
									} else {
										row = append(row, auditRun{X: x, Text: text, Style: index})
									}
									x += width
								}
								shot.Rows = append(shot.Rows, row)
							}
							result.Shots = append(result.Shots, shot)
						}
						if scene == "main" && language == "ko" {
							roles := map[string]tcell.Style{"body": tcell.StyleDefault.Foreground(s.colors().text).Background(s.colors().surface), "hint": tcell.StyleDefault.Foreground(s.colors().muted).Background(s.colors().background), "field": s.fieldStyle(), "placeholder": tcell.StyleDefault.Foreground(s.colors().placeholder).Background(s.colors().field), "selection": s.focusStyle(), "button": s.buttonStyle(), "button_focus": s.buttonFocusStyle(), "editor": s.editorStyle(), "grid_base": s.gridBaseStyle(3, 1), "grid_null": s.gridBaseStyle(2, 3), "grid_header": s.gridHeaderStyle(false), "grid_active": s.gridActiveStyle(true), "edited_cell": s.editedCellStyle()}
							for kind := sqlKeyword; kind <= sqlBind; kind++ {
								roles[fmt.Sprintf("sql_token_%d", kind)] = s.profileSQLStyle(kind, s.editorStyle())
							}
							for name, style := range roles {
								result.Roles = append(result.Roles, auditRole{Theme: theme, Colors: count, Role: name, Style: auditStyleOf(style, count)})
							}
						}
					})
				}
			}
		}
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(output, "raw.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("captured %d scenes across 12 capability/theme combinations (%d frames)", len(scenes), len(result.Shots))
}

func auditResult() oracle.Result {
	return oracle.Result{Columns: []string{"ID", "이름", "금액", "상태"}, Rows: [][]string{{"1", "첫 번째 행", "123.45", "정상"}, {"2", "수정한 값", "0", "NULL"}, {"3", "긴 텍스트 · 원문은 셀 상세에서 확인합니다", "-12.50", "대기"}}, Values: [][]oracle.Cell{{{Text: "1", Type: "NUMBER"}, {Text: "첫 번째 행", Type: "VARCHAR2"}, {Text: "123.45", Type: "NUMBER"}, {Text: "정상", Type: "VARCHAR2"}}, {{Text: "2", Type: "NUMBER"}, {Text: "수정한 값", Type: "VARCHAR2"}, {Text: "0", Type: "NUMBER"}, {Null: true, Type: "VARCHAR2"}}, {{Text: "3", Type: "NUMBER"}, {Text: "긴 텍스트 · 원문은 셀 상세에서 확인합니다", Type: "VARCHAR2"}, {Text: "-12.50", Type: "NUMBER"}, {Text: "대기", Type: "VARCHAR2"}}}}
}
func auditEdit(s *screen) {
	r := auditResult()
	r.EditID = 1
	r.EditObject = oracle.Object{Owner: "LAZYORA", Name: "SAMPLE_ROWS", Kind: "TABLE"}
	r.EditRowIDs = []string{"r1", "r2", "r3"}
	r.EditColumnNames = []string{"ID", "TITLE", "AMOUNT", "STATE"}
	s.render(r)
	s.connected = true
	s.app.SetFocus(s.table)
	s.table.Select(2, 1)
}
func auditObject(s *screen, program bool) {
	if program {
		s.browserTypes.SetCurrentOption(3)
	}
	s.activateBrowser(s.browser.GetCurrentNode())
}
func auditPrepare(t *testing.T, s *screen, scene, fixture string) {
	delegate := func(p tview.Primitive) { s.app.SetFocus(p) }
	r := auditResult()
	s.render(r)
	switch scene {
	case "main":
	case "production":
		s.connected = true
		s.connectionIdentity.Environment = "prod"
		s.refreshConnectionTitles()
	case "results":
		s.app.SetFocus(s.table)
	case "range":
		s.app.SetFocus(s.table)
		s.resultSelection = resultSelection{active: true, anchorRow: 1, anchorCol: 0, endRow: 2, endCol: 2}
		s.highlightResults()
	case "edited":
		auditEdit(s)
		s.markEditedCell(s.editedCellIdentity(s.resultBrowser.source, 1, 1))
	case "schema":
		s.app.SetFocus(s.browser)
	case "resize":
		s.resizePanelsCommand()
	case "busy":
		s.busy = true
		s.cancel = func() {}
		s.setStatus("running", "execute")
	case "error":
		s.setStatus("error", errors.New("ORA-00942: 검토용 오류 메시지"))
	case "menu":
		s.showMenu(menuEdit)
	case "submenu":
		s.showMenu(menuView)
		s.menuRowAction(0)
	case "context":
		s.app.SetFocus(s.table)
		s.showContextMenu()
	case "connect", "connect_input", "connect_choice":
		s.connectionForm()
		if scene == "connect_input" {
			focusForm(s, s.connectForm, connectionAlias)
		}
		if scene == "connect_choice" {
			focusForm(s, s.connectForm, connectionMode)
			s.connectForm.GetFormItem(connectionMode).(*choiceField).InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), delegate)
		}
	case "settings", "theme_choice":
		s.settingsForm()
		if scene == "theme_choice" {
			focusForm(s, s.prefsForm, 3)
			s.prefsForm.GetFormItem(3).(*choiceField).InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), delegate)
		}
	case "help", "help_body":
		s.showHelp()
		if scene == "help_body" {
			s.app.SetFocus(s.helpText)
		}
	case "guide":
		s.showGuide()
	case "release_history":
		s.showReleaseHistory()
	case "diagnostics":
		s.showDiagnostics()
	case "issue_report":
		s.showIssueReport()
	case "about":
		s.showAbout()
	case "comparison", "comparison_detail":
		s.showKeymapComparison()
		if scene == "comparison_detail" {
			s.app.SetFocus(s.comparison.detail)
		}
	case "workspaces":
		s.showWorkspaces()
	case "master_create":
		s.masterDialog("create", func() {})
	case "master_unlock":
		s.masterDialog("unlock", func() {})
	case "vault":
		s.passwordManager()
	case "bind":
		s.withBinds("SELECT :name,:amount FROM dual", func(map[string]oracle.BindValue) {})
	case "export":
		s.showExport()
	case "search":
		s.showSearch(true)
	case "history":
		s.sqlHistory = []string{"SELECT 1 FROM dual", "SELECT title FROM sample_rows"}
		s.showHistory()
	case "completion":
		s.editor.SetText("se", false)
		s.showCompletion(false)
	case "object":
		auditObject(s, false)
	case "definition_ddl", "definition_constraints", "definition_relations":
		auditObject(s, false)
		s.showDefinition(strings.TrimPrefix(scene, "definition_"))
	case "source", "source_errors", "arguments", "editor_actions":
		auditObject(s, true)
		if scene == "arguments" {
			s.showArguments()
		} else {
			s.showSource()
			if scene == "source_errors" {
				s.renderDiagnostics([]oracle.Diagnostic{{Line: 3, Position: 5, Attribute: "ERROR", Text: "PLS-00201: 검토용 오류"}})
			}
			if scene == "editor_actions" {
				s.showSourceEdit()
			}
		}
	case "routines", "routine_input":
		obj := oracle.Object{Owner: "LAZYORA", Name: "SAMPLE_PROCEDURE", Kind: "PROCEDURE"}
		s.connected = true
		s.showRoutine(obj, "")
		s.acceptRoutines([]oracle.Routine{{Object: obj, Member: "SAMPLE_PROCEDURE", Arguments: []oracle.RoutineArgument{{Argument: oracle.Argument{Name: "P_VALUE", DataType: "VARCHAR2", Direction: "IN", Position: 1}}}}})
		if scene == "routine_input" {
			s.showRoutineInputs(oracle.Routine{Object: obj, Member: "SAMPLE_PROCEDURE", Arguments: []oracle.RoutineArgument{{Argument: oracle.Argument{Name: "P_VALUE", DataType: "VARCHAR2", Direction: "IN", Position: 1}}}})
		}
	case "routine_results", "routine_grid":
		s.showRoutineResults(oracle.RoutineResult{Routine: "SAMPLE_PROCEDURE", Scalars: r})
		if scene == "routine_grid" {
			s.routineResultList.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), delegate)
		}
	case "copy":
		s.clipboardText = "ID\tTITLE\n1\t샘플 데이터"
		s.copyPreview()
	case "describe":
		s.describe(true)
	case "result_filter":
		s.showResultFilter()
	case "cell":
		s.table.Select(1, 1)
		s.showCellDetail()
	case "grid_edit":
		auditEdit(s)
		s.showGridEdit()
	case "grid_inline", "grid_inline_null":
		auditEdit(s)
		if scene == "grid_inline_null" {
			s.table.Select(2, 3)
		}
		s.startInlineEdit(nil)
	case "confirm":
		s.showProgramConfirm("변경 전·후를 확인하세요.\n새 값: 샘플 데이터", func() {})
	case "sql_confirm":
		s.confirmActionPreview("DELETE FROM sample_rows", "주의: WHERE 없는 변경의 검토 예시입니다.", "SQL 확인", s.tr("execute"), func() {})
	case "exit_clean", "exit_pending", "exit_uncertain":
		s.pendingChanges = scene != "exit_clean"
		s.transactionUncertain = scene == "exit_uncertain"
		s.showExitConfirmation()
	case "reconnect":
		s.pendingChanges = true
		s.requestReconnect(oracle.Config{Host: "oracle.example.invalid", Port: 1521, User: "LAZYORA", Mode: "service", Name: "FREEPDB1"})
	case "pending":
		s.pendingWork = oracle.PendingWork{Statements: []oracle.StatementRecord{{SQL: "UPDATE sample_rows SET title=:title WHERE id=:id", At: time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC), Affected: 1}}}
		s.showPendingWork()
	case "plan":
		s.showPlan("SELECT * FROM sample_rows", []oracle.PlanStep{{ID: 0, Operation: "SELECT STATEMENT"}, {ID: 1, Parent: 0, HasParent: true, Operation: "TABLE ACCESS", Options: "FULL", Object: "SAMPLE_ROWS"}})
	case "output":
		s.outputLines = []string{"DBMS_OUTPUT: 샘플 출력", "완료: 3행"}
		s.showOutput()
	case "script":
		s.scriptReport = oracle.ScriptReport{Total: 2, Steps: []oracle.ScriptStep{{Statement: oracle.Statement{Text: "SELECT 1 FROM dual"}, Result: r}, {Statement: oracle.Statement{Text: "SELECT missing FROM dual"}, Error: "ORA-00904: 검토용 오류"}}}
		s.showScriptReport()
	case "file":
		s.document.Path = filepath.Join(fixture, "example.sql")
		s.showFileDialog("open", nil)
	case "file_unsaved":
		s.guardDocument(func() {})
	case "tab_rename":
		s.showTabRename()
	default:
		t.Fatalf("unknown scene %s", scene)
	}
}
