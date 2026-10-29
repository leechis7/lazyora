# 작업 규칙

- 문서를 새로 작성하거나 수정할 때는 한국어와 영어를 함께 작성·갱신한다. 한국어를 기본으로 유지하며 README·사용/개발 안내·릴리스 설명 등 대응 문서의 내용·버전·명령·단축키·검증 범위와 언어 전환 링크를 맞춘다. 기존 문서의 변경 없는 과거 이력을 일괄 번역하는 작업은 별도 요청 때 수행한다.

- 이 저장소와 대화에서는 LazyOra만 개발한다. Tuivory 자체 개발·견본·일감은 별도 Tuivory 프로젝트에서 관리한다. UI 스타일 작업은 Tuivory에 게시된 Go 스타일 패키지의 버전/커밋을 확인하고, 공개 패키지를 import하여 변경된 스타일을 LazyOra의 LORA 일감으로 적용·검증한다. 확인한 출처와 적용/보류 판단은 `docs/tuivory-sync.md`에 기록한다.

- 공통 UI의 그리기·테두리/겹침 보호·컨트롤·스타일은 Tuivory 공개 API를 우선 사용한다. LazyOra에는 Oracle 동작과 데이터/상태·명령 연결 및 필요한 표시 어댑터만 둔다. 공통 기능이 부족하면 Tuivory 프로젝트의 제공 범위를 확인해 보완을 요청하고, 같은 UI 수정을 양쪽에 독립 구현하지 않는다. 패키지에 기능이 제공되면 기존 우회 구현을 제거하고 import 기반으로 전환·회귀 검증한다.
- Shared UI policy: prefer published Tuivory APIs for rendering, borders/overlap repair, controls and styles. Keep Oracle behavior, application data/state/commands and required display adapters in LazyOra. Check/request missing shared capabilities in the separate Tuivory project rather than maintaining parallel UI fixes. Replace local workarounds with public package APIs once available and verify integration regressions.

- Tuivory는 TUI 컴포넌트 라이브러리다. 공통 컨트롤의 표시·내부 조작·스타일·옵션은 Tuivory를 우선 사용하되, 실제 화면 조합·배치·크기/순서·컴포넌트 사이 포커스·앱 전역 단축키·데이터/명령 연결은 LazyOra 책임이다. 앱에 UI 조합 코드가 존재한다는 이유만으로 미이관 결함으로 판단하거나 전체 화면을 라이브러리로 옮기지 않는다.
- Component boundary: Tuivory owns reusable control rendering, internal interaction, styles and options. LazyOra owns actual screen composition/layout/sizing/order, cross-component focus, global shortcuts and data/action binding. Host UI composition is normal; do not classify all UI code as an unfinished migration or transfer entire application screens into the library.

