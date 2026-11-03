# LORA-76 UI 전수 조사 / Runtime UI audit

[한국어 적용 기록](../tuivory-sync.md) | [English integration record](../en/tuivory-sync.md)

후속 / Follow-up: [v0.10.0 적용·재검증 / integration and revalidation](lora-76-tuivory0100.md). 아래 내용은 v0.9.0 당시 기록입니다. The findings below describe the v0.9.0 baseline.

## 한국어

2026-11-02 현재 `feature/LORA-76-tuivory-v080`의 미커밋 작업을 조사했습니다. **배포 가능한 정상 상태로 판정하지 않습니다.** 상단 메뉴 위치 수정은 로컬 검증을 통과했지만, 한글 소실·저색상 대비·좁은 화면의 확인창 결함이 남았습니다. 이번 요청은 조사이며 추가 제품 수정·커밋·게시·릴리스를 수행하지 않았습니다.

### 기준과 조사 방법

- 공개 `github.com/leechis7/tuivory/go v0.9.0`, 태그 `go/v0.9.0`, Origin `3aadb547a40f8be58b55268300dc613b6f1b8db1`, Sum `h1:fY/1eZcDws90Z9Vb60JQ4GruNNBuX9Y9dU450o3Bkz8=`. 정식 LazyOra v0.12.1의 SDK v0.5.0과 구분합니다.
- 실제 앱 생성자와 `Application.ForceDraw()`를 사용해 BeforeDraw·포커스·하단 안내 갱신을 실행했습니다. 기존 개별 위젯 속성 검사만으로 정상이라고 판단하지 않았습니다.
- 53개 화면/변형 × 한국어·영어 × 다크·라이트·흑백 × 8·16·256·truecolor = **1,272개 조합**. 각 조합을 120×40 → 80×24 → 40×16 → 120×40으로 그려 **5,088개 크기 상태**를 조사했습니다. 메인·메뉴·접속·설정·도움말·보안·편집·내보내기·객체·루틴·확인창을 포함합니다.
- 조합 검사 **1,248개 통과, 24개 실패**. 실패는 모두 40열 재접속 확인창의 포커스 버튼 위치입니다. 한글 의미 보존과 대비 결함은 별도 화면 내용 검사/독립 SDK 재현으로 발견했으므로 이 통과 수치에 가려서는 안 됩니다.
- 16종 대화상자의 Tab/Shift+Tab 검사 **192개 조합 통과**. 각 80×24·40×16에서 각각 18회씩 양방향 이동했습니다. 위 재접속 확인창은 이 별도 Tab 행렬에 포함되지 않습니다.
- 접속·설정·스키마의 선택 컨트롤 열기/닫기 **72개 조합 통과**. 한국어 화면과 열린 테마 목록을 포함한 스냅샷 120개를 저장했습니다.
- 진단 코드는 `internal/ui/runtime_audit_test.go`에 있습니다. 일반 테스트에서는 건너뛰는 선택 실행 검사입니다. `LAZYORA_UI_AUDIT=1 go test ./internal/ui -run '^TestRuntimeUI(Audit|FocusAudit|ChoiceAudit)$' -count=1`로 실행합니다. 현재 알려진 재접속 결함 때문에 이 명령의 실패는 예상됩니다. `LAZYORA_UI_AUDIT_DIR`를 지정하면 선택한 화면의 JSON/TXT 캡처를 저장합니다.

### 확인한 결함과 우선순위

| 우선순위 | 문제와 재현 | 원인·담당 | 상태 |
| --- | --- | --- | --- |
| 높음 | 키맵 비교표의 한글 항목이 지워지고 ASCII만 남음. 같은 상세 TextView의 한글은 정상 | 공개 SDK `WideSafeScreen`과 Table의 셀 배경 재칠 충돌. 독립 SDK 모듈에서도 재현 | 미수정, TVR-7에 근거 전달 |
| 중간 | 라이트/8색에서 가로·세로 분할선이 보이지 않음. 80×24 캡처의 비공백 81칸이 같은 전경/배경 | SDK `SplitterStyle()`이 silver/silver를 반환 | 미수정, TVR-7에 근거 전달 |
| 중간 | 재접속 확인창의 기본 취소 버튼이 40열 화면 밖에 배치됨. 한국어 x=40, 폭=10; 영어 x=33, 폭=8 | 앱의 긴 버튼 라벨 + 공유 Modal의 단일 줄 배치. 모든 테마·색상에서 재현 | 미수정, LORA-76에 기록. 기존 코드에도 같은 구성 존재 |
| 중간 | 다크/8색에서 SQL 키워드·함수 등 대부분 silver, 바인드만 olive여서 강조 구분이 약함 | SDK 구문 스타일 정책. 파서가 호출되지 않는 현상과 다름 | 저색상 가독성 개선 요청. 사용자 터미널 원인은 미확정 |
| 낮음 | 비교표 일부 구분선의 전경색과 선택 배경색이 같음 | 공유 비교표 스타일/배경 조합 | 미수정, TVR-7에 함께 전달 |
| 수정 검증됨 | F10 팝업이 실제 상단 라벨과 다른 위치에 열림 | 앱의 메뉴별 화면 균등 분할 계산을 공개 `BarLayout()`/`MenuPlacement()`로 교체 | 앞선 로컬 수정, 미커밋·미배포. 80개 위치 사례 및 실제 PTY 검증 통과 |

