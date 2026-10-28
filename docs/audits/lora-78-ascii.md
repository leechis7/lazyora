# LORA-78 문자 표시 검증 / Character display validation

2026-10-28 · `feature/LORA-78-ascii-display` · 미커밋·미배포 / uncommitted, unreleased

## 한국어

- 공개 Tuivory Go v0.1.0(태그 go/v0.1.0, 커밋 c034a00b1db92ee7ad4abd1ea5956907f1c1a124)을 유지하며 트리 ASCII 옵션을 실제 표시 모드에 연결했습니다. 공통 소스 복사나 로컬 replace는 없습니다.
- 설정의 문자 표시(Alt+D)에 자동/Unicode/ASCII를 추가했습니다. 저장 즉시 현재/다른 접속 작업공간에 적용하고 저장 실패·취소 시 기존 값을 유지합니다. CLI `--display-mode`는 한 번의 실행에만 우선하며 저장된 모드나 한국어 선호를 덮어쓰지 않습니다.
- 자동은 출력 문자셋과 tcell의 필수 문자 출력 추정치를 사용합니다. TERM·색상·마우스 지원과 문자 지원을 구분하며 실제 폰트 지원은 감지하지 못합니다. ASCII 또는 한글 출력 불가 시 영어 UI를 사용하되 언어 선호는 보존합니다.
- displayLayout/asciiScreen은 그리는 문자만 바꾸며 비 ASCII 데이터의 대체 문자는 ?와 원래 셀 폭으로 표시합니다. SQL·객체 식별·DB 값·앱 복사/내보내기·설정 원문은 변환하지 않습니다. 데모로 시작한 뒤 실제 접속한 결과가 설정 저장 시 데모 문구로 바뀌지 않도록 기존 표시 조건도 보완했습니다.
- 새 전용 테스트 7개(설정 1·UI 6): 과거 설정 호환·유효성·저장, TERM 4종×색상 4종 분리, 일부 문자/한글 불가, 강제/자동 우선순위, 12가지 테마/색상×12종 화면의 ASCII 셀 검사, 한글/이모지/결합문자·복사 보존, 설정 취소/실패와 작업공간 동기화, 넓은 문자 뒤의 셀 위치, 잘못된 CLI와 데모 이후 실제 결과 보호를 확인했습니다.
- 기존 전체 UI 검사 통과. 최종 `go test -race ./...` 통과(UI 패키지 61.525초: 테스트 실행 시간이며 제품 성능 수치가 아님). `go vet ./...`, Linux ARM64 `make build` 통과. CLI --help/--version과 바이너리 모듈 정보 확인.
- 실제 로컬 PTY의 --demo 시작/종료 3종 통과: UTF-8에서 강제 ASCII(출력 20853바이트, ASCII 전용), TERM 빈 값·locale C에서 자동 ASCII(21455바이트, ASCII 전용), TERM=ansi·UTF-8에서 자동 Unicode(32473바이트, Unicode 인코딩 포함). 모두 Ctrl+Q 확인 종료 코드 0, 설정 파일 바이트 불변. 이 수치는 해당 시험 출력량으로 성능 비교가 아닙니다.
- 한영 문서 로컬 링크와 각 내장 가이드 8개 목차 검사를 통과했습니다. 실제 원격 폰트·SecureCRT/MobaXterm·CentOS와 신규 실DB 연결은 별도 검증 대상입니다. GitHub CI·게시·배포는 미실행이며 정식 v0.10.2 자산과 버전은 변경하지 않았습니다.
- Jira에 검토 전환이 없어 진행 중 상태에서 검토 대기 댓글을 남깁니다. 사용자 검토 완료 전 완료 처리하지 않습니다. 기존 test/oracle/run.sh의 사용자 실행 권한 변경은 보존했습니다.

## English

- Keep published Tuivory Go v0.1.0 (go/v0.1.0, c034a00b1db92ee7ad4abd1ea5956907f1c1a124) and connect the tree ASCII option to the effective mode without copying sources or local replacements.
- Add saved Auto/Unicode/ASCII selection under Settings (Alt+D), immediate workspace-wide application, cancel/failure preservation and a one-run --display-mode override. Preserve saved Korean and mode preferences independently of effective output.
- Auto uses encoding/terminfo estimates for required glyphs, independently of TERM/color/mouse. Actual font coverage cannot be detected. Use English UI for ASCII or unavailable Korean encoding without changing preferences.
- Only rendering is substituted, with ? placeholders preserving terminal-cell positions. Original SQL, identities, database values and application copy/export stay intact. Guard real results after a demo-launched session from being replaced with demo labels when saving settings.
- Seven dedicated tests cover old settings/validation/persistence, 4 TERM values × 4 color counts, partial glyph/Korean limitations, precedence, 12 theme/color profiles × 12 screen types, Unicode/combining/emoji/copy preservation, cancel/failure/workspace sync, wide-cell positions, invalid CLI and real-result preservation.
- Existing UI regression, final full race tests (UI package 61.525 seconds), vet and Linux ARM64 build passed. Test runtime is not a product-performance benchmark. Checked CLI help/version and binary module metadata.
- Three real local PTY --demo runs passed startup and confirmed Ctrl+Q exit (code 0): forced ASCII under UTF-8 (20853 bytes, ASCII-only), Auto under empty TERM/locale C (21455 bytes, ASCII-only), and Auto under TERM=ansi/UTF-8 (32473 bytes, Unicode encoded). Preference files were byte-identical after each run. Byte counts describe those runs, not comparative performance or actual font coverage.
- Local documentation links and eight sections per embedded language guide passed. Real remote fonts, SecureCRT/MobaXterm/CentOS and fresh live-DB connections remain unverified. No remote CI/publication/release; existing v0.10.2 assets/version are untouched. Leave Jira In Progress with a review-pending comment until Review exists and the user approves completion. Preserve the user’s run.sh mode change.
