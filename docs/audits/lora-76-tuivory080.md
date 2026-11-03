# LORA-76: Tuivory v0.8.0 전환 검토 / Migration review

[반영 기록](../tuivory-sync.md) · [English record](../en/tuivory-sync.md)

## 범위 / Scope

공개 `github.com/leechis7/tuivory/go v0.8.0`, 태그 `go/v0.8.0`, Origin `46cc506d0e9234cdd1c7f1b0a2033269e39476c7`, 모듈 Sum `h1:z9afgCzv5OShNQLFH92dEm5pROEGB5SnenONPGLZuMo=`를 확인했습니다. 실제 저장소 적용 전 `/tmp/lora080-review` 격리 복사본에서 검증한 변경안입니다. 이 문서는 커밋·게시·릴리스 완료를 뜻하지 않습니다. 공개 v0.12.1 바이너리는 Tuivory v0.5.0을 유지합니다.

Verified the published module/tag/source/checksum above. This proposal was validated in isolated `/tmp/lora080-review` before applying it to the working repository. It does not imply commit, publication or release; published v0.12.1 still contains Tuivory v0.5.0.

- 기존 앱/테스트 tview import 114파일(프로덕션 53·테스트 61), tcell import 119파일(프로덕션 50·테스트 69). 두 집합의 합집합 127파일을 루트 Tuivory import로 전환합니다. AST import 검사로 두 backend의 직접 import 0개를 검증하며, 검증용 문자열의 단순 검색 결과와 구별합니다.
- Migrate the union of 127 files: 114 tview importers (53 production/61 tests) and 119 tcell importers (50 production/69 tests). AST-based checks verify zero backend imports rather than counting diagnostic string literals.
- SDK가 제공하는 호환 `Widget…` 타입/생성자와 root 키·마우스·화면·색상 API를 사용합니다. SDK 내부 backend 의존은 유지되며 LazyOra 로컬 shim은 만들지 않습니다. 앱의 화면 선언·Oracle 데이터/상태·명령과 전역 단축키를 보존합니다.
- Use published `Widget…` compatibility types/constructors and root key/mouse/screen/color APIs. Backend dependencies remain inside the SDK; no local LazyOra import shim. Preserve host screen declarations, Oracle data/state/actions and global keys.
- `SurfaceView`로 불투명 borderless 팝업 카드를 구성하고 이전 Flex/Box 표면 우회를 제거합니다. SDK가 채우기·문자 경계 보호를 수행합니다. 기존 모달 선택/입력 차단과 1×1 그림자 정책은 유지합니다.
- Replace the Flex/Box surface workaround with public opaque borderless `SurfaceView`, delegating fill/wide-glyph protection while preserving modal routing and one-cell shadow offsets.

## 남은 편집기 의존 / Remaining editor dependency

v0.8.0의 root `TextArea` → `/ui.TextArea` → backend `TextArea` 호환 별칭에는 기존 `SetTextStyleFunc`가 없습니다. `TextDocument.SetByteSyntax`와 새 Editor/Buffer는 제공되지만 기존 선택/Replace 바이트 위치·스크롤·키맵·편집/되돌리기·긴 문서 계약을 그대로 유지하는 drop-in 확장은 아닙니다. `third_party/tview`의 기존 렌더링 확장과 replace를 유지하여 SQL 원문과 문법 색상을 보존합니다. 코드 import 0개와 포크 제거 완료를 혼동하지 않습니다. 포크 제거·의미 모델 기반 편집기 재연결은 TVR-24/LORA-76의 별도 검토 범위로 남깁니다. Tuivory 소스를 이 저장소에 복사하거나 SDK 자체를 수정하지 않습니다.

The root TextArea aliases the stock backend through /ui and does not offer SetTextStyleFunc. New TextDocument.SetByteSyntax/Editor/Buffer APIs are available, but do not preserve the complete existing byte-selection/Replace, scrolling, keymaps, undo and large-document contract as a drop-in extension. Retain the existing third_party/tview rendering extension/replacement to preserve original SQL and colors. Zero consumer imports does not mean fork removal; adapting the semantic editor remains separately tracked under TVR-24/LORA-76. No copied or modified Tuivory implementation.