모든 문제를 이번 SDK 갱신이 새로 만들었다고 판정하지 않습니다. F10 위치 계산과 재접속 버튼 구성은 기존 HEAD에도 있었습니다. 한글 소실은 현재 공개 SDK만으로 재현되지만 최초 발생 버전은 이 조사에서 이분 탐색하지 않았습니다. 컨텍스트 메뉴가 포인터 위치 대신 화면 좌상단에 열리는 기존 동작도 개선 후보이며 새 회귀로 확정하지 않았습니다.

### 한글 소실의 독립 재현

LazyOra 소스와 `replace`가 없는 별도 Go 모듈에서 Tuivory v0.9.0을 사용했습니다. 다음과 같이 같은 표를 원본 화면과 보호 화면에 각각 그리면 원본은 `한글 SQL 전체 선택`, 보호 화면은 공백과 `SQL`만 남습니다. 배경색이 지정된 표 셀의 후처리가 한글 두 번째 칸을 다시 쓰고, 보호 화면이 이를 겹침으로 판단해 첫 칸까지 지웁니다.

```go
raw := tuivory.NewSimulationScreen("UTF-8")
if err := raw.Init(); err != nil { panic(err) }
defer raw.Fini()
raw.SetSize(60, 10)
table := tuivory.NewTable()
table.SetRect(0, 0, 50, 5)
table.SetCell(0, 0, tuivory.NewTableCell("한글 SQL 전체 선택").
    SetBackgroundColor(tuivory.ColorBlue))
table.Draw(tuivory.WideSafeScreen(raw)) // Compare with table.Draw(raw).
raw.Show()
// Inspect raw.GetContent(x, 0): the Korean glyphs are now spaces.
```

공통 수정의 수용 기준은 실제 겹침 복구를 유지하면서 한글/CJK/emoji의 배경·선택·속성 재칠, SurfaceView/ShadowView 조합, 크기 변경·Show/Sync 후 원문을 보존하는 것입니다. LazyOra에 별도 공통 렌더러 우회를 추가하지 않았습니다. 실제 재현은 Go backend이며 Rust에서도 발생한다고 단정하지 않습니다. 언어 중립 계약으로 TVR-7에 전달했습니다.

### 정상 범위와 한계

- truecolor 실제 실행파일 캡처에서 SQL 키워드 `#8db8ff`, 함수 `#73d7ce`, 일반 식별자 `#e5edf5`를 확인했습니다. 16·256색에서도 구문 역할 색상이 구분됩니다. 흑백 테마의 단색 표시는 의도된 동작입니다. 사용자 실행 환경의 TERM/색상 수는 아직 확인하지 못했으므로 사용자 증상을 해결했다고 표시하지 않습니다.
- 이번 53개 변형 검사는 시뮬레이션 터미널의 실제 앱 그리기입니다. 이전 실제 PTY 검사는 메인·메뉴·접속 등의 제한된 경로에 해당하며, 53개 모두를 실제 원격 터미널에서 확인한 것은 아닙니다. 폰트·SecureCRT·MobaXterm·CentOS 실환경, Windows/macOS는 이번에 시험하지 않았습니다.
- Oracle 메타데이터·결과는 가짜 데이터입니다. SQL 전송을 차단했고 실제 접속·트랜잭션·대용량 실DB 동작을 검증하지 않았습니다.
- 직접 tview/tcell import는 first-party 앱·테스트에서 0개입니다. 그러나 SDK 호환 별칭과 기존 `third_party/tview` 편집기 replace가 남습니다. 전체 의미 컴포넌트 전환 완료와 같은 뜻이 아닙니다. 새 Editor의 Tab/Ctrl+Left·대용량 재배치 문제는 기존 TVR-24에 전달한 별도 의존성입니다.
- 테스트 통과만으로 색상 대비·문자 원문·전체 화면 품질을 보증하지 못했습니다. 이후 회귀 기준에는 셀 메타데이터뿐 아니라 실제 그린 문자 보존·전경/배경 대비를 포함해야 합니다.

### 조사 후 일반 검증

일반 `go test ./internal/ui`는 15.422초에 통과했습니다. 선택 실행 감사 검사의 알려진 실패와 일반 테스트 통과를 구분합니다. `git diff --check`와 이번 보고서·한영 적용 기록의 로컬 링크 37개도 통과했습니다. 제품 코드를 추가 수정하지 않아 바이너리를 다시 빌드하거나 CI를 실행하지 않았습니다.

## English

