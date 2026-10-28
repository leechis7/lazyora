# 문서 색인

[한국어 (기본)](index.md) | [English](en/index.md)

현행 사용법은 아래 안내를 따릅니다. 앱 버전의 원본은 `internal/appinfo/info.go`, 의존 버전은 `go.mod`, 실제 배포 파일은 [GitHub Releases](https://github.com/leechis7/lazyora/releases)에서 확인합니다. **v0.11.2는 Tuivory v0.2.0을 공통 UI 패키지로 사용**합니다. 이전 TUIS·독립 프로젝트 준비안은 현재 의존성이나 미완료 작업을 뜻하지 않습니다.

## 현행 안내

| 분야 | 한국어 | English |
| --- | --- | --- |
| 소개·설치·핵심 키 | [README](../README.md) | [README](../README.en.md) |
| 기능별 상세 사용법 | [사용 안내](usage.md) | [Usage](en/usage.md) |
| 실행파일에 포함되는 간단 가이드 | [내장 가이드](guide.ko.md) | [Offline guide](guide.en.md) |
| 터미널·색상·마우스·복사 | [터미널](terminal.md) | [Terminal](en/terminal.md) |
| 암호 저장·잠금·복구 | [암호 관리](credentials.md) | [Credentials](en/credentials.md) |
| 소스 빌드·검증 범위 | [빌드](build.md) | [Build](en/build.md) |
| 개발·PR·문서 게시·릴리스 | [개발 절차](development.md) | [Development](en/development.md) |
| 공통 UI 규칙과 앱 어댑터 | [UI 스타일](ui-style.md) | [UI styles](en/ui-style.md) |
| Tuivory 버전·API·갱신 | [적용 기록](tuivory-sync.md) | [Integration](en/tuivory-sync.md) |
| 호환성과 날짜별 검증 구분 | [호환성](compatibility.md) | [Compatibility](en/compatibility.md) |
| 로컬 Oracle Free·실DB 시험 | [시험 환경](../test/oracle/README.md) | [Oracle testing](en/oracle-testing.md) |
| 기여 방법 | [기여 안내](../CONTRIBUTING.md) | [Contributing](en/contributing.md) |
| 문의·비공개 취약점 제보 | [지원](../SUPPORT.md) · [보안](../SECURITY.md) | [Support/security](en/support.md) |
| 행동 규칙 | [한국어](../CODE_OF_CONDUCT.md) | [English](../CODE_OF_CONDUCT.en.md) |
| v0.11.2 릴리스 설명 | [한국어](releases/v0.11.2.md) | [English](en/releases/v0.11.2.md) |
| v0.11.1 릴리스 설명 | [한국어](releases/v0.11.1.md) | [English](en/releases/v0.11.1.md) |
| v0.11.0 릴리스 설명 | [한국어](releases/v0.11.0.md) | [English](en/releases/v0.11.0.md) |
| v0.10.2 릴리스 설명 | [한국어](releases/v0.10.2.md) | [English](en/releases/v0.10.2.md) |

[이번 문서 전수 점검 기록](audits/lora-71-docs.md)과 [기계 판독용 목록](audits/lora-71-docs-inventory.json)에서 점검 범위를 확인할 수 있습니다.

## 원본·보관 기록

- [CHANGELOG](../CHANGELOG.md)는 버전별 변경 이력입니다. 지난 버전의 기록을 현재 기능표처럼 고쳐 쓰지 않습니다. 이후 변경은 `Unreleased`에 기록합니다.
- `docs/releases/`의 기존 릴리스 설명은 해당 버전의 범위이며 [정식 릴리스](https://github.com/leechis7/lazyora/releases)와 함께 읽습니다. 변경 없는 과거 버전 전체를 소급 번역하지 않습니다.
- [LORA-69 조사](audits/lora-69.md), [적용](audits/lora-69-implementation.md), [LORA-70 패키지 전환 검증](audits/lora-70.md), 같은 폴더의 HTML/CSV/JSON은 해당 실행의 증거입니다. 당시 ‘미게시/CI 대기’와 지금의 배포 상태를 구분합니다.
- [최초 스타일 제안](ui-style-proposal.md), [초기 공통 원칙](tui-style-guide/principles.md), [프로젝트 준비 계획](tui-style-guide/project-plan.md)은 설계 유래입니다. 현행 팔레트는 후보 CSV가 아니라 고정한 Tuivory 패키지입니다.
- [PL/SQL 디버거 타당성 검토](plsql-debugger.md) · [English status](en/plsql-debugger.md)는 구현되지 않은 디버거의 설계 검토입니다. 소스 편집·컴파일·루틴 실행과 구분합니다.
- [2026-10-20 내보내기 벤치마크](../test/oracle/benchmarks/export-2026-10-20.md)는 당시 환경의 측정값입니다. 이번 문서 정리에서 새 성능 시험을 수행한 것은 아닙니다.
- `third_party/tview/`의 원본 문서·라이선스는 의존성 출처이며 제품 문서 갱신 대상으로 덮어쓰지 않습니다. 로컬 변경 설명은 [README.lazyora.md](../third_party/tview/README.lazyora.md)에 있습니다.

## 문서 유지 원칙

새로 작성하거나 수정하는 현행 안내는 한·영을 함께 갱신합니다. 명령·단축키·버전·보호 범위와 실제 시험/모의 검사/교차 빌드의 구분을 맞춥니다. 내장 가이드 원문 수정은 다음 실행파일 빌드에 반영되며 이미 배포한 바이너리는 바뀌지 않습니다. 문서 게시만으로 기존 태그·릴리스를 다시 만들지 않습니다.