## 검증 / Validation

- 전체 UI 회귀 검사 통과(15.234초). SQL 문법 색상·원문/선택·Undo, 팔레트/한글 경계·물리 Show 출력, 키맵·메뉴·그리드·트랜잭션 보호를 기존 검사로 확인합니다. 공개 import 경계 검사를 추가했습니다.
- Full UI suite passed in 15.234 seconds, including existing syntax/original-text/selection/undo, palettes/wide glyphs/post-Show output, keymaps/menus/grid/transaction guard checks. Added a public-import architecture regression.
- `make check`와 `make build` 통과. Linux ARM64만 로컬 빌드했으며 `go version -m`에서 Tuivory v0.8.0과 기존 tview replace를 확인했습니다. 앱 버전은 개발 중 기존 v0.12.1을 유지합니다.
- Vet/default ARM64 build passed; module metadata confirms Tuivory v0.8.0 and the retained editor replacement. The development app version remains v0.12.1.
- 한글/영어 Unicode·영어 ASCII의 실제 격리 PTY 3개에서 메인 4회·메뉴 4회·접속창 3회씩 총 33회 크기 변경 후 exit 0·설정 파일 무변경을 확인했습니다. ASCII 접속창은 7비트 출력 검사도 통과했습니다.
- Three real isolated PTYs (Korean/English Unicode and English ASCII) passed 33 total resizes (4 main/4 menu/3 connection per mode), normal exit and unchanged settings. ASCII connection output passed seven-bit validation.
- 기존 v0.12.1과 한영 메인·메뉴·접속창의 JSON 캡처 셀 6개 화면이 모두 동일했습니다. 비교 필드는 좌표·문자·전경/배경·굵기·밑줄이며 실제 단말 폰트·Dim 밝기까지 인증하는 검사는 아닙니다.
- All six Korean/English main/menu/connection JSON captures matched v0.12.1 in coordinates, text, foreground/background, bold and underline. This does not certify physical font rendering or terminal Dim brightness.
- 실 Oracle/CentOS/원격 SSH·다른 OS 실행·GitHub CI는 이번 검증에 포함하지 않았습니다. 게시·배포는 별도 승인 대상입니다.
- No live Oracle/CentOS/remote SSH/other-OS execution or GitHub CI in this validation. Publication and release require separate approval.

- 전체 `GOMAXPROCS=2 GOMEMLIMIT=768MiB go test -race ./...` 통과: UI 190.381초, Oracle Docker 테스트 패키지는 실행 조건에 따른 모의/건너뜀 범위이며 실 DB 검증으로 주장하지 않습니다. 문서 로컬 링크 422개와 최종 패치의 `git apply --check`를 통과했습니다.
- Full race suite passed (UI 190.381 seconds). The Oracle Docker test package follows its opt-in/mock/skip conditions; this is not a live database test. All 422 local document links and patch applicability checks passed.

## 로컬 적용 상태 / Working-tree status

격리 검증과 패치 적용 가능성 확인 뒤 `feature/LORA-76-tuivory-v080`에 동일 패치를 적용했습니다. 커밋·게시·GitHub CI·배포 전이며 사용자 검토 대기입니다. 기존 `test/oracle/run.sh` 실행 권한 변경은 보존했고 업그레이드 패치에 포함하지 않았습니다. 전체 LORA-76을 완료 처리하지 않습니다.

Applied the verified patch to feature/LORA-76-tuivory-v080 after isolated validation and applicability checks. Awaiting user review; no commit, publication, GitHub CI or release. Preserve the pre-existing executable-mode change on test/oracle/run.sh outside the upgrade patch. Do not mark the broader LORA-76 complete.

적용 후 실제 작업 트리의 import 경계·팝업 한글 경계·SQL 문법 색상/원문/선택 핵심 회귀 검사를 다시 통과했습니다(7.481초). ARM64 바이너리를 원본 작업 트리에서도 다시 빌드하고 Tuivory v0.8.0 포함을 확인했습니다.

Post-application targeted import-boundary, popup wide-glyph and SQL-color/original-text/selection regressions passed in the actual working tree (7.481 seconds). Rebuilt its ARM64 binary and verified Tuivory v0.8.0 metadata.
