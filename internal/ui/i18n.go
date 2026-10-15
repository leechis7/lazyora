package ui

import "fmt"

// SQL, database object names and server errors keep their original content.
var messages = map[string][2]string{
	"saved_connections":     {"Saved connections (Alt+R)", "저장된 접속 (Alt+R)"},
	"new_connection":        {"New connection", "새 접속"},
	"alias":                 {"Alias (Alt+A)", "별칭 (Alt+A)"},
	"alias_required":        {"Enter an alias to save this connection.", "저장할 접속의 별칭을 입력하세요."},
	"connection_saved":      {"Connection saved without password. Ctrl+Enter connects.", "비밀번호를 제외한 접속 정보를 저장했습니다. Ctrl+Enter로 접속하세요."},
	"connection_save_error": {"Could not save connection: %s", "접속 정보를 저장하지 못했습니다: %s"},
	"connection_load_error": {"Could not load connections (saving disabled): %s", "접속 정보를 읽지 못했습니다 (저장 불가): %s"},
	"help":                  {"Help", "도움말"},
	"connect":               {"Connect", "접속"},
	"settings":              {"Settings", "설정"},
	"execute":               {"Run", "실행"},
	"commit":                {"Commit", "커밋"},
	"rollback":              {"Rollback", "롤백"},
	"cancel":                {"Cancel", "취소"},
	"quit":                  {"Quit", "종료"},
	"save":                  {"Save", "저장"},
	"close":                 {"Close", "닫기"},
	"sql_title":             {" SQL — one statement or PL/SQL block ", " SQL — 한 문장 또는 PL/SQL 블록 "},
	"results_title":         {" Results ", " 조회 결과 "},
	"hint":                  {"Ctrl+Enter Run | F9 SQL | F10 Results | Tab Focus | Ctrl+Q Quit", "Ctrl+Enter 실행 | F9 SQL | F10 결과 | Tab 전환 | Ctrl+Q 종료"},
	"disconnected":          {"Disconnected. F2 opens connection settings.", "연결되지 않았습니다. F2로 접속 화면을 여세요."},
	"connect_first":         {"Connect first with F2.", "먼저 F2로 DB에 접속하세요."},
	"cancelling":            {"Cancelling operation before exit…", "작업을 취소하고 종료하는 중…"},
	"running":               {"Running %s… F8 cancels. Oracle DDL can implicitly commit.", "%s 중… F8 취소. Oracle DDL은 암시적으로 커밋할 수 있습니다."},
	"error":                 {"Error: %s", "오류: %s"},
	"connected":             {"Connected: %s@%s:%d/%s (%s). Manual commit; DDL implicitly commits.", "접속됨: %s@%s:%d/%s (%s). 수동 커밋; DDL은 암시적 커밋."},
	"rows":                  {"%d rows · %s", "%d행 · %s"},
	"rows_limited":          {"%d rows · %s · limited to 1000 rows", "%d행 · %s · 최대 1,000행"},
	"limit":                 {" · limited to 1000 rows", " · 최대 1,000행"},
	"executed":              {"Statement completed · affected rows: %d · %s · F5 Commit / F6 Rollback", "실행 완료 · 처리 행: %d · %s · F5 커밋 / F6 롤백"},
	"committed":             {"Commit completed.", "커밋했습니다."},
	"rolled_back":           {"Rollback completed.", "롤백했습니다."},
	"demo":                  {"Demo mode. F2 connects to Oracle; no statements run without a connection.", "데모 모드입니다. F2로 Oracle에 접속해야 SQL을 실행할 수 있습니다."},
	"demo_cell":             {"Demo — no database connected", "데모 — DB 연결 없음"},
	"host":                  {"Host (Alt+H)", "호스트 (Alt+H)"},
	"port":                  {"Port (Alt+P)", "포트 (Alt+P)"},
	"username":              {"Username (Alt+U)", "계정 (Alt+U)"},
	"password":              {"Password (Alt+W)", "비밀번호 (Alt+W)"},
	"connection_type":       {"Connection type (Alt+T)", "접속 방식 (Alt+T)"},
	"service_name":          {"Service Name / SID (Alt+N)", "Service Name / SID (Alt+N)"},
	"connection_title":      {" Oracle connection — reconnect rolls back previous session ", " Oracle 접속 — 재접속 시 이전 세션 롤백 "},
	"required":              {"Host, username and service name/SID are required.", "호스트, 계정과 Service Name/SID를 입력하세요."},
	"port_error":            {"Port must be between 1 and 65535.", "포트는 1~65535 사이의 숫자여야 합니다."},
	"language":              {"Language (Alt+L)", "언어 (Alt+L)"},
	"mouse":                 {"Mouse (Alt+M)", "마우스 (Alt+M)"},
	"mouse_label":           {"Enable mouse clicks and scrolling", "마우스 클릭·스크롤 사용"},
	"settings_title":        {" Settings — Ctrl+S saves / Esc cancels ", " 설정 — Ctrl+S 저장 / Esc 취소 "},
	"settings_saved":        {"Settings saved.", "설정을 저장했습니다."},
	"settings_error":        {"Could not save settings: %s", "설정을 저장하지 못했습니다: %s"},
	"settings_load_error":   {"Using default settings: %s", "기본 설정으로 시작합니다: %s"},
	"help_title":            {" Help — F1 / Esc closes ", " 도움말 — F1 / Esc 닫기 "},
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
	s.editor.SetTitle(s.tr("sql_title"))
	s.table.SetTitle(s.tr("results_title"))
	if s.hint != nil {
		s.hint.SetText(s.tr("hint"))
	}
	for i, b := range s.buttons {
		b.SetLabel(toolbarActions[i].key + " " + s.tr(toolbarActions[i].action))
	}
	if s.connectForm != nil {
		s.localizeConnection()
	}
	if s.demo && s.table.GetRowCount() > 1 {
		s.table.GetCell(1, 1).SetText(s.tr("demo_cell"))
	}
	if s.statusKey != "" {
		s.setStatus(s.statusKey, s.statusArgs...)
	}
}
