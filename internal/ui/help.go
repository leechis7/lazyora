package ui

import (
	"fmt"

	"github.com/rivo/tview"
	"lazyora/internal/appinfo"
)

var shortcutHelp = [2]string{
	`Keyboard shortcuts

F1 / Esc             Open / close help
F2                   Connect / reconnect
F3                   Settings (language and mouse)
Ctrl+Enter / F4       Run SQL
F5 / F6              Commit / rollback
F8                   Request cancellation
F7                   Show / hide schema browser
F9 / F10             Focus SQL editor / results
Tab / Shift+Tab      Change focus
Enter                Newline in the SQL editor
Ctrl+Q / Ctrl+C      Confirm exit (also when no changes are pending)

Connection: Ctrl+Enter or Alt+C connects; Esc or Alt+X cancels.
Alt+H/P/U/W/T/N selects host/port/user/password/type/service name.
Alt+R selects saved connections; Alt+A selects alias; Ctrl+S/Alt+S saves.
White fields accept text; dark fields with v select an option.
The upper section loads profiles; the lower section edits details, starting with Alias.
Tab/Shift+Tab cycles both sections. New connection restores defaults and clears passwords.
Connection profiles never store passwords. Selecting one clears the password.
Settings: Ctrl+S or Alt+S saves; Esc or Alt+X cancels.
Alt+L selects language; Alt+M selects the mouse checkbox (Space toggles).
Mouse: click toolbar buttons, fields or result cells; wheel scrolls.
Schemas: Enter/click opens schemas, groups or object details; Left/Right folds/expands.
Ctrl+R refreshes schemas (browser) or metadata (object detail); Esc closes detail.
Tab cycles SQL → results → visible schemas; Shift+Tab cycles in reverse.
F7 preserves the tree when hidden; showing it focuses the browser.
Tables/views/materialized views show columns and column comments; other objects show name, type and status.
Functions, procedures, packages/bodies, sequences, synonyms, indexes, triggers and types/bodies are listed.
Alt+S: source editor for functions/procedures/packages/types/bodies/triggers.
Alt+A: arguments, return values and overloads for functions/procedures/packages.
Source: Ctrl+F9 compile/apply, Ctrl+R reload, Esc close; Tab switches diagnostics.
Compilation reports error lines/columns and requires confirmation for implicit DDL commits.
Unsaved edits require discard confirmation. Browsing does not change the current schema.

Run executes the whole editor as one SQL statement or BEGIN/DECLARE block.
Results are limited to 1,000 rows. DML needs manual commit; DDL can commit implicitly.
Ctrl+Enter needs a terminal that reports modifiers; otherwise use F4.
Exit: commit and exit / roll back and exit / keep working when changes may be pending.
Esc or the default button keeps working. Uncertain transaction outcomes require reconnection.
Arrow keys and PageUp/PageDown scroll this help.`,
	`키보드 단축키

F1 / Esc             도움말 열기 / 닫기
F2                   접속 / 재접속
F3                   설정 (언어·마우스)
Ctrl+Enter / F4       SQL 실행
F5 / F6              커밋 / 롤백
F8                   실행 취소 요청
F7                   스키마 창 표시 / 숨김
F9 / F10             SQL 편집기 / 결과 표로 이동
Tab / Shift+Tab      포커스 전환
Enter                SQL 편집기에서 줄바꿈
Ctrl+Q / Ctrl+C      종료 확인 (변경 내역이 없어도 확인)

접속 화면: Ctrl+Enter 또는 Alt+C 접속, Esc 또는 Alt+X 취소
Alt+H/P/U/W/T/N: 호스트/포트/계정/비밀번호/접속 방식/서비스 이름 선택
Alt+R: 저장된 접속 선택, Alt+A: 별칭 입력, Ctrl+S/Alt+S: 접속 정보 저장
흰 배경은 직접 입력, 어두운 배경과 v 표시는 선택박스입니다.
위쪽은 저장된 접속 불러오기, 아래쪽은 별칭부터 시작하는 접속 정보 편집 영역입니다.
Tab/Shift+Tab으로 두 영역을 순환합니다. 새 접속은 기본값을 복원하고 비밀번호를 비웁니다.
비밀번호는 저장하지 않으며 저장된 접속 선택 시 비밀번호 칸을 비웁니다.
설정 화면: Ctrl+S 또는 Alt+S 저장, Esc 또는 Alt+X 취소
Alt+L: 언어 선택, Alt+M: 마우스 항목 선택 (Space로 켜기·끄기)
마우스: 버튼·입력 항목·결과 셀 클릭, 휠 스크롤
스키마: Enter/클릭으로 스키마·분류·객체 정보 열기, 좌/우로 접기·펼치기
Ctrl+R: 탐색기에서는 스키마 목록, 객체 상세에서는 컬럼·기본 정보 새로고침. Esc로 상세 닫기
Tab: SQL → 결과 → 표시 중인 스키마 순환, Shift+Tab: 역순 이동
F7로 숨겨도 트리 상태를 유지하며 다시 열면 탐색기에 포커스를 둡니다.
테이블·뷰·구체화 뷰는 컬럼과 코멘트를, 그 밖의 객체는 이름·종류·상태를 표시합니다.
함수·프로시저·패키지/본문·시퀀스·동의어·인덱스·트리거·타입/본문을 탐색합니다.
Alt+S: 함수·프로시저·패키지/타입 및 본문·트리거 소스 조회·편집
Alt+A: 함수·프로시저·패키지의 인자·반환값·오버로드 조회
소스: Ctrl+F9 적용·컴파일, Ctrl+R 새로고침, Esc 닫기, Tab 오류 목록 이동
컴파일 오류의 행·열을 표시하며 DDL의 암시적 커밋은 실행 전에 확인합니다.
미적용 편집을 버릴 때 확인합니다. 탐색으로 현재 스키마를 변경하지 않습니다.

편집기 전체 내용을 SQL 한 문장 또는 BEGIN/DECLARE 블록 하나로 실행합니다.
결과는 최대 1,000행입니다. DML은 수동 커밋, DDL은 암시적 커밋이 가능합니다.
Ctrl+Enter를 구분하지 못하는 터미널에서는 F4를 사용하세요.
미커밋 가능 내역이 있으면 커밋 후 종료 / 롤백 후 종료 / 계속 사용을 선택합니다.
Esc와 기본 버튼은 계속 사용입니다. 트랜잭션 처리 실패 시 재접속해 결과를 확인하세요.
위/아래 화살표와 PageUp/PageDown으로 도움말을 스크롤할 수 있습니다.`,
}

