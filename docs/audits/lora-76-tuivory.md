# LORA-76 공통 UI 적용 검토 / Shared UI adoption review

2026-10-28 · `feature/LORA-76-tuivory-common-ui` · 미커밋·미배포 / uncommitted, unreleased

## 출처 / Source

공개 `github.com/leechis7/tuivory/go v0.2.0`, 태그 `go/v0.2.0`, 커밋 `987d1a6786033a6bf063235b5e7fea7b565f0af0`. Go 모듈 Origin과 릴리스 메타데이터가 일치합니다. Sum `h1:q+NJW7/inCfX1cbGLIlEihF8mo/i86cGHozvl9VL6Gw=`. GoModSum `h1:d/qS2dVqqaFo/t/ktK9lq+GScrqjbQIK86IMqYEC+vE=`. 공개 Go 모듈을 import하며 upstream 변경·복사·로컬 replace는 없습니다.

Public module Origin agrees with release metadata; version, tag, commit and checksums are listed above. Import the public module; no upstream edits, copied implementations or local replacement.

## 적용 / Adopted

| 항목 / Area | 적용 판단 / Decision |
| --- | --- |
| 팝업 경계 / Popup edges | `WideSafeScreen`으로 공통 팝업 전체를 감쌉니다. 독자 `popupFill`을 제거합니다. Delegate the entire composite to public overlap protection; remove local repair. |
| 폭 제한 라벨 / Clipped labels | `CellText`로 위임하고 공통 `~` 표식을 사용합니다. Delegate clipping; preserve original stored labels. |
| 기존 폼·버튼·목록·표·트리 / Existing controls | 기존 공개 Styler/Choice/Tree API 유지. 색상 값은 바뀌지 않으며 Choice는 새 공통 수리를 내부 사용합니다. Keep public APIs and palettes; Choice adopts upstream repair. |
| ASCII 표시 / ASCII display | Put/SetCell/PutStr 계열도 기존 어댑터에 연결합니다. 겹침 수리는 Tuivory, 원문/언어/표시 정책은 LazyOra 책임입니다. Bridge writing methods; do not duplicate overlap repair. |

## 검토 후 보류 / Reviewed and deferred

| API | 차이와 후속 범위 / Difference and follow-up |
| --- | --- |
| `DimRegion` | 항상 Dim 속성을 사용합니다. 현재 8/16색 배경은 공개 Styles.PopupBackdrop의 제한색 정책을 사용하므로 그 정책은 유지합니다. Keep existing public limited-color backdrop styling instead of forcing dim attributes. |
| `PanelView` / `SplitView` | 기본 내부 여백·단일선·후보 Palette와 좁은 화면 한쪽 숨김이 기존 이중선·동시 SQL/결과·백분율 크기/최대화 복원과 다릅니다. 원칙 없이 일괄 교체하지 않고 LORA-76의 후속 패널/분할 적용으로 유지합니다. Default padding/borders/palette/compact hiding differ; full migration remains follow-up scope of LORA-76. |
| `StatusView` / `Help` | 우선순위 생략/개수 표시는 현재 클릭 버튼·포커스별 키·고정 버전 영역과 상태 연결이 필요합니다. LORA-76 후속 하단 적용 범위입니다. Requires binding existing clickable keys, focus context and fixed version area; remains follow-up scope. |
| `TabView` / `BreadcrumbView` | SQL 탭·접속 문맥/소유권 표시는 LORA-73, 결과/출력은 LORA-74, 객체 상세는 LORA-75에서 다룹니다. Apply product state in their existing issues. |
| `Form` / `FocusStack` / `Menu` / `GridView` | 새 상태 모델에 비동기 접속·검증·암호 저장·확인·SQL 키맵·ROWID 편집·복사/대량 내보내기를 연결해야 합니다. 현재 공개 Styler를 계속 사용하며, 공통 기능 부족은 Tuivory 보완 후 소비합니다. Bind host validation/actions/data safety before replacing models; retain existing public styling in this pass. |

LORA-76 전체의 패널/분할/하단 재구성 완료를 뜻하지 않습니다. 이번 적용은 공통 팝업 중복 제거와 안전한 패키지 갱신이며, 남은 범위가 있는 일감은 완료 처리하지 않습니다.

This pass does not complete all LORA-76 panel/split/footer work. It removes duplicated popup repair and safely updates the public package. Do not mark the broader issue Done while remaining scope exists.

## 검증 범위 / Verification scope

