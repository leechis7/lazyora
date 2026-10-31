# LORA-76: Tuivory v0.5.0 우선 적용 / Tuivory-first integration

[한국어](#한국어) | [English](#english) · [한국어 적용 기록](../tuivory-sync.md) · [English integration](../en/tuivory-sync.md)

## 한국어

2026-10-31 사용자 요청으로 tview 직접 UI 처리를 줄입니다. 공개 Go `github.com/leechis7/tuivory/go v0.5.0`, 태그 `go/v0.5.0`, Origin `26da505af4b818bb8c3c272e5970c411d9bde92b`, Sum `h1:KjY4+GAHeY6SULi6++Gv3TaUUS/jTZ2UEpgDsRGmKxo=`를 확인했습니다. 소스 복사·수정·로컬 replace 없이 공개 패키지를 소비합니다. 기존 tview SQL 편집 확장용 replace는 별도 아래 요청에 해당합니다.

### 최종 기준 정정: 직접 tview import 0개

사용자가 목표를 명확히 했습니다: LazyOra는 tview를 직접 import하지 않고 Tuivory가 UI 전체를 감싸야 합니다. 배치 계산/컨테이너도 SDK로 옮기며 앱에는 화면 선언·데이터/명령 연결만 둡니다. **[TVR-25](https://leechis7.atlassian.net/browse/TVR-25)**(배치 컨테이너), **[TVR-26](https://leechis7.atlassian.net/browse/TVR-26)**(언어 중립 UI facade(Go·Rust 우선))를 요청했습니다. SDK 내부 타입 별칭·생성자 forwarder는 허용하고, 앱 안의 로컬 tview shim으로 import를 숨기는 전환은 하지 않습니다. 앞선 요청의 과도한 ‘별칭 불허’ 문구도 이 기준에 맞춰 정정했습니다.

현재 직접 import는 first-party Go 114파일(프로덕션 53·테스트 61)에 남아 있습니다. v0.5.0은 필요한 public Primitive/모든 생성자·컨테이너/편집기 facade를 아직 제공하지 않아 전체 제거 완료로 볼 수 없습니다. 테스트까지 포함해 import가 0개가 되고 UI 동작/호환 검사가 통과해야 최종 목표 달성입니다. 간접 backend 의존성은 허용하며 소비자의 SQL 편집용 tview replace 제거 경로는 TVR-24/26에 포함합니다.

### 이번 전환

- 공통 폼 팝업의 직접 셀 순회·카드 채우기·검은 그림자 그리기·추가 WideSafeScreen 래퍼를 제거했습니다. 공개 `ShadowView`가 그림자·자식 경계 보호·카드 좌표·입력/붙여넣기를 맡고, 배경은 `DimRegion`으로 감쇠합니다. 배경의 원래 색은 유지하며 Dim만 적용합니다.
- 기존 카드 크기/위치를 유지하기 위해 폼 팝업에는 `ShadowAuto`, 오른쪽/아래 **1칸/1행**을 명시합니다. 공간이 없는 축은 줄이고 양쪽 모두 없으면 Off입니다. SDK의 기본 Off 및 Auto 기본 2칸/1행을 바꾸지 않습니다. 기존 메인 메뉴에는 그림자를 새로 켜지 않습니다.
- 소스 편집 명령의 native `tview.List`를 `Menu`/`MenuView`/`PanelView` 팝업으로 전환했습니다. 원래 편집기·실행 명령·클립보드·키맵 연결은 앱에 유지합니다. 공개 `StyleChildren` 계약으로 래퍼 안의 폼 버튼/열린 Choice를 탐색합니다.
- Deprecated 소비를 정리했습니다: `FieldStyle()` → `TextStyle(Field, Body)`, `Colors.SelectionText` → `Styles.SelectionFG`. PopupCardStyle/PopupShadowStyle/PopupBackdrop 직접 소비는 팝업 전환과 함께 없어졌습니다. `FooterButtonStyle`, `Styler.Table`, `Grid.Click`은 이미 직접 호출하지 않았으며 각각 TextStyle/TableOn/Pointer 계열을 사용합니다.

### 잔여 UI 분류와 요청

| 영역 | 공개 API와 현재 처리 | 적용 판단 |
| --- | --- | --- |
| 메인 메뉴·스키마 트리·결과 그리드 | Menu/MenuView, TreeView, GridView를 실제 렌더링에 사용 | 유지·v0.5.0 회귀 검증 |
| 소스 편집 명령 메뉴·공통 팝업 | MenuView/PanelView, ShadowView/DimRegion | 이번에 전환 |
| SQL 탭·경로 | Tabs/TabView/BreadcrumbView 제공 | API 부재가 아님. LORA-73의 문서/세션 상태 연결로 순차 적용 |
| 결과/객체 상세의 목록·표·전환 | ListModel/ListView, Grid, TabView 제공 | API 부재가 아님. 제품 원본 매핑·상세/선택·읽기 정책과 연결할 어댑터 작업 |
| 패널·분할선·하단 안내 | PanelView/SplitView/StatusView 제공 | API 부재가 아님. LORA-76의 남은 크기/최대화 복원·버튼/포커스/폭 우선순위를 연결해야 함 |
| 테두리 없는 불투명 카드 | ShadowView는 그림자만 담당하고 PanelView.Draw는 Border=false도 테두리를 그림 | **[TVR-22](https://leechis7.atlassian.net/browse/TVR-22)** 요청. 현재는 표준 tview Flex.Box를 불투명 Box로 둔 최소 배치 어댑터 유지. 독자 픽셀 렌더러는 만들지 않음 |
| 보안 입력·독립 버튼·제품용 폼 | 기본 Form/FormView와 Styler/Choice는 제공. 마스킹·선택/클립보드·독립 버튼 API는 부족 | **[TVR-23](https://leechis7.atlassian.net/browse/TVR-23)** 요청. TVR-9의 기본 폼 완료 범위는 존중하며 별도 확장 요구. 암호를 일반 text로 대체하지 않음 |
| SQL/소스 편집·긴 도움말/DDL/로그 | Form textarea/RowsView로 전체 편집·선택/되돌리기·스크롤 원문 뷰를 대체할 수 없음 | **[TVR-24](https://leechis7.atlassian.net/browse/TVR-24)** 요청. 공개 Editor/Buffer/TextView 제공 뒤 기존 TextArea 확장/뷰 제거 |
| Application/Pages/Flex·화면 조합 | 단말 I/O와 화면 선언/업무 연결은 앱, 배치 엔진·컨테이너·일반 입력 전달은 SDK | TVR-25/26 공개 후 직접 tview 사용 제거. 어떤 화면을 구성할지는 앱이 선언 |
| Table/TreeNode 상태·Oracle 객체 참조 | 기존 데이터/선택 호환 어댑터가 native 타입을 보관하는 경우도 있음 | 타입 보관과 실제 tview 렌더링을 구분. SDK에서 공개한 호환 타입은 허용. 앱 내부에만 별칭을 만들어 직접 backend import를 남기지 않음 |

요청 3개는 Tuivory 프로젝트에 백로그로 등록했으며 여기서 upstream 구현을 하지 않습니다. 준비된 공통 API의 앱 연결과 아직 없는 기능 요청은 별개입니다. v0.5.0 갱신만으로 모든 native UI 사용이 사라진 것은 아닙니다. 후속 컴포넌트가 공개되면 이 최소 어댑터를 걷어내고 회귀 검증합니다.

### 검증 기록

- 새 12팔레트 검사는 ShadowView 카드/그림자 좌표, 좁은 화면 왕복, 그림자 클릭 차단과 한글 붙여넣기 전달·SQL 원문 보존을 확인합니다. 기존 한영/12팔레트 팝업 물리 Show 출력·한글/CJK/이모지/결합 문자·테마·드롭다운·배경 복원 회귀를 유지합니다.
- 최초 전체 UI 검사는 래퍼를 통과하지 못하는 기존 버튼 탐색 때문에 실패했고, 확인창 선택이 그대로여서 작업 채널을 기다리는 테스트를 진단용 SIGQUIT로 종료했습니다. SDK deadlock으로 단정하지 않았습니다. 공개 StyleChildren을 통한 탐색으로 수정한 뒤 버튼/확인창 검사 0.030초, 전체 UI 14.989초를 통과했습니다. 중단한 실행은 통과로 계산하지 않습니다.
- 최종 동시성·정적 분석·ARM64·실제 PTY 결과는 아래에 추가합니다. 새 실DB·원격 폰트/SSH·CentOS·다른 OS 실제 UI 시험은 이 범위가 아닙니다.

- 최종 전체 race 통과(UI 188.771초), make check와 Linux ARM64 make build/--version 및 Tuivory v0.5.0 내장을 확인했습니다. 실행파일 표시 버전은 v0.12.0입니다.
- 실제 격리 PTY 3종(한글/영어 Unicode·영어 ASCII)에서 팝업 크기 왕복과 종료0/설정 불변을 통과했습니다. 기존 화면/메뉴 각각4회와 새 접속 팝업3회, 총33회 SIGWINCH 뒤 동작을 확인했고 ASCII 접속 팝업은 전체7비트 표시를 검사했습니다. 초기 캡처 스크립트의 오래된 버전 기대값은 v0.12.0으로 수정해 재시험했으며 실패한 시도는 성공으로 계산하지 않았습니다. 새 소스 편집 메뉴는 UI 회귀 검사 범위이며 실DB 소스 편집을 새로 시험한 것은 아닙니다.

정식 앱 버전은 v0.12.0을 유지하며 이 변경은 미배포입니다. 커밋·게시·CI·배포는 아직 수행하지 않았습니다. `test/oracle/run.sh` 사용자 실행권한 변경은 보존합니다. 남은 LORA-76 전체 범위를 완료 처리하지 않습니다.

## English

On 2026-10-31, reduce direct tview UI handling using public Go `github.com/leechis7/tuivory/go v0.5.0`, tag `go/v0.5.0`, Origin `26da505af4b818bb8c3c272e5970c411d9bde92b`, Sum `h1:KjY4+GAHeY6SULi6++Gv3TaUUS/jTZ2UEpgDsRGmKxo=`. No upstream source copy/modification/local replacement. The existing tview editor extension is covered by a separate request below.

### Clarified target: zero direct tview imports

The user explicitly requires Tuivory to wrap the entire UI, including layout execution, with no direct tview imports in LazyOra. Request [TVR-25](https://leechis7.atlassian.net/browse/TVR-25) for layout containers and [TVR-26](https://leechis7.atlassian.net/browse/TVR-26) for the complete language-neutral facade (Go/Rust first). SDK-owned aliases/forwarders are acceptable; a consumer-local tview shim is not completion. Earlier blanket restrictions on SDK aliases were corrected.

Direct imports remain in 114 first-party Go files (53 production, 61 tests). Public v0.5.0 does not yet wrap every necessary Primitive/constructor/container/editor interface, so this is not full migration. Acceptance includes zero imports in tests plus preserved UI behavior. Indirect backend dependencies are allowed; TVR-24/26 includes retiring the app's tview editor replace.

### Adopted

- Replace popup cell scanning/card fill/black shadow drawing and the extra WideSafeScreen wrapper with public ShadowView and DimRegion. Shared components own shadow/child boundary repair/card geometry/input/paste. Preserve underlying colors with Dim rather than repainting them in a host-specific muted palette.
- Preserve existing form-card geometry with explicit Auto right-one/down-one offsets; reduce unavailable axes or turn shadows Off when neither fits. Do not change the SDK's default Off or Auto's default right-two/down-one. Do not newly enable main-menu shadows.
- Replace the source-edit command native List with Menu/MenuView/PanelView in a popup. Host editor actions, keymaps and clipboard remain. Traverse public StyleChildren for native forms/choices nested within SDK wrappers.
- Replace deprecated FieldStyle with TextStyle(Field, Body), and the Colors.SelectionText snapshot with live Styles.SelectionFG. PopupCardStyle/PopupShadowStyle/PopupBackdrop consumption disappears with popup migration. FooterButtonStyle/Styler.Table/Grid.Click were already unused directly; retain TextStyle/TableOn/Pointer paths.

### Remaining UI and requests

| Area | Current public support | Decision |
| --- | --- | --- |
| Main menu/schema tree/results | MenuView/TreeView/GridView actually render them | Retain and regress v0.5.0 |
| Source action menu/common popups | MenuView/PanelView/ShadowView/DimRegion | Adopted now |
| SQL tabs/path | TabView/BreadcrumbView exist | Host integration in LORA-73, not an upstream gap |
| Detail lists/grids/switching | ListView/Grid/TabView exist | Bind host original mappings/detail/selection policies; not an upstream gap |
| Panels/splits/footer | PanelView/SplitView/StatusView exist | Remaining LORA-76 sizing/maximize/focus/click/width-policy adapters |
| Borderless opaque card | ShadowView delegates card draw; PanelView draws borders even with Border=false | **[TVR-22](https://leechis7.atlassian.net/browse/TVR-22)**. Keep only a standard opaque host Flex.Box adapter, not a duplicate renderer |
| Secret input/standalone buttons/product forms | Basic Form/View and Styler/Choice exist; masking/selection/clipboard/standalone buttons are missing | **[TVR-23](https://leechis7.atlassian.net/browse/TVR-23)**, an extension beyond completed TVR-9. Never substitute unmasked ordinary text for secrets |
| SQL/source editor/long help/DDL/log reader | Basic textarea/RowsView do not supply rich editing/selection/undo/scrolling original-text viewers | **[TVR-24](https://leechis7.atlassian.net/browse/TVR-24)**, public Editor/Buffer/TextView before retiring native extensions/views |
| Application/Pages/Flex/screen composition | Host terminal I/O/screen declarations/business bindings; SDK layout execution/containers/common input forwarding | Remove direct backend UI use after TVR-25/26; app still chooses screen structure |
| Native Table/TreeNode metadata | Some remain as data/selection compatibility adapters | Distinguish stored types from actual rendering; SDK-owned compatibility types are valid; do not retain direct backend imports in a consumer-local shim and claim completion |

All three requests are in the separate Tuivory backlog; no upstream implementation here. Available SDK features requiring host binding are distinct from missing APIs. This upgrade does not eliminate every native control. Remove minimal remaining adapters when suitable published APIs arrive.

### Evidence

- New twelve-palette coverage checks card/shadow geometry, narrow-size round trips, shadow-click exclusion, Korean paste forwarding and original SQL preservation. Retain bilingual/twelve-palette physical Show regressions for Korean/CJK/emoji/combining glyphs, themes, dropdowns and backdrop restoration.
- Initial full UI testing failed because old button traversal could not cross the new wrapper. A test waiting for the unchanged confirmation choice was terminated with diagnostic SIGQUIT; do not mislabel it an SDK deadlock or count it as passed. Public StyleChildren traversal fixed the connection; button/confirmation tests passed in 0.030 s and full UI rerun in 14.989 s.
- Final race/vet/ARM64/actual-PTY results follow below. New live Oracle/remote fonts/SSH/CentOS/other-OS UI checks remain outside scope.

- Final full race passed (UI 188.771 s), with vet/default Linux ARM64 build/--version and embedded Tuivory v0.5.0 verified; displayed app version is v0.12.0.
- Three isolated actual PTYs (Korean/English Unicode, English ASCII) passed popup resizing, exit 0 and unchanged settings: four resizes each for main/menu plus three for the connection popup, 33 SIGWINCH events total. ASCII popup output was seven-bit. Corrected a stale capture-script version expectation to v0.12.0 before rerunning; the failed attempt was not counted as passed. New source-action menus are covered by UI regressions, not a new live-DB source-edit run.

Displayed app version stays v0.12.0; these changes are unreleased/uncommitted/unpublished with no remote CI. Preserve the user's run.sh executable-mode change and leave broader LORA-76 scope open.

### 언어 범위 / Language scope

Tuivory는 현재 Go/Rust를 지원하며 향후 Python·JavaScript·C·C++도 지원해야 합니다. 공통 UI·배치·이벤트·문자 좌표 계약은 언어 중립적으로 설계하고, 언어별 backend/호환 API를 구분합니다. TVR-22~26을 이 기준으로 갱신했습니다. LazyOra의 Go 모듈/직접 tview import 사례는 소비자 검증이며 제품 전체를 Go로 제한하지 않습니다.

Tuivory supports Go/Rust today and must accommodate Python/JavaScript/C/C++ later. Shared UI/layout/event/text-coordinate contracts are language-neutral with language-specific backends/adapters. TVR-22–26 reflect this. LazyOra's Go-module/import-zero fixture is a consumer example, not the product's language boundary.