func (s *screen) showHelp() {
	s.helpFocus = s.app.GetFocus()
	index := 0
	if s.prefs.Language == "ko" {
		index = 1
	}
	info := fmt.Sprintf("%s v%s\n%s: %s\n%s: %s\n%s: GNU AGPL v3 (%s)\n%s\nCopyright (C) 2026 Chaekang Lee\n\n",
		appinfo.Name, appinfo.Version, s.tr("developer"), appinfo.Developer,
		s.tr("email"), appinfo.Email, s.tr("license"), appinfo.License, appinfo.LicenseURL)
	help := tview.NewTextView().SetText(info + shortcutHelp[index]).SetWrap(true).SetWordWrap(true).SetScrollable(true)
	help.SetBorder(true).SetTitle(s.tr("help_title"))
	styleBox(help.Box)
	help.SetTextColor(palette.text)
	close := clickable(tview.NewButton(s.tr("close") + " (Esc)").SetSelectedFunc(s.closeHelp))
	root := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(help, 0, 1, true).AddItem(close, 1, 0, false)
	s.pages.AddPage("help", root, true, true)
	s.app.SetFocus(help)
}

func (s *screen) closeHelp() {
	s.pages.RemovePage("help")
	if s.helpFocus != nil {
		s.app.SetFocus(s.helpFocus)
	}
	s.helpFocus = nil
}
