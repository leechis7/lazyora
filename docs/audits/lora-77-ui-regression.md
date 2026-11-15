# LORA-77 UI 통합 회귀 / Integrated UI regression

[한국어 적용 기록](../tuivory-sync.md) · [English integration record](../en/tuivory-sync.md) · [Jira LORA-77](https://leechis7.atlassian.net/browse/LORA-77)

## 한국어

2026-11-13, LazyOra v0.20.0 이후 작업 브랜치에서 공개 [Tuivory Go v0.12.4](https://github.com/leechis7/tuivory/releases/tag/go/v0.12.4)를 사용했습니다. 원본 커밋은 `dd8dc51862c8a139d051cd8eddf412e041b593d9`, 모듈 체크섬은 `h1:kqpqHrX7r3NrZBdq8KUJUzKIBk4eFOP5+vR9Mlq7OTs=`입니다. 패키지 버전은 유지했습니다.

- 40×16 화면에서 F5 문장 목록이 높이 0이 되는 문제를 확인했습니다. 높이가 낮으면 요약보다 문장 목록·결과에 공간을 먼저 주고, DBMS_OUTPUT도 본문을 우선합니다. Tab은 그려지지 않은 항목을 건너뜁니다.
- 40열에서 객체 상세를 열면 Tuivory 분할 뷰가 왼쪽 스키마만 그리는데 오른쪽 탭이 포커스를 받았습니다. 기존 분할 API로 좁을 때 상세를 우선 표시하도록 지정했습니다. 넓어지면 분할 배치로 돌아갑니다.
- 스키마 시험은 첫 행이 SAMPLE_ROWS라고 가정했습니다. 같은 계정에 다른 테이블이 있는 환경에서도 Alt+I로 이름을 찾은 뒤 컬럼·DDL·제약·인덱스를 확인하도록 바꿨습니다. DBMS_OUTPUT 비우기는 화면의 부분 갱신 문자열 대신 복사 결과로 확인합니다.

한영 × 다크·라이트·흑백 × 8·16·256·True Color에서 56개 화면/상태의 **1,344개 조합**, 120×40 → 80×24 → 40×16 → 120×40의 **5,376개 크기 상태**가 통과했습니다(54.22초). Tab·Shift+Tab 192개 조합(53.30초)과 선택 목록 열기·닫기 72개 조합(5.79초)도 통과했습니다. 일반 Go 테스트, vet, Linux ARM64 빌드도 통과했습니다.

Linux ARM64 PTY와 Oracle Free 23.26.3에서 결과 탭 전환, F9/F5 편집기 포커스, DBMS_OUTPUT 복사·비우기, 두 접속의 미커밋 데이터 격리와 개별 롤백을 확인했습니다. 전체화면 토글·Esc 복원은 linux·ansi·xterm·xterm-256color에서 확인했습니다.

2026-11-14 후속 시험에서 스키마 데이터·DDL 탭, 루틴 이후 롤백, 실제 드래그 복사·미리보기, F5·바인드·XLSX, 2,507행 조회·정렬·필터·전체 CSV, 실행 중 도움말과 재접속 보호가 통과했습니다. 이전 시험은 F10을 문맥 메뉴로 쓰고, 활성 커서가 사라져도 비활성 메뉴를 같은 횟수로 이동하며, 화면 차분에 긴 문구가 항상 연속으로 포함된다고 가정했습니다. 현재 키와 실제 복사·결과 동작으로 검증을 보완했습니다. 최종 전체 실행은 미커밋 SQL 조회·복사, 커밋·롤백 후 내역 초기화, 종료 취소와 정상 종료까지 통과했습니다. 루틴 뒤 롤백도 같은 실행에서 통과했으며 제품의 롤백 실패는 재현되지 않았습니다. 시험 뒤 Oracle Free를 내렸습니다. 별도의 PTY 시험에서 TERM 미설정·빈 값·ansi·xterm 기동과 종료, 도움말 4개 언어·테마 조합, VS Code·IntelliJ 키맵도 통과했습니다.

[40열 한글 화면](evidence/lora-77-compact-ko.txt)은 모의 터미널에서 실제로 그린 문자입니다. 좁은 폭에서 일부 버튼·하단 안내가 잘리는 현상은 남습니다. 기존 README 캡처는 일반 크기의 전체 화면 예시로 유지했습니다. 실제 단말의 폰트·SecureCRT·MobaXterm·CentOS·Windows/macOS는 이번 검증 대상이 아닙니다.

별도 사용자 제보에서 `SELECT ROWID, a.* FROM sample_rows a`의 다중 셀 수정과 행 삭제를 함께 적용할 때 오류를 재현했습니다. 삭제를 빼도 원본 `AMOUNT=-0.0001` 비교에서 실패했습니다. 숫자 바인드의 십진 표현을 정규화해 원본 값 비교를 유지하면서 거짓 충돌을 없앴습니다.

## English

On 2026-11-13 the branch after LazyOra v0.20.0 retained published [Tuivory Go v0.12.4](https://github.com/leechis7/tuivory/releases/tag/go/v0.12.4), origin commit `dd8dc51862c8a139d051cd8eddf412e041b593d9` and checksum `h1:kqpqHrX7r3NrZBdq8KUJUzKIBk4eFOP5+vR9Mlq7OTs=`.

- At 40×16 the F5 statement list could receive zero rows. Short panes now prioritize statements and results over the summary, give DBMS_OUTPUT room for its body, and skip undrawn controls when tabbing.
- At 40 columns, the compact object workspace showed the schema browser while focus remained in hidden details. The existing Tuivory split API now shows details first; widening restores the split.
- The schema test assumed SAMPLE_ROWS was the first table. It now filters by name with Alt+I before checking columns, DDL, constraints, and indexes. Output clearing is checked through clipboard behavior instead of relying on a partial terminal redraw.

Across Korean/English, Dark/Light/Black-White and 8/16/256/True Color, **1,344 combinations** of 56 views passed, for **5,376 size states** at 120×40 → 80×24 → 40×16 → 120×40 (54.22s). The 192 Tab/Shift+Tab cases (53.30s) and 72 choice-open/close cases (5.79s) also passed. The normal Go suite, vet and Linux ARM64 build passed.

On Linux ARM64 PTY and Oracle Free 23.26.3, we verified result tabs, editor focus after F9/F5, DBMS_OUTPUT copying and clearing, and isolation and rollback across two live connections. Fullscreen toggling and Escape restoration passed with linux, ansi, xterm and xterm-256color.

Follow-up runs on 2026-11-14 passed docked Data/DDL tabs, rollback after routines, drag-copy and preview, F5/binds/XLSX, 2,507-row paging/sort/filter/full CSV export, help during execution, and reconnect protection. The old test used F10 for a context menu, counted disabled entries after cursor exhaustion, and assumed terminal diffs contained complete long labels. It now follows current keys and checks copied data and results. The final full run passed pending-SQL viewing/copying, clearing the journal after commit/rollback, canceled exit and clean shutdown. Rollback after the routine passed in the same run; a product rollback failure was not reproduced. Oracle Free was stopped afterward. Separate PTY tests passed startup/exit with absent, empty, ansi and xterm TERM, four help language/theme combinations, and VS Code/IntelliJ keymaps.

The [40-column English capture](evidence/lora-77-compact-en.txt) shows the simulated terminal’s rendered text. Some buttons and footer labels remain clipped at this width. The README images remain examples of the standard-width layout. Actual terminal fonts, SecureCRT, MobaXterm, CentOS, Windows and macOS were outside this pass.

Oracle Free reproduced the reported error when applying several cell edits and a row deletion from `SELECT ROWID, a.* FROM sample_rows a`. The comparison with the unchanged `AMOUNT=-0.0001` also failed without the deletion. Normalizing decimal binds prevents the false conflict while preserving the original-value check.

```sh
LAZYORA_UI_AUDIT=1 go test ./internal/ui -run '^TestRuntimeUI(Audit|FocusAudit|ChoiceAudit)$' -count=1
```
