# LORA-72 트리 적용 검증 / Tree integration validation

2026-10-28 · 개발 브랜치 `feature/LORA-72-tuivory-tree` · 미커밋·미배포 / uncommitted and unreleased

## 한국어

- 공개 모듈: `github.com/leechis7/tuivory/go v0.1.0`, 태그 `go/v0.1.0`, 커밋 `c034a00b1db92ee7ad4abd1ea5956907f1c1a124`. 모듈 합계 `h1:v/9M5vOlgsZkB3A6VCAhv6BviCNZuQlFYuyG1por648=`.
- `Tree/TreeState/TreeView`를 import하여 실제 탐색·렌더링을 위임합니다. 기존 공개 Styles를 새 컴포넌트의 의미 역할에 연결하고 후보 팔레트를 혼합하지 않습니다. Oracle 데이터·비동기 작업·상세 동작은 LazyOra가 소유합니다.
- 기존 선택 사용자 단위 객체 목록 조회와 유형/이름의 로컬 필터는 유지합니다. 각 테이블·뷰의 컬럼은 펼칠 때만 별도 조회합니다. 모든 객체의 컬럼을 일괄 조회하지 않습니다. 접힘·필터·종류 변경 후 재사용하며 목록 새로고침/사용자 변경/재접속 시 폐기합니다.
- 전용 테스트 6개: 컬럼 펼침/재사용과 SQL·미커밋 상태 보존, 조회 실패/취소 후 재시도 및 새 목록에 도착한 옛 응답 차단, 여러 빈 가지·필터 캐시, 12개 테마/색상 모의 렌더링·포커스 이중선·한글/리터럴 이름, 5천 객체 탐색, 컬럼의 설명·편집 대상이 부모 객체임을 검증했습니다.
- `go test -race ./...` 통과. 최종 UI 패키지 53.672초는 전체 테스트 실행 시간이며 제품의 조회 성능 측정값이 아닙니다. `go vet ./...`, `make build` 통과. 바이너리의 `go version -m`에서 Tuivory v0.1.0, Linux ARM64를 확인했습니다.
- `--version`은 릴리스 준비 전이므로 기존 v0.10.2입니다. 정식 v0.10.2 태그·릴리스·다운로드 파일을 수정하지 않았습니다. 신규 기능 릴리스 준비 때 기존 minor 버전 규칙을 적용합니다.
- 기존 UI 회귀와 로컬 문서 링크·한영 가이드 목차 검사 통과. 실제 SecureCRT/MobaXterm/CentOS 단말·실DB 신규 연결은 시험하지 않았습니다. Oracle 패키지 테스트의 캐시 통과를 새 실DB 검증으로 해석하지 않습니다. GitHub CI·게시·배포는 미실행입니다.
- LORA-73~76의 다른 신규 컴포넌트 통합은 별도 작업입니다. Jira에 검토 상태가 아직 없어 LORA-72는 진행 중 상태로 검토 대기 댓글을 남깁니다. 완료는 사용자 검토 후 처리합니다.

## English

- Public module: `github.com/leechis7/tuivory/go v0.1.0`, tag `go/v0.1.0`, commit `c034a00b1db92ee7ad4abd1ea5956907f1c1a124`; checksum `h1:v/9M5vOlgsZkB3A6VCAhv6BviCNZuQlFYuyG1por648=`.
- Import public Tree/TreeState/TreeView for rendering/navigation. Adapt existing public Styles roles without mixing the separate candidate palette. Oracle data, asynchronous requests and details actions remain application-owned.
- Preserve the existing owner-scoped object-list query and local type/name filtering. Fetch each table/view’s columns only on expansion, never all objects’ columns in bulk. Reuse collapsed/filtered/type-switched branches within the list; discard on refresh, owner change or reconnect.
- Six dedicated tests cover expansion/caching and work preservation, failure/cancellation retry and stale response rejection, multiple empty branches and filtering, simulated rendering across 12 theme/color profiles with focused double borders and Korean/literal names, 5,000-object navigation, and column actions targeting their parent object.
- Full `go test -race ./...`, `go vet ./...` and Linux ARM64 `make build` passed. The final UI package test runtime of 53.672 seconds is not a product query-performance benchmark. Binary module metadata confirms Tuivory v0.1.0 and Linux ARM64.
- `--version` remains v0.10.2 until release preparation. Existing v0.10.2 tags/releases/downloads are untouched. A future feature release follows the existing minor-version policy.
- Existing UI regression, local document links and eight sections in each embedded language guide passed. No fresh live DB or SecureCRT/MobaXterm/CentOS terminal test was performed. Cached Oracle package tests are not new live-database evidence. Remote CI/publication/release was not run.
- Other new components remain in LORA-73–76. Until Jira exposes Review, leave LORA-72 In Progress with a review-pending comment; Done requires explicit user review approval.
