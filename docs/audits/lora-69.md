# LORA-69 색상·테마 조사

> 날짜별 검증 보관 기록 / Dated verification record. 아래 수치와 커밋·게시 상태는 해당 검사 시점의 기록입니다. 현재 v0.10.2에는 Tuivory v0.0.1이 배포되어 있습니다. / Tuivory v0.0.1 is now shipped in v0.10.2. [현행 문서 / Current documentation](../index.md).

2026-10-23 · 기준: v0.10.0, dev `e7d40a0c3732d758987a791b449a7404c02a6e28`

8색과 16색은 앱에서 같은 제한 팔레트를 사용하지만 실제 표시 결과까지 같지는 않습니다. tcell의 색상 근사, 16색에서 유지되는 밝은 색, 터미널의 굵게 처리 방식이 함께 영향을 줍니다. 따라서 16색이 항상 더 나쁘다는 결론보다는 아래의 실제 결함과 표시 조건을 나누어 수정하는 것이 적절합니다.

이번 변경은 조사 도구와 보고서뿐입니다. 제품 팔레트·UI 동작·버전은 변경하지 않았으며 DB 접속이나 SQL 실행도 하지 않았습니다.

## 비교 자료

- [오프라인 화면 비교](lora-69-gallery.html): 화면, 한·영 UI, 초기/테마 왕복, 굵게 처리 방식을 선택합니다. 각 그림을 클릭하면 확대됩니다. 최신 Chromium 계열 브라우저에서 파일을 여세요. 외부 서버나 네트워크 연결은 필요 없습니다.
- [화면별 점검표 CSV](lora-69-matrix.csv): 6,432개 분석 행. 같은 화면에 글꼴 굵기만 적용하는 모델과 밝은 전경색을 적용하는 모델을 각각 계산합니다.
- [공통 역할별 색상 CSV](lora-69-roles.csv): 원본 색과 제한 팔레트에서의 색, 대비를 비교합니다.

| 테마 | 8색 | 16색 | 256색 | True Color |
| --- | --- | --- | --- | --- |
| 다크 | 조사 | 조사 | 조사 | 조사 |
| 라이트 | 조사 | 조사 | 조사 | 조사 |
| 흑백 | 조사 | 조사 | 조사 | 조사 |

각 조합은 67개 화면·상태 × 2개 언어 × 초기/왕복 2회로 268프레임이며, 전체 3,216프레임입니다. 모든 프레임은 128×40 셀입니다. 코드의 고정 이름 `AddPage` 42종 모두 캡처에 포함됩니다. 이것은 가능한 모든 입력값·창 크기·포커스·DB 오류를 전부 시험했다는 뜻은 아닙니다.

## 확인한 문제와 개선 위치

| 우선순위 | 재현 조건·증상 | 근거·원인 | 수정 위치·제안 |
| --- | --- | --- | --- |
| 높음 | 마스터 암호 생성/해제 및 암호 관리 안내문이 다크 8/16색, 흑백 모든 색상 수에서 보이지 않음 | `master_create`, `master_unlock`, `vault` 초기 프레임에서 안내문 전경/배경이 모두 검정. 다크 256/True Color에서도 대비가 낮음 | 공통 `styleForm`과 읽기 전용 Form TextView의 역할 분리. 입력 글자색을 안내문에 적용하지 않도록 개선 |
| 높음, 터미널 설정에 따라 발생 | 선택 행·수정 셀이 흐릿하고 라이트 바인드 변수가 거의 안 보임 | 기본 0~7색을 굵게 표시할 때 8~15색으로 바꾸는 모델에서 검정→회색. 수정 셀 회색/올리브 대비 1.062, 현재 셀 회색/청록 1.209. 라이트 16색 바인드 노랑/흰색 1.074 | 공통 선택·그리드·SQL 토큰 스타일. 굵게 속성과 전경색을 함께 검토하고 두 굵게 모델에서 대비 유지 |
| 중간 | 테마를 바꾸고 원래대로 돌아오면 처음과 다른 색이 남음 | `main`, 다크 16색 초기 SQL 탭은 흰 글자/남색 배경, 왕복 뒤 검정/은색. 공통 색상 치환에서 역할이 합쳐진 뒤 복원이 불완전함 | `applyTheme`의 색상 값 치환을 역할별 스타일 재적용으로 개선. SQL 탭·메뉴·팝업 등을 함께 확인 |
| 중간 | 흑백 테마의 ‘복사 내용 보기’ 제목이 사라짐 | `copy`, 모든 색상 수: 제목 흰색/흰색. `copyPreview`가 입력 배경을 적용한 뒤 `refreshBorders`가 일반 제목색을 재적용 | 공통 테두리/제목 스타일에 실제 배경 대비를 반영. 본문 입력 글자색은 유지 |
| 낮음 | 접속 전 DBMS_OUTPUT ‘출력 켜기’ 버튼이 8색에서 안 보임 | `output`: 비활성 버튼의 남색 전경/파랑 배경이 둘 다 남색으로 근사됨 | 비활성 버튼 공통 스타일 검토. 활성 버튼 결함과 구분해야 함 |
| 낮음 | 라이트 8색 일부 비활성 메뉴 문구가 안 보임 | `submenu`, `context`: 회색 계열이 같은 은색으로 근사됨. 실행할 수 없는 항목임 | 비활성임을 유지하되 읽을 수 있는 스타일로 통일 |

