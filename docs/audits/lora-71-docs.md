# LORA-71 문서 전수 점검 / Documentation review

2026-10-24 · 기준 / Baseline: v0.10.2, Tuivory v0.0.1 · 로컬 검증 기록 / Local verification record

## 한국어

작성 문서 전체를 현행 안내·날짜별 기록·설계 보관 자료로 분류했습니다. 의존성의 원본 `third_party/tview/` 문서·라이선스는 덮어쓰지 않았으며, 로컬 포크 설명을 확인했습니다. 문서별 목록과 링크 검사 결과는 [점검 목록](lora-71-docs-inventory.json)에 있습니다.

- 한국어 README에 Tuivory 도입 시점·정확한 모듈·버전·프레임워크 역할과 별도 설치 불필요를 추가하고 영문 설명과 맞췄습니다.
- F5 DBMS_OUTPUT ‘미지원’을 수집 지원으로, 모든 SQL 탭의 거래 공유를 같은 접속 작업 공간 안의 탭으로, 현재 행의 굵기 표시를 배경 표시로 수정했습니다. 마스터 암호 8자 이상/1,024바이트 이하 조건도 맞췄습니다.
- 상세 영문 사용·개발·UI·패키지 갱신·호환성·Oracle 시험 안내와 v0.10.2 영문 설명을 보완했습니다. 내장 한영 가이드는 기존 8개 주제 구조를 유지하고 Tuivory·검색·패널 크기 조절을 반영했습니다.
- GitHub 이슈/PR 템플릿을 한영 병기로 갱신했습니다. 보안 안내에는 선택적 암호화 보관함과 같은 OS 계정/설정 경로 공유 범위를 명시했습니다.
- 완료된 패키지 도입·CI와 오래된 준비 계획·당시 미게시/대기 상태를 구분했습니다. 지난 릴리스·벤치마크의 측정값이나 당시 검증 주장을 현재 결과로 고쳐 쓰지 않았습니다.

문서 링크·앵커, README/앱 버전, go.mod의 Tuivory 모듈, Makefile·시험 명령, 내장 가이드 주제 수와 실제 소스의 문장/행/바인드/암호 한도를 대조합니다. 내장 가이드와 CHANGELOG는 실행파일에 포함되므로 회귀/race·vet·기본 ARM64 빌드로 확인합니다. 실제 Oracle·구형 커널·원격 터미널·성능 시험을 새로 수행한 것은 아닙니다. 로컬 검사와 이후 GitHub 게시/CI 근거는 Jira LORA-71에서 구분해 기록합니다. 문서 게시만으로 v0.10.2 태그나 기존 바이너리를 바꾸지 않습니다.

## English

All authored Markdown is classified as current guidance, dated evidence or design history. Upstream third-party documents/licenses are not rewritten; the local tview fork notes were checked. The [inventory](lora-71-docs-inventory.json) records file scope and link results.

The refresh documents Tuivory's exact module/version and responsibilities in both README languages; corrects F5 output support, workspace-local transaction sharing and current-row styling; aligns master-password limits; expands English usage/development/UI/integration/compatibility/testing guidance; and updates both offline guides without changing their eight-topic structure. Issue/PR templates are bilingual. Credential/security guidance distinguishes profiles from optional encrypted storage and shared-account scope.

Current published integration/CI is separated from dated preparation plans and earlier unpublished/pending records. Historical release facts and measured benchmark results are preserved rather than reinterpreted as new verification.

Checks cover local links/anchors, app/README/module versions, documented Make targets/test commands, guide topics and source-defined limits. Embedded guide/changelog changes are validated with regression/race, vet and the default ARM64 build. No fresh Oracle, old-kernel, remote-terminal or benchmark tests are implied. Jira LORA-71 separates local evidence from subsequent publication/CI. Publishing docs does not replace the existing v0.10.2 tag or binaries.

## 로컬 검사 결과 / Local results

- Markdown 60개(현행 36, 날짜별 기록 18, 설계 기록 6), 로컬 링크·앵커 340개: 오류 0. / 60 Markdown files (36 current, 18 dated, 6 design), 340 local links/anchors: zero errors.
- 앱/한영 README v0.10.2, Tuivory v0.0.1, 내장 가이드 8개 주제, 소스 한도와 Oracle Make 타깃 대조 완료. / Version/module, eight guide topics, source limits and Oracle Make targets verified.
- `go test -race ./...`, `go vet ./...`, `make build`: 통과 / passed. `--version`: `LazyOra v0.10.2`.
- `git diff --check`: 통과 / passed. 로그 / log: `/tmp/lazyora-docs-refresh-validation.log`.

이 결과는 커밋 전 로컬 검사이며 이후 게시/CI 기록은 Jira LORA-71을 기준으로 확인합니다. / These are pre-commit local results; subsequent publication/CI evidence is recorded in Jira LORA-71.
