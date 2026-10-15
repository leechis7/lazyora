# 개발과 릴리스

`main`은 릴리스용 브랜치, `dev`는 다음 버전의 개발 내용을 통합하는 브랜치입니다. 기능·버그 수정·문서 작업은 `dev`에서 분기합니다. 커밋과 GitHub 업로드는 사용자의 명시적인 승인 후 진행합니다.

## 기능 개발

먼저 Jira 관련 일감을 확인하고, 없으면 사용자 승인 후 등록합니다. 작업 브랜치에는 Jira 번호와 주제를 넣습니다.

```bash
git switch dev
git pull --ff-only origin dev
git switch -c feature/LORA-17-schema-browser
```

개발 및 로컬 검증 후 승인받아 한국어 메시지로 커밋하고 브랜치를 게시합니다. `dev` 대상 PR을 만들면 CI가 실행됩니다. 테스트·vet·Linux AMD64/ARM64 빌드가 통과한 뒤 PR을 병합하고, `dev` 푸시에서도 통합 검증을 수행합니다. 병합 후 끝난 feature 브랜치는 정리합니다.

CI에서 만든 바이너리는 실행 결과의 `linux-binaries` 아티팩트에서 7일 동안 받을 수 있습니다. GitHub 아티팩트 다운로드 시 실행 권한이 사라졌다면 `chmod +x`를 적용합니다. CI에는 실DB·구형 커널 검증이 포함되지 않으므로 Oracle·CentOS 시험은 별도로 진행합니다.

## 릴리스

1. `dev`에서 릴리스 준비용 feature 브랜치를 만들고 `internal/appinfo/info.go`와 README의 버전을 갱신합니다. 버전 태그는 `vX.Y.Z` 형식으로 사용합니다.
2. 변경을 `dev`로 병합하고 `dev → main` PR을 생성합니다.
3. CI 통과와 필요한 실환경 시험을 확인한 뒤 사용자 승인으로 `main`에 병합합니다. `dev → main` 병합은 merge commit을 사용하여 두 브랜치의 커밋 관계를 유지합니다.
4. 사용자 릴리스 요청 후 아래와 같이 해당 `main` 커밋에 태그를 붙여 게시합니다. 아래 버전 번호는 예시입니다.

```bash
git switch main
git pull --ff-only origin main
git tag -a v0.0.2 -m 'v0.0.2 릴리스'
git push origin v0.0.2
```

Release Actions는 태그와 소스 버전의 일치 여부 및 태그 커밋의 `main` 포함 여부를 확인하고 테스트·vet·빌드를 수행합니다. 성공하면 Linux AMD64/ARM64 실행파일, LICENSE, README.md, SHA256SUMS를 첨부한 정식 GitHub Release를 발행합니다. `main` 병합만으로 릴리스를 발행하지 않습니다. 기존 태그와 릴리스는 덮어쓰지 않습니다.

릴리스 후 `main`의 merge commit 또는 긴급 수정이 `dev`에도 포함되도록 `main → dev`를 merge commit으로 동기화합니다. 정상 릴리스 시 `dev → main` PR을 squash/rebase 방식으로 병합하지 않습니다.

## 자동화 설정

- `.github/workflows/ci.yml`: `dev`·`main` 대상 PR 및 두 브랜치 푸시 검증
- `.github/workflows/release.yml`: 버전 태그 푸시 시 정식 릴리스 발행
- CI·Release의 Go 버전은 `mise.toml`과 함께 갱신합니다.
- 외부 Actions는 커밋 SHA로 고정합니다. 버전 변경 시 SHA와 주석을 함께 갱신합니다.

GitHub에서 CI 성공을 병합의 필수 조건으로 강제하려면 `dev`와 `main`에 브랜치 규칙을 설정하고 `Test, vet and build` 검사를 필수로 지정합니다. 워크플로 파일만으로 병합이 차단되지는 않습니다.
