# 기여 안내

[한국어](CONTRIBUTING.md) | [English](docs/en/contributing.md)

버그 제보, 기능 제안, 문서와 코드 수정을 환영합니다. 한국어와 영어로 작성할 수 있습니다. 참여 시 [행동 규칙](CODE_OF_CONDUCT.md)을 따라 주세요.

## 이슈와 작업 조율

기존 [GitHub 이슈](https://github.com/leechis7/lazyora/issues)를 먼저 확인하고 중복이 없으면 템플릿으로 등록해 주세요. 큰 변경은 구현 전에 범위와 화면 동작을 논의합니다. 유지관리자는 관련 Jira 작업을 확인하고 연결합니다. 외부 기여자가 Jira 계정을 만들 필요는 없습니다.

보안 취약점은 공개 이슈 대신 [보안 안내](SECURITY.md)에 따라 비공개로 전달해 주세요. DB 비밀번호, 접속 문자열의 인증정보, 실제 업무 데이터는 이슈·로그·스크린샷·커밋에 포함하지 마세요.

## 개발과 검증

Go 1.24 이상과 make를 사용합니다. 선택적으로 `mise.toml`의 Go 버전을 맞출 수 있습니다.

```bash
git switch dev
git pull --ff-only origin dev
git switch -c feature/LORA-번호-주제
go test -race ./...
go vet ./...
make build
```

Jira 번호는 유지관리자와 확인해 정합니다. 외부 기여는 fork 저장소의 작업 브랜치에서 진행할 수 있습니다. 자세한 브랜치·릴리스 절차는 [개발과 릴리스](docs/development.md)를 참고하세요.

Go 파일은 `gofmt`로 정리하고, 동작 변경에 필요한 검증을 추가합니다. DB 관련 변경은 기존 수동 트랜잭션·실행 취소 동작을 고려합니다. Oracle Docker 시험 자료는 [test/oracle](test/oracle/README.md)에 모읍니다. UI 변경은 한국어·영어, 키보드·마우스, 8·16색 환경도 고려합니다.

## Pull Request

- 일반 변경의 대상 브랜치는 `dev`입니다. `main`은 릴리스 통합용입니다.
- 해결하는 문제, 변경 후 동작, 관련 GitHub/Jira 일감, 검증 결과와 남은 제한을 적어 주세요.
- 커밋 설명은 한국어로 작성합니다. `feat:`·`fix:`·`docs:`·`ci:` 등의 접두어를 사용할 수 있습니다.
- 문서를 새로 작성하거나 수정할 때는 한국어와 영어를 함께 작성·갱신합니다. 한국어를 기본으로 하고 대응 문서의 내용·버전·명령·단축키·검증 범위·언어 링크를 맞춥니다.
- 사용자 동작이 바뀌면 한국어/영어 README·해당 사용 문서·도움말과 CHANGELOG의 `Unreleased`를 갱신합니다.
- CI 통과를 확인한 뒤 병합합니다. 릴리스 버전·태그 변경은 유지관리자와 조율합니다.

기여한 코드는 저장소의 [AGPL-3.0-only 라이선스](LICENSE)로 제공됩니다. 별도의 기여자 계약이나 서명 절차는 현재 두고 있지 않습니다.
