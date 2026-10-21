package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/appinfo"
)

type helpSection struct {
	id       string
	title    [2]string
	commands []string
	notes    [2]string
}

var helpSections = []helpSection{
	{"sql", [2]string{"SQL editor", "SQL 편집"}, []string{"file_new", "file_open", "file_save", "file_save_as", "file_close", "tab_next", "tab_previous", "execute", "script", "script_report", "plan", "output_view", "output_toggle", "complete", "complete_refresh", "format_code", "duplicate_line", "copy_line", "cut_line", "delete_line", "line_up", "line_down", "transpose_line", "routine_execute", "routine_results", "undo", "redo", "cut", "copy", "paste", "select_all", "find", "find_next", "find_previous", "replace", "clear_sql", "history"}, [2]string{
		"UTF-8 SQL files: Open/Save/Save As in File; the title shows the file name and * for unsaved edits.\nSQL tabs (Alt+G): New/Close/Next/Previous tab. Ctrl+N opens a new tab. Ctrl+Tab / Alt+PgDn next, Ctrl+Shift+Tab / Alt+PgUp previous. Tabs share one DB transaction; switching/closing is locked while running.\nClose/Open/Exit ask Save, Discard or Cancel for unsaved SQL. File saving does not commit the database.\nIf Ctrl+Shift+S is indistinguishable from Ctrl+S, use File > Save SQL as.\nIN binds prompt for Text/NUMBER/DATE/TIMESTAMP/NULL; up to 32 names. Values stay in memory for the tab's last SQL; reconnect/clear removes them. General SQL binds are IN-only. Object details / Arguments > Execute (Alt+E) supports routine OUT, IN OUT and REF CURSOR.\nDBMS_OUTPUT: Alt+P opens output. Enable (Alt+E) before SQL; F9/F5 capture after execution. Ctrl+C copies, Alt+D clears the retained display. Session-specific, no commit/rollback, bounded output and collection warning shown separately.\nEnter inserts a newline. Alt+Enter / Shift+F9 also execute.\nExecute selected text; without a selection, execute the statement at the cursor. SQL uses semicolons; separate PL/SQL units with a standalone /.\nFind is literal and case sensitive; next/previous wraps.\nHistory: this run only, up to 100 entries in memory. Loading does not execute SQL.\nDDL and UPDATE/DELETE without an outer WHERE ask before execution. The default is Cancel; Esc also cancels.",
		"파일 메뉴에서 UTF-8 SQL 열기/저장/다른 이름 저장. 제목에 파일명과 미저장 표시(*)가 나타납니다.\nSQL 탭(Alt+G) 메뉴에서 새 탭/닫기/다음/이전 선택. Ctrl+N 새 탭, Ctrl+Tab / Alt+PgDn 다음, Ctrl+Shift+Tab / Alt+PgUp 이전 탭. DB 트랜잭션은 공유하며 실행 중 탭 전환·닫기가 잠깁니다.\n닫기/열기/종료 전 미저장 SQL은 저장·버리기·취소를 확인합니다. 파일 저장은 DB 커밋과 별개입니다.\nCtrl+Shift+S를 구분하지 못하는 터미널은 파일 > SQL 다른 이름으로 저장을 사용하세요.\nIN 바인드는 문자열/NUMBER/DATE/TIMESTAMP/NULL을 입력합니다(최대 32개). 값은 탭 최근 SQL의 메모리에만 유지하며 재접속/지우기로 제거합니다. 일반 SQL 바인드는 IN 전용입니다. 객체 상세 / 인자 > 실행(Alt+E)에서 루틴 OUT·IN OUT·REF CURSOR를 지원합니다.\nDBMS_OUTPUT: Alt+P로 출력 창을 열고 실행 전에 Alt+E로 켭니다. F9/F5 실행 후 수집하며 Ctrl+C 복사·Alt+D 표시 지우기를 제공합니다. 접속별로 적용하고 커밋·롤백하지 않습니다. 출력 제한과 수집 경고는 별도로 안내합니다.\nEnter는 줄바꿈입니다. Alt+Enter / Shift+F9로도 실행합니다.\n선택한 SQL을 실행하며, 선택이 없으면 커서 위치 문장을 실행합니다. SQL은 세미콜론, PL/SQL 단위는 별도 줄 /로 구분합니다.\n찾기는 대소문자를 구분하는 일반 문자열 검색이며 끝에서 순환합니다.\n이력은 이번 실행의 SQL 최대 100개를 메모리에만 보관하며, 불러와도 자동 실행하지 않습니다.\nDDL과 최상위 WHERE 없는 UPDATE/DELETE는 실행 전 확인합니다. 기본 선택과 Esc는 취소입니다."}},
	{"results", [2]string{"Results and copying", "조회 결과 · 복사"}, []string{"copy", "copy_row", "export", "grid_widen", "grid_narrow", "grid_auto", "fetch_more", "result_sort_asc", "result_sort_desc", "result_filter", "result_view_reset", "cell_detail"}, [2]string{
		"Export (Alt+W / Ctrl+Shift+E): CSV, TSV, text-cell XLSX, INSERT. Choose loaded all/range/row or Full DB result. Full export reviews and requeries the original SQL/binds, streams rows with progress/Esc cancel, and has no app row/size/time cap. Data may change; no auto-commit/rollback. XLSX splits sheets. Reconnect/clear removes replay information. INSERT needs reviewed table/columns; no DB execution.\nClick a cell. Drag or Shift+arrows selects a rectangle; arrows / PgUp / PgDn scroll.\nPaste copied text uses the app clipboard populated by Copy/Cut; use the terminal paste shortcut for OS clipboard text.\nCopy format: TSV, with quoted tabs/newlines inside values; NULL stays NULL.\nIf clipboard transfer is blocked: Edit > Copy preview, then select and copy using your terminal.",
		"Alt+W / Ctrl+Shift+E 내보내기: CSV·TSV·텍스트 셀 XLSX·INSERT. 가져온 결과 전체/범위/행 또는 DB 전체 결과를 선택합니다. 전체 저장은 원 SQL·바인드를 확인 후 재조회하며 진행 행 수·Esc 취소와 앱의 행 수·크기·시간 제한 없는 순차 저장을 제공합니다. 데이터가 달라질 수 있고 자동 커밋·롤백하지 않습니다. Excel은 시트를 분할하며 재접속·바인드 지우기는 재조회 정보를 제거합니다. INSERT는 대상 테이블·컬럼을 확인하며 DB에 실행하지 않습니다.\n셀을 클릭하거나 드래그·Shift+방향키로 범위를 선택합니다. 방향키·PgUp·PgDn으로 이동합니다.\n편집 > 복사한 내용 붙여넣기는 앱에서 복사·잘라낸 내용을 사용합니다. OS 클립보드는 터미널의 붙여넣기를 사용하세요.\n복사 형식은 TSV이며, 값 안의 탭·줄바꿈은 인용 처리하고 NULL은 그대로 복사합니다.\n클립보드 전송이 막히면 편집 > 복사 내용 보기에서 터미널의 선택·복사를 사용하세요."}},
	{"navigation", [2]string{"Windows, menus and buttons", "화면 · 메뉴 · 버튼"}, []string{"fullscreen_sql", "fullscreen_results", "focus_toggle", "browser", "context_menu", "help"}, [2]string{
		"Tab / Shift+Tab cycles visible panels. Repeat the fullscreen key to restore split.\nView > Maximize retains menus/status/footer; Alt+B / View > Schemas shows/hides the browser.\nAlt+M opens menus; Alt+F/G/E/S/X/V/T/H opens File/SQL tabs/Edit/Session/Execute/View/Tools/Help.\nMenus: arrows select, Enter runs, Esc closes. F10 / right-click opens a context menu.\nButtons: Tab reaches the row, Left/Right selects, Enter runs. Focus is white bold underlined text on dark blue.",
		"Tab / Shift+Tab으로 표시 중인 창을 순환합니다. 전체화면 키를 다시 누르면 분할 화면으로 복원합니다.\n보기 > 창 최대화는 메뉴·상태·푸터를 유지합니다. Alt+B 또는 보기 > 스키마로 탐색기를 표시·숨깁니다.\nAlt+M 메뉴 열기. Alt+F/G/E/S/X/V/T/H는 파일/SQL 탭/편집/세션/실행/보기/도구/도움말입니다.\n메뉴는 방향키 선택, Enter 실행, Esc 닫기입니다. F10 / 우클릭은 문맥 메뉴입니다.\n버튼은 Tab으로 접근, 좌우로 선택, Enter로 실행합니다. 선택은 진한 파랑 배경·흰색 굵은 밑줄 글자입니다."}},
	{"connection", [2]string{"Connection and settings", "접속 · 환경설정"}, []string{"connect"}, [2]string{
		"Startup opens login; Alt+L reopens it. Session > Connect (Alt+S, Enter)   |   Tools > Settings (Alt+T, Enter)\nConnection: profiles above; editable fields below, starting with Alias.\n  Alt+R profiles, Alt+A alias, Alt+H/P/U/W/T/N host/port/user/password/type/name\n  Ctrl+Enter / Alt+Enter / Alt+C connect, Ctrl+S / Alt+S save, Esc cancel\nSettings: Alt+L language, Alt+M mouse, Space toggle, Ctrl+S save.\nWhite fields accept text; dark fields with v select options. Passwords are not saved by default. Enable encrypted saving and press Save to opt in.\nAlt+K / Tools > Passwords: unlock, lock, change master, reset. Unlock needs the master password; it cannot be recovered.",
		"시작하면 접속 창이 열립니다. Alt+L로 다시 열거나 세션 > 접속(Alt+S, Enter)   |   도구 > 설정(Alt+T, Enter)\n접속 화면 위쪽은 저장된 접속 선택, 아래쪽은 별칭부터 시작하는 정보 편집입니다.\n  Alt+R 저장된 접속, Alt+A 별칭, Alt+H/P/U/W/T/N 호스트/포트/계정/암호/방식/이름\n  Ctrl+Enter / Alt+Enter / Alt+C 접속, Ctrl+S / Alt+S 저장, Esc 취소\n환경설정: Alt+L 언어, Alt+M 마우스, Space 선택, Ctrl+S 저장.\n흰 배경은 텍스트 입력, 어두운 배경과 v는 선택박스입니다. 비밀번호는 기본 미저장이며, 암호화 저장을 선택하고 저장해야 보관합니다.\nAlt+K / 도구 > 암호 관리: 잠금 해제·잠금·마스터 변경·초기화. 마스터 암호는 복구할 수 없습니다."}},
	{"schema", [2]string{"Schema, object details and source", "스키마 · 객체 상세 · 소스"}, []string{"browser_user", "browser_type", "browser_refresh", "describe", "quick_describe"}, [2]string{
		"Browser: choose User and Type above the object list. The login user and Tables are selected initially.\nAlt+U selects User; Alt+O selects Type. Tab cycles the selectors and list.\nArrows move, Enter/click opens, Ctrl+R refreshes. Selecting a browser user does not change the SQL session.\nObject: Alt+D DDL, Alt+C constraints, Alt+I indexes, Alt+G grants/relations, Alt+S source/edit, Alt+A arguments, Ctrl+R refresh, Esc back.\nSource: F9 apply/compile, Alt+R reload, Alt+X close, F10 edit menu, Tab cycles source/errors/buttons.\nCompilation is DDL and may commit prior DML, even when compilation fails.\nDiscarding unapplied source edits requires confirmation.",
		"탐색기 위쪽에서 사용자와 객체 종류를 선택합니다. 기본값은 접속 계정과 테이블입니다.\nAlt+U 사용자, Alt+O 객체 종류로 이동하고 Tab으로 선택칸과 목록을 순환합니다.\n방향키 이동, Enter/클릭 상세 조회, Ctrl+R 새로고침. 사용자 선택은 SQL 세션을 변경하지 않습니다.\n객체 상세: Alt+D DDL, Alt+C 제약조건, Alt+I 인덱스, Alt+G 권한·의존성, Alt+S 소스·편집, Alt+A 인자, Ctrl+R 새로고침, Esc 돌아가기.\n소스: F9 적용·컴파일, Alt+R 다시 조회, Alt+X 닫기, F10 편집 메뉴, Tab 소스/오류/버튼 이동.\n컴파일은 DDL이므로 실패하더라도 이전 DML이 커밋될 수 있습니다.\nDB에 적용하지 않은 소스 편집을 버릴 때는 확인합니다."}},
	{"transactions", [2]string{"Commit, rollback and exit", "커밋 · 롤백 · 종료"}, []string{"commit", "rollback", "pending_work", "cancel", "quit"}, [2]string{
		"DML uses manual commit. DDL / PL/SQL may commit implicitly.\nSession > View pending work lists SQL, time, outcome and affected rows since the last successful commit/rollback.\nExit > View pending work (Alt+V) opens the same details; Esc returns without resolving anything.\nThis is a memory-only execution journal (latest 100 statements, up to 64 KiB SQL each), not a row-level diff.\nExit always asks. Pending changes: commit and exit / rollback and exit / keep working.\nThe default is keep working; Esc cancels exit. Forced termination cannot show confirmation.",
		"DML은 수동 커밋입니다. DDL·PL/SQL은 암시적 커밋을 수행할 수 있습니다.\n세션 > 미커밋 내역 보기에서 마지막 커밋·롤백 이후 SQL·시각·결과·처리 행 수를 확인합니다.\n종료창의 내역 보기(Alt+V)도 같은 상세를 열며 Esc로 돌아가도 DB 상태는 바뀌지 않습니다.\n메모리 실행 기록(최근 100개, SQL당 64 KiB)이며 행별 변경 비교는 아닙니다.\n종료는 항상 확인합니다. 미커밋 가능 내역이 있으면 커밋 후 종료 / 롤백 후 종료 / 계속 사용을 선택합니다.\n기본 선택은 계속 사용이며 Esc는 종료를 취소합니다. 강제 종료에는 확인창을 보장하지 않습니다."}},
	{"terminal", [2]string{"Terminal compatibility and limits", "터미널 호환성 · 지원 범위"}, []string{"parse", "debug_continue"}, [2]string{
		"If Ctrl+Enter arrives as plain Enter, use F9 or Alt+Enter.\nSecureCRT: Session Options > Terminal > Emulation > Mapped Keys > Send String: \\e[13;5u\nClipboard transfer uses OSC 52 and depends on terminal permissions. See docs/terminal.md.\nF9 results fetch 1,000 rows per page (View/F10), up to 10,000 rows / estimated 64 MiB. F5 runs scripts (max 100 statements, 60 seconds total), stopping on first error/cancel. SQL*Plus directives and debugging are unsupported.",
		"Ctrl+Enter가 일반 Enter로 전달되면 F9 또는 Alt+Enter를 사용하세요.\nSecureCRT: Session Options > Terminal > Emulation > Mapped Keys > Send String: \\e[13;5u\n클립보드 전송은 OSC 52와 터미널 권한에 따라 동작합니다. docs/terminal.md를 참고하세요.\nF9 결과는 보기/F10에서 1,000행씩 이어 가져옵니다(최대 10,000행·추정 64 MiB). F5는 최대 100문장·전체 60초로 스크립트를 실행하고 첫 오류·취소에서 중단합니다. SQL*Plus 명령·디버거는 지원하지 않습니다."}},
}

