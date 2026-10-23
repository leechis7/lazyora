# TUI 스타일 가이드 — 독립 프로젝트 준비 자료

Reusable design principles for terminal user interfaces, with explicit color profiles and verifiable component states.
Developed from LazyOra findings; framework adapters and application-specific examples remain separate.

2026-10-23 · LORA-69 · 상태: 문서 초안, 별도 프로젝트 생성 전

색상·테마·키보드·컴포넌트 상태의 기준을 다른 TUI에서도 사용할 수 있도록 정리한 자료입니다. 현재 LazyOra 저장소 안에서 관리하며, 나중에 독립 저장소를 시작할 때 이 폴더를 출발점으로 삼습니다.

## 읽는 순서

1. [공통 원칙](principles.md): 제품·언어·프레임워크와 무관하게 적용할 설계 계약.
2. [프로젝트 준비 계획](project-plan.md): 범위, 문서/도구 구조, 이관 대상, 출처, 검증과 미결정 사항.
3. [LazyOra 스타일 통합 제안](../ui-style-proposal.md): Oracle 도구에 적용할 구체적인 사례.
4. [색상 후보표](../ui-style-palette-candidate.csv): 12프로필 × 16역할의 검토용 예시.
5. [조사와 비교 자료](../audits/lora-69.md): 실제 결함에서 원칙을 도출한 근거와 검증 한계.

## 자료의 경계

| 구분 | 담을 내용 | 현재 위치 |
| --- | --- | --- |
| 공통 가이드 | 역할 기반 스타일, 색상 프로필, 포커스/선택/변경 상태, 모달, 셀 레이아웃 | 이 폴더 |
| 프레임워크 적용 | tcell/tview의 색상·위젯·이벤트 API 연결 | 현재는 LazyOra 적용 제안, 이후 어댑터 문서로 추출 |
| 제품 사례 | Oracle SQL, ROWID, 트랜잭션, Toad 키맵, LazyOra 메뉴 | LazyOra 문서·코드에 유지 |
| 검증 도구 | 색상/대비 분석, 단말 견본, 프레임 비교, 증거 기록 | 기존 tools와 감사용 테스트; 독립성 확인 후 추출 |

‘공통 가이드’와 ‘기본 테마’도 구분합니다. 특정 파랑·노랑·Nord 계열을 모든 TUI의 표준색으로 정하지 않습니다. 역할·상태·대비 계약을 공통 기준으로 두고, 현재 색상 후보는 그 계약을 구현하는 예시로 취급합니다.

## 현재 준비 상태

- 공식 자료 조사와 LazyOra의 67장면/12조합 비교 자료가 있습니다.
- 프레임워크에 독립적인 원칙과 향후 이관 범위를 문서로 정리했습니다.
- LazyOra에는 공통 프로필·배경 역할·컴포넌트 스타일과 테마 재적용을 로컬 구현했습니다. 적용 근거는 [구현 기록](../audits/lora-69-implementation.md)에 보관합니다.
- 범용 토큰 스키마, 공개 사이트, 독립 실행형 컴포넌트 견본, 범용 검증 CLI는 아직 구현하지 않았습니다.
- 별도 저장소 생성·패키지 추출·라이선스 변경·게시·버전 발행은 출범 시 결정합니다.

새 설계 원칙은 ‘문제 → 규칙 → 예외 → 검증 방법 → 사례’ 순서로 보완합니다. 예쁜 화면만 모으기보다 다른 개발자가 같은 판단을 재현할 수 있는 가이드를 목표로 합니다.
