# LORA-76: Tuivory 메뉴·그리드 적용 검증

[한국어](#한국어) | [English](#english) · [적용 기록](../tuivory-sync.md) · [English integration](../en/tuivory-sync.md)

## 한국어

2026-10-29 미배포 개발 소스의 검증입니다. 실행파일 표시 버전은 0.11.3이며 정식 v0.11.3 자산을 교체하지 않았습니다. 커밋·GitHub 게시·CI·릴리스는 이번 구현에 포함하지 않습니다.

### 출처와 경계

공개 Go `github.com/leechis7/tuivory/go v0.3.0`, 태그 `go/v0.3.0`, Origin `dabab8b8d9b23e9c641231b5358301e3fbb45431`, 모듈 Sum `h1:yvisKQpR9gB90Hge+K0b8NvJJI38LT6gMnH2YE3VTkQ=`를 확인했습니다. `go.mod`와 실행파일의 모듈 정보를 확인했습니다. Tuivory 소스 수정·복사·로컬 replace는 없고 기존 tview SQL 편집 확장은 유지합니다.

| 범위 | 공통 패키지 | LazyOra 연결 |
| --- | --- | --- |
| 메뉴 | Menu/MenuView/MenuBarView: 표시·비활성·구분선·중첩/뒤로·포인터·스크롤 | 명령/분류/가능 조건·단일선 카드 배치·기존 전역 키·호출 패널 복원 |
| 그리드 | Grid/GridView: data 옵션·표시·행번호·구분선·줄무늬·숫자 정렬·셀/범위/편집 상태·상태줄·포인터/열 너비 | 원본/표시 어댑터·조회 행/원본 컬럼·정렬/필터·DB 편집·확인·커밋/롤백·복사/내보내기 |
| 좌표·출력 | Geometry, CellText, WideSafeScreen | 공통 좌표에 인라인 편집 배치·기존 ASCII 표시 변환 |

레거시 Table은 기존 앱 메타데이터·선택 호환용으로 유지하지만 본체 Draw는 호출하지 않습니다. 모든 결과 본체는 GridView를 통해 그립니다. 실제 데이터를 새 MemoryRows에 전량 복제하지 않고 GridSource가 창 단위로 원문을 제공합니다. 기존 제품 데이터/메타데이터 보관 한도와 서버 전체 결과 내보내기 정책은 바꾸지 않습니다.

### 검증 결과

- 기존 편집·복사·ROWID 숨김/원본 매핑·부분 LOB 차단·정렬/필터·페이징·커밋/롤백 재조회·탭/작업공간·메뉴 포커스 복원·키맵 회귀를 유지했습니다.
- 새 통합 검사 3개: 한글/영어 × 다크/라이트/흑백 × 8/16/256/True Color의 24조합에서 각 3개 폭의 메뉴/그리드 144회 그리기·가시성·원문/트랜잭션 보존, 공통 탐색/헤더 드래그/포인터 캡처와 F10/F12 충돌 방지, 실제 변경 셀 배경과 이웃 셀 보존을 확인했습니다. 기존 메뉴 배경 보존 검사에는 단일선 모서리 확인을 추가했습니다.
- `GOMAXPROCS=2 GOMEMLIMIT=768MiB go test -race ./...` 통과(UI 197.257초). 이후 메뉴 카드·바 입력·제목 폭과 포인터 캡처를 정돈했으며, 관련 최종 race 검사도 통과했습니다(UI 4.646초). 전체 UI 재실행 28.141초 통과와 최종 `make check`, `make build`(Linux ARM64)를 확인했습니다. 검증 시점의 전체/집중 범위를 구분하며 원격 CI 결과로 표현하지 않습니다.
- 1,000행 × 20열, 120×28 모의 터미널에서 선택 이동과 전체 그리기를 포함한 `BenchmarkGridDraw1000Rows`: 163회, 약 7.31 ms/op, 274,439 B/op, 21,140 allocs/op. 로컬 ARM64의 관측값이며 변경 전 대비 개선이나 모든 단말의 성능을 보증하는 수치가 아닙니다.
- 설정과 DB 환경을 격리한 로컬 ARM64 실제 PTY에서 한영 트리 펼침·메뉴·중첩/복귀·결과 범위 복사·셀 상세·접속 팝업·종료를 확인했습니다. 영어 ASCII 모드도 7비트 출력과 같은 조작을 확인했습니다. 모두 종료 0이며 임시 설정 파일은 바뀌지 않았습니다. DB 연결 없는 `--demo`입니다. 초기 시나리오는 스키마 포커스에서 F6 한 번만 눌러 SQL로 이동한 뒤 내용을 바꿨기 때문에 종료 확인이 하나 더 필요했으며, 해당 시도를 성공으로 계산하지 않았습니다. 결과창 포커스까지 명시한 최종 시나리오로 재검증했습니다.
- 한국어·영어 README의 화면은 이 미배포 로컬 실행파일의 실제 PTY 데이터를 렌더링한 이미지로 갱신했습니다. 캡션에 미배포/표시 버전을 구분합니다. 실제 폰트·원격 SSH·CentOS·Windows/macOS·새 Oracle 실DB 접속 검증은 수행하지 않았습니다.

### 표시 차이와 보류

현재 행 표식은 `>`, 공통 폭 생략 표식은 `~`입니다. 기존 원문·TSV 복사·NULL/탭/줄바꿈 표시, 변경 셀 배경·상태줄 `*`는 유지합니다. 행 Edited를 전달하면 모든 셀이 편집색이 되므로 셀 단위만 연결하며 행번호의 `*`는 생략합니다. 헤더 클릭 정렬과 공통 데모의 F5~F12 키는 설치하지 않고 기존 정렬/전역 키를 유지합니다. 새 헤더 구분선 드래그는 앱의 열 너비 상태에 반영됩니다.

TVR-17 옵션/프리셋은 공개 패키지에서 소비했습니다. LORA-76의 패널·분할선·하단 안내와 LORA-73~75의 앱 작업공간은 남아 있으며 전체 일감을 완료 처리하지 않습니다. 기존 `test/oracle/run.sh` 사용자 실행권한 변경은 보존합니다.

## English

This records unreleased development-source validation on 2026-10-29. The displayed executable version remains 0.11.3; released v0.11.3 assets are unchanged. No commit, GitHub publication/CI or release is part of this implementation.

### Source and ownership

Verified public Go module `github.com/leechis7/tuivory/go v0.3.0`, tag `go/v0.3.0`, Origin `dabab8b8d9b23e9c641231b5358301e3fbb45431` and Sum `h1:yvisKQpR9gB90Hge+K0b8NvJJI38LT6gMnH2YE3VTkQ=` in module metadata and the executable. No Tuivory source modification/copy/local replacement; retain the existing tview SQL-editing extension.

| Area | Shared package | LazyOra binding |
| --- | --- | --- |
| Menus | Menu/MenuView/MenuBarView rendering, disabled/separator/nesting/back, pointer navigation and scrolling | Commands/grouping/eligibility, single-border card placement, global keys and caller-panel restoration |
| Grid | Grid/GridView data options, rendering, numbers/dividers/stripes/alignment, cell/range/edited state, status, pointer/width handling | Original/display data, row/column mapping, sorting/filtering, DB edit/confirmation/transactions and clipboard/export |
| Coordinates/output | Geometry, CellText, WideSafeScreen | Inline-editor placement and the existing display-only ASCII adapter |

The legacy Table stores application metadata/selection compatibility but never draws the result body; GridView does. GridSource provides original values by window instead of duplicating the full dataset into MemoryRows. Existing application retention limits and full-server export policy are unchanged.

### Evidence

- Retained edit/copy/hidden-ROWID/original mapping, partial-LOB guards, sorting/filtering/paging, transaction refresh, tabs/workspaces, caller restoration and keymap regressions.
- Three integration tests cover 24 language/theme/color combinations and 144 menu/grid draws over three widths, visibility/original-data/transaction preservation, common navigation/header resize/pointer capture, F10/F12 priority and actual edited-versus-neighbor backgrounds. Existing backdrop tests additionally assert single menu borders.
- Full `GOMAXPROCS=2 GOMEMLIMIT=768MiB go test -race ./...` passed (UI 197.257 s). Later card/bar/title/pointer refinements passed the final targeted race check (UI 4.646 s). Full UI rerun passed in 28.141 s, with final vet (`make check`) and Linux ARM64 `make build`. These are local checks, not remote CI results; full and later focused scopes are distinguished.
- `BenchmarkGridDraw1000Rows`, 1,000 rows × 20 columns in a 120×28 simulation, includes selection movement and drawing: 163 iterations, about 7.31 ms/op, 274,439 B/op and 21,140 allocs/op. A local ARM64 observation, not a before/after improvement claim or terminal-independent guarantee.
- Isolated actual ARM64 PTYs exercised Korean/English tree expansion, menus/nesting/back, result range copy, cell details, connection popup and exit. English ASCII mode also checked 7-bit output. Successful runs exited 0 with unchanged temporary settings; all used DB-free `--demo`. An initial script used only one F6 from schema focus, reached SQL and modified the draft, requiring another exit guard; that attempt was not counted as passed. The corrected scenario explicitly reaches result focus before exercising grid keys.
- Refreshed both README screenshots from actual unreleased PTY data, with development/display-version captions. Real remote SSH/fonts/CentOS/Windows/macOS and new live Oracle connections remain outside this verification.

### Differences and remaining scope

Current-row/clipping markers are `>`/`~`. Preserve original values, TSV, NULL/tab/newline display, edited-cell backgrounds and status `*`; omit row-number `*`, because supplying a row Edited state would color every cell. Keep cell-level edit metadata. Do not install header-click sort or generic demo F5–F12 bindings; preserve existing sort/global commands. Header-divider dragging updates host column widths.

Published TVR-17 options/presets are consumed. Panels/splitters/footer in LORA-76 and application workspaces in LORA-73–75 remain; do not mark the entire issue Done. Preserve the user's `test/oracle/run.sh` executable-mode change.