func (s *screen) helpLanguage() int {
	if s.prefs.Language == "ko" {
		return 1
	}
	return 0
}

func (s *screen) mainHelp() string {
	var out strings.Builder
	for topic := 1; topic <= 9; topic++ {
		out.WriteString(s.helpArticle(topic))
		out.WriteString("\n\n")
	}
	return out.String()
}

// Keep navigation short; detailed titles and explanations belong in the article.
var helpTopicNames = [][2]string{
	{"Current screen", "현재 화면"}, {"SQL editor", "SQL 편집·실행"},
	{"Results / copy", "결과·복사·내보내기"}, {"Windows / menus", "화면·메뉴"},
	{"Connection", "접속·환경설정"}, {"Schema / source", "스키마·소스"},
	{"Transactions", "커밋·롤백·종료"}, {"Terminal / limits", "터미널·지원 범위"},
	{"About", "앱 정보"}, {"Editor keymaps", "편집기 키맵"},
}

func (s *screen) helpSubheading(title string) string {
	color := fmt.Sprintf("#%06x", palette.accent.Hex())
	if s.limitedColors {
		color = "teal"
	}
	return fmt.Sprintf("[%s::b]%s[-::-]\n\n", color, tview.Escape(title))
}

func (s *screen) helpCommands(ids []string) string {
	var out strings.Builder
	for _, id := range ids {
		// No fixed-width padding: long labels and key combinations wrap naturally.
		fmt.Fprintf(&out, "[::b]%s[::-]  %s\n", tview.Escape(s.editorBindingLabels(id, 2)), tview.Escape(s.tr(commandFor(id).label)))
	}
	out.WriteByte('\n')
	return out.String()
}

