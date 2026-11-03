# LORA-76 Tuivory v0.10.0 적용 / Integration

[한국어 적용 기록](../tuivory-sync.md) | [English integration record](../en/tuivory-sync.md)

## 한국어

2026-11-02 사용자가 요청한 공개 패키지 갱신입니다. 기존 `feature/LORA-76-tuivory-v080` 브랜치를 유지합니다. 새 SDK를 소비하며 SDK 소스를 복사·수정하거나 공통 UI 우회를 추가하지 않습니다. 이전 [전수 조사](lora-76-ui-runtime-audit.md)는 v0.9.0 시점의 기록이며 아래 재검증으로 해소 여부를 구분합니다.

### 출처와 적용 판단

- 모듈: `github.com/leechis7/tuivory/go v0.10.0`
- 태그: `go/v0.10.0`
- Origin: `1697765b366facfc82120ef4f6c9a18a73437524`
- Sum: `h1:MZKtI82SYZpVhFTj3zRNJc5x3wtoWEn06Lbj9CNGcp8=`
- 기존 공개 루트 import를 유지합니다. 호환 확인창 생성자가 새 반응형 Modal을 반환하므로 기존 앱의 취소·커밋·롤백 결정을 그대로 연결한 상태에서 버튼 줄바꿈을 소비합니다. 별도 앱 라벨 축약/배치 엔진은 추가하지 않았습니다.
- 문자 겹침 보호, 분할선 스타일, 구문 역할 스타일, 비교표 스타일을 같은 공개 API로 소비하므로 SDK의 수정이 적용됩니다.
- 비교표의 새 공통 스타일은 헤더/본문 표면을 통일해 구분선 대비를 확보하고, 현재 키맵 헤더에 강조색·밑줄·별표를 유지합니다. 기존의 ‘모든 헤더는 본문과 배경색이 달라야 한다’는 검사 대신 실제 읽기 가능한 전경/배경, 선언된 색상, 현재 열의 구별과 선택 보존을 검사합니다. 앱에서 기존 배경색을 강제로 덮어쓰지 않습니다. 문자 보존·선택 이동·구분선 대비는 별도 실제 출력 검사로 추가했습니다.
- 새 탭/트리 드래그 API는 게시된 기능이지만, 이번 회귀 개선을 위해 객체 재정렬 업무를 새로 정의하지 않습니다. 새 기능을 적용 완료했다고 표시하지 않습니다.
- 편집기 `third_party/tview` replace 유지 판단은 바뀌지 않습니다. 직접 tview/tcell import 0개와 포크 제거 완료를 혼동하지 않습니다. 이번 패키지 갱신이 Oracle 실행·승인·암호·키맵 정책을 바꾸지 않습니다.

### 이전 결함의 재검증

| 이전 문제 | v0.10.0 결과 |
| --- | --- |
| Table+WideSafeScreen의 한글 소실 | LazyOra 없는 독립 SDK 모듈과 실제 앱 비교표에서 한글 원문 보존. 양쪽 모두 `한글 SQL 전체 선택` 유지 |
| 라이트/8색 분할선이 silver/silver | 공개 스타일이 black/silver. 앱의 가로·세로 분할선이 배경과 구별됨 |
| 40열 재접속 기본 취소 버튼이 화면 밖 | 24개 실패 조합 모두 해소. 취소 라벨 표시와 기본 Enter의 무작업 취소도 검증 |
| 다크/8색 SQL 강조가 거의 단색 | 실제 앱 화면에서 일곱 문법 역할이 색상·속성 조합으로 구분됨. 흑백 테마의 단색 의도는 유지 |
| 비교표 구분선 대비 | 실제 비교표의 한글·선택 이동·재그리기·Sync 후 구분선 전경/배경 구별 |

### 검증

- 이전과 같은 실제 앱 생성자/ForceDraw 기반 53변형 × 2언어 × 3테마 × 4색상 프로필: **1,272개 조합 / 5,088개 크기 상태 전부 통과**.
- Tab/Shift+Tab **192개 조합**, 선택 컨트롤 열기/닫기 **72개 조합 통과**. 세 검사를 함께 실행한 시간 82.227초입니다. 프레임 120개를 별도 경로에 저장해 이전 캡처를 덮어쓰지 않았습니다.
- 일반 회귀 검사 `TestTuivoryRenderedUpgradeRegressions`를 추가했습니다. 24언어/테마/프로필 조합에서 실제 출력의 한글 헤더/본문, 구분선, 취소 버튼·기본 취소, 원문 SQL 보존을 검사합니다. 다크/8색은 분석된 일곱 역할의 실제 출력 스타일도 검사합니다.
- 새 회귀 검사 및 기존 비교표 헤더 검사는 1.949초에 통과했습니다. 전체 검사·빌드·PTY 결과는 아래 최종 검증 항목에 기록합니다.

실제 원격 CentOS/SecureCRT/MobaXterm, Windows/macOS 및 실DB 동작을 새로 확인한 것은 아닙니다. 스냅샷 검사는 시뮬레이션 터미널의 실제 앱 출력이며, 로컬 PTY 검사는 별도로 구분합니다. 현재 앱 버전 표시는 v0.12.1이며 미커밋 개발 바이너리입니다. 커밋·게시·CI·릴리스는 별도 승인 전 수행하지 않습니다.

