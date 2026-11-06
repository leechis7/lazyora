# Tuivory 패키지 적용 기록

[한국어](tuivory-sync.md) | [English](en/tuivory-sync.md)

LazyOra는 [Tuivory](https://github.com/leechis7/tuivory)의 공통 스타일 원칙을 제품 UI에 적용합니다. Tuivory의 가이드·견본·도구 개발은 별도 저장소와 Tuivory Jira에서 관리하고, LazyOra의 적용 작업은 LORA Jira에서 관리합니다.

## v0.12.3 릴리스 준비

v0.12.3은 공개 Tuivory Go v0.11.1을 고정하고 셀 편집 UI·실제 커서·독립 변경 표식을 통합합니다. [릴리스 설명](releases/v0.12.3.md). 아래 미커밋·미게시 표기는 각각의 검증 시점 이력입니다. 사용자 배포 승인에 따라 기능 커밋과 한영 근거를 분리했으며 원격 CI·태그 게시를 확인합니다. 전체 LORA-76 완료나 SQL 편집기 포크 제거를 의미하지 않습니다.

## LORA-76: Tuivory v0.11.1 공통 셀 편집 전환 (2026-11-06, 미배포)

공개 `github.com/leechis7/tuivory/go v0.11.1`, 태그 `go/v0.11.1`, Origin `aacaa369a830ad22fe5d5bed7a1e4c25830771c5`, Sum `h1:g7Rp4Rlo039vVdrfJ1Xl3s7/C4HK4jB6HXrOyN9y5to=`를 적용했습니다. 이전 v0.11.0의 커서 보류 사유가 해소되어 공개 `Grid.BeginEdit`/`Input`/`EditInput`/`EditSession`/`ResolveEdit`와 `NewGridView`로 셀 내 편집을 전환했습니다. 로컬 `inlineCellEditor.Draw`/`MouseHandler`, InputField·전체화면 입력 페이지와 개별 좌표/클리핑 처리를 제거했습니다. 남은 객체는 Oracle 대상 ID·원본 컬럼·물리 ROWID 식별과 공개 세션의 연결 상태만 보관합니다.

공개 `PreserveCursorTty`를 기존 TTY 화면 생성 경로에 연결하고 소비자 DECSCUSR 필터는 제거했습니다. 이 경로는 터미널 사용자 커서 모양/깜빡임 설정을 보존하며 기존 TERM 미지정 ANSI 선택과 Win32 키 릴리스 누출 차단은 별개로 유지합니다. Windows의 기존 native console 선택은 유지하며 실제 Windows/macOS 콘솔·SecureCRT·CentOS에서 깜빡임을 시험한 것은 아닙니다. [독립 재현 코드](audits/evidence/lora-76-tuivory0111-cursor.go.txt)와 [출력](audits/evidence/lora-76-tuivory0111-cursor.txt)은 실제 커서 `(9,1,true)`와 고정 `|` 제거를 확인합니다.

Oracle 편집 자격/초깃값은 `Prepare` 콜백으로 공급합니다. Enter 제출은 SDK 승인 대기로 두고 LazyOra 변경 확인에서 승인해야 `ResolveEdit` 후 UPDATE 작업을 보냅니다. 확인 취소는 같은 SDK 초안을 재개하며 원본은 바꾸지 않습니다. 소스 재조회·탭 교체는 세션을 취소하고 오래된 승인에는 SQL을 보내지 않습니다. Alt+D 별도 편집창, Ctrl+U 지우기, Alt+Delete NULL, 원문 복사·변경 셀/독립 행 표식은 유지합니다. SDK 데모 전역 키·Tab 편집 이동·범위 붙여넣기·열 고정·찾기 등은 새 제품 명령으로 노출하지 않습니다.

커서·입력 버퍼·문자 폭·마우스·스타일·그리기는 SDK가 담당하며 강제 커서 escape나 새 로컬 편집 렌더러를 만들지 않습니다. API 제공과 제품 통합 완료를 구분하며, SQL 편집기의 기존 `third_party/tview` replace 및 다른 LORA-76 범위는 남아 있습니다. 커밋·게시·배포하지 않았습니다.

검증: 전체 테스트(UI 25.650초), 관련 race(68.936초)와 최종 승인/포커스 보호 race(1.704초), vet·ARM64 빌드, 실제 커서 96조합·표식 24조합·PTY 4환경을 통과했습니다. [한영 검증 보고서](audits/lora-76-tuivory0111.md)에서 실DB·물리 터미널 미검증 범위를 확인하세요. 사용자 검토 대기이며 커밋·게시·배포하지 않았습니다.

## LORA-76: Tuivory v0.11.0 적용 및 편집 UI 전환 판단 (2026-11-06, 미배포)

공개 `github.com/leechis7/tuivory/go v0.11.0`, 태그 `go/v0.11.0`, Origin `ebe38b07fceb357b3537a355ba45fffde8832ad1`, Sum `h1:U85LlO5WQzQDcRSfvfiaiXo4ufYXk1WeuquABSvUDWw=`를 적용했습니다. `SetRowEdited`로 변경 행 번호의 `*`를 복원하고 기존 셀 `Edited`와 색상을 독립적으로 공급합니다. 스타일과 표식 그리기는 공개 SDK가 담당하며 행 전체 변경색 우회는 추가하지 않습니다. 과거 아래의 ‘행 표식 생략’ 기록은 해당 버전의 이력입니다.

**편집 UI 전체 교체는 보류합니다.** v0.11.0은 승인형 `BeginEdit`/`EditInput`/`ResolveEdit`를 제공하지만 공개 `NewGridView`의 실제 출력은 커서를 고정 `|` 문자로 그립니다. 독립 재현의 `GetCursor()`는 `(-1,-1,false)`이며 실제 커서를 표시하지 않습니다. [재현 코드](audits/evidence/lora-76-tuivory0110-cursor.go.txt) · [출력](audits/evidence/lora-76-tuivory0110-cursor.txt). 요청한 터미널 커서 동작을 유지하기 위해 기존 InputField 어댑터를 남기고, [TVR-7 보완 요청](https://leechis7.atlassian.net/browse/TVR-7?focusedCommentId=11002)에 공통 backend 커서 위치/가시성·포커스/클리핑·NULL/승인 대기·종료 처리를 전달했습니다. 강제 커서 모양 escape나 새 소비자 커서 우회는 만들지 않습니다. ‘API 없음’이라는 11월 5일 기록은 당시 v0.10.2 기준이며 현재는 커서 호환성 차이로 보류합니다.

새 열 고정/숨김/순서·범위 붙여넣기·찾기·보기 상태·Tab 편집 이동은 SDK에 제공되지만 이번 적용에서 제품 명령으로 연결하지 않았습니다. 기존 Oracle 승인·ROWID·트랜잭션·원문 복사와 전역 키를 유지합니다. 커밋·게시·배포 승인 전이며 사용자 `test/oracle/run.sh` 실행 모드는 보존합니다.

검증: 전체 `go test ./...`(UI 22.429초), 관련 편집/표식 `-race`(13.624초), `go vet ./...`, Linux ARM64 `make build`를 통과했습니다. 새 소비자 표식 검사는 다크/라이트/흑백 ×8/16/256/truecolor ×Unicode/ASCII 24조합에서 셀·행 표식 분리, 포커스 이동과 상태 해제를 확인합니다. 기존 한영 Enter/더블클릭 실제 커서 검사도 통과했습니다. 실DB·CentOS·물리 터미널·타 OS 로컬 빌드·GitHub CI는 이번에 실행하지 않았습니다. 사용자 검토 대기이며 커밋·게시·배포하지 않았습니다.

## LORA-76: 공통 셀 편집 API 전환 요청 (2026-11-05, 의존성 대기)

공개 Go v0.10.2(Origin `5e95114869356297c38cf69dc3b1cc46675475eb`)의 `Grid.Handle("activate")`는 `edit` 이벤트를 제공하고 `NewGridView`는 이를 호스트에 전달합니다. 셀 내 입력칸 배치·편집 수명·확정/취소를 소유하는 공개 컨트롤은 아직 없으므로, 현재 `inlineCellEditor.Draw`/`MouseHandler`와 페이지·포커스 관리를 제거하지 않습니다. 기존 Enter·더블클릭 편집은 유지하며 공통 API로 전환 완료했다고 표시하지 않습니다.

사용자 승인으로 [TVR-7 요청](https://leechis7.atlassian.net/browse/TVR-7?focusedCommentId=10993)에 언어 중립 계약을 전달했습니다. Tuivory는 셀 좌표·클리핑·리사이즈/스크롤·커서·포커스·키/마우스 전달·시작/제출/취소 및 설정 가능한 NULL 조작을 담당합니다. LazyOra는 안정적인 행/열 ID와 리비전, 편집 허용 여부·초깃값·NULL/형식·스타일을 연결하고 Oracle ROWID/원본 컬럼 검증·변경 승인·UPDATE·트랜잭션을 담당합니다. 제출은 소스를 즉시 바꾸지 않고 호스트 승인/거절을 지원해야 하며 접속·탭·소스 교체 후 오래된 편집은 무효화해야 합니다.

공개 패키지 제공 후 기존 로컬 UI 우회를 제거하고 Enter·더블클릭·타이핑·취소·NULL·승인 거절·데이터 교체·좁은 화면/크기 변경·한글/ASCII·색상별 회귀 검증을 진행합니다. 별도 Alt+D 편집창은 유지합니다. Go/Rust 제공 완료와 계획을 구분하며 여기서는 Tuivory 구현을 수정하지 않습니다. 이번 기록은 소스/API 확인과 요청 정리이며 제품 변경·추가 테스트·커밋·배포는 없습니다.

변경 셀 표시도 [TVR-7 추가 요청](https://leechis7.atlassian.net/browse/TVR-7?focusedCommentId=10995)에 포함했습니다. 기존 셀 `Edited`/`EditedStyle` 지원을 바탕으로 Tuivory가 테마·색상 수·문자 모드별 배경/글자/보조 표식과 선택·NULL 등 상태 조합 우선순위를 담당합니다. LazyOra는 변경 셀의 안정적인 식별과 실제 미커밋 상태, 승인·실패 및 커밋/롤백/재조회에 따른 해제만 공급합니다. 행 변경 표식은 셀 배경과 독립적으로 제공하여 하나의 변경 셀 때문에 행 전체가 변경 색으로 칠해지지 않도록 요청했습니다. 공개 API 보완 대기이며 기존 표시 동작은 유지합니다.

## LORA-76: Enter·더블클릭 셀 내 편집 명확화 (미배포, 2026-11-03)

일반 SQL 결과에서 Enter·더블클릭을 같은 셀 내 편집 명령으로 연결합니다. 편집 가능한 셀은 기존 공개 InputField로 진입하고 크기 변경 후에도 커서를 셀 안에 유지합니다. 편집 불가 시 상세창 대신 그리드에 사유를 안내합니다. Alt+D 별도 편집창, 문맥/보기 메뉴의 셀 상세, 기존 Oracle 허용 조건과 ROWID 숨김을 유지하며 F4는 객체 상세로 남깁니다. 공통 렌더러나 강제 커서 모양 제어 코드는 추가하지 않았고, 깜빡임/모양은 SecureCRT 호환을 위해 터미널 설정을 따릅니다. 사용자 검토 대기이며 미커밋·미게시입니다.

## LORA-76: 그리드 전후 기능 감사 (2026-11-03, 제품 수정 없음)

v0.11.3 자체 렌더러와 v0.12.2/공개 SDK v0.10.2를 비교했습니다. 버전별 관찰 47개와 동일한 기존 검사 24개를 확인했고, 탐색·휠 선택/복사 대상·헤더·스타일 속성·수정 행 표식의 조작/표시 차이 9가지를 정리했습니다. 조건을 갖춘 결과의 인라인/별도창 편집은 유지됐습니다. [한영 보고서·증거](audits/lora-76-grid-regressions.md)에 SDK와 Oracle 어댑터의 담당 및 실DB 미검증 범위를 구분했습니다. 제품 수정·커밋·게시 없음.

## v0.12.2 릴리스 적용

v0.12.2는 공개 Tuivory Go v0.10.2(Origin `5e95114869356297c38cf69dc3b1cc46675475eb`)을 고정합니다. 앱·테스트의 직접 tview/tcell import는 0개이며 기존 편집기 replace는 입력·성능 호환성 때문에 유지합니다. 공개 SQL 분석기, 실제 라벨 기준 F10 위치, 한글·대비 수정과 오른쪽 1칸·하단 반 줄 그림자를 포함합니다. [릴리스 설명](releases/v0.12.2.md)을 확인하세요. 아래 날짜별 미커밋·미배포 표기는 당시 검증 시점의 이력이며 전체 배치·편집기 전환은 남아 있습니다.

## LORA-76: v0.10.2 Windows 파사드 수정 (2026-11-03)

공개 Go v0.10.2, Origin `5e95114869356297c38cf69dc3b1cc46675475eb`, Sum `h1:iPVsFzd0TLAsADWvMRnP6S0kD0OsZsZzglGAyOeob+Y=`를 적용합니다. 플랫폼 조건을 보존한 SDK 생성기가 Windows import 실패를 해결하며 소비자 shim이나 Windows 지원 제외로 우회하지 않습니다. upstream에서 v0.10.1을 회수했으며 아래 날짜별 검증은 과거 기록입니다. PR #70에서 Windows/전체 CI로 v0.12.2 배포를 재개합니다. [적용 근거](audits/lora-76-tuivory0102.md).

## LORA-76: Tuivory v0.10.1 반 줄 그림자 (2026-11-03, 미커밋·미배포)

공개 Go v0.10.1(태그 `go/v0.10.1`, Origin `287ab1d17e3834c13359645d4a4ec132e83ed345`, Sum `h1:Q/cz8UMFjfANg0Xi/jKtSWcuhtA79vBwW1xs3myr78E=`)을 적용했습니다. 기존 ShadowView/ShadowAuto로 오른쪽 1칸·하단 반 줄을 소비하며 Unicode는 `▀`, ASCII는 `.`으로 표시합니다. 시각적으로 반 줄이어도 배치에는 1행을 예약합니다. 공통 렌더러를 별도 구현하지 않았습니다. [한영 적용·검증 기록](audits/lora-76-tuivory0101.md)을 확인하세요.

## LORA-76: Tuivory v0.10.0 회귀 개선 적용 (2026-11-02, 미커밋·미배포)

공개 Go v0.10.0(태그 `go/v0.10.0`, Origin `1697765b366facfc82120ef4f6c9a18a73437524`, Sum `h1:MZKtI82SYZpVhFTj3zRNJc5x3wtoWEn06Lbj9CNGcp8=`)을 적용합니다. 한글 보존·분할선/구문/비교표 스타일·반응형 확인창의 SDK 수정을 기존 공개 API로 소비하며 앱 우회 배치를 추가하지 않습니다. 이전 24개 실패를 포함한 1,272개 화면 조합이 모두 통과했습니다. [한영 적용·재검증 기록](audits/lora-76-tuivory0100.md)을 확인하세요. 아래 v0.9.0 조사 결과는 수정 전 이력입니다.

## LORA-76: UI 전수 조사 (2026-11-02)

실제 앱 그리기 1,272개 조합을 검사했습니다. 한글 소실·8색 분할선 대비·40열 재접속 버튼 잘림이 남아 배포 가능한 상태로 판정하지 않습니다. 공개 SDK만의 재현과 앱 원인을 분리해 기존 LORA-76/TVR-7에 기록했습니다. [한영 조사 보고서](audits/lora-76-ui-runtime-audit.md)에서 범위·결함·정상 동작·미검증 환경을 확인하세요. 추가 제품 수정·커밋·게시는 하지 않았습니다.

## LORA-76: Tuivory v0.9.0 SQL 분석기 적용 (2026-11-01, 미커밋·미배포)

공개 `github.com/leechis7/tuivory/go v0.9.0`, 태그 `go/v0.9.0`, Origin `3aadb547a40f8be58b55268300dc613b6f1b8db1`, Sum `h1:fY/1eZcDws90Z9Vb60JQ4GruNNBuX9Y9dU450o3Bkz8=`를 확인했습니다. 기존 v0.8.0 작업 브랜치에서 공개 SQL 분석기를 연결해 독자 강조 렉서·색상 어휘 분류표를 제거했습니다. 최소 표시 어댑터가 SDK의 원문 바이트 범위·공통 역할을 기존 편집기 스타일 콜백에 연결하며, 분석 오류 시 이전 색을 쓰지 않고 기본 텍스트로 표시합니다. 자동완성 후보 268개는 강조 분류와 분리된 앱 데이터로 보존합니다. SQL 실행/위험 구문 판단은 이 강조 결과를 사용하지 않습니다.

first-party 앱·테스트의 tview/tcell 직접 import는 0개를 유지합니다. 그러나 직접 import 제거와 기존 편집기 replace 제거는 별개입니다. 새 편집기의 Tab·Ctrl+← 동작과 긴 문서 재배치 비용이 기존 계약과 달라 전면 교체는 보류했습니다. 새 분석기·바이트 편집 API가 없다는 의미가 아닙니다. 공통 입력/성능 개선은 TVR-24에 재현 근거를 전달했으며 LazyOra에 중복 편집 엔진을 구현하지 않습니다. [v0.9.0 적용·검증 기록](audits/lora-76-tuivory090.md)을 확인하세요. 정식 v0.12.1은 여전히 Tuivory v0.5.0입니다.

상단 F10 팝업은 화면을 메뉴 개수로 균등 분할하던 앱 계산을 제거하고, 공개 `Menu.BarLayout()`의 실제 라벨 좌표와 `MenuPlacement()`의 경계 보정에 연결합니다. 한영 메뉴·좁은 폭·크기 왕복·부모 좌표 이동에서 검증합니다.

## LORA-76: Tuivory v0.8.0 단일 import 전환안 (2026-11-01, 미커밋·미배포)

- 공개 모듈 `github.com/leechis7/tuivory/go v0.8.0`, 태그 `go/v0.8.0`, Origin `46cc506d0e9234cdd1c7f1b0a2033269e39476c7`, Sum `h1:z9afgCzv5OShNQLFH92dEm5pROEGB5SnenONPGLZuMo=`를 확인했습니다. SDK 소스 복사나 로컬 Tuivory replace는 없습니다.
- first-party 앱/테스트의 tview·tcell 직접 import를 공개 루트 facade로 전환합니다. 이름이 기존 의미 모델과 겹치는 호환 위젯은 `WidgetForm`, `WidgetTextView` 등 게시된 이름을 사용하며 소비자 shim을 만들지 않습니다. 전역 테두리 값은 `GetBorders()`로 읽습니다.
- 공개 `SurfaceView`가 팝업의 불투명·테두리 없는 카드 채우기를 담당합니다. 이전 Flex/Box 표면 우회를 제거하고 테마·ASCII·입력 차단·그림자·선택 동작을 보존합니다.
- **직접 import 제거와 편집기 포크 제거는 다릅니다.** v0.8.0 `TextArea`는 기본 backend 타입의 호환 별칭이며 기존 `SetTextStyleFunc` 콜백이 없습니다. 의미 모델 `TextDocument.SetByteSyntax`는 제공되지만 기존 편집기의 바이트 선택/Replace·Undo/Redo·대용량 표시 계약 전체를 그대로 대체하지 않습니다. 문법 색상과 원문·키맵을 보존하기 위해 `third_party/tview` replace는 유지하며, 전체 포크 제거가 끝났다고 표시하지 않습니다.
- 릴리스 v0.12.1의 Tuivory는 여전히 v0.5.0입니다. 이번 v0.8.0 변경안의 적용·커밋·게시·CI·배포 여부와 검증은 [적용 검토 기록](audits/lora-76-tuivory080.md)에 구분합니다. 기존 날짜별 항목은 당시 기록입니다.

## v0.12.1 릴리스 적용

v0.12.1은 공개 Tuivory Go v0.5.0을 포함합니다. 아래 팝업·소스 명령 메뉴 전환과 Deprecated 정리를 사용자 승인으로 발행하며 한영 문서·실제 PTY 캡처를 갱신합니다. 전체 tview 직접 import 제거는 TVR-22~26의 언어 중립 API 제공에 의존하는 미완료 목표입니다. 아래 미배포/미커밋 표기는 검증 당시 시점을 보존한 기록이며 정식 v0.12.0은 여전히 v0.4.0을 포함합니다.


## LORA-76: Tuivory v0.5.0 우선 적용 (2026-10-31, 미배포)

현재 개발 의존성은 `github.com/leechis7/tuivory/go v0.5.0`, 태그 go/v0.5.0, Origin `26da505af4b818bb8c3c272e5970c411d9bde92b`, Sum `h1:KjY4+GAHeY6SULi6++Gv3TaUUS/jTZ2UEpgDsRGmKxo=`입니다. 공개 ShadowView/DimRegion으로 팝업 직접 그리기를 제거하고 소스 편집 명령 목록을 공통 메뉴로 바꾸며 Deprecated 소비를 정리합니다. 기존 폼 크기에 맞춰 Auto 그림자를 오른쪽/아래 1칸으로 명시하고 배경은 원래 색에 Dim만 적용합니다.

모든 tview 사용을 제거한 것은 아닙니다. 이미 있는 Tab/List/Panel/Split/Status API의 제품 연결과, SDK 기능 부족을 구분합니다. TVR-22(테두리 없는 표면), TVR-23(보안 입력/버튼/폼), TVR-24(Editor/TextView)를 사용자 요청에 따라 별도 프로젝트 백로그에 등록했습니다. [직접 UI 사용 분류·요청·검증](audits/lora-76-tuivory050.md)을 참고하세요. 정식 v0.12.0은 여전히 v0.4.0이며 이번 변경은 커밋·게시·배포 전입니다.


## v0.12.0 릴리스 적용

v0.12.0은 Tuivory Go v0.4.0을 고정하고 아래 메뉴·그리드·표면/비활성 개선을 포함합니다. 사용자 배포 승인에 따라 구현·문서·릴리스 준비를 분리해 dev/main PR·CI와 정식 태그로 발행합니다. 아래 미배포 표기·검증 기록은 당시의 구현/검토 시점을 보존한 이력이며 현재 의존성은 v0.4.0입니다. 전체 LORA-76의 남은 앱 패널/분할/하단 작업은 포함하지 않습니다.


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

현재 개발 소스의 패키지는 **`github.com/leechis7/tuivory/go v0.5.0`**입니다. 정식 LazyOra v0.11.0·v0.11.1은 v0.1.0을, v0.10.2는 v0.0.1을 포함합니다. 아래 TUIS 관련 항목은 이전 적용 이력이며 현행 의존성이 아닙니다.

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

최종 UI 목표는 **LazyOra 앱/테스트의 직접 tview import 0개**입니다. Tuivory가 컨트롤·배치 컨테이너·타입/생성자·편집기까지 감싸는 [TVR-26](https://leechis7.atlassian.net/browse/TVR-26), 일반 배치 API인 [TVR-25](https://leechis7.atlassian.net/browse/TVR-25)를 추가 요청했습니다. SDK 내부 호환 별칭은 허용하며 LazyOra 내부 로컬 shim으로 숨기지 않습니다. 현재 v0.5.0만으로 전체 목표를 달성한 것은 아닙니다.

Tuivory의 공통 규격은 현재 Go/Rust와 향후 Python·JavaScript·C·C++를 고려한 언어 중립 설계입니다. LazyOra의 Go 소비와 Tuivory 전체 지원 범위를 구분하며, TVR-22~26도 공통 계약과 언어별 어댑터를 나눠 요청했습니다. 계획 언어를 이미 구현된 것으로 표시하지 않습니다.
