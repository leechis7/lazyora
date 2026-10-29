# Tuivory 패키지 적용 기록

[한국어](tuivory-sync.md) | [English](en/tuivory-sync.md)

LazyOra는 [Tuivory](https://github.com/leechis7/tuivory)의 공통 스타일 원칙을 제품 UI에 적용합니다. Tuivory의 가이드·견본·도구 개발은 별도 저장소와 Tuivory Jira에서 관리하고, LazyOra의 적용 작업은 LORA Jira에서 관리합니다.

## LORA-76: Tuivory v0.4.0 단일선·표면·비활성 개선 (2026-10-30, 미배포)

공개 모듈 `github.com/leechis7/tuivory/go v0.4.0`, [go/v0.4.0](https://github.com/leechis7/tuivory/releases/tag/go/v0.4.0), Origin `fef7a7b2247d1c68ef0fb1469e6a2fc2e1cec9b5`, Sum `h1:rhyWAiaCladiGiRSs/K97VpCyDj4FCma539665KazeU=`를 확인했습니다. 현재 메뉴·그리드 브랜치에서 v0.3.0을 갱신하며 소스 복사·수정·로컬 replace는 하지 않습니다.

- 메뉴는 공개 PanelView에 원래 MenuView를 직접 넣어 단일선·제목 폭 자르기·표면 채우기·문자 경계 보호를 맡깁니다. 이전의 별도 Box 수동 그리기와 메뉴의 추가 WideSafeScreen 래퍼를 제거합니다. 앱은 위치·크기·제목·여백 0과 명령/포커스 연결만 관리합니다. 일반 포커스 패널의 이중선 정책은 유지합니다.
- 일반 텍스트를 공개 TextOnSurface로 실제 표면에 연결합니다. 본문·힌트·안내/경고의 배경 불일치와 낮은 대비를 공통 API로 보정하며, 입력·선택·편집 셀 등의 명시적 상태 배경은 유지합니다. GridView의 공개 PreserveBackgrounds 기본 설정으로 결과 그리드 스타일을 보존합니다.
- 비활성 색은 공개 DisabledStyle을 통해 소비하고 공통 메뉴의 회색/Dim·접두사 제거·포커스/클릭/실행 차단을 적용합니다. 일반 텍스트 참조 대비 검사는 4.5:1을 유지하고 비활성만 새 공통 설계의 2:1 목표와 구별 표시를 검사합니다. 8색 Dim의 실제 밝기는 터미널 구현에 의존합니다.
- upstream 크기 변경 멈춤 수정은 playground 내부 재배치 큐에 있습니다. LazyOra는 그 private 앱 코드를 가져오지 않고 자체 단말 수명주기를 유지하며 실제 PTY의 창 크기 왕복과 조작 지속을 별도로 확인합니다.

아래 v0.3.0 항목은 앞선 로컬 적용 단계입니다. 현재 검증은 [v0.4.0 한영 기록](audits/lora-76-tuivory040.md)을 기준으로 확인합니다. 정식 v0.11.3은 v0.2.0이며 커밋·게시·배포는 아직 수행하지 않았습니다. 이 작업으로 남은 LORA-76 전체 패널/분할/하단 범위를 완료하지 않습니다.


## LORA-76: Tuivory v0.3.0 메뉴·결과 그리드 (2026-10-29, 미배포)

- 공개 모듈: `github.com/leechis7/tuivory/go v0.3.0`, [go/v0.3.0](https://github.com/leechis7/tuivory/releases/tag/go/v0.3.0), Origin `dabab8b8d9b23e9c641231b5358301e3fbb45431`, Sum `h1:yvisKQpR9gB90Hge+K0b8NvJJI38LT6gMnH2YE3VTkQ=`. 공개 모듈을 import하며 Tuivory 소스 복사·수정·로컬 replace는 없습니다. 기존 tview의 SQL 편집기 확장용 replace는 유지합니다.
- 메뉴: 공개 `Menu`·`MenuView`·`MenuBarView`가 표시·구분선·비활성 항목·중첩/뒤로·스크롤·포인터 탐색을 담당합니다. 기존 List 렌더링과 자체 목록 탐색을 제거했습니다. 앱은 명령 분류/실행 가능 조건, F10·Shift+F10·Alt 진입, 기존 ←/→ 상위 메뉴 이동 정책, 호출 창 복원과 단일선 카드 배치를 담당합니다. 카드 전체는 기존 `WideSafeScreen` 래퍼로 보호합니다.
- 그리드: 동일 공개 `Grid`의 `data` 프리셋과 옵션으로 행 번호·컬럼 구분선·줄무늬·숫자 정렬·현재 셀/행·범위·변경 셀·상태줄을 구성하고 `GridView`로 그립니다. 독자 결과 본체 그리기·행번호/구분선 덧그리기를 제거했습니다. `GridSource`가 원문을 보이는 창 단위로 제공하며 표시문자·NULL·숫자·편집 상태는 `GridCellInfo`로 연결합니다. 레거시 Table은 기존 제품 메타데이터/선택 호환용이며 본체를 그리지 않습니다. 인라인 편집 좌표도 공개 `Geometry`와 일치합니다.
- 보존: Oracle ROWID/원본 컬럼·정렬/필터/페이징·트랜잭션·편집 확인·부분 LOB 복사 차단·TSV/OSC52·전역 키는 LazyOra 책임입니다. 라이브러리의 데모 F5~F12 바인딩을 앱에 그대로 설치하지 않습니다. 헤더 클릭 정렬은 비활성화하고 기존 메뉴 정렬을 유지합니다.
- 표시 차이: 현재 행 번호 앞 표식은 `>`이며 `~`는 공통 컴포넌트의 폭 생략 표식입니다. 변경한 셀의 배경과 상태줄 `*`는 유지하되 행 번호 전체의 `*`는 표시하지 않습니다. 공통 모델의 행 Edited 상태는 모든 셀의 편집색도 바꾸므로 셀 단위 Edited만 전달합니다. 상태줄은 제품의 한영 문구를 공통 StatusLine에 전달합니다.
- TVR-17의 그리드 옵션이 공개되어 이번에 소비합니다. 패널/분할선/하단 안내 및 LORA-73~75의 앱 작업공간 요구까지 완료한 것은 아닙니다. [이번 검증과 제한](audits/lora-76-menu-grid.md)을 참고하세요. 정식 v0.11.3은 Tuivory v0.2.0이며 이번 소스 변경은 아직 커밋·게시·릴리스하지 않았습니다.


## LORA-76: 컴포넌트와 앱 책임 정정 (2026-10-29, 검토)

Tuivory는 TUI 컴포넌트 라이브러리입니다. 재사용 가능한 컨트롤의 표시·내부 포커스/선택/입력·스타일·옵션·문자 겹침 보호는 공개 API를 우선 사용합니다. 실제 접속/설정/도움말 화면 조합, 앱 전체 배치·크기/순서·컴포넌트 간 포커스·전역 단축키·데이터/명령 연결은 LazyOra가 담당합니다. UI 조합 코드가 제품에 남는 것 자체는 미이관 결함이 아니며, 전체 화면을 Tuivory로 이전하자는 앞선 해석은 철회합니다. [정정한 책임 표와 소스 위치](audits/lora-76-tuivory.md)를 참고하세요.

TVR-7 본문과 과도한 이전 요청 댓글을 한영으로 정정했습니다. [TVR-17](https://leechis7.atlassian.net/browse/TVR-17)의 단일 간결 그리드 옵션/프리셋 요청과 비교 이미지 2장은 유지합니다. 공통 컨트롤의 중복만 줄이며 앱 화면 조합까지 패키지에 맡기지 않습니다. 검토 기록 변경이며 코드·정식 배포 자산을 추가 변경하지 않았습니다.

## LORA-76: 컬럼 말단 중복 아이콘 생략 (2026-10-29, v0.11.3)

공개 Tuivory Go v0.2.0 / go/v0.2.0 / 987d1a6786033a6bf063235b5e7fea7b565f0af0을 유지합니다. TreeView의 기본 말단 표식(Unicode · / ASCII .)에 LazyOra의 컬럼 아이콘(· / C)이 중복됐으므로 공개 TreeNode.Icon/ASCIIIcon을 빈 값으로 둡니다. 각 컬럼 줄에서 두 칸을 줄이고 기본 말단 표시·들여쓰기·식별·선택·상세 열기를 보존합니다. 공통 렌더러를 복사하거나 수정하지 않습니다. 기본 표식 자체의 선택 옵션이 필요하면 Tuivory 쪽에서 제공한 뒤 소비합니다. 이 표시 개선은 v0.11.3에 포함하며 기존 v0.11.2 자산은 변경하지 않습니다.

## LORA-76: Tuivory v0.2.0 공통 UI 우선 적용 (2026-10-29, v0.11.2)

- 출처: [go/v0.2.0 공개 릴리스](https://github.com/leechis7/tuivory/releases/tag/go/v0.2.0), 커밋 `987d1a6786033a6bf063235b5e7fea7b565f0af0`. 모듈 Origin·Sum `h1:q+NJW7/inCfX1cbGLIlEihF8mo/i86cGHozvl9VL6Gw=`를 확인하고 공개 모듈을 고정합니다. Go 1.24·tcell/tview·AGPL-3.0-only 요구는 호환됩니다. 로컬 replace나 패키지 소스 복사는 없습니다.
- 적용: 공통 팝업 전체(배경·카드·그림자·자식 테두리)를 공개 `WideSafeScreen`으로 그립니다. 독자 `popupFill`을 제거하고 라벨 폭 자르기도 `CellText`로 전환합니다. 좁은 라벨의 생략 표식은 공통 API의 `~`를 사용합니다. LORA-79의 물리 출력 회귀 검사는 유지합니다.
- 제품 어댑터: 새 그리기 API가 `Put`을 사용하므로 `asciiScreen`의 Put/SetCell/PutStr 계열도 기존 표시 전용 변환으로 연결합니다. 원문·셀 폭·복사·설정 정책은 보존하며 두 칸 문자 겹침 수리 코드를 재구현하지 않습니다.
- 검토: 기존 스타일/위젯/트리 API는 유지합니다. 기존 스타일 값은 변경되지 않았고 Choice는 새 공통 겹침 보호를 내부 사용합니다. 새 Form/FocusStack/Menu/Grid/Tab/Panel/Split/Status의 상태 모델을 제품 전체에 일괄 교체하지 않습니다. 적용/보류의 구체적 동작 차이와 관련 후속 일감은 [검토 기록](audits/lora-76-tuivory.md)에 구분합니다.
- 정책: 공통 UI 문제는 Tuivory 공개 API로 해결하고, 부족한 기능은 별도 Tuivory 프로젝트에서 제공한 뒤 소비합니다. 제품 Oracle/명령/데이터 어댑터와 공통 컴포넌트 책임을 분리합니다. v0.11.2에 적용하며 기존 v0.11.1 바이너리는 변경하지 않습니다.

## LORA-79: 공통 팝업 한글 경계 수정 (2026-10-28, v0.11.1)

공개 의존성은 `github.com/leechis7/tuivory/go v0.1.0`, 태그 `go/v0.1.0`, 출처 커밋 `c034a00b1db92ee7ad4abd1ea5956907f1c1a124`를 유지합니다. Go 모듈 Origin을 다시 확인했습니다. Tuivory의 열린 Choice에는 두 칸 문자 경계 보호가 있으나 이번 결함은 LazyOra가 직접 배치하는 공통 팝업 카드·그림자에 있습니다. 공개 패키지를 그대로 import하고 upstream 수정·소스 복사·로컬 replace는 하지 않습니다.

`form_popup.go`의 `popupFill`에서 카드·그림자의 양쪽 경계에 걸친 배경 문자의 두 칸을 함께 지웁니다. 경계 밖의 온전한 글자와 스타일은 유지합니다. 기존 검사가 논리 셀과 한 칸 배경만 확인해 놓친 문제이므로 `Show()` 이후 실제 출력 버퍼를 비교하는 회귀 검사를 추가합니다. 한영 UI·3종 팝업·12개 테마/색상·크기 왕복·한글/CJK/이모지/결합 문자·배경 복원 및 SQL/결과/포커스 보존을 검사합니다. [한영 검증 기록](audits/lora-79-popup.md)을 참고하세요. 이 수정은 LazyOra v0.11.1부터 포함되며 기존 v0.11.0 자산은 변경하지 않습니다. Tuivory의 일반 팝업/오버레이용 공개 경계 보호 API는 별도 Tuivory 프로젝트에서 보완한 뒤 LazyOra 후속 일감으로 적용할 범위입니다.

## LORA-78: 문자 지원 감지·ASCII 표시 (2026-10-28, v0.11.0)

공개 모듈은 v0.1.0, 태그 go/v0.1.0, 커밋 c034a00b1db92ee7ad4abd1ea5956907f1c1a124를 유지합니다. TreeView.ASCII를 실제 적용 모드에 연결합니다. 기존 tview 화면은 LazyOra의 displayLayout/asciiScreen이 그릴 때만 호환 문자로 대체합니다. 데이터·언어 선호·터미널 입출력 프로토콜은 제품 책임으로 유지하며 Tuivory 자체 개발이나 소스 복사는 하지 않습니다. 아직 사용하지 않는 Tuivory Panel/Split 등의 ASCII 옵션은 해당 컴포넌트 적용 일감에서 연결합니다.

자동 감지는 CharacterSet과 CanDisplay의 추정치를 사용하며 실제 폰트 글리프는 확인하지 못합니다. 수동 모드·설정 저장·CLI 우선순위와 원문 보존 정책은 [터미널 안내](terminal.md)에 기록합니다. 전체 race/vet/ARM64 빌드·144종 모의 화면·3종 로컬 PTY 시험을 통과했습니다. 원문 보존과 실제 원격 폰트/실DB 미검증 범위는 [검증 기록](audits/lora-78-ascii.md) 및 LORA-78 검토 댓글에 남깁니다.


## 반영 절차

1. 게시된 릴리스나 main의 정확한 커밋과 문서를 확인합니다. 로컬의 미게시 변경을 게시 기준과 섞지 않습니다.
2. 마지막 확인 기준과 비교해 원칙·토큰·컴포넌트·상태별 차이를 정리합니다. 테마의 검토용 후보와 실제 권장 규칙을 구분합니다.
3. LORA 관련 일감에 요구사항을 추가하고 적용 작업을 시작하면 진행 중으로 전환합니다.
4. Tuivory의 공개 Go 패키지를 import하여 공통 스타일·컴포넌트에 적용합니다. Oracle 트랜잭션·SQL 편집·결과 복사·Toad 키맵처럼 제품 고유 동작은 해당 요구사항에 맞춰 보존합니다.
5. 변경 영향에 맞는 렌더링/상태 전이·회귀 검사를 수행합니다. 8/16/256색·True Color × 다크/라이트/흑백에서 포커스·선택·입력·팝업·테마 왕복을 확인하고, 모의 화면과 실제 단말 확인 범위를 구분합니다.
6. 출처·적용 위치·검증·보류를 아래에 기록합니다. 커밋·게시·배포는 사용자의 명시적 승인 후 진행합니다.

색상 후보를 일괄 교체하는 대신 실제 배경과 상태 합성에서 대비·표시 의미를 검증하여 제품 값을 정합니다. 자동 정기 동기화나 Tuivory 프로젝트의 일감/스프린트 변경은 이 절차에 포함하지 않습니다.

현재 개발 소스의 패키지는 **`github.com/leechis7/tuivory/go v0.4.0`**입니다. 정식 LazyOra v0.11.0·v0.11.1은 v0.1.0을, v0.10.2는 v0.0.1을 포함합니다. 아래 TUIS 관련 항목은 이전 적용 이력이며 현행 의존성이 아닙니다.

## LORA-72: Tuivory v0.1.0 트리 적용 (2026-10-28, v0.11.0)

- 공개 태그: [go/v0.1.0](https://github.com/leechis7/tuivory/releases/tag/go/v0.1.0), 소스 커밋 `c034a00b1db92ee7ad4abd1ea5956907f1c1a124`. 공개 Go 모듈의 Origin 및 체크섬을 확인하고 `v0.0.1 → v0.1.0`으로 갱신했습니다. 로컬 replace나 소스 복사는 없습니다.
- 적용: `browser_tree.go`가 공개 Tree/TreeState/TreeView의 렌더링·탐색·스크롤을 사용합니다. Oracle 객체 식별과 비동기 작업 참조는 LazyOra 어댑터에서 유지합니다. `browser_columns.go`에서 테이블·뷰의 컬럼을 펼칠 때만 조회하고 현재 목록에 캐시합니다.
- 기존 사용자·유형 선택·검색·단일 클릭/Enter 상세 보기·포커스 이중선은 유지합니다. 타입 선택·필터로 숨긴 노드의 컬럼 캐시는 유지하고, Ctrl+R·사용자 변경·재접속으로 목록을 교체하면 폐기합니다. 지연 응답은 원래 노드 인스턴스가 현재 캐시에 있는 경우에만 반영합니다.
- 색상: 신규 공통 후보 토큰과 기존 NewStyles 팔레트는 서로 다른 자료입니다. 트리는 기존 공개 Styles의 의미 역할에 연결하여 같은 화면에서 팔레트를 혼합하지 않습니다.
- 보류: Tab·Panel·Split·Breadcrumb·Status 컴포넌트는 이번 릴리스에 추가됐지만 LORA-73~76에서 순차 적용·검토합니다. 이번 작업으로 해당 일감을 완료 처리하지 않습니다.
- 검증 완료: 기존 UI 회귀, 새 트리 전용 6개 테스트(12개 테마/색상 조합 포함), 5천 객체 탐색, 전체 `go test -race ./...`, `go vet ./...`, Linux ARM64 `make build`를 통과했습니다. 문서 로컬 링크 93개와 한영 내장 가이드 각 8개 목차도 확인했습니다. 실제 원격 단말·실DB 신규 접속은 이번에 시험하지 않았습니다. 상세 기록은 통합·게시 전의 검증 시점을 기준으로 합니다. [검증 기록](audits/lora-72-tree.md)을 참고하세요.

## LORA-70: Tuivory 전환 (2026-10-24)

- 사용자 요청으로 TUIS 소비를 종료하고 [Tuivory go/v0.0.1](https://github.com/leechis7/tuivory/releases/tag/go/v0.0.1)로 교체합니다.
- 실제 Go 모듈: `github.com/leechis7/tuivory/go v0.0.1`. 공개 패키지 이름은 `tuivory`이며 모든 소비 코드가 이 경로 하나를 import합니다. 기존 두 하위 패키지 import와 TUIS 모듈은 제거합니다.
- 릴리스 커밋: `cf06d95f4334f65436197ae4f12faf35783f5b7a`. `go/` 하위 모듈이므로 태그는 `go/v0.0.1`, Go 의존 버전은 `v0.0.1`입니다. 로컬 replace 없이 공개 모듈을 사용합니다.
- 구조 변경: 저장소 루트의 언어 중립 설계 자료와 Go 하위 모듈 분리, 단일 공개 API와 internal 구현, 토큰 형식 식별자 변경. LazyOra는 후보 JSON 토큰을 직접 읽지 않으므로 형식 식별자 변경의 영향은 없습니다.
- 실제 다운로드한 모듈의 `internal/style`·`internal/widgets` 구현을 이전 `packages/go/tcell`·`packages/go/tview`와 비교했습니다. 패키지 선언·내부 import 경로·설명 이외에 기존 스타일/위젯 런타임 로직의 차이는 없습니다. 신규 공개 진입점에서 타입·함수·상수를 제공합니다. 릴리스 이후 공개 main의 추가 1개 커밋도 추적/검증 문서만 변경합니다.
- Go·tcell·tview 버전과 AGPL-3.0-only 라이선스는 호환됩니다. LazyOra의 tview 로컬 포크는 유지합니다. Tuivory 저장소 자체는 변경하지 않습니다.
- 소비 API: `tuivory.NewStyles`, `ProfileForColors`, `Styles`, `Styler`, `NewChoice`, `Choice`, `Surface`, `TextRole`, `CellState`, `SyntaxKind`, `BorderColor`와 스타일 메서드.
- 이후 갱신: `go get github.com/leechis7/tuivory/go@<버전>` 및 `go mod tidy` 후 LORA 일감에서 검증합니다.

이 전환의 전체 회귀 및 12프로필 화면 검사는 완료했고 v0.10.2로 배포했습니다. [검증 기록](audits/lora-70.md)에 이전 결과와 구분해 기록했습니다. 현재 문서 게시 후 main/dev CI도 통과했으며, 현행 동작은 현재 의존성과 이 Tuivory 항목을 기준으로 확인합니다.


## 이전 단계 보관 기록 / Previous migration stages

다음은 당시의 TUIS 적용 이력입니다. 현재 패키지 선택·갱신 지침이 아닙니다.

## 2026-10-24 기준 확인

| 항목 | 내용 |
| --- | --- |
| TUIS 출처 | main [e2118e8d9d005ed8a8bc03da008548790b3b6b65](https://github.com/leechis7/tuis/commit/e2118e8d9d005ed8a8bc03da008548790b3b6b65) |
| 원칙 | [docs/principles.md](https://github.com/leechis7/tuis/blob/e2118e8d9d005ed8a8bc03da008548790b3b6b65/docs/principles.md) |
| 후보 예시 | [themes/lazyora-candidate.csv](https://github.com/leechis7/tuis/blob/e2118e8d9d005ed8a8bc03da008548790b3b6b65/themes/lazyora-candidate.csv) |
| 비교 대상 | LazyOra의 초기 공통 원칙 보관본과 후보 CSV, v0.10.1의 공통 스타일 적용 기록 |
| 결과 | 원칙 본문과 후보 데이터 동일. TUIS 문서의 소개/링크만 변경 |
| 이번 적용 | 작업 범위·출처·반영 절차 문서화. 새 UI/색상 값 변경 없음 |
| 관련 일감 | [LORA-69](https://leechis7.atlassian.net/browse/LORA-69) |

기존 공통 스타일의 구현·검증 근거는 [LORA-69 적용 기록](audits/lora-69-implementation.md), 현행 규칙은 [UI 스타일 규칙](ui-style.md)에 있습니다. 이후 Tuivory 변경을 반영할 때는 새 기준과 차이를 별도 기록으로 추가합니다.


## LORA-70: Go 패키지 적용 (2026-10-24)

- 출처: TUIS 공개 main [`ea155a284e80f7d45c5dbaf054ba070c35b9425f`](https://github.com/leechis7/tuis/commit/ea155a284e80f7d45c5dbaf054ba070c35b9425f).
- 고정 의존성: `github.com/leechis7/tuis v0.0.0-20261023175236-ea155a284e80`. 정식 태그가 없어 Go가 생성한 커밋 기반 버전을 사용합니다. 로컬 replace나 TUIS 소스 복사 없이 공개 모듈을 다운로드합니다.
- 사용 API: `packages/go/tcell`의 `NewStyles`, `ProfileForColors`, 상태별 스타일 메서드와 `packages/go/tview`의 `Styler`, `Choice`.
- 의존성: 양쪽 모두 Go 1.24 이상, tcell v2.13.10, tview v0.42.0입니다. LazyOra의 `replace github.com/rivo/tview => ./third_party/tview`가 모듈 전체에 적용되어 문법 강조용 TextArea 확장도 유지됩니다.
- 라이선스: TUIS는 AGPL-3.0-only이며 LazyOra와 동일합니다. 공개 모듈의 LICENSE와 출처 문서는 해당 고정 커밋에 포함됩니다.

| LazyOra 적용 위치 | TUIS가 제공하는 내용 |
| --- | --- |
| `style_tokens.go` | 테마/색상 프로필 해석, 팔레트, 표면, 상세 표 스타일 |
| `theme.go`, `widgets.go` | 입력·폼·버튼·목록·체크박스·편집기·확인창·테두리 스타일 |
| `choice.go` | 공개 `Choice` 컴포넌트. 폼 색 덮어쓰기와 열린 목록의 한글 겹침 처리 포함 |
| `grid.go`, `grid_inline.go` | 그리드 팔레트, 헤더·현재 셀·행·NULL·변경 셀·행번호·구분선·상태줄 스타일 |
| `sql_colors.go` | 호스트 lexer가 분류한 토큰에 테마/프로필별 구문색 적용 |
| `keymap_comparison.go`, `tabs.go`, `toolbar.go` | 비교표, 기존 탭·하단 버튼 스타일 재적용 |
| `form_popup.go`, `panel_resize.go` | 팝업 배경·카드·그림자, 분할선 스타일 |

`themeColors`와 `gridColors`의 소문자 필드는 기존 제품 렌더러를 위한 값 어댑터입니다. 별도 팔레트 값을 소유하지 않습니다. 테마 변경 시 기존 위젯과 숨겨진 탭을 순회하는 수명 관리, 화면별 의미 역할, 포커스, 팝업 입력 차단, SQL lexer·Oracle 실행·미커밋 상태·그리드 좌표 및 복사는 LazyOra 책임으로 유지합니다. `Styler.Apply`로 앱의 순회 전체를 교체하지 않는 이유는 이러한 제품 상태를 정확히 재적용하기 위해서입니다.

이전 TUIS 단계에서는 `go get github.com/leechis7/tuis@<버전 또는 커밋>`을 사용했습니다. 이 명령은 현재 적용 절차가 아닙니다. 현행 갱신은 위 Tuivory 항목을 따릅니다.

검증 결과는 [LORA-70 검증 기록](audits/lora-70.md)에 기록합니다. 커밋·GitHub CI·게시·릴리스는 이번 구현 요청에 포함되지 않습니다.


## LORA-70: 정식 v0.1.0 적용 (2026-10-24)

- 이전 단계의 의존성: `github.com/leechis7/tuis v0.1.0` ([공개 릴리스](https://github.com/leechis7/tuis/releases/tag/v0.1.0)). 이전 pseudo-version을 정식 태그로 교체했습니다.
- Go 모듈 다운로드의 Origin은 `refs/tags/v0.1.0`, 커밋 `ea155a284e80f7d45c5dbaf054ba070c35b9425f`입니다. GitHub compare와 다운로드한 두 모듈 디렉터리의 `diff -qr` 모두 소스 동일성을 확인했습니다.
- API·팔레트·위젯 동작 변경이 없어 소비 코드 수정은 필요하지 않습니다. `go.mod`와 `go.sum`만 정식 버전으로 갱신했습니다. 앞서 수행한 3,216프레임 비교는 동일한 소스에 대한 결과로 유지하며 이번 갱신으로 재생성한 결과라고 기록하지 않습니다.
- 갱신 후 검사와 실행파일 의존성 확인은 [LORA-70 검증 기록](audits/lora-70.md)의 정식 버전 항목을 참조합니다.