func helpParagraphs(text string) string {
	var out strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out.WriteString("• " + tview.Escape(line) + "\n\n")
		}
	}
	return out.String()
}

func (s *screen) helpArticle(topic int) string {
	language := s.helpLanguage()
	if topic == 0 {
		return s.helpSubheading(s.tr("help_current")) + helpParagraphs(s.helpContextText) +
			s.helpSubheading(s.tr("help_reading")) + helpParagraphs(s.tr("help_reading_hint"))
	}
	if topic == 8 {
		return s.helpSubheading(s.tr("help_about")) + fmt.Sprintf("[::b]%s v%s[::-]\n\n%s: %s\n\n%s: %s\n\n%s: %s\n%s\n\nCopyright (C) 2026 Chaekang Lee\n", appinfo.Name, appinfo.Version, s.tr("developer"), appinfo.Developer, s.tr("email"), appinfo.Email, s.tr("license"), appinfo.License, appinfo.LicenseURL)
	}
	if topic == 9 {
		return s.helpSubheading(helpTopicNames[9][language]) + helpParagraphs(s.keymapHelp())
	}
	section := helpSections[topic-1]
	var out strings.Builder
	out.WriteString(s.helpSubheading(section.title[language]))
	if topic == 1 {
		for _, group := range []struct {
			title    string
			commands []string
		}{
			{"help_run", []string{"execute", "script", "script_report", "plan", "output_view", "output_toggle", "routine_execute", "routine_results"}},
			{"help_files", []string{"file_new", "file_open", "file_save", "file_save_as", "file_close", "tab_next", "tab_previous"}},
			{"help_edit", []string{"undo", "redo", "cut", "copy", "paste", "select_all", "find", "find_next", "find_previous", "replace", "complete", "complete_refresh", "format_code", "duplicate_line", "copy_line", "cut_line", "delete_line", "line_up", "line_down", "transpose_line", "clear_sql", "history"}},
		} {
			out.WriteString(s.helpSubheading(s.tr(group.title)))
			out.WriteString(s.helpCommands(group.commands))
		}
	} else {
		out.WriteString(s.helpCommands(section.commands))
	}
	out.WriteString(s.helpSubheading(s.tr("help_details")))
	out.WriteString(helpParagraphs(section.notes[language]))
	if topic == 1 {
		out.WriteString(helpParagraphs(s.tr("format_hint")))
	}
	if topic == 2 {
		out.WriteString(helpParagraphs(s.tr("result_filter_hint") + "\n" + s.tr("result_paging_hint") + "\n" + s.tr("cell_hint")))
	}
	return out.String()
}

