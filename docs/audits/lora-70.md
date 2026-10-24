# LORA-70 스타일 패키지 전환 검증

2026-10-24 · `feature/LORA-70-tuis-package` · 커밋·게시 전 로컬 검증

현재 적용 대상은 **Tuivory v0.0.1**입니다. 아래 TUIS 초기 적용과 정식 태그 갱신은 이전 검증 이력이며, Tuivory 후속 검증은 별도 항목으로 기록합니다.

공통 스타일을 복사해 보관하던 방식에서 TUIS의 공개 Go 패키지를 import하는 방식으로 전환했습니다. 적용 API·책임·의존성·갱신 절차는 [패키지 적용 기록](../tuivory-sync.md)에 있습니다.

## 비교 기준

- 변경 전: LazyOra dev `4530a1156853fa8629a16f654f87680fe2b52e01`, 앱 버전 v0.10.1. `git archive`로 `/tmp`에 추출해 기존 작업 트리를 변경하지 않고 화면을 생성했습니다.
- 변경 후: TUIS `v0.0.0-20261023175236-ea155a284e80`를 실제 import한 이 작업 트리.
- 동일한 `TestThemeAudit` 사용: 다크/라이트/흑백 × 8/16/256/True Color × 한·영 × 67장면 × 초기/테마 왕복 = 각 3,216프레임.
- 두 자료의 `Rows`, `Styles`, `W`, `H`, `Page`를 비교했습니다. 색상 원본 값·참조 팔레트 변환·굵기·밑줄 등 검사기가 캡처한 스타일 필드와 위치를 그대로 비교합니다.
- 파일/내보내기 화면 96프레임의 차이는 Go 테스트의 무작위 임시 경로였습니다. **텍스트 중 `/tmp/TestThemeAudit[0-9]+/`만 같은 값으로 정규화**한 뒤 비교했으며, 위치·스타일은 정규화하지 않았습니다. 역할별 자료는 Go map의 순회 순서를 제외하기 위해 테마·색상 수·역할로 정렬했습니다.

[기계 판독용 비교 결과와 원본 자료 SHA256](lora-70-comparison.json)

## 결과

| 항목 | 결과 |
| --- | --- |
| 전환 전후 3,216프레임 | 임시 경로 정규화 후 차이 0, 누락 0 |
| 초기/테마 왕복 1,608쌍 | 차이 0 |
| 공통 역할·SQL 토큰 스타일 | 모든 캡처 값 동일 |
| 굵기 해석 2종의 6,432개 대비 분석 | 동일 전경/배경 및 4.5:1 미만 텍스트 후보 0 |
| 공통 역할 참조 대비 최저 | 4.64:1 |
| 전체 `go test -race ./...` | 통과 |
| `go vet ./...` | 통과 |
| `make build` | Linux ARM64만 빌드, 통과 |
| `go version -m bin/lazyora-linux-arm64` | 실제 TUIS 고정 버전과 tview 로컬 포크 사용 확인 |
| `bin/lazyora-linux-arm64 --version` | `LazyOra v0.10.1` (이번 작업에서 제품 버전 변경 없음) |

기존 회귀 검사는 SQL/암호 입력, 선택·포커스, 편집기의 UTF-8/스크롤/실행취소, 그리드 미커밋 표시·복사, 열린 콤보박스의 한글 겹침과 기존 접속/키맵 동작을 포함합니다. 비공개 필드 접근에 의존하던 선택 필드 테스트는 공개 컴포넌트의 동일 인스턴스와 실제 그리기 결과를 검사하도록 바꿨습니다.

## 재현

```sh
go test -race ./...
go vet ./...
make build
LAZYORA_THEME_AUDIT_DIR=/tmp/lazyora-lora70-final \
  go test ./internal/ui -run '^TestThemeAudit$' -count=1
python tools/theme_audit.py /tmp/lazyora-lora70-final/raw.json \
  /tmp/lazyora-lora70-final/report
```

변경 전 자료도 위 기준 커밋을 별도 임시 디렉터리에 `git archive`로 추출한 뒤 같은 테스트로 생성합니다. 생성된 JSON의 화면 키는 `(Scene, Theme, Colors, Variant, Language)`입니다. 위 경로 정규화 후 렌더링 필드를 비교합니다. 이 실행의 전체 검사 로그는 `/tmp/lazyora-lora70-validation.log`, 변경 전 자료는 `/tmp/lazyora-lora70-baseline/raw.json`, 변경 후 자료·HTML 갤러리는 `/tmp/lazyora-lora70-final/`에 있습니다. 대용량 생성 자료는 Git에 추가하지 않습니다.

