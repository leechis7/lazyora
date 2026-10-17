# 변경 이력

사용자에게 영향을 주는 변경을 버전별로 기록합니다. 날짜는 `YYYY-MM-DD` 형식이며, `Unreleased`는 다음 릴리스 후보입니다.

## [Unreleased]

### 추가

- 항상 표시하는 종료 확인 팝업. 미커밋 가능 내역이 있으면 커밋 후 종료·롤백 후 종료·계속 사용을 선택하고 변경이 없어도 종료 여부를 확인합니다.

### 수정

- 메인 화면에서 SQL 편집기·결과 표의 포커스와 관계없이 F5 커밋·F6 롤백을 표시합니다.
- 짧은 TNS ACCEPT 응답을 처리하는 go-ora 업스트림 수정을 적용해 구형 Oracle 접속 시 발생한 슬라이스 범위 패닉을 방지합니다.
- 일부 SecureCRT 환경에서 기본 커서 모양 명령의 `q`가 출력되는 문제를 방지합니다.
- 8·16색 터미널에서 입력창 테두리는 파랑, 결과·대화창 테두리는 청록으로 표시하고 포커스된 창의 선을 굵게 표시합니다. 256색·True Color 테마는 유지합니다.

### 개발·문서

- `dev` 기반 feature 개발과 PR 검증, `main` 버전 태그 기반 릴리스 절차를 추가합니다.
- GitHub Actions에서 테스트·vet·Linux AMD64/ARM64 빌드와 릴리스 파일 생성을 구성합니다.
- 변경 이력, 기여·보안·지원·행동 규칙과 이슈·PR 템플릿을 추가합니다.

## [0.0.1] - 2026-10-15

첫 공개 릴리스입니다. 대상 커밋은 `8db2685`입니다.

### 추가

- Oracle Service Name/SID 접속 설정과 별칭별 접속 정보 저장. 비밀번호는 저장하지 않습니다.
- SQL 편집기, 최대 1,000행의 조회 결과 표, 수동 COMMIT/ROLLBACK 및 실행 취소 요청.
- F1 도움말, Ctrl+Enter/F4 실행, 포커스에 따라 바뀌는 하단 단축키 안내.
- 한글·영문 UI, 언어·마우스 설정, 마우스 클릭·스크롤과 입력·선택 영역을 구분하는 테마.
- Docker Compose 기반 Oracle Free 시험 환경과 실DB 검사 도구.
- Linux AMD64/ARM64 정적 실행파일과 SHA256SUMS.
- 화면 우측하단과 `--version` 버전 표시, 도움말의 개발자·AGPL-3.0-only 라이선스 정보.

### 검증 및 제한

- Linux ARM64에서 자동 검사와 실행, Docker Oracle Free에서 Service Name 접속·자료형 조회·DML 롤백을 확인했습니다.
- 이 릴리스 시점에는 실제 CentOS 7.4, 구형 Oracle 및 SID 접속 검증이 완료되지 않았습니다. 이후 호환성 수정은 `Unreleased`에 기록합니다.
- 한 번에 SQL 한 문장 또는 BEGIN/DECLARE 블록 하나를 실행합니다. 여러 문장 일괄 실행, SQL*Plus 스크립트, 바인드 입력, DBMS_OUTPUT 출력은 지원하지 않습니다.

[Unreleased]: https://github.com/leechis7/lazyora/compare/v0.0.1...dev
[0.0.1]: https://github.com/leechis7/lazyora/releases/tag/v0.0.1