This audit covers the uncommitted `feature/LORA-76-tuivory-v080` work as of 2026-11-02. **It is not approved as release-ready.** It consumes published Tuivory Go v0.9.0, tag `go/v0.9.0`, Origin `3aadb547a40f8be58b55268300dc613b6f1b8db1`, checksum `h1:fY/1eZcDws90Z9Vb60JQ4GruNNBuX9Y9dU450o3Bkz8=`. Released LazyOra v0.12.1 still contains SDK v0.5.0. This request performed an audit, without additional product fixes, commits, publication or release.

### Coverage and findings

The real application constructor and `Application.ForceDraw()` executed application draw hooks, focus and footer updates on simulated screens. We exercised 53 screen variants × two languages × three themes × four color profiles (8/16/256/truecolor): **1,272 combinations**, each resized through 120×40, 80×24, 40×16 and 120×40 (**5,088 size states**). Main/menu, connection/settings, help/security, editing/export, object/routine and confirmation screens were included. **1,248 combinations passed and 24 failed**, all for the reconnect confirmation's offscreen default button at 40 columns. Additional text-content/contrast inspection found defects that geometry checks did not detect.

Separate Tab/Shift+Tab coverage passed **192 combinations** across 16 dialogs, two languages, three themes and 8/truecolor, with 18 moves in each direction at both 80×24 and 40×16. This separate matrix excludes reconnect. Opening/closing all choice fields in connection/settings/schema passed **72 combinations**. We saved 120 selected Korean/expanded-choice snapshots.

| Priority | Finding | Ownership/status |
| --- | --- | --- |
| High | Korean labels disappear from the keymap comparison table while ASCII survives; its detail TextView remains correct | SDK WideSafeScreen erases wide glyphs during Table's per-cell background repaint. Reproduced without LazyOra or a backend replacement; unresolved, evidence sent to TVR-7 |
| Medium | Light/8 splitters disappear; 81 nonblank divider cells have identical foreground/background in the 80×24 capture | SDK SplitterStyle returns silver/silver; unresolved, reported to TVR-7 |
| Medium | Default Cancel button overflows at 40 columns: Korean x=40,width=10; English x=33,width=8 | Host long labels plus shared Modal's single-row buttons; unresolved, tracked in LORA-76. Same configuration exists in HEAD |
| Medium | Dark/8 SQL syntax is mostly silver, with olive binds, providing weak role distinction | SDK low-color style policy; enhancement requested. The user's actual terminal cause remains unconfirmed |
| Low | Some comparison separators match selected backgrounds | Shared comparison style/background combinations; unresolved, reported to TVR-7 |
| Locally fixed | F10 popups did not align with their top-bar labels | Previous host fix uses public BarLayout/MenuPlacement; 80 placement cases and actual PTY checks passed, still uncommitted/unreleased |

The SDK-only Korean reproduction above draws the same colored Table cell on raw versus WideSafeScreen screens. Raw retains all text; the protected screen retains only ASCII `SQL`. Acceptance must preserve CJK/emoji during style-only repaint, selection, SurfaceView/ShadowView composition, resize and Show/Sync, while retaining real overlap repair. No duplicate consumer rendering workaround was added. Actual evidence is for Go; Rust is not claimed affected. The requested preservation contract is language-neutral.

Not every finding is a new migration regression. The old F10 placement formula and reconnect labels predate this work; the first SDK version with glyph loss was not bisected. Context menus opening at the viewport origin rather than pointer position are another existing improvement candidate, not a proven new regression.

### Reproduction and limits

Run `LAZYORA_UI_AUDIT=1 go test ./internal/ui -run '^TestRuntimeUI(Audit|FocusAudit|ChoiceAudit)$' -count=1`; the known reconnect failures currently make this diagnostic command fail. Set `LAZYORA_UI_AUDIT_DIR` for selected JSON/TXT captures. These opt-in diagnostics skip in the ordinary suite.

Actual truecolor binary captures distinguish SQL keyword `#8db8ff`, function `#73d7ce` and ordinary identifier `#e5edf5`; 16/256 profiles also distinguish syntax roles. Monochrome theme is intentionally monochrome. The user's TERM/color capability is unknown, so the reported highlighting symptom is not declared resolved. Earlier actual PTY checks cover limited main/menu/connection paths, not all 53 variants. This audit does not certify real remote terminal fonts, SecureCRT/MobaXterm/CentOS, Windows or macOS. Oracle data was synthetic and SQL submission was blocked; no live database, transaction or large-result behavior was verified.

First-party application/tests have zero direct tview/tcell imports, but SDK compatibility aliases and the existing `third_party/tview` editor replacement remain. This is not full semantic component migration. Shared Editor Tab/Ctrl+Left and large-document layout concerns were separately reported in TVR-24. Future regression gates must inspect rendered glyph preservation and contrast, beyond widget metadata or passing unit tests.

The ordinary `go test ./internal/ui` passed in 15.422 seconds; this does not override the opt-in audit failures. `git diff --check` and 37 local documentation links passed. No further product-code change required a binary rebuild, and GitHub CI was not triggered.
