# LORA-79 팝업 경계 검증 / Popup boundary verification

2026-10-28 · `feature/LORA-79-popup-wide-glyph` · 미커밋·미배포 / uncommitted, unreleased

## 한국어

- v0.11.0에서 접속 팝업의 왼쪽 경계 직전에 두 칸 한글을 놓으면 `GetContent`에는 테두리가 있어도 `Show` 이후 실제 `GetContents`에는 빈 칸으로 남는 결함을 재현했습니다. Tuivory Choice 없이 단순 tview Box에서도 발생합니다. 공개 Tuivory Go v0.1.0 / c034a00b1db92ee7ad4abd1ea5956907f1c1a124의 문제로 판정하지 않습니다.
- LazyOra 공통 카드·그림자의 각 행을 칠하기 전에 양쪽 경계를 가로지르는 두 칸 문자를 온전히 지웁니다. 온전한 경계 밖 문자와 스타일을 유지하고 SQL·결과 원문을 바꾸지 않습니다. 팝업을 닫은 뒤 원래 화면을 다시 그릴 수 있습니다.
- 새 회귀 검사의 수정 전 실패를 확인했습니다(접속창 논리 `┌`, 실제 출력 누락). 수정 후 접속·설정·앱 정보 × 한영 × 다크/라이트/흑백 × 8/16/256/True Color를 검사합니다. 각 조합에서 120×36 → 80×24 → 41×16 → 120×36으로 크기를 왕복하고, 한글·CJK·이모지와 결합 문자를 경계에 놓아 매번 2회 그립니다(1,728회 모의 팝업 출력 프레임). 보이는 셀의 문자·결합 문자·스타일을 물리 버퍼와 비교하고 SQL·결과·포커스 보존을 확인합니다.
- 별도 검사에서 외부의 온전한 두 칸 문자·스타일을 보존하고, 팝업을 닫고 배경을 다시 그린 뒤 물리 출력의 보이는 글자·스타일이 원래대로 복원되는지 확인합니다. tcell의 두 번째 칸은 독립 출력 셀이 아니므로 선행 칸과 문자열 폭을 기준으로 검사합니다.
- 이 검사는 tcell 모의 단말 출력 경로 검증입니다. 실제 원격 SecureCRT/MobaXterm·CentOS·폰트 및 신규 실DB 접속 시험은 별도입니다. 기존 ASCII 전 화면·열린 콤보박스·팝업 입력/크기 검사를 함께 수행합니다.
- 커밋·GitHub 게시·CI·배포는 수행하지 않습니다. 검토 상태가 없는 Jira에서는 진행 중을 유지하고 검토 대기를 기록합니다. `test/oracle/run.sh`의 기존 사용자 실행 권한 변경은 보존합니다.

## English

- Reproduced v0.11.0: place a two-cell Korean glyph immediately before the modal's left edge. Logical `GetContent` contains a border, while physical `GetContents` after `Show` omits it. A plain tview Box without Tuivory Choice also reproduces it. This is not attributed to public Tuivory Go v0.1.0 / c034a00b1db92ee7ad4abd1ea5956907f1c1a124.
- Before painting each shared card/shadow row, LazyOra clears both cells of background glyphs crossing either boundary. Preserve intact outside glyphs/styles and original SQL/results, allowing the original page to redraw on close.
- Confirmed the new regression fails before the fix (logical connection-border `┌`, missing physical cell). Cover connection/settings/About × Korean/English × three themes × four color profiles, resizing 120×36 → 80×24 → 41×16 → 120×36. Inject Korean/CJK/emoji and combining marks and draw each twice (1,728 simulated popup output frames). Compare visible characters, combining marks and styles against physical output; preserve SQL/results/focus.
- A separate test preserves intact outside wide glyphs/styles and restores the original visible physical background after closing and repainting. Inspect leading cells and glyph widths because tcell continuation slots are not independently emitted cells.
- This validates simulated tcell output, not new remote-terminal/font/CentOS/live-Oracle sessions. Existing whole-layout ASCII, open-choice, popup-input and resizing regressions are also run.
- No commit, GitHub publication, CI or release is performed. Keep Jira In Progress with review-pending evidence when no Review status exists. Preserve the pre-existing user executable-mode change in `test/oracle/run.sh`.

## 실행 결과 / Results

- 통과: 수정 전 재현 실패 확인 → 수정 후 팝업·폼·열린 Choice·ASCII 대상 검사(8.148초), 전체 `make test` race 검사(UI 123.472초), `make check`, Linux ARM64 `make build` 및 `--version` 확인. 이 시간은 검사 실행 시간이며 제품 성능 수치가 아닙니다.
- 변경 문서의 로컬 링크 20개와 `git diff --check`를 통과했습니다. Tuivory 의존 버전·출처는 유지했습니다. 기본 바이너리 `bin/lazyora-linux-arm64`에 로컬 수정이 반영됐으며 버전은 기존 0.11.0입니다. 정식 릴리스 자산은 갱신하지 않았습니다.
- Passed: confirmed regression failure before the fix; targeted popup/form/open-choice/ASCII checks (8.148s), full `make test` race suite (UI 123.472s), `make check`, Linux ARM64 `make build` and `--version`. These are test durations, not product performance measurements.
- Validated 20 local links in changed documents and `git diff --check`. Tuivory version/source are unchanged. Local `bin/lazyora-linux-arm64` includes the fix and retains version 0.11.0; published release assets are unchanged.