- 작업 시작 전에 Jira의 관련 일감을 확인한다. 기존 일감이 있으면 요구사항과 진행 내역을 그 일감에 추가한다. 없으면 사용자에게 등록 여부를 묻고, 등록하기로 승인받으면 등록한 뒤 작업한다.
- 2026-10-27 사용자 지시로 스크럼에서 칸반으로 전환한다. 새 스프린트 생성·기간 설정·일감 배정을 하지 않고 백로그 우선순위와 `할 일 → 진행 중 → 검토 → 완료` 상태로 관리한다. 기존 스프린트 이력은 보존한다. 사용자 지시가 없는 한 한 번에 한 일감을 진행하고, Tuivory 패키지 등 선행 의존성이 준비되지 않은 작업은 할 일에 둔 채 사유를 기록한다.
- 실제 작업을 시작하면 Jira 일감을 `진행 중`으로 전환한다. 2026-10-27 검토 단계 추가 지시가 2026-10-21 자동 완료 기준보다 우선한다. 구현·필요한 검증을 마치면 근거와 남은 CI/게시 여부를 댓글로 남기고 `검토`로 전환한다. 사용자 검토 완료가 명시된 뒤에만 `완료`로 전환한다. 수정 요청을 받아 재작업할 때는 `진행 중`으로 되돌리고 검증 후 다시 `검토`로 전환한다. 검토 상태가 아직 Jira에 없으면 `진행 중`을 유지하며 검토 대기임을 댓글로 기록하고, `완료`로 대체하지 않는다. 미구현 작업이나 하위 작업이 남은 에픽은 완료 처리하지 않는다. 아래 CI 일괄 실행 기간에는 로컬 검증 완료와 GitHub 통합 CI 대기를 구분해 기록한다.
- Kanban policy (2026-10-27): prioritize the backlog without creating or assigning sprints, preserve historical sprints, and use To Do → In Progress → Review → Done. Work on one issue at a time unless instructed otherwise. Leave dependency-waiting work in To Do with a recorded reason. After implementation and required verification, move to Review with evidence and any pending CI/publication noted; move to Done only after explicit user review approval. Return requested rework to In Progress. If Jira has no Review status yet, keep In Progress with a review-pending comment rather than using Done. This supersedes the 2026-10-21 automatic completion rule; do not close epics with remaining work. Existing commit, publication and sequential-branch approval rules still apply.
- 사용자가 해당 변경의 커밋을 명시적으로 승인할 때만 커밋한다. 구현·수정 요청만으로 자동 커밋하지 않는다.
- GitHub 업로드는 사용자가 명시적으로 요청하거나 승인할 때만 한다. 구조 정리 중인 변경은 로컬에 보존한다.
- 커밋 메시지의 설명은 한국어로 작성한다. Git 훅이 요구하는 `feat:`·`fix:` 등의 접두어는 사용할 수 있다.
- 릴리스에 신규 기능이 포함되면 minor 버전을 올리고 patch를 0으로 초기화한다(예: 0.6.0 → 0.7.0). 버그 수정만 포함되면 patch를 올린다(예: 0.6.0 → 0.6.1). 0.x 버전에도 동일하게 적용한다.
- major는 제품 버전으로 사용자가 결정한다. 에이전트는 major를 임의로 올리지 않는다. 기능 개수·완성도·호환성 변경을 이유로 자동 상향하지 않으며, 사용자의 명시적인 major 변경 지시 전까지 현재 major를 유지한다(예: 0.9.0 다음 기능 릴리스는 0.10.0).
- 개발은 최신 `dev`에서 `feature/LORA-번호-주제` 브랜치를 만들어 진행한다. 완성한 변경은 검증 후 `dev` 대상 PR로 통합하며 릴리스 준비 시 `dev`를 `main`에 병합한다. 정식 릴리스는 `main`에 포함된 커밋의 `vX.Y.Z` 태그로 발행한다. 세부 절차는 `docs/development.md`를 따른다.
- 브랜치 작업은 순차적으로 진행한다. 현재 브랜치의 구현·검증, 사용자 검토, 승인된 커밋·병합과 브랜치 정리를 마친 뒤 다음 브랜치를 시작한다. 검토·승인 대기 중에도 임의로 다른 브랜치를 만들거나 전환하지 않는다. 사용자가 다른 브랜치로 옮기라고 명시한 경우에만 예외로 한다.
- 2026-10-21 사용자가 승인한 남은 백로그 작업은 CI를 일괄 실행한다. 일감별 로컬 구현·검증·승인된 커밋·로컬 dev 병합·작업 브랜치 정리를 순차 진행하고, 전체 구현 후 통합 게시와 GitHub CI를 진행한다. 중간 게시·PR·CI 대기는 생략하고 기존 실행 중인 CI는 유지한다. 게시 전 로컬 dev가 origin/dev보다 앞설 수 있으므로 원격 상태로 덮어쓰지 않는다.
- Oracle Docker 시험 환경의 설정, 환경 파일, 샘플 SQL, 실행 스크립트와 실DB 테스트는 `test/oracle/`에 모은다.

- 로컬 개발·검증의 기본 바이너리는 Linux ARM64만 빌드한다(`make build`). 다른 OS·아키텍처의 로컬 바이너리는 사용자가 요청할 때만 생성한다. 전체 지원 대상 빌드는 GitHub CI·Release에서 `make build-all`로 수행한다.
