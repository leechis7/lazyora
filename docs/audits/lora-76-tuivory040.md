# LORA-76: Tuivory v0.4.0 적용 검증 / Integration evidence

[한국어](#한국어) | [English](#english) · [적용 기록](../tuivory-sync.md) · [English integration](../en/tuivory-sync.md)

## 한국어

2026-10-30, 기존 미커밋 `feature/LORA-76-tuivory-menu-grid`의 v0.3.0 적용 위에서 갱신합니다. 공개 모듈 `github.com/leechis7/tuivory/go v0.4.0`, 태그 `go/v0.4.0`, Origin `fef7a7b2247d1c68ef0fb1469e6a2fc2e1cec9b5`, Sum `h1:rhyWAiaCladiGiRSs/K97VpCyDj4FCma539665KazeU=`를 확인했습니다. Tuivory 소스를 수정/복사/로컬 replace하지 않았습니다. 기존 tview의 SQL 편집 확장용 replace는 유지합니다.

- 메뉴 카드의 별도 Box 수동 그리기·제목 자르기·추가 WideSafeScreen 래퍼를 제거하고 공개 PanelView에 원래 MenuView를 직접 연결합니다. 패키지가 메뉴를 인식해 단일선을 사용하며 제목/배경/경계 보호도 맡습니다. 앱은 배치·크기·여백 0·내용·명령/호출 포커스만 연결합니다. 일반 패널 이중선은 유지합니다.
- 텍스트는 TextOnSurface에 의미 역할과 실제 표면을 전달합니다. 명시적인 필드/선택/편집 배경을 바꾸지 않습니다. GridView의 PreserveBackgrounds와 기존 제품 팔레트 어댑터를 유지합니다. 공개 DisabledStyle의 비활성 색과 공통 메뉴의 접두사 제거·입력 차단을 소비합니다.
- 새 비활성 회색은 기존 모든 텍스트 4.5:1 검사와 충돌했습니다. 테스트를 생략하거나 팔레트를 별도 수정하지 않고 공개 API가 명시한 비활성 목표 2:1을 검사 기준에 구분했습니다. 일반 본문·힌트·경고·SQL·입력/선택 등의 4.5:1 검사는 유지합니다. 실제 12개 팔레트의 회색/Dim 구별과 비활성 클릭 차단도 검사합니다. 8색 Dim의 실제 밝기는 원격 단말별 확인이 필요합니다.
- upstream의 멈춤 수정은 private playground의 BeforeDraw/reflow 큐에 있습니다. LazyOra는 해당 예제 앱을 import/복사하지 않으며, 자체 앱에서 실제 SIGWINCH 크기 왕복 후 키 조작·정상 종료를 별도로 검사합니다.

검증 결과는 아래에 기록합니다. 이전 단계의 근거는 [v0.3.0 기록](lora-76-menu-grid.md)이며 이번 결과와 혼동하지 않습니다. 실행파일 표시 버전은 0.11.3을 유지하고 정식 자산은 바꾸지 않습니다. 커밋·게시·CI·배포는 수행하지 않습니다. 새 실DB·원격 SSH/폰트·CentOS·다른 OS 검증은 범위 밖입니다. 전체 LORA-76의 패널/분할/하단 구성은 여전히 남아 있으며 `test/oracle/run.sh` 사용자 모드 변경은 보존합니다.

### 검증 결과

- 전체 `GOMAXPROCS=2 GOMEMLIMIT=768MiB go test -race ./...` 통과(UI 208.583초). 기존 메뉴·그리드·SQL/키맵·편집·트랜잭션/복사 회귀와 24개 언어/테마/색상 조합을 포함합니다. 새 12팔레트 통합 검사는 실제 비활성 메뉴의 회색/Dim·접두사 제거·클릭/포커스/실행 차단과 5종 표면의 텍스트 배경/원문 보존을 확인합니다.
- `make check`, 기본 Linux ARM64 `make build` 통과. 다운로드한 공개 모듈과 실행파일 내장 의존성이 v0.4.0인 것을 확인했습니다. 실행파일 표시 버전은 `LazyOra v0.11.3`이며 미배포 코드입니다.
- 설정·DB 환경을 격리한 실제 ARM64 PTY 3종(한국어/영어 Unicode, 영어 ASCII)이 정상 종료 0·설정 바이트 불변을 통과했습니다. 각 시나리오에서 일반 화면과 열린 메뉴 각각 40×12→80×24→140×45→128×38로 변경해 총 24회 SIGWINCH 리사이즈 후 메뉴/복귀·범위 복사·셀 상세·접속 팝업·종료가 계속 동작함을 확인했습니다. ASCII 화면/메뉴는 7비트 표시를 검사했습니다. DB 없는 `--demo`이며 실제 Oracle 신규 접속 검증이 아닙니다.
- [한국어 화면](../images/lazyora-overview-ko.png)과 [English screen](../images/lazyora-overview.png)을 실제 PTY 데이터로 갱신하고 메뉴 단일선·긴 한글 제목·회색 비활성 항목을 시각 검토했습니다. 문서 로컬 링크 399개, 내장 가이드 목차 한영 각 8개와 `git diff --check`를 확인했습니다.

## English

On 2026-10-30, upgrade the existing uncommitted v0.3.0 menu/grid work on `feature/LORA-76-tuivory-menu-grid`. Verified public `github.com/leechis7/tuivory/go v0.4.0`, tag `go/v0.4.0`, Origin `fef7a7b2247d1c68ef0fb1469e6a2fc2e1cec9b5`, Sum `h1:rhyWAiaCladiGiRSs/K97VpCyDj4FCma539665KazeU=`. No Tuivory source copy/modification/local replacement; retain the existing tview SQL-editing extension.

- Replace manually drawn menu border Box/title clipping/extra WideSafeScreen layer with public PanelView containing the original MenuView. The package recognizes menus for single borders and owns titles/surfaces/wide-glyph protection. LazyOra supplies placement/size/zero padding/content/actions/caller focus. Ordinary panels keep double borders.
- Feed semantic text roles and actual surfaces to TextOnSurface without changing explicit field/selection/edited backgrounds. Keep GridView PreserveBackgrounds and the product palette adapter. Consume public DisabledStyle plus shared prefix removal and input exclusion.
- New disabled gray conflicted with the previous uniform 4.5:1 test. Do not skip tests or fork the palette: distinguish the public API's 2:1 disabled target while retaining 4.5:1 for body/hint/warning/SQL/input/selection and other active content. Test gray/Dim distinction and disabled-click exclusion in all 12 palettes. Actual eight-color Dim brightness needs terminal-specific verification.
- Upstream's deadlock fix belongs to its private playground BeforeDraw/reflow queue. LazyOra does not import/copy that app; separately exercise actual SIGWINCH resizing, continued keyboard interaction and normal exit in its own application.

Record results below. The [v0.3.0 evidence](lora-76-menu-grid.md) is a previous stage, not this run. Keep displayed version 0.11.3 and published assets unchanged. No commit/publication/CI/release. New live DB/remote SSH/fonts/CentOS/other OS runs are outside scope. Broader LORA-76 panel/split/footer work remains; preserve the user's run.sh mode change.

### Validation results

- Full `GOMAXPROCS=2 GOMEMLIMIT=768MiB go test -race ./...` passed (UI 208.583 s), including existing menu/grid/SQL/keymap/edit/transaction/copy regressions and 24 language/theme/color combinations. The new 12-palette integration test checks actual disabled gray/Dim, no ordinary prefix, click/focus/activation exclusion and original text/backgrounds across five surface roles.
- `make check` and default Linux ARM64 `make build` passed. Verified downloaded public module and executable dependency v0.4.0. Displayed `LazyOra v0.11.3` still denotes unreleased source changes.
- Three isolated actual ARM64 PTYs (Korean/English Unicode and English ASCII) exited 0 with unchanged temporary settings. Each resized both the main screen and an open menu through 40×12→80×24→140×45→128×38: 24 SIGWINCH resizes total, followed by working menu/back/range-copy/cell-detail/connection/exit operations. Checked 7-bit ASCII screen/menu output. All are DB-free `--demo`, not new live Oracle integration runs.
- Refreshed [Korean](../images/lazyora-overview-ko.png)/[English](../images/lazyora-overview.png) screenshots from actual PTY data and visually inspected menu single borders, long Korean titles and gray disabled items. Verified 399 local document links, eight guide sections per language and `git diff --check`.
