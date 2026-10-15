package ui

import "github.com/rivo/tview"

var shortcutHelp = [2]string{
	`Keyboard shortcuts

F1 / Esc             Open / close help
F2                   Connect / reconnect
F3                   Settings (language and mouse)
Ctrl+Enter / F4       Run SQL
F5 / F6              Commit / rollback
F8                   Request cancellation
F9 / F10             Focus SQL editor / results
Tab / Shift+Tab      Change focus
Enter                Newline in the SQL editor
Ctrl+Q / Ctrl+C      Quit and roll back pending changes

Connection: Ctrl+Enter or Alt+C connects; Esc or Alt+X cancels.
Alt+H/P/U/W/T/N selects host/port/user/password/type/service name.
Alt+R selects saved connections; Alt+A selects alias; Ctrl+S/Alt+S saves.
Connection profiles never store passwords. Selecting one clears the password.
Settings: Ctrl+S or Alt+S saves; Esc or Alt+X cancels.
Alt+L selects language; Alt+M selects the mouse checkbox (Space toggles).
Mouse: click toolbar buttons, fields or result cells; wheel scrolls.

Run executes the whole editor as one SQL statement or BEGIN/DECLARE block.
Results are limited to 1,000 rows. DML needs manual commit; DDL can commit implicitly.
Ctrl+Enter needs a terminal that reports modifiers; otherwise use F4.
Arrow keys and PageUp/PageDown scroll this help.`,
	`키보드 단축키

F1 / Esc             도움말 열기 / 닫기
F2                   접속 / 재접속
F3                   설정 (언어·마우스)
Ctrl+Enter / F4       SQL 실행
F5 / F6              커밋 / 롤백
F8                   실행 취소 요청
F9 / F10             SQL 편집기 / 결과 표로 이동
Tab / Shift+Tab      포커스 전환
Enter                SQL 편집기에서 줄바꿈
Ctrl+Q / Ctrl+C      종료 (미완료 트랜잭션 롤백)

접속 화면: Ctrl+Enter 또는 Alt+C 접속, Esc 또는 Alt+X 취소
Alt+H/P/U/W/T/N: 호스트/포트/계정/비밀번호/접속 방식/서비스 이름 선택
Alt+R: 저장된 접속 선택, Alt+A: 별칭 입력, Ctrl+S/Alt+S: 접속 정보 저장
비밀번호는 저장하지 않으며 저장된 접속 선택 시 비밀번호 칸을 비웁니다.
설정 화면: Ctrl+S 또는 Alt+S 저장, Esc 또는 Alt+X 취소
Alt+L: 언어 선택, Alt+M: 마우스 항목 선택 (Space로 켜기·끄기)
마우스: 버튼·입력 항목·결과 셀 클릭, 휠 스크롤

편집기 전체 내용을 SQL 한 문장 또는 BEGIN/DECLARE 블록 하나로 실행합니다.
결과는 최대 1,000행입니다. DML은 수동 커밋, DDL은 암시적 커밋이 가능합니다.
Ctrl+Enter를 구분하지 못하는 터미널에서는 F4를 사용하세요.
위/아래 화살표와 PageUp/PageDown으로 도움말을 스크롤할 수 있습니다.`,
}

func (s *screen) showHelp() {
	s.helpFocus = s.app.GetFocus()
	index := 0
	if s.prefs.Language == "ko" {
		index = 1
	}
	help := tview.NewTextView().SetText(shortcutHelp[index]).SetWrap(true).SetWordWrap(true).SetScrollable(true)
	help.SetBorder(true).SetTitle(s.tr("help_title"))
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
