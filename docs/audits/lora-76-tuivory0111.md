# LORA-76 · Tuivory v0.11.1 공통 셀 편집 검증 / Shared cell-edit integration

**한국어(기본)** · English below · [적용 기록](../tuivory-sync.md) · [English integration](../en/tuivory-sync.md)

## 한국어

공개 `go/v0.11.1`, Origin `aacaa369a830ad22fe5d5bed7a1e4c25830771c5`, Sum `h1:g7Rp4Rlo039vVdrfJ1Xl3s7/C4HK4jB6HXrOyN9y5to=`를 현재 작업 브랜치에 적용했습니다. 정식 LazyOra 릴리스가 아닌 미커밋 로컬 검토 결과입니다.

- `NewGridView`가 셀 편집 그리기·문자 폭·클리핑·실제 커서를 담당합니다. 공개 Grid 세션 API가 버퍼·입력·마우스·승인 대기·재시도를 담당합니다. 자체 InputField 렌더러/MouseHandler/입력 페이지를 제거했습니다.
- LazyOra는 Oracle 대상/ROWID/원본 컬럼과 편집 자격·초깃값을 공급합니다. 제출 요청은 변경 확인 전에는 UPDATE를 보내지 않고 취소하면 같은 SDK 초안으로 돌아갑니다. 실제 SQL·트랜잭션과 Alt+D 별도 편집창은 제품에 남습니다.
- `PreserveCursorTty`로 backend 커서 모양/깜빡임 강제 명령을 차단합니다. 독자 DECSCUSR 필터는 제거하고 TERM 미지정 ANSI 선택과 별도 Win32 키 보고 호환 규칙은 유지했습니다.
- 변경 셀과 독립 행 표식은 기존 공개 상태/스타일과 `SetRowEdited`를 사용합니다. SDK의 다른 신규 업무 기능은 이번에 제품 명령으로 노출하지 않습니다.

### 결과

| 검증 | 결과 |
| --- | --- |
| 전체 `go test ./...` | 성공, UI 25.650초 |
| 관련 편집/터미널/표식 `-race` | 성공, 68.936초 |
| 최종 클릭/포커스 보호 `TestSharedInline` 추가 `-race` | 성공, 1.704초 |
| `go vet ./...` / `make build` | 성공, Linux ARM64 실행파일의 SDK v0.11.1 확인 |
| 실제 앱 Enter/더블클릭 커서 | 96조합: 한영 ×3테마 ×4색상수 ×2문자모드 ×2진입방식; 120×36→80×24→40×16→120×36, 커서 위치/가시성·타이핑·Esc·Alt+D 확인 |
| 변경 셀/행 표식 | 24조합: 셀/행 표식 분리·무관 셀 보존·포커스 이동·표식 해제 |
| 승인/초안/오래된 응답 | 대기 중 커서 숨김, 확인 취소 후 복원, NULL 후 타이핑, 실제 `|`·CJK·이모지·결합 문자·긴 값, 선택 복사, 바깥 클릭 차단, SQL 편집기로 커서 복귀, 소스 교체 뒤 승인 차단 |
| 기존 편집 보호 | 정렬/원본 컬럼·ROWID 투영, 미연결/스냅샷 없음/읽기전용/결과 불확실/실행 중 보호, 별도창·원문 복사 유지 |
| 독립 SDK 재현 | [코드](evidence/lora-76-tuivory0111-cursor.go.txt) · [출력](evidence/lora-76-tuivory0111-cursor.txt): 실제 커서 `(9,1,true)`, 고정 `|` 없음 |
| 실제 ARM64 PTY | TERM 미지정/빈 값/ansi/xterm 4환경의 데모 시작·종료 확인. DECSCUSR·커서 깜빡임 강제 명령·Win32 보고 활성화 누출 없음 |

PTY는 기존 `test/oracle/tui_terminal_smoke.py`에 제어문자 검사만 추가한 임시 진단으로 실행했습니다. Oracle 연결·실DB UPDATE·물리 SecureCRT·CentOS·Windows/macOS 콘솔의 실제 깜빡임은 이번에 검증하지 않았습니다. 터미널 기본 커서 설정을 보존하는 코드와 제어문자·좌표/가시성을 확인한 범위입니다. 기타 OS 바이너리·GitHub CI·커밋·게시·배포는 수행하지 않았습니다. 기존 SQL 편집기의 `third_party/tview` replace는 별개로 남아 있습니다. 사용자 `test/oracle/run.sh` 실행 모드 변경을 보존했습니다.

## English

Integrated published `go/v0.11.1`, Origin `aacaa369a830ad22fe5d5bed7a1e4c25830771c5`, checksum `h1:g7Rp4Rlo039vVdrfJ1Xl3s7/C4HK4jB6HXrOyN9y5to=` on the current branch. This is an uncommitted local review, not an official LazyOra release.

`NewGridView` owns cell rendering, glyph widths, clipping and the real cursor; public Grid session APIs own buffers, input/pointer handling, pending approval and retry. Remove the local InputField renderer/MouseHandler/input page. LazyOra supplies Oracle identity/eligibility and initial values, requires host review before enqueueing UPDATE, and retains SQL/transactions and Alt+D. Rejection resumes the same SDK draft. Use `PreserveCursorTty` rather than the local DECSCUSR filter, preserving independent TERM fallback and Win32-report rules. Dirty-cell/gutter presentation remains through public state/styles and `SetRowEdited`; other new SDK workbench capabilities are not exposed as product commands.

Validation passed: full `go test ./...` (UI 25.650s), focused editor/terminal/marker race (68.936s), final click/focus-guard race (1.704s), vet and ARM64 build with SDK version inspection. Actual app cursor tests cover 96 language/theme/color/glyph/entry combinations through four sizes, typing/Esc and Alt+D. Another 24 combinations verify independent cell/gutter cues, focus changes and clearing. Approval tests cover hidden pending cursors, rejection recovery, NULL-to-typing, literal pipes/CJK/emoji/combining/long text, selection copying, outside-click isolation, SQL-cursor restoration and stale approval rejection. Existing identity/projection/read-only/busy guards pass.

The [standalone reproducer](evidence/lora-76-tuivory0111-cursor.go.txt) and [output](evidence/lora-76-tuivory0111-cursor.txt) show cursor `(9,1,true)` without a fake pipe. Four actual ARM64 demo PTYs (unset/empty TERM, ansi, xterm), using the existing startup smoke script with temporary escape-sequence assertions, pass startup/confirmed clean exit without forced cursor-shape/blink or Win32-report-enable leakage. No live Oracle UPDATE, physical SecureCRT/CentOS or Windows/macOS console blinking was tested. Evidence covers coordinates/visibility and emitted controls; actual blinking follows terminal preferences. No other-OS binary, remote CI, commit/publication/release. Retain the separate SQL-editor backend replacement and the user's script executable-mode change.