## English

This is the user-requested published-package upgrade dated 2026-11-02, on the existing `feature/LORA-76-tuivory-v080` branch. Consume `github.com/leechis7/tuivory/go v0.10.0`, tag `go/v0.10.0`, Origin `1697765b366facfc82120ef4f6c9a18a73437524`, checksum `h1:MZKtI82SYZpVhFTj3zRNJc5x3wtoWEn06Lbj9CNGcp8=`. No copied/modified SDK source or duplicate host rendering workaround was added. The [previous audit](lora-76-ui-runtime-audit.md) records the v0.9.0 baseline.

Existing public APIs now consume shared wide-glyph preservation, splitter/syntax/comparison styles and the responsive Modal constructor. Keep host transaction decisions, cancellation and input bindings; do not add host button-shortening/layout machinery. The new comparison design shares header/body surfaces to retain separator contrast, while accent/underline/star distinguish the current keymap. Update the old background-only assertion to verify actual readability, declared colors, current-column distinction and stable selection; add separate physical-output glyph/separator checks. Do not force the old palette locally. Published tab/tree drag APIs are available but no new Oracle-object reordering workflow was introduced. The editor replacement remains; zero direct tview/tcell imports does not mean the fork is retired.

### Revalidation

- Independent SDK-only and actual application output retain the complete Korean Table text with WideSafeScreen; both raw/protected output contain `한글 SQL 전체 선택`.
- Light/8 SplitterStyle now returns black/silver; both application splitters remain visible.
- All 24 prior 40-column reconnect failures are resolved. Verify visible Cancel text and default Enter cancellation without changing transactions or SQL.
- Dark/8 SQL has seven distinct rendered syntax styles through colors/attributes; intentionally monochrome themes remain monochrome.
- Comparison text and separator contrast survive selection changes, redraw, resizing and Sync.

The same 53 variants × two languages × three themes × four profiles passed **all 1,272 combinations / 5,088 resize states**. Focus and choice matrices passed **192 and 72 combinations**; the combined diagnostic command took 82.227s. Save 120 snapshots separately from previous evidence. New normal regression coverage exercises 24 language/theme/profile combinations using physical application output, including Korean text, splitters, separators, responsive Cancel, SQL preservation and seven dark/8 syntax styles. New regression plus existing header tests passed in 1.949s. Final full-suite/build/PTY results are recorded below.

These checks do not newly certify remote CentOS/SecureCRT/MobaXterm, Windows/macOS or live Oracle behavior. Simulated-screen output and local PTY checks are distinguished. The uncommitted development binary still reports LazyOra v0.12.1; no commit, publication, CI or release is implied.

## 최종 검증 / Final verification

- `go vet ./...`: 통과 / passed.
- `make build`: Linux ARM64 빌드 통과. `go version -m`으로 SDK v0.10.0과 체크섬을 확인했습니다. / Linux ARM64 build passed; binary module metadata confirms SDK v0.10.0 and its checksum.
- 실제 PTY: 한국어·영어·영문 ASCII 3모드, 크기 변경 33회, 정상 종료 0, 임시 설정 원문 보존. / Actual PTY: Korean, English and English ASCII modes; 33 resizes, exit 0 and unchanged temporary settings.
- 실제 PTY 구문색: 키워드 `#8db8ff`, 함수 `#73d7ce`. / Actual PTY syntax colors: keyword `#8db8ff`, function `#73d7ce`.
- 한영 적용·개발·스타일 문서와 조사 보고서의 로컬 링크 83개 통과. / All 83 local links in updated integration/development/style/audit documents passed.
- Jira에 검토 상태가 없어 이번 적용은 진행 중에 검토 대기로 기록합니다. 사용자 검토 완료 전 완료로 옮기지 않습니다. / Jira has no Review transition; record this integration as review-pending while keeping In Progress until user approval.

추가 실제 PTY 검증 / Additional actual PTY check: 한국어 `TERM=xterm`, `TCELL_TRUECOLOR=disable`의 8색 모드에서도 크기 변경 11회·메뉴/접속/종료가 통과했습니다. 키워드와 함수가 색상/속성으로 구분되며 종료 0·임시 설정 보존을 확인했습니다. 위 3모드와 합계 4모드·44회 크기 변경입니다. / Korean xterm/8-color mode also passed 11 resizes, menus/connection/exit, distinguishable keyword/function attributes, exit 0 and unchanged preferences; combined coverage is four modes and 44 resizes.

전체 `go test -race ./...` 재확인은 종료 코드 0으로 통과했습니다(UI 265.499초). 첫 실행은 모든 패키지 통과 로그(UI 232.524초)를 남겼으나 세션 종료 코드가 143이어서 성공으로 단정하지 않고 같은 명령의 정상 종료를 재확인했습니다. / The full `go test -race ./...` confirmation passed with exit 0 (UI 265.499s). The initial run printed passing package results (UI 232.524s) but its session returned 143, so the same command's normal completion was explicitly rechecked instead of treating that session as successful.