func (s *screen) helpContext(page string) string {
	switch page {
	case "cell":
		return s.tr("footer_cell") + "\n" + s.tr("cell_hint")
	case "result_filter":
		return s.tr("footer_result_filter") + "\n" + s.tr("result_filter_hint")
	case "definition":
		return s.tr("footer_definition") + "\n" + s.tr("definition_note") + "\n" + s.tr("definition_limits")
	case "editor_actions":
		return s.tr("menu_hint") + "\n" + s.keymapHelp()
	case "completion":
		return s.tr("footer_completion") + "\n" + s.tr("completion_hint")
	case "export":
		return s.tr("footer_export")
	case "plan":
		return s.tr("footer_plan") + "\n" + s.tr("plan_hint")
	case "output":
		return s.tr("footer_output") + "\n" + s.tr("output_note")
	case "routines", "routine_input", "routine_results", "routine_grid":
		return s.tr("footer_routine") + "\n" + s.tr("routine_hint") + "\n" + s.tr("routine_result_hint")
	case "bind":
		return s.tr("footer_bind") + "\n" + s.tr("bind_hint")
	case "script":
		return s.tr("footer_script")
	case "file":
		return s.tr("footer_file") + "\n" + s.tr("file_picker_hint")
	case "file_unsaved":
		return s.tr("footer_file_unsaved") + "\n" + s.tr("file_unsaved")
	case "sql_confirm":
		return s.tr("footer_sql_confirm") + "\n" + s.tr("sql_risk_limits")
	case "master":
		return s.tr("footer_master") + "\n" + s.tr("vault_master_hint")
	case "vault":
		return s.tr("footer_vault") + "\n" + s.tr("vault_hint")
	case "connect":
		return s.tr("footer_connect") + "\nCtrl+Enter / Alt+C: " + s.tr("connect") + " | Ctrl+S: " + s.tr("save")
	case "settings":
		return s.tr("footer_settings") + "\nCtrl+S: " + s.tr("save") + " | Esc: " + s.tr("cancel")
	case "object":
		return s.tr("footer_object") + "\nAlt+S: " + s.tr("source") + " | Alt+A: " + s.tr("arguments")
	case "source":
		return s.sourceKeymapFooter() + "\n" + s.tr("syntax_colors") + "\n" + s.tr("format_hint")
	case "arguments":
		return s.tr("footer_arguments")
	case "search":
		return s.tr("footer_search")
	case "pending":
		return s.tr("footer_pending")
	case "history":
		return s.tr("footer_history")
	case "copy":
		return s.tr("copy_preview_hint")
	}
	if s.browserHasFocus() {
		return s.tr("footer_browser")
	}
	if s.table.HasFocus() {
		return fmt.Sprintf(s.tr("footer_results"), shortcut("copy"), shortcut("copy_row"), shortcut("focus_toggle"), "F2 / Shift+F2")
	}
	hint := fmt.Sprintf(s.tr("footer_sql"), bindingLabels("execute", 2), shortcut("describe"), shortcut("focus_toggle"), "F2 / Shift+F2")
	if s.prefs.EditorKeymap == "notepad++" {
		hint = strings.ReplaceAll(hint, "Ctrl+T", "Ctrl+Space")
	}
	return s.keymapHint() + "\n" + hint + "\n" + s.tr("syntax_colors")
}

