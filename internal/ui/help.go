package ui

import (
	"fmt"

	"github.com/rivo/tview"
	"lazyora/internal/appinfo"
)

var contextHelp = [2]string{
	`Tab / Shift+Tab: cycle SQL, results and visible schemas; maximized panels switch SQL/results.
Enter inserts a newline. Ctrl+Enter and Alt+Enter execute SQL only in the main screen.
F11 maximizes the focused panel while retaining menus/status/footer. Ctrl+F11 shows only that panel fullscreen. Press the same key again to restore split. F7 restores split layout before toggling schemas. Help/menu overlays return to fullscreen when closed.
Results: click a cell; drag a rectangle or Shift+arrows to select; Ctrl+C copies cell/range, Alt+Y copies a row.
Copied data is TSV; tabs/newlines/quotes inside values are quoted. NULL is copied as displayed.
Copy uses OSC 52. Terminal permissions can block it; Edit > Copy preview disables mouse reporting for native terminal selection.
SecureCRT Ctrl+Enter mapping: Session Options > Terminal > Emulation > Mapped Keys > Send String: \e[13;5u
If Ctrl+Enter is indistinguishable from Enter, use Alt+Enter or F9. See docs/terminal.md.

Connection: upper profile picker, lower editable fields starting with Alias.
Alt+R profiles; Alt+A alias; Alt+H/P/U/W/T/N host/port/user/password/type/service.
Ctrl+Enter / Alt+Enter / Alt+C connect; Ctrl+S / Alt+S save; Esc / Alt+X cancel.
White fields accept text; dark fields with v select options. Profiles never store passwords.
Settings: Alt+L language, Alt+M mouse checkbox (Space toggles); Ctrl+S saves, Esc cancels.

Schema browser: arrows navigate/collapse/expand; Enter/click opens details; Ctrl+R refreshes.
Object details: Alt+S source/edit, Alt+A arguments, Ctrl+R refresh, Esc close.
Source: Ctrl+F9 applies/compiles; Ctrl+R reloads; Tab switches diagnostics; Esc closes.
DDL compilation requires confirmation and can commit prior DML. Unapplied edits require discard confirmation.
Exit always asks. Pending changes allow commit/rollback and exit or keep working (default).

The whole SQL editor is executed as one statement or BEGIN/DECLARE block. Script execution (F5) is unavailable.
At most 1,000 result rows. DML needs manual commit; DDL can commit implicitly.`,
	`Tab / Shift+Tab: 쿼리·결과·표시 중인 스키마 순환. 전체화면에서는 쿼리/결과 전환.
Enter는 줄바꿈입니다. Ctrl+Enter·Alt+Enter는 메인 화면에서 SQL을 실행합니다.
F11: 메뉴·상태·푸터를 유지하는 창 최대화. Ctrl+F11: 선택한 창만 터미널 전체 표시. 같은 키를 다시 누르면 분할 화면으로 복원합니다. F7은 분할 화면 복원 후 스키마를 토글합니다. 도움말·메뉴를 닫으면 전체화면으로 돌아갑니다.
결과: 셀 클릭, 마우스 드래그/Shift+방향키 범위 선택. Ctrl+C 셀·범위 복사, Alt+Y 한 행 복사.
복사 형식은 TSV이며 값 안의 탭·줄바꿈·따옴표는 인용 처리합니다. NULL은 화면 표시대로 복사합니다.
OSC 52 전송은 터미널 설정에 따라 차단될 수 있습니다. 편집 > 복사 내용 보기는 마우스 보고를 꺼 터미널 기본 선택·복사를 지원합니다.
SecureCRT Ctrl+Enter 매핑: 세션 옵션 > Terminal > Emulation > Mapped Keys > Send String: \e[13;5u
일반 Enter와 구분되지 않으면 Alt+Enter 또는 F9를 사용하세요. docs/terminal.md 참고.

접속: 위쪽은 저장된 접속 선택, 아래쪽은 별칭부터 시작하는 정보 편집입니다.
Alt+R 저장된 접속, Alt+A 별칭, Alt+H/P/U/W/T/N 호스트/포트/계정/암호/방식/서비스.
Ctrl+Enter / Alt+Enter / Alt+C 접속, Ctrl+S / Alt+S 저장, Esc / Alt+X 취소.
흰 배경은 직접 입력, 어두운 배경과 v 표시는 선택박스입니다. 비밀번호는 저장하지 않습니다.
설정: Alt+L 언어, Alt+M 마우스 항목(Space 토글), Ctrl+S 저장, Esc 취소.

스키마: 방향키 이동·접기·펼치기, Enter/클릭 상세 조회, Ctrl+R 새로고침.
객체 상세: Alt+S 소스·편집, Alt+A 인자, Ctrl+R 새로고침, Esc 닫기.
소스: Ctrl+F9 적용·컴파일, Ctrl+R 새로고침, Tab 오류 목록 이동, Esc 닫기.
컴파일 DDL은 이전 DML을 커밋할 수 있어 확인합니다. 미적용 편집을 버릴 때도 확인합니다.
종료는 항상 확인하며, 변경 가능 내역이 있으면 커밋/롤백 후 종료 또는 계속 사용(기본)을 선택합니다.

SQL 편집기 전체를 한 문장 또는 BEGIN/DECLARE 블록으로 실행합니다. 스크립트 실행(F5)은 아직 지원하지 않습니다.
최대 1,000행을 표시합니다. DML은 수동 커밋, DDL은 암시적으로 커밋할 수 있습니다.`}

func (s *screen) showHelp() {
	s.helpFocus = s.app.GetFocus()
	index := 0
	if s.prefs.Language == "ko" {
		index = 1
	}
	info := fmt.Sprintf("%s v%s\n%s: %s\n%s: %s\n%s: GNU AGPL v3 (%s)\n%s\nCopyright (C) 2026 Chaekang Lee\n\n",
		appinfo.Name, appinfo.Version, s.tr("developer"), appinfo.Developer,
		s.tr("email"), appinfo.Email, s.tr("license"), appinfo.License, appinfo.LicenseURL)
	help := tview.NewTextView().SetText(info + s.mainHelp() + "\n" + contextHelp[index]).SetWrap(true).SetWordWrap(true).SetScrollable(true)
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
