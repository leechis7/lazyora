# LORA-76: Tuivory v0.9.0 SQL 분석기 적용 / SQL analyzer adoption

[한국어 반영 기록](../tuivory-sync.md) · [English integration](../en/tuivory-sync.md)

## 출처와 적용 / Source and adoption

- 공개 모듈 / Public module: `github.com/leechis7/tuivory/go v0.9.0`
- 태그 / Tag: `go/v0.9.0`
- Origin: `3aadb547a40f8be58b55268300dc613b6f1b8db1`
- Sum: `h1:fY/1eZcDws90Z9Vb60JQ4GruNNBuX9Y9dU450o3Bkz8=`
- 공개 사용법 / Published migration guide: [syntax.en.md](https://github.com/leechis7/tuivory/blob/go/v0.9.0/docs/syntax.en.md)

기존 `feature/LORA-76-tuivory-v080`의 미커밋 전환 작업을 이어갑니다. 직접 tview/tcell import 0개와 공개 root facade·SurfaceView 적용을 유지합니다. `sql_colors.go`의 독자 렉서·키워드/함수/타입 분류표를 제거하고 선택적 `NewSQLAnalyzer().AnalyzeSyntax()`에 Oracle SQL/PLSQL 강조를 위임합니다. 한 번만 지연 생성한 분석기는 원문 바이트 범위와 공통 역할을 반환합니다. 앱에는 역할을 기존 스타일 콜백에 연결하는 최소 어댑터와 변경 시 재분석/그 외 토큰 캐시만 둡니다. 분석 실패 시 이전 구문의 색상을 남기지 않고 일반 텍스트로 표시합니다.

Continue the existing uncommitted feature branch. Retain zero direct tview/tcell imports and public root-facade/SurfaceView integration. Remove the host highlighting lexer and keyword/function/type classification table; delegate Oracle SQL/PLSQL highlighting to optional NewSQLAnalyzer().AnalyzeSyntax(). Initialize it lazily once. Keep only a byte-range/semantic-role display adapter and change-triggered analysis/token caching; failures render plain text instead of stale highlighting.

기존 268개 오프라인 자동완성 후보는 `completion_words.go`의 별도 앱 데이터로 보존합니다. 강조 분석기가 이 후보를 분류표로 참조하지 않습니다. SDK는 현재 공개 어휘 열거 API가 없으며 기존 완성 동작을 잃지 않습니다. SQL 실행 파서·위험 구문 판단·바인드 변수 실행 처리·Oracle 메타데이터/트랜잭션은 바꾸지 않습니다. Go/Rust 소스 강조 분석기가 추가됐다고 표시하지 않습니다.

Preserve 268 existing offline completion candidates as separate application data; the highlighter never uses that list for classification. The SDK currently exposes no vocabulary-enumeration API. Execution parsing, dangerous-SQL policy, bind execution, metadata and transactions remain unchanged. Do not claim Go/Rust source analyzers are implemented.

## 전체 편집기 교체 판단 / Full editor replacement decision

새 `NewEditor/NewEditorView`, `SelectBytes/ReplaceBytes`, `SetSyntaxAnalyzer`와 `SyntaxStyleFunc`는 실제 제공됩니다. 직접 import 0개와 로컬 편집기 포크 제거는 별개이며 다음 소비 계약 차이로 전면 교체는 보류합니다. LazyOra에 범용 편집기를 다시 구현해 우회하지 않습니다.

New Editor/View, byte editing, automatic analysis and SyntaxStyleFunc APIs are available. Zero direct consumer imports does not mean fork retirement. Defer full editor replacement for the differences below rather than implementing another local shared editor.

| 확인 항목 / Check | 관찰 / Observation |
| --- | --- |
| Tab | stock SDK 호환 TextArea는 원문 탭 삽입, 새 EditorView는 삽입 없음 / Stock SDK TextArea inserts a literal tab; new EditorView inserts nothing. |
| Ctrl+← | `SELECT value` 끝 바이트 위치: stock 12→7, 새 EditorView 12→12 / No word movement in the new view. |
| 1,000행·33KB / 1,000 lines | SQL 바인딩·초기 Layout 뒤 100×20 불변 문서 Layout 1회: 46.678ms, 15,350,344B 할당 / One warmed layout. |
| 10,000행·330KB / 10,000 lines | 동일 조건 563.646ms, 161,602,376B 할당 / Same conditions. |

위 값은 이 ARM64 서버의 단일 진단 관측으로, 다른 환경의 일반 성능 보장이나 stock과의 동등 벤치마크가 아닙니다. 공개 v0.9.0만 의존하는 별도 모듈 `/tmp/lora090-probe`에서 비교했고 upstream 소스는 수정하지 않았습니다. Ctrl+Home 등의 차이는 앱 전역 키 정책과 구분하므로 모두 SDK 결함으로 단정하지 않습니다. SDK Layout의 전체 문서 그라핌/행/구문 역할 재구성 경로와 Tab·단어 이동 계약을 TVR-24에 전달했습니다.

These are single diagnostic observations on this ARM64 host, not portable guarantees or a matched stock-renderer benchmark. A standalone module depending only on public v0.9.0 reproduced them; upstream sources were not changed. Do not classify all Ctrl+Home/global-key differences as SDK defects. Reported full-document layout allocation, Tab and word-navigation compatibility to [TVR-24](https://leechis7.atlassian.net/browse/TVR-24).

현재 `third_party/tview` replace는 SQL 색상과 기존 편집/스크롤/키맵 동작을 보존하기 위해 유지합니다. 새 SDK가 SQL 분석이나 바이트 편집을 제공하지 않기 때문이 아닙니다. 공통 입력·긴 문서 표시 개선이 게시되면 재검증하고 이 의존성을 제거합니다.

Retain the existing tview replacement to preserve highlighting and editing/scroll/keymap behavior, not because the SDK lacks SQL analysis or byte editing. Re-evaluate and remove it after shared input/large-document improvements are published.

## 검증 및 게시 상태 / Validation and publication

핵심 SQL 토큰·문법 색상·원문/선택/Undo·소스 편집·자동완성·직접 import 경계 검사는 통과했습니다(0.041초). `make check`, 기본 Linux ARM64 `make build`와 실제 포함 Tuivory v0.9.0 확인도 통과했습니다. 애플리케이션 버전은 아직 v0.12.1이며, 공식 배포본 v0.12.1의 Tuivory는 v0.5.0입니다. 커밋·게시·GitHub CI·배포는 별도 승인 전 수행하지 않습니다. 전체 LORA-76 완료가 아닙니다. 기존 run.sh 실행 권한 변경은 보존합니다.

Targeted SQL/token/color/literal-text/selection/undo/source/completion/import-boundary tests passed in 0.041s. Vet and default Linux ARM64 build passed; module metadata confirms Tuivory v0.9.0. The development app version remains v0.12.1; published v0.12.1 contains Tuivory v0.5.0. No commit/publication/GitHub CI/release without separate approval; the broader LORA-76 remains incomplete. Preserve the existing run.sh mode change.

실제 한글/영어 Unicode·영어 ASCII PTY 3개에서 각 메인 4회·메뉴 4회·접속창 3회, 총 33회 크기 변경 후 정상 exit 0과 설정 보존을 확인했습니다. ASCII 접속창 7비트 출력 검사도 통과했습니다. v0.8.0 한영 메인/메뉴/접속창 6개 JSON 캡처의 좌표·문자·전경/배경·굵기·밑줄은 모두 동일합니다. 실제 폰트·Dim 밝기 인증, 실 Oracle/CentOS·다른 OS 실행 검증은 아닙니다.

Three actual PTYs (Korean/English Unicode, English ASCII) passed 33 resizes, normal exit and unchanged settings; ASCII connection output passed seven-bit validation. All six bilingual main/menu/connection JSON captures match v0.8.0 in coordinates/text/FG/BG/bold/underline. This does not certify physical fonts/Dim brightness or live Oracle/CentOS/other-OS execution.

전체 `GOMAXPROCS=2 GOMEMLIMIT=768MiB go test -race ./...`도 통과했습니다(UI 188.848초). Oracle Docker 테스트 패키지 성공은 기본 모의/opt-in 건너뜀 조건의 결과이며 실 DB 시험으로 주장하지 않습니다. 로컬 문서 링크 428개와 `git diff --check`를 확인했습니다. Jira에 검토 전환이 없어 진행 중 상태에 검토 대기를 남깁니다.

The full race suite also passed (UI 188.848 seconds). The Oracle Docker package follows its default mock/opt-in skip conditions; this is not a live database validation. All 428 local documentation links and diff whitespace checks passed. Jira has no Review transition, so retain In Progress with review pending.

## 상단 F10 메뉴 위치 재작업 / Top-menu placement rework

사용자가 상단 F10 메뉴 위치 오류를 보고했습니다. 원인은 LazyOra 팝업의 X를 실제 라벨 좌표가 아닌 `menuIndex * 화면폭 / 8`로 계산하던 앱 배치 코드입니다. 공개 `Menu.BarLayout()`의 현재 표시 셀 좌표를 앵커로 사용하고 `MenuPlacement()`에 화면 경계 보정을 맡겼습니다. 실제 메뉴바가 배치된 경우 부모 원점을 포함한 좌표·메뉴바 아래 위치를 사용하고, 아직 배치되지 않은 경우 메뉴 페이지 좌표를 기준으로 공개 BarLayout 결과를 사용합니다. 한영 라벨·상단 메뉴 스크롤·좁은 폭과 크기 변경을 따라갑니다. 새로운 범용 UI 배치 엔진이나 테두리 그리기를 앱에 만들지 않았습니다.

The reported top F10 placement bug came from the host's equally spaced menuIndex*viewportWidth/8 anchor. Use visible public Menu.BarLayout cell coordinates and MenuPlacement viewport clamping instead. Respect nonzero parent origins and the bottom of the laid-out bar; before layout, use public label geometry relative to the menu page. Follow bilingual labels, bar scrolling, narrow widths and resizing without implementing another local layout/border engine.

한영 8개 메뉴·5개 화면 크기(축소/복원)·부모 원점 이동의 80개 배치 사례와 기존 메뉴 회귀를 통과했습니다(0.540초). 전체 UI 16.089초, 메뉴 관련 race 2.902초, 기본 ARM64 재빌드도 통과했습니다. 실제 한영 PTY에서 22회 resize·정상 종료·설정 보존을 확인하고 한글 편집 메뉴 캡처의 라벨 아래 배치를 육안 확인했습니다. 본 변경은 로컬 검토 대기이며 커밋·게시·배포하지 않았습니다.

Passed 80 placement cases (two languages, eight groups, five resize states with nonzero parent origin) plus existing menu checks in 0.540s; full UI in 16.089s, menu race checks in 2.902s and ARM64 rebuild. Two actual bilingual PTYs passed 22 resizes, normal exit and settings preservation; visually inspected Korean Edit popup alignment. This rework awaits local review, with no commit/publication/release.
