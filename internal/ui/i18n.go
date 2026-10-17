package ui

import "fmt"

// SQL, database object names and server errors keep their original content.
var messages = map[string][2]string{
	"fullscreen":             {"Fullscreen / restore", "전체화면 / 복원"},
	"maximize_short":         {"Max", "최대화"},
	"fullscreen_short":       {"Full", "전체화면"},
	"copy_empty":             {"No data selected to copy.", "복사할 데이터를 선택하세요."},
	"copy_requested":         {"Clipboard request sent (%d bytes). Edit > Copy preview if unsupported.", "클립보드 전송 요청 (%d바이트). 미지원 시 편집 > 복사 내용 보기."},
	"copy_preview":           {"Copy preview", "복사 내용 보기"},
	"copy_preview_hint":      {"Mouse reporting is off. Select text with your terminal and copy; arrows/PgUp/PgDn scroll, Esc closes.", "터미널에서 글자를 선택해 복사하세요. 방향키/PgUp/PgDn 스크롤, Esc 닫기 (마우스 보고 일시 중지)."},
	"copy_selection":         {"Copy cell / range", "셀 / 범위 복사"},
	"copy_row":               {"Copy row", "행 복사"},
	"cancel_execution":       {"Cancel execution", "실행 취소"},
	"script_unavailable":     {"Run script (not supported yet)", "스크립트 실행 (아직 지원하지 않음)"},
	"maximize":               {"Maximize panel / restore", "창 최대화 / 복원"},
	"maximized":              {"Maximized", "최대화"},
	"focus_sql":              {"Focus SQL editor", "쿼리창 이동"},
	"focus_results":          {"Focus results", "결과창 이동"},
	"describe":               {"Describe object", "객체 상세 조회"},
	"object_reference":       {"Object / schema.object", "객체 / 스키마.객체"},
	"object_reference_error": {"Enter object name or schema.object.", "객체명 또는 스키마.객체명을 입력하세요."},
	"object_not_found":       {"Object not found.", "객체를 찾을 수 없습니다."},
	"footer_describe":        {"Enter Describe | Esc Close", "Enter 상세 조회 | Esc 닫기"},
	"menu_hint":              {"Alt+M Menu | Left/Right Menus | Up/Down Select | Enter Run | Esc Close", "Alt+M 메뉴 | 좌우 메뉴 이동 | 위아래 선택 | Enter 실행 | Esc 닫기"},
	"menu_file":              {"File", "파일"},
	"menu_edit":              {"Edit", "편집"},
	"menu_session":           {"Session", "세션"},
	"menu_execute":           {"Execute", "실행"},
	"menu_view":              {"View", "보기"},
	"menu_tools":             {"Tools", "도구"},
	"menu_help":              {"Help", "도움말"},

	"source":                  {"Source", "소스"},
	"source_button":           {"Source / Edit (Alt+S)", "소스 / 편집 (Alt+S)"},
	"arguments":               {"Arguments", "인자"},
	"arguments_button":        {"Arguments (Alt+A)", "인자 (Alt+A)"},
	"compile":                 {"Compile", "컴파일"},
	"compile_source":          {"Compiling object", "객체 컴파일"},
	"compile_button":          {"Apply / Compile (Ctrl+F9)", "적용 / 컴파일 (Ctrl+F9)"},
	"reload_button":           {"Reload (Ctrl+R)", "다시 조회 (Ctrl+R)"},
	"reload":                  {"Reload", "다시 조회"},
	"proceed":                 {"Proceed", "진행"},
	"discard_source":          {"Source edits have not been applied to Oracle.\nDiscard them for this action?", "소스 편집 내용을 DB에 적용하지 않았습니다.\n편집 내용을 버리고 진행하시겠습니까?"},
	"compile_confirm":         {"Apply/compile this object in Oracle?\nDDL may commit prior changes even if compilation fails.\nSavepoint/rollback cannot undo CREATE OR REPLACE.", "Oracle에 소스를 적용·컴파일하시겠습니까?\nDDL은 컴파일 실패 시에도 이전 변경을 커밋할 수 있습니다.\nCREATE OR REPLACE는 롤백으로 되돌릴 수 없습니다."},
	"compiling":               {"Applying/compiling source…", "소스 적용·컴파일 중…"},
	"source_ready":            {"Edit source; Ctrl+F9 applies/compiles. * means unapplied edits.", "소스를 편집하고 Ctrl+F9로 적용·컴파일합니다. *는 미적용 편집입니다."},
	"compile_status":          {"Object status: %s · %d compiler messages", "객체 상태: %s · 컴파일 메시지 %d개"},
	"diagnostics":             {" Compiler errors / warnings ", " 컴파일 오류 / 경고 "},
	"line":                    {"Line", "행"},
	"position":                {"Position", "위치"},
	"message":                 {"Message", "메시지"},
	"member":                  {"Member", "멤버"},
	"overload":                {"Overload", "오버로드"},
	"argument":                {"Argument", "인자"},
	"direction":               {"Mode", "방향"},
	"depth":                   {"Depth", "깊이"},
	"return_value":            {"(RETURN)", "(반환값)"},
	"arguments_empty":         {"No visible arguments (parameterless routines may have no rows).", "조회 가능한 인자가 없습니다 (인자 없는 루틴은 행이 없을 수 있습니다)."},
	"footer_source":           {"Ctrl+F9 Apply/Compile | Ctrl+R Reload | Tab Source/Errors | Esc Back", "Ctrl+F9 적용/컴파일 | Ctrl+R 다시 조회 | Tab 소스/오류 | Esc 돌아가기"},
	"footer_arguments":        {"↑↓←→ Move | PgUp/PgDn Page | Esc Back", "↑↓←→ 이동 | PgUp/PgDn 스크롤 | Esc 돌아가기"},
	"footer_confirm":          {"←→ / Tab Select | Enter Confirm | Esc Cancel", "←→ / Tab 선택 | Enter 확인 | Esc 취소"},
	"materialized_views":      {"Materialized views", "구체화 뷰"},
	"functions":               {"Functions", "함수"},
	"procedures":              {"Procedures", "프로시저"},
	"packages":                {"Packages", "패키지"},
	"package_bodies":          {"Package bodies", "패키지 본문"},
	"sequences":               {"Sequences", "시퀀스"},
	"synonyms":                {"Synonyms", "동의어"},
	"indexes":                 {"Indexes", "인덱스"},
	"triggers":                {"Triggers", "트리거"},
	"types":                   {"Types", "타입"},
	"type_bodies":             {"Type bodies", "타입 본문"},
	"object_details":          {" Object details ", " 객체 정보 "},
	"property":                {"Property", "항목"},
	"value":                   {"Value", "값"},
	"object_status":           {"Status", "상태"},
	"object_info_only":        {"Object metadata. Available actions are shown below.", "객체 기본 정보입니다. 가능한 동작은 아래 버튼을 확인하세요."},
	"object_info_loaded":      {"Object details loaded: %s", "객체 정보를 불러왔습니다: %s"},
	"exit_clean":              {"No uncommitted changes are tracked.\nExit LazyOra?", "앱에서 추적한 미커밋 변경 내역이 없습니다.\n종료하시겠습니까?"},
	"exit_pending":            {"Statements that may change data have not been committed or rolled back by LazyOra.\nCommit and exit, roll back and exit, or keep working?", "변경 가능한 SQL 실행 후 앱에서 커밋·롤백하지 않은 내역이 있습니다.\n커밋 후 종료, 롤백 후 종료 또는 계속 사용을 선택하세요."},
	"exit_now":                {"Exit", "종료"},
	"exit_commit":             {"Commit and exit", "커밋 후 종료"},
	"exit_rollback":           {"Roll back and exit", "롤백 후 종료"},
	"exit_cancel":             {"Keep working", "계속 사용"},
	"exit_finishing":          {"Finishing the transaction before exit…", "종료 전 트랜잭션 처리 중…"},
	"exit_failed":             {"Could not finish the transaction. LazyOra will stay open.\nReconnect to check the transaction outcome.", "트랜잭션 처리를 확인하지 못해 종료하지 않았습니다.\n접속을 다시 열어 DB 처리 결과를 확인하세요."},
	"footer_exit":             {"←→ / Tab Select | Enter Confirm | Esc Keep working", "←→ / Tab 선택 | Enter 확인 | Esc 계속 사용"},
	"browser_retry":           {"Ctrl+R: load schemas", "Ctrl+R: 스키마 불러오기"},
	"schemas":                 {"Schemas", "스키마"},
	"tables":                  {"Tables", "테이블"},
	"views":                   {"Views", "뷰"},
	"browser_title":           {" Schemas (F7) ", " 스키마 (F7) "},
	"browser_connect":         {"F2: connect to browse", "F2: 접속 후 탐색"},
	"browser_expand":          {"Enter: load objects", "Enter: 객체 불러오기"},
	"browser_empty":           {"(empty)", "(없음)"},
	"browser_loading":         {"Loading metadata… F8 cancels.", "메타데이터 조회 중… F8로 취소합니다."},
	"browser_error":           {"Metadata query failed", "메타데이터 조회 실패"},
	"schemas_loaded":          {"%d schemas loaded.", "스키마 %d개를 불러왔습니다."},
	"objects_loaded":          {"%s: %d accessible objects.", "%s: 접근 가능한 객체 %d개."},
	"columns_loaded":          {"%d columns loaded.", "컬럼 %d개를 불러왔습니다."},
	"columns_empty":           {"No accessible columns. The object may have been removed or privileges changed.", "조회 가능한 컬럼이 없습니다. 객체 삭제 또는 권한 변경을 확인하세요."},
	"columns_title":           {" Columns ", " 컬럼 "},
	"owner":                   {"Schema", "스키마"},
	"object_name":             {"Object", "객체"},
	"object_type":             {"Type", "종류"},
	"column_position":         {"#", "순서"},
	"column_name":             {"Column", "컬럼"},
	"data_type":               {"Data type", "데이터 타입"},
	"column_comment":          {"Comment", "코멘트"},
	"nullable":                {"Nullable", "NULL 허용"},
	"yes":                     {"Yes", "예"},
	"no":                      {"No", "아니요"},
	"footer_browser":          {"F7 Hide | ↑↓ Move | ←→ Fold/Expand | Enter Open | Ctrl+R Refresh | Tab SQL", "F7 숨김 | ↑↓ 이동 | ←→ 접기/펼치기 | Enter 열기 | Ctrl+R 새로고침 | Tab SQL"},
	"footer_object":           {"↑↓←→ Move | PgUp/PgDn Page | Ctrl+R Refresh | Esc Back", "↑↓←→ 이동 | PgUp/PgDn 스크롤 | Ctrl+R 새로고침 | Esc 돌아가기"},
	"developer":               {"Developer", "개발자"},
	"email":                   {"Email", "이메일"},
	"license":                 {"License", "라이선스"},
	"footer_sql":              {"%s Run | %s | Tab Results", "%s 실행 | %s | Tab 결과"},
	"footer_results":          {"%s Copy | %s Row | Shift+Arrows | PgUp/PgDn | %s", "%s 복사 | %s 행 | Shift+방향키 | PgUp/PgDn | %s"},
	"footer_connect":          {"Tab Fields | Alt+R Profiles | Alt+A Alias | Alt+W Password", "Tab 항목 이동 | Alt+R 저장된 접속 | Alt+A 별칭 | Alt+W 비밀번호"},
	"footer_settings":         {"Alt+L Language | Alt+M Mouse | Space Toggle | Tab Fields", "Alt+L 언어 | Alt+M 마우스 | Space 선택 | Tab 항목 이동"},
	"footer_help":             {"↑↓ / PgUp/PgDn Scroll | F1 / Esc Close", "↑↓ / PgUp/PgDn 스크롤 | F1 / Esc 닫기"},
	"footer_busy":             {"F8 Cancel", "F8 실행 취소"},
	"previous_page":           {"Previous", "이전"},
	"next_page":               {"Next", "다음"},
	"saved_connections_title": {" Load saved connection ", " 저장된 접속 불러오기 "},
	"connection_details":      {" Connection details — reconnect rolls back previous session ", " 접속 정보 편집 — 재접속 시 이전 세션 롤백 "},
	"saved_connections":       {"Saved connections (Alt+R)", "저장된 접속 (Alt+R)"},
	"new_connection":          {"New connection", "새 접속"},
	"alias":                   {"Alias (Alt+A)", "별칭 (Alt+A)"},
	"alias_required":          {"Enter an alias to save this connection.", "저장할 접속의 별칭을 입력하세요."},
	"connection_saved":        {"Connection saved without password. Ctrl+Enter connects.", "비밀번호를 제외한 접속 정보를 저장했습니다. Ctrl+Enter로 접속하세요."},
	"connection_save_error":   {"Could not save connection: %s", "접속 정보를 저장하지 못했습니다: %s"},
	"connection_load_error":   {"Could not load connections (saving disabled): %s", "접속 정보를 읽지 못했습니다 (저장 불가): %s"},
	"help":                    {"Help", "도움말"},
	"connect":                 {"Connect", "접속"},
	"settings":                {"Settings", "설정"},
	"execute":                 {"Run", "실행"},
	"commit":                  {"Commit", "커밋"},
	"rollback":                {"Rollback", "롤백"},
	"cancel":                  {"Cancel", "취소"},
	"quit":                    {"Quit", "종료"},
	"save":                    {"Save", "저장"},
	"close":                   {"Close", "닫기"},
	"sql_title":               {" SQL — one statement or PL/SQL block ", " SQL — 한 문장 또는 PL/SQL 블록 "},
	"results_title":           {" Results ", " 조회 결과 "},
	"disconnected":            {"Disconnected. F2 opens connection settings.", "연결되지 않았습니다. F2로 접속 화면을 여세요."},
	"connect_first":           {"Connect first with F2.", "먼저 F2로 DB에 접속하세요."},
	"cancelling":              {"Cancelling operation before exit confirmation…", "종료 확인 전에 실행 중인 작업을 취소하는 중…"},
	"running":                 {"Running %s… Oracle DDL can implicitly commit.", "%s 중… Oracle DDL은 암시적으로 커밋할 수 있습니다."},
	"error":                   {"Error: %s", "오류: %s"},
	"connected":               {"Connected: %s@%s:%d/%s (%s). Manual commit; DDL implicitly commits.", "접속됨: %s@%s:%d/%s (%s). 수동 커밋; DDL은 암시적 커밋."},
	"rows":                    {"%d rows · %s", "%d행 · %s"},
	"rows_limited":            {"%d rows · %s · limited to 1000 rows", "%d행 · %s · 최대 1,000행"},
	"limit":                   {" · limited to 1000 rows", " · 최대 1,000행"},
	"executed":                {"Statement completed · affected rows: %d · %s · manual commit", "실행 완료 · 처리 행: %d · %s · 수동 커밋"},
	"committed":               {"Commit completed.", "커밋했습니다."},
	"rolled_back":             {"Rollback completed.", "롤백했습니다."},
	"demo":                    {"Demo mode. F2 connects to Oracle; no statements run without a connection.", "데모 모드입니다. F2로 Oracle에 접속해야 SQL을 실행할 수 있습니다."},
	"demo_cell":               {"Demo — no database connected", "데모 — DB 연결 없음"},
	"host":                    {"Host (Alt+H)", "호스트 (Alt+H)"},
	"port":                    {"Port (Alt+P)", "포트 (Alt+P)"},
	"username":                {"Username (Alt+U)", "계정 (Alt+U)"},
	"password":                {"Password (Alt+W)", "비밀번호 (Alt+W)"},
	"connection_type":         {"Connection type (Alt+T)", "접속 방식 (Alt+T)"},
	"service_name":            {"Service Name / SID (Alt+N)", "Service Name / SID (Alt+N)"},
	"connection_title":        {" Oracle connection — reconnect rolls back previous session ", " Oracle 접속 — 재접속 시 이전 세션 롤백 "},
	"required":                {"Host, username and service name/SID are required.", "호스트, 계정과 Service Name/SID를 입력하세요."},
	"port_error":              {"Port must be between 1 and 65535.", "포트는 1~65535 사이의 숫자여야 합니다."},
	"language":                {"Language (Alt+L)", "언어 (Alt+L)"},
	"mouse":                   {"Mouse (Alt+M)", "마우스 (Alt+M)"},
	"mouse_label":             {"Enable mouse clicks and scrolling", "마우스 클릭·스크롤 사용"},
	"settings_title":          {" Settings — Ctrl+S saves / Esc cancels ", " 설정 — Ctrl+S 저장 / Esc 취소 "},
	"settings_saved":          {"Settings saved.", "설정을 저장했습니다."},
	"settings_error":          {"Could not save settings: %s", "설정을 저장하지 못했습니다: %s"},
	"settings_load_error":     {"Using default settings: %s", "기본 설정으로 시작합니다: %s"},
	"help_title":              {" Help — F1 / Esc closes ", " 도움말 — F1 / Esc 닫기 "},
}