func (s *screen) showHelp() {
	if page, _ := s.pages.GetFrontPage(); page == "help" {
		return
	}
	s.helpFocus = s.app.GetFocus()
	page, _ := s.pages.GetFrontPage()
	s.helpContextText = s.helpContext(page)
	help := tview.NewTextView().SetDynamicColors(true).SetWrap(true).SetWordWrap(true).SetScrollable(true)
	help.SetBorder(true).SetTitle(s.tr("help_title"))
	help.SetBorderPadding(1, 1, 2, 2)
	styleBox(help.Box)
	help.SetTextColor(palette.text)
	singleLineBorder(help.Box)
	topics := tview.NewList().ShowSecondaryText(false).SetHighlightFullLine(true).SetSelectedFocusOnly(false)
	topics.SetMainTextColor(palette.text).SetSelectedStyle(buttonFocusStyle().Underline(false))
	topics.SetBorder(true).SetTitle(s.tr("help_topics"))
	styleBox(topics.Box)
	singleLineBorder(topics.Box)
	s.helpText, s.helpTopics = help, topics
	for i, title := range helpTopicNames {
		topics.AddItem(fmt.Sprintf("%d  %s", i, title[s.helpLanguage()]), "", 0, nil)
	}
	topics.SetChangedFunc(func(index int, _, _ string, _ rune) { s.selectHelpTopic(index) })
	topics.SetSelectedFunc(func(_ int, _, _ string, _ rune) { s.app.SetFocus(help) })
	body := tview.NewFlex().AddItem(topics, 26, 0, false).AddItem(help, 0, 1, true)
	// Narrow terminals retain both navigation and readable article width.
	body.SetDrawFunc(func(_ tcell.Screen, x, y, width, height int) (int, int, int, int) {
		if width < 76 {
			body.SetDirection(tview.FlexRow).ResizeItem(topics, min(5, max(3, height/4)), 0)
		} else {
			body.SetDirection(tview.FlexColumn).ResizeItem(topics, 26, 0)
		}
		return x, y, width, height
	})
	close := clickable(tview.NewButton(s.tr("close") + " (Esc)").SetSelectedFunc(s.closeHelp))
	root := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(body, 0, 1, true).AddItem(close, 1, 0, false)
	s.selectHelpTopic(0)
	s.pages.AddPage("help", root, true, true)
	s.app.SetFocus(help)
}

func (s *screen) helpJump(section int) {
	if s.helpTopics == nil || section < 0 || section >= len(helpTopicNames) {
		return
	}
	s.helpTopics.SetCurrentItem(section)
	s.selectHelpTopic(section)
	s.app.SetFocus(s.helpText)
}

func (s *screen) selectHelpTopic(topic int) {
	s.helpText.SetText(s.helpArticle(topic)).ScrollToBeginning()
}

func (s *screen) closeHelp() {
	s.pages.RemovePage("help")
	if s.helpFocus != nil {
		s.app.SetFocus(s.helpFocus)
	}
	s.helpFocus = nil
	s.helpText, s.helpTopics, s.helpContextText = nil, nil, ""
}
