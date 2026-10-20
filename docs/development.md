# 개발과 릴리스

`main`은 릴리스용 브랜치, `dev`는 다음 버전의 개발 내용을 통합하는 브랜치입니다. 기능·버그 수정·문서 작업은 `dev`에서 분기합니다. 커밋과 GitHub 업로드는 사용자의 명시적인 승인 후 진행합니다.

소스 빌드 방법과 호환성·검증 기록은 [빌드와 검증](build.md)을 참고하세요.

## 기능 개발

먼저 Jira 관련 일감을 확인하고, 없으면 사용자 승인 후 등록합니다. 작업 브랜치에는 Jira 번호와 주제를 넣습니다.

```bash
git switch dev
git pull --ff-only origin dev
git switch -c feature/LORA-17-schema-browser
```

로컬 개발·검증의 기본 빌드는 `make build`로 Linux ARM64 하나만 생성합니다. 다른 OS/아키텍처의 로컬 바이너리는 사용자가 요청할 때만 개별 타깃으로 만듭니다. GitHub CI·Release는 `make build-all`로 전체 지원 대상을 빌드합니다.

개발 및 로컬 검증 후 승인받아 한국어 메시지로 커밋하고 브랜치를 게시합니다. `dev` 대상 PR을 만들면 CI가 실행됩니다. 테스트·vet·Linux/Windows/macOS AMD64·ARM64 빌드가 통과한 뒤 PR을 병합하고, `dev` 푸시에서도 통합 검증을 수행합니다. 병합 후 끝난 feature 브랜치는 정리합니다.

CI에서 만든 바이너리는 실행 결과의 `binaries` 아티팩트에서 7일 동안 받을 수 있습니다. GitHub 아티팩트 다운로드 시 실행 권한이 사라졌다면 `chmod +x`를 적용합니다. CI에는 실DB·구형 커널 검증이 포함되지 않으므로 Oracle·CentOS 시험은 별도로 진행합니다.

Windows·macOS 실행기에서는 대소문자가 다른 사용자 지정 설정명으로 설정·프로필·암호 파일을 실제 생성·갱신하여 서로 덮어쓰지 않는지도 검사합니다. 이 검사는 실제 TUI·Oracle 접속 검증을 대신하지 않습니다.

macOS 설정 경로 검사는 `macos-15-intel`에서 실행합니다. CPU와 무관한 파일명 충돌 검사이며, ARM 실행기의 용량 부족 대기 지연을 줄이기 위해 실행기 풀을 고정합니다. macOS AMD64·ARM64 실행파일은 별도 전체 빌드 단계에서 모두 검증합니다.

## 릴리스

신규 기능이 포함된 릴리스는 **minor**를 올리고 patch를 0으로 초기화합니다(예: `0.6.0 → 0.7.0`). 버그 수정만 포함하면 **patch**를 올립니다(예: `0.6.0 → 0.6.1`). `0.x` 버전에도 같은 원칙을 적용합니다. 사용자가 릴리스 버전을 명시하면 그 지정을 따릅니다.

**major는 제품 버전으로 사용자만 결정합니다.** 에이전트는 기능 개수·제품 완성도·호환성 변경을 이유로 major를 임의로 올리지 않습니다. 사용자의 명시적인 major 변경 지시 전까지 현재 major를 유지하며, `0.9.0` 다음 기능 릴리스는 `0.10.0`입니다.

1. `dev`에서 릴리스 준비용 feature 브랜치를 만들고 `internal/appinfo/info.go`와 README의 버전을 갱신합니다. CHANGELOG의 `Unreleased` 변경을 해당 버전과 날짜로 옮기고 비교 링크도 갱신합니다. `docs/releases/vX.Y.Z.md`에 기능·단축키 변경·다운로드·지원 범위를 적으면 Release Actions가 해당 문서를 릴리스 본문으로 사용합니다. 버전 태그는 `vX.Y.Z` 형식으로 사용합니다.
2. 변경을 `dev`로 병합하고 `dev → main` PR을 생성합니다.
3. CI 통과와 필요한 실환경 시험을 확인한 뒤 사용자 승인으로 `main`에 병합합니다. `dev → main` 병합은 merge commit을 사용하여 두 브랜치의 커밋 관계를 유지합니다.
4. 사용자 릴리스 요청 후 아래와 같이 해당 `main` 커밋에 태그를 붙여 게시합니다. 아래 버전 번호는 예시입니다.

```bash
git switch main
git pull --ff-only origin main
git tag -a v0.0.2 -m 'v0.0.2 릴리스'
git push origin v0.0.2
```

Release Actions는 태그와 소스 버전의 일치 여부 및 태그 커밋의 `main` 포함 여부를 확인하고 테스트·vet·Linux/Windows/macOS 빌드를 수행합니다. 성공하면 Linux AMD64/ARM64 실행파일과 Windows AMD64/ARM64 `.exe`와 macOS AMD64/ARM64 실행파일, LICENSE, README.md, SHA256SUMS를 첨부한 정식 GitHub Release를 발행합니다. `main` 병합만으로 릴리스를 발행하지 않습니다. 기존 태그와 릴리스는 덮어쓰지 않습니다.

릴리스 후 `main`의 merge commit 또는 긴급 수정이 `dev`에도 포함되도록 `main → dev`를 merge commit으로 동기화합니다. 정상 릴리스 시 `dev → main` PR을 squash/rebase 방식으로 병합하지 않습니다.

## 자동화 설정

SQL 구문 색상은 tview v0.42.0의 로컬 렌더링 확장을 사용합니다. [원본과 변경 범위](../third_party/tview/README.lazyora.md)를 확인하세요. 의존성 갱신 시 원본과 다른 Go 파일은 textarea.go 하나여야 하며, 렌더링 회귀 검사를 함께 수행합니다. 릴리스에는 MIT 라이선스 고지를 TVIEW-LICENSE.txt로 첨부합니다.

- `.github/workflows/ci.yml`: `dev`·`main` 대상 PR 및 두 브랜치 푸시 검증
- `.github/workflows/release.yml`: 버전 태그 푸시 시 정식 릴리스 발행
- CI·Release의 Go 버전은 `mise.toml`과 함께 갱신합니다.
- CI·Release의 Go 모듈 다운로드는 일시적인 네트워크 오류에 대비해 최대 3회 재시도합니다. 실제 테스트·vet·빌드는 실패를 그대로 보고합니다.
- 외부 Actions는 커밋 SHA로 고정합니다. 버전 변경 시 SHA와 주석을 함께 갱신합니다.

GitHub에서 CI 성공을 병합의 필수 조건으로 강제하려면 `dev`와 `main`에 브랜치 규칙을 설정하고 `Test, vet and build` 검사를 필수로 지정합니다. 워크플로 파일만으로 병합이 차단되지는 않습니다.

현재 브랜치의 구현·검증·사용자 검토와 승인된 커밋·병합·정리를 마친 뒤 다음 브랜치를 시작합니다. 사용자 명시적 전환 지시가 없으면 승인 대기 중에도 다른 브랜치로 이동하지 않습니다.