func (s *screen) tr(key string) string {
	value, ok := messages[key]
	if !ok {
		return key
	}
	if s.prefs.Language == "ko" {
		return value[1]
	}
	return value[0]
}

func (s *screen) setStatus(key string, args ...any) {
	s.statusKey, s.statusArgs = key, args
	if key == "running" && len(args) == 1 {
		args = []any{s.tr(args[0].(string))}
	}
	if len(args) == 0 {
		s.status.SetText(s.tr(key))
		return
	}
	s.status.SetText(fmt.Sprintf(s.tr(key), args...))
}

func (s *screen) refreshLabels() {
	s.refreshMenuLabels()
	s.localizeBrowser()
	s.editor.SetTitle(s.tr("sql_title"))
	s.table.SetTitle(s.tr("results_title"))
	s.refreshFooter()
	if s.connectForm != nil {
		s.localizeConnection()
	}
	if s.demo && s.table.GetRowCount() > 1 {
		s.table.GetCell(1, 1).SetText(s.tr("demo_cell"))
		if len(s.resultRows) > 0 && len(s.resultRows[0]) > 1 {
			s.resultRows[0][1] = s.tr("demo_cell")
		}
	}
	if s.statusKey != "" {
		s.setStatus(s.statusKey, s.statusArgs...)
	}
}