기존 LORA-79의 한영·12프로필·3종 팝업·폭/높이 왕복·1,728 물리 출력 프레임 및 복원 검사를 유지합니다. 단순 버전 갱신 때 ASCII 화면으로 Unicode가 새어 나오는 실패를 재현했고, 새 쓰기 API 어댑터와 양쪽 래퍼 순서 검사로 보완합니다. 전체 UI, ASCII 전 화면, 폼/메뉴/테마/포커스/최대화/분할/복사·원문 보존을 검증합니다. 실제 원격 터미널·폰트·CentOS·신규 실DB 검증은 별도입니다.

Retain the LORA-79 physical-output regression (1,728 frames across dialogs/languages/profiles/resizing and restoration). Reproduced Unicode leakage in ASCII after the initial update; bridge new writing methods and test both wrapper orders. Verify existing UI/ASCII/forms/menus/themes/focus/maximization/resizing/copy/original-data behavior. Fresh remote-terminal/font/CentOS/live-DB verification is separate.

## 실행 결과 / Results

- 전체 UI `go test ./internal/ui -count=1` 16.610초, 전체 `make test` race 검사 통과(UI 185.784초). `make check`, Linux ARM64 `make build`, `--version`, 모듈 정보의 Tuivory v0.2.0 확인 통과. 시간은 검사 소요 시간이며 제품 성능 지표가 아닙니다.
- 실제 로컬 PTY의 `--demo --display-mode unicode|ascii`, Alt+L → Esc → Ctrl+Q 확인 종료 키 시나리오 2종이 종료 코드 0으로 끝났습니다. 임시 한국어/Unicode 설정 파일 바이트는 유지됐고 ASCII 출력은 7비트 문자만 포함했습니다. 이 검사는 실제 원격 폰트의 육안 검사를 대신하지 않습니다.
- 변경 문서 로컬 링크 95개와 `git diff --check` 통과. 공통 `popupFill` 정의는 0개입니다. `test/oracle/run.sh` 사용자 실행 권한 변경은 보존했습니다. 커밋·게시·CI·배포는 하지 않았습니다. 앱 표시 버전은 기존 0.11.1이며 로컬 ARM64 바이너리에만 개발 변경이 반영됐습니다.
- Full UI tests passed (16.610s), as did the full race suite (UI 185.784s), vet, Linux ARM64 build/version and embedded Tuivory v0.2.0 verification. Durations describe tests, not product performance.
- Two local PTY demo key scenarios (Unicode/ASCII, Alt+L → Esc → confirmed Ctrl+Q) exited 0. Temporary Korean/Unicode preferences were byte-identical; ASCII output contained only 7-bit bytes. This is not a visual inspection of a remote terminal/font.
- Verified 95 local documentation links and diff formatting; no local `popupFill` definitions remain. Preserve the user's run.sh mode change. No commit/publication/CI/release; the local ARM64 build contains development changes and still displays 0.11.1.

## 2026-10-29 컴포넌트와 앱 책임 정정 / Component and application boundary correction

사용자가 Tuivory는 TUI 컴포넌트 라이브러리임을 명확히 했습니다. 이전 검토에서 앱의 화면 조합 코드까지 모두 이전 대상으로 분류한 해석은 철회합니다. 아래는 ‘미이관 목록’이 아니라 현재 코드의 책임을 나눈 표입니다. 색상만 재사용한 것과 공통 컨트롤을 재사용한 것은 구분하되, 화면 조합·배치·앱 명령 연결이 호스트에 남는 것은 정상입니다.

