# 터미널 키와 복사

메인 SQL 실행은 F9, Ctrl+Enter, Alt+Enter입니다. 일반 Enter는 줄바꿈입니다. 접속 화면에서도 Ctrl+Enter·Alt+Enter·Alt+C로 접속할 수 있습니다.

## SecureCRT에서 Ctrl+Enter가 줄바꿈으로 들어올 때

서버 OS보다 접속 프로그램이 보내는 키 바이트가 중요합니다. 일반 Enter와 Ctrl+Enter가 모두 CR(`0d`)이면 앱이 두 키를 구분할 수 없습니다. `TERM=xterm`만 설정해도 수정 키 보고 방식이 달라지는 것은 아닙니다.

사용하는 세션의 Options → Session Options → Terminal → Emulation → Mapped Keys에서 Map a Key로 Ctrl+Enter를 선택하고, Send String에 다음을 지정합니다. 기본 세션 전체를 덮어쓰지 말고 시험 중인 세션에 적용합니다.

```text
\e[13;5u
```

이 문자열은 ESC `[13;5u`(CSI-u Ctrl+Enter)를 보냅니다. 드라이버는 xterm modifyOtherKeys 형식 `\e[27;5;13~`도 지원합니다. 실제 앱에서 일반 Enter 줄바꿈과 두 Ctrl+Enter 형식, Alt+Enter를 `make -C test/oracle tui-test`로 검증합니다. 직접 시험할 때 세션 → 접속과 메인 SQL 실행을 각각 확인하세요. 매핑을 사용할 수 없으면 F9 또는 Alt+Enter를 사용합니다.

근거: [VanDyke 키 매핑 안내](https://www.vandyke.com/support/tips/map-keyboard-shortcut-to-perform-menu-toolbar-function-securecrt.html), [Send String 이스케이프 표기](https://www.vandyke.com/support/tips/literal-backslash-character-in-send-string-function.html), [키보드 프로토콜](https://sw.kovidgoyal.net/kitty/keyboard-protocol/). 설치된 tcell 입력 처리 코드에서도 위 두 형식을 확인했습니다. CentOS·SecureCRT 실기에서 매핑 적용 후의 추가 확인은 필요합니다.

## Linux·구형 터미널의 Shift+기능키

`TERM=linux` 등에서 Shift+F2가 `Esc[26~`로 전달되면 키 라이브러리가 F14로 해석합니다.
LazyOra는 이 입력도 Shift+F2로 처리하여 결과 전체화면·복원을 지원합니다.
xterm 형식 `Esc[1;2Q`·`Esc[12;2~`도 동일하게 동작하며 일반 F2는 SQL 전체화면으로 유지합니다.
같은 방식으로 수정 키 없는 F13~F24를 Shift+F1~F12로 처리합니다.
Shift+F3 이전 찾기, Shift+F9 SQL 실행, Shift+F10 문맥/소스 편집 메뉴도 이 경로를 지원합니다.
배정되지 않은 Shift 조합에 새 기능을 추가하지는 않습니다.

## MobaXterm에서 Shift+F2가 F12로 전달될 때

일부 MobaXterm 설정에서는 Shift+F2가 `Esc[24~`(일반 F12)로 전달됩니다.
LazyOra는 결과 전체화면·복원에 Shift+F2와 F12를 함께 배정하므로 두 입력 모두 사용할 수 있습니다.
실제 F12도 같은 동작을 수행합니다. 클라이언트가 보내는 형식은 터미널 설정마다 다르며
서버의 `TERM` 값만 바꿔 수정 키 정보를 복원할 수는 없습니다.

## 8색 시험과 마우스

Unix에서 `TERM`이 없거나 빈 값이면 LazyOra가 앱 내부에서 `ansi`를 기본 사용합니다.
직접 지정한 `TERM`과 셸 환경변수는 바꾸지 않으며 Windows는 기본 콘솔 선택 방식을 유지합니다.

색상 수와 마우스 프로토콜은 별개입니다. `TERM=ansi`는 마우스 기능을 선언하지 않아
마우스 설정을 켜도 클릭을 받을 수 없습니다. xterm 호환 터미널에서 8색과 마우스를 함께
시험하려면 `TERM=xterm TCELL_TRUECOLOR=disable ./bin/lazyora-linux-arm64`로 실행하세요.
SecureCRT를 사용한다면 세션의 Terminal → Emulation도 Xterm으로 맞춥니다.
실제 터미널이 지원하는 종류를 지정해야 하며 서버의 TERM 값만 바꾸어 기능이 추가되지는 않습니다.
도움말 → 환경 진단에서 `TERM`, `Colors`, `Terminal mouse support`와 앱의 `Mouse` 설정을 확인할 수 있습니다.

## 결과 복사

결과에 포커스를 둔 상태에서 Ctrl+C는 현재 셀 또는 선택 범위, Alt+Y는 현재 행 전체를 복사합니다. Shift+방향키 또는 마우스로 셀 사각형 범위를 선택하고 Esc로 범위를 해제합니다. 범위를 드래그한 뒤 Ctrl+C를 누릅니다. 복사 대상은 생략·색상 태그 처리 전의 원본 값입니다. 셀은 탭, 행은 줄바꿈으로 구분하며 값 안의 탭·줄바꿈·따옴표는 TSV 인용 규칙으로 보존합니다. NULL은 표시값 `NULL`로 복사합니다.

클립보드에는 tcell의 OSC 52 경로로 전송 요청합니다. 터미널 정책·중계 프로그램에 따라 차단되거나 크기가 제한될 수 있습니다. 전송 성공 응답이 없어 UI는 PC 클립보드에 저장됐다고 단정하지 않습니다. 비밀번호나 세션 전체를 자동으로 복사하지 않으며 사용자가 선택한 SQL·결과만 전송합니다.

미지원 환경에서는 복사한 다음 **편집 → 복사 내용 보기**를 엽니다. 이 화면에서는 마우스 보고를 잠시 꺼 터미널 자체 텍스트 선택을 사용할 수 있습니다. 방향키·PgUp/PgDn으로 이동하고 SecureCRT의 Ctrl+Shift+C 등 터미널 복사를 사용합니다. Esc로 닫으면 기존 마우스 설정과 포커스로 돌아옵니다. 긴 행은 가로 스크롤해 선택하거나 TSV를 나눠 복사하세요.

근거: [tcell 클립보드 API](https://pkg.go.dev/github.com/gdamore/tcell/v2#Screen), [SecureCRT 복사 키](https://www.vandyke.com/support/tips/nl102004.html). Windows 콘솔의 클립보드 전달 지원은 선택된 tcell 백엔드에 따라 달라지며, 전달되지 않으면 복사 내용 보기에서 터미널 기본 복사를 사용합니다.