숨김 여부와 관계없이 표의 대비 수치는 tcell 내장 RGB 참조값을 사용한 계산입니다. 실제 터미널에서 사용자가 ANSI 색을 변경하면 수치가 달라집니다.

입력 placeholder 역할도 라이트/흑백 8색에서 은색/은색이 될 수 있습니다. 다만 현재 일반 입력의 실제 placeholder 사용은 제한적이므로 이를 광범위한 현행 화면 결함으로 집계하지 않았습니다. NULL 직접 편집 장면은 별도 캡처했으며 Dim 처리는 아래 분석 제한을 적용합니다.

## 16색 개선 방향

1. 안내문과 입력 텍스트를 분리하고, 제목은 실제 배경을 기준으로 정합니다. 이는 색상 수와 무관한 공통 컴포넌트 문제입니다.
2. 제한 색상에서 선택·포커스·미커밋 셀의 전경/배경을 먼저 정한 뒤 굵기와 밑줄을 보조 수단으로 사용합니다. 굵게가 밝은색으로 변하는 환경에서도 내용을 읽을 수 있어야 합니다.
3. 16색에서 흰색·밝은 청록·밝은 자홍 등이 그대로 남아 8색보다 강하게 보입니다. 8색과 16색의 역할별 목표색을 명시하고 강조색을 줄이는 방향이 적절합니다. 단순히 전체 색상 수를 8로 강제하는 방식은 권하지 않습니다.
4. 테마 왕복 시 현재 테마의 역할에서 스타일을 다시 만듭니다. 흑백으로 합쳐진 색상 값을 역으로 추측하지 않습니다.
5. 라이트 편집기의 흰 바탕 요구사항은 유지하고 SQL 토큰과 선택 스타일을 조정합니다.

예상 수정량은 공통 폼·제목 0.5~1일, 선택/ANSI 스타일 0.5~1일, 테마 복원 1~2일, 회귀 및 실제 터미널 비교 약 1일로 **약 3~5일**입니다. 개발 계획용 추정이며 실제 원격 터미널 검증 결과에 따라 달라질 수 있습니다. 후속 제품 수정은 별도 사용자 지시 후 진행합니다.

## 방법과 해석 제한