| 영역 | Tuivory에 적합한 공통 컴포넌트 | LazyOra에 남는 정상적인 UI·기능 연결 |
| --- | --- | --- |
| 폼·팝업 | 입력/버튼/선택·팝업 표면·내부 포커스/입력 계약·문자 겹침 보호 | 접속/설정 등 실제 화면의 항목·배치·크기·저장/닫기 동작. [form_popup.go](../../internal/ui/form_popup.go), [buttons.go](../../internal/ui/buttons.go)의 조합 전체를 이전 대상으로 보지 않음. |
| 메뉴 | 메뉴·하위 메뉴·구분선·컴포넌트 내 탐색 API | 앱 메뉴 분류·명령/활성 조건·진입 위치와 기능 실행. [menu.go](../../internal/ui/menu.go), [menu_groups.go](../../internal/ui/menu_groups.go). |
| 탭·작업공간 | 탭 표시·넘침·선택/닫기 요청 등 범용 조작 | SQL 문서/세션 소유권·저장/닫기 검증·탭 배치와 이벤트 연결. [tabs.go](../../internal/ui/tabs.go). |
| 패널·분할선 | 패널/분할선의 표시·최소 크기·크기 조절 기능 | 어떤 패널을 어느 위치/크기로 구성하고 최대화/복원 명령과 연결할지. [panel_resize.go](../../internal/ui/panel_resize.go), [layout.go](../../internal/ui/layout.go). |
| 하단 상태/키 | 상태/키 항목의 공통 표시·폭/우선순위·생략 옵션 | 현재 화면의 키·버전·메시지 선택, 앱 단축키와 명령 연결·전체 배치. [toolbar.go](../../internal/ui/toolbar.go). |
| 도움말·상세 | 목록·본문·패널·분할·표 등 재사용 가능한 부품 | 도움말 화면 전체 조합·목차/내용·비교표 데이터·화면 간 이동. [help.go](../../internal/ui/help.go), [reference.go](../../internal/ui/reference.go), [keymap_comparison.go](../../internal/ui/keymap_comparison.go). |
| 검색·자동완성 | 검색 입력·후보 목록·선택 이벤트 등 범용 컨트롤 | 팝업을 여는 시점/배치, SQL 문맥·후보 생성·삽입 동작. [completion.go](../../internal/ui/completion.go). |
| 편집기·표시 모드 | 공통 편집/표시 훅, 문자 폭·겹침·호환 표시 API | SQL lexer·편집 정책·언어/모드 설정·단말 수명주기·필요한 표시 어댑터. [display.go](../../internal/ui/display.go). |
| 데이터 그리드 | [TVR-17](https://leechis7.atlassian.net/browse/TVR-17)의 단일 그리드 옵션/프리셋, 선택/뷰포트/공통 표시 | 열/자료형·원본 매핑·SQL/검증/트랜잭션·실제 복사/파일 I/O. [grid.go](../../internal/ui/grid.go). |

공통 컨트롤을 재구현하지 않는 원칙은 유지합니다. 다만 Tuivory의 컴포넌트 옵션이 필요한 것과 앱의 화면 구성 책임을 분리합니다. 기본 외형 차이는 공통 옵션 보완을 검토하되, 앱 전체 화면 템플릿을 Tuivory가 제공해야 한다고 요구하지 않습니다. Form/FocusStack/Layout 도구도 앱의 포커스 순서나 화면 구조를 결정하지 않습니다.

TVR-7 본문에 이 경계를 한영으로 추가하고, 과도한 전체 UI 이전 요청 댓글(10824)을 정정했습니다. 기존 에픽 요구/배포 이력과 상태는 보존합니다. TVR-17의 단일 간결 그리드 + 옵션/프리셋 요청과 비교 PNG 2장은 유효합니다. 이 검토는 새 컨트롤 구현이나 전체 이관 완료를 뜻하지 않습니다. 현재 미커밋 트리 아이콘 변경과 사용자 run.sh 모드 변경을 보존했습니다.

### English

The user clarified that Tuivory is a TUI component library. Withdraw the earlier classification of all host screen-composition code as migration work. The table distinguishes reusable controls from normal host UI and feature binding; styling reuse and component reuse are separate, but host composition is not a defect.

| Area | Tuivory | LazyOra |
| --- | --- | --- |
| Forms/popups | Reusable fields/buttons/choices, popup surfaces, internal focus/input and overlap protection | Actual connection/settings screen fields/layout/size, save/close decisions and composition. |
| Menus | Generic menus/submenus/separators and internal navigation | App command groups/availability, entry placement and action binding. |
| Tabs/workspaces | Tab display/overflow/selection/close requests | Documents/sessions, save/close guards, placement and event binding. |
| Panels/splits | Component rendering/minimum sizes/resizing | Panel composition, application layout and maximize/restore commands. |
| Footer/status | Common item rendering/width/priority/omission options | Which keys/messages/version to show and global shortcut/action/layout choices. |
| Help/detail viewers | Lists/text/panels/splits/tables | Full help-screen composition, contents/data and navigation policy. |
| Completion/search | Reusable filter/list/selection controls | When/where to open, SQL context/candidates and insertion. |
| Editor/display | Common render/edit hooks, width/overlap/fallback APIs | SQL lexer, edit policy, preferences, terminal lifetime and required display adapters. |
| Grid | One configurable grid with presets (TVR-17) | Columns/types/original mappings, SQL/validation/transactions and real clipboard/file I/O. |

Avoid reimplementing shared controls without transferring entire screens to the library. Distinguish reusable option gaps from host composition. Focus/layout primitives do not dictate app-wide focus order or structure. The epic description and overly broad comment 10824 were corrected in Korean/English, preserving original requirements/release history/status. TVR-17 and both images remain valid. No new component implementation or completion claim; preserve existing leaf-icon and run.sh changes.