이 수치는 모의 화면과 tcell 참조 팔레트 기준입니다. 분석에서 Dim 배경·공백·장식선을 제외하며, 모든 실제 SecureCRT/MobaXterm·사용자 팔레트·작은 창 크기의 검증을 뜻하지 않습니다. Oracle 서버를 새로 띄우지 않았으며 실DB 검증을 수행한 것으로 기록하지 않습니다.

GitHub CI·커밋·게시·릴리스는 실행하지 않았습니다. `test/oracle/run.sh`의 기존 실행 권한 변경은 보존했습니다. TUIS 저장소는 수정하지 않았습니다.


## 후속 갱신: TUIS 정식 v0.1.0

사용자 요청에 따라 의존성을 `github.com/leechis7/tuis v0.1.0`으로 갱신했습니다. 태그의 원본 커밋은 위에서 검증한 `ea155a284e80f7d45c5dbaf054ba070c35b9425f`와 같습니다. GitHub compare의 변경 파일 0건과 다운로드한 두 모듈 디렉터리의 `diff -qr` 통과로 소스 동일성을 확인했습니다.

정식 버전 적용 후 `go test -race ./...`, `go vet ./...`, `make build`를 다시 실행해 통과했습니다. `go version -m bin/lazyora-linux-arm64`에 TUIS **v0.1.0**이 포함됨을 확인했습니다. 로그는 `/tmp/lazyora-tuis-v010-validation.log`입니다.

기존 3,216프레임 검증은 동일한 패키지 소스에 대한 근거로 유지합니다. 태그 교체 후 화면 자료를 새로 생성한 것은 아닙니다. 제품 버전·커밋·GitHub 게시·릴리스 상태는 변경하지 않았습니다.

## 후속 전환: Tuivory v0.0.1

사용자 요청으로 TUIS 의존성을 제거하고 **`github.com/leechis7/tuivory/go v0.0.1`**의 단일 공개 패키지 `tuivory`로 전환했습니다. 공개 태그는 `go/v0.0.1`, 커밋은 `cf06d95f4334f65436197ae4f12faf35783f5b7a`입니다. 현재 작업 브랜치는 같은 LORA-70 브랜치를 이름만 바꾼 `feature/LORA-70-tuivory-package`입니다.

저장소 구조·Go 하위 모듈·단일 공개 진입점·내부 구현 배치·후보 토큰 식별자가 달라졌습니다. 다운로드한 이전 TUIS의 스타일/위젯 구현 9개 파일을 Tuivory의 대응 파일과 비교했습니다. 패키지 선언·import 경로·설명 이름을 정규화하면 9개 모두 일치합니다. 소비 코드는 공개 API만 사용하며 `internal` 패키지에 의존하지 않습니다. 후보 JSON 토큰은 직접 소비하지 않습니다.

이번 전환 후 검사를 **새로 실행**했습니다.

| 항목 | Tuivory 적용 후 결과 |
| --- | --- |
| `go test -race ./...` | 통과 |
| `go vet ./...` | 통과 |
| `make build` | Linux ARM64만 빌드, 통과 |
| 실행파일 모듈 정보 | Tuivory v0.0.1 포함, TUIS 미포함 |
| 12프로필 × 67장면 × 한·영 × 초기/왕복 | 새로 3,216프레임 생성 |
| 변경 전 dev 자료와 비교 | 무작위 테스트 경로 정규화 후 차이 0, 누락 0 |
| 테마 왕복 1,608쌍 | 차이 0 |
| 공통 역할·SQL 스타일 자료 | 기준 자료와 동일 |
| 6,432개 참조 대비 분석 행 | 동일색 및 4.5:1 미만 텍스트 후보 0 |

[Tuivory 전환 비교 결과와 자료 해시](lora-70-tuivory-comparison.json). 경로 정규화와 실제 단말/실DB 미검증 등의 해석 범위는 위 초기 검증과 같습니다.

로그: `/tmp/lazyora-tuivory-validation.log`. 원본 화면 자료·분석·갤러리: `/tmp/lazyora-tuivory-audit/`. 현행 패키지 적용 기록은 [tuivory-sync.md](../tuivory-sync.md)로 옮겼습니다. 이전 TUIS 기록은 이전 단계로 명시해 보존합니다. 제품 버전은 v0.10.1 그대로이며 커밋·GitHub 게시·CI·릴리스는 실행하지 않았습니다. 기존 `test/oracle/run.sh` 권한 변경을 보존했고 외부 Tuivory 저장소는 수정하지 않았습니다.