- `internal/ui/theme_audit_test.go`는 실제 앱 생성자와 위젯을 tcell SimulationScreen으로 그립니다. `Colors()`만 변경해 제한 팔레트 선택을 재현하고, 셀 스타일은 tcell v2.13.10의 `FindColor` 및 내장 팔레트로 근사합니다.
- DB 작업자는 정지시키고 요청을 메모리 채널에만 받습니다. 샘플 스키마·SQL·결과·암호 안내·오류를 사용하며 네트워크 접속, 실제 암호 저장, 커밋/롤백은 수행하지 않습니다. 임시 설정 파일은 Go 테스트 임시 폴더에만 생성됩니다.
- 왕복은 각 화면을 유지한 상태에서 dark → light → blackwhite → 원래 테마 순으로 변경합니다. 1,608개 초기/왕복 쌍 중 1,188쌍에서 렌더링 차이가 있었습니다. 이 수에는 레이아웃·텍스트·스타일 분할 변화가 포함되므로 **1,188개 버그를 의미하지 않습니다.** 보고서의 SQL 탭 차이는 개별 확인한 사례입니다.
- 색상 수는 단말 능력 모델입니다. 실제 `TERM=ansi`, `linux`, `xterm`, `xterm-256color`의 입력·마우스·커서 호환성을 검증한 것이 아닙니다. SecureCRT·MobaXterm 실제 실행, 사용자 ANSI 팔레트, 폰트, 작은 창/다양한 배율은 **별도 확인 필요**입니다.
- 대비 4.5:1은 일반 텍스트의 조사 기준으로만 사용합니다. [W3C 설명](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html)의 비활성 컨트롤 예외 등을 고려해야 하며 이 보고서는 터미널 접근성 인증이 아닙니다. CSV의 낮은 대비 개수를 곧바로 결함 수로 해석하지 마세요.
- [xterm 공식 문서](https://www.invisible-island.net/xterm/manpage/xterm.html)의 `boldColors`는 기본 0~7 전경색을 8~15로 표시할 수 있음을 설명합니다. 이 동작을 켠 모델과 글꼴 굵기만 바꾸는 모델을 비교했으며 모든 터미널이 동일하게 동작한다고 가정하지 않았습니다.
- 자동 대비 분석은 영문·숫자·한글이 포함된 run을 집계하며 공백·선만 있는 run은 제외합니다. Dim run은 팝업 뒤 배경의 오탐을 줄이기 위해 제외합니다. 따라서 활성 Dim NULL·장식 테두리는 CSV 집계만으로 판단할 수 없으며 갤러리에서 확인해야 합니다. 비활성 메뉴는 따로 제외하지 않으므로 보고서에서 구분했습니다.
- 갤러리의 Dim 투명도와 글꼴은 시각화용입니다. 실제 에뮬레이터 스크린샷이 아닙니다. 불투명 팝업 내부에 이전 화면의 같은 좌표 문구가 나타나는 뚜렷한 사례는 이번 비교에서 확인하지 못했지만, 모든 조합의 마우스·재그리기 동작을 보장하지 않습니다.

## 재실행

```sh
LAZYORA_THEME_AUDIT_DIR=/tmp/lazyora-theme-audit go test ./internal/ui -run '^TestThemeAudit$' -v -count=1
python3 tools/theme_audit.py /tmp/lazyora-theme-audit/raw.json /tmp/lazyora-theme-audit/report
sh tools/terminal-colors.sh
```

Go가 mise로 관리된다면 위 `go` 앞에 `mise exec --`를 붙일 수 있습니다. 일반 테스트에서는 이 점검을 건너뜁니다. 기본 산출물은 테스트 임시 파일이 아닌 지정한 `/tmp` 경로에 남습니다. 대용량 raw JSON과 상세 분석 JSON은 저장소 산출물에 포함하지 않습니다.

마지막 명령은 실제 단말의 색과 굵게 효과를 비교하는 ANSI 견본만 출력합니다. TERM·사용자 팔레트·터미널 설정을 변경하지 않습니다. 실제 단말 검증 시 이 출력과 갤러리의 main/edited/connect/vault/copy/comparison 장면을 비교하면 됩니다.

## 검증 기록

- 12조합 × 67장면 × 한·영 × 초기/왕복 = 3,216프레임 생성 통과.
- 고정 이름 AddPage 42종 중 누락 0종.
- 8/16색 × 3테마 × main/vault/comparison × 한·영 대표 조합 race 검사 통과.
- `go vet ./internal/ui` 통과, 환경변수를 지정하지 않은 일반 실행에서 점검 테스트 Skip 확인.
- Python 구문 검사, POSIX 셸 구문 검사, Chromium에서 비교 화면 로딩 확인.

## 화면 목록

about, arguments, bind, busy, cell, comparison, comparison_detail, completion, confirm, connect, connect_choice, connect_input, context, copy, definition_constraints, definition_ddl, definition_relations, describe, diagnostics, edited, editor_actions, error, exit_clean, exit_pending, exit_uncertain, export, file, file_unsaved, grid_edit, grid_inline, grid_inline_null, guide, help, help_body, history, issue_report, main, master_create, master_unlock, menu, object, output, pending, plan, production, range, reconnect, release_history, resize, result_filter, results, routine_grid, routine_input, routine_results, routines, schema, script, search, settings, source, source_errors, sql_confirm, submenu, tab_rename, theme_choice, vault, workspaces

42개 페이지 ID: about, arguments, bind, cell, completion, connect, copy, definition, describe, editor_actions, exit, export, file, file_unsaved, grid_edit, grid_inline, help, history, keymap_compare, main, master, menu, object, output, pending, plan, program_confirm, reconnect, reference, result_filter, routine_grid, routine_input, routine_results, routines, script, search, settings, source, sql_confirm, tab_rename, vault, workspaces

## 공식 자료 조사와 스타일 통합 설계

추가 사용자 요청에 따라 TUI 공식 문서와 UI 디자인 시스템을 조사했습니다. [LazyOra TUI 스타일 가이드 제안](../ui-style-proposal.md)에 근거 자료, 테마/색상 수 정책, 컴포넌트 상태, 간격, 이관 순서와 검증 기준을 정리했습니다. [12개 프로필의 핵심 색상 후보](../ui-style-palette-candidate.csv)는 제품에 아직 적용하지 않은 검토 자료입니다.

추가 문서화: 향후 독립 TUI 스타일 가이드 프로젝트를 위한 [공통 원칙과 이관 계획](../tui-style-guide/README.md)을 준비했습니다. 현재는 문서 정리 단계이며 별도 저장소·범용 라이브러리 구현은 시작하지 않았습니다.

후속 사용자 승인에 따른 공통 스타일 적용 결과는 [구현·검증 기록](lora-69-implementation.md)에서 확인합니다. 원본 조사 자료는 수정 전 근거로 보존합니다.
