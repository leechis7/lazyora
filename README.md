# LazyOra

터미널에서 Oracle SQL을 작성·실행하고 스키마 객체를 탐색하는 TUI입니다.
[lazysql](https://github.com/jorgerojas26/lazysql)의 화면 구성을 참고했습니다.

현재 버전: **0.5.0** · **Linux / Windows / macOS · AMD64 / ARM64**

- SQL 파일·여러 편집 탭, 커서 문장·F5 스크립트 실행과 IN 바인드 입력
- 예상 실행 계획, 결과 조회·복사와 CSV·TSV·Excel·INSERT 내보내기, 수동 커밋·롤백
- 스키마 객체 탐색, 컬럼·코멘트·소스·인자 조회와 소스 편집·컴파일
- 한국어·영어 UI, 키보드·마우스 조작

## 빠른 시작

[GitHub Releases](https://github.com/leechis7/lazyora/releases/latest)에서 환경에 맞는 실행파일을 받으세요. Go나 Oracle Instant Client를 따로 설치할 필요가 없습니다.

| 환경 | 실행파일 |
| --- | --- |
| Linux x86_64 (CentOS 포함) | `lazyora-linux-amd64` |
| Linux ARM64 | `lazyora-linux-arm64` |
| Windows x86_64 | `lazyora-windows-amd64.exe` |
| Windows ARM64 | `lazyora-windows-arm64.exe` |
| macOS Intel | `lazyora-darwin-amd64` |
| macOS Apple Silicon | `lazyora-darwin-arm64` |

Linux ARM64 실행 예시입니다. Linux의 다른 아키텍처나 macOS에서는 해당 파일명을 사용하세요.

```bash
chmod +x lazyora-linux-arm64
./lazyora-linux-arm64
```

Windows에서는 PowerShell에서 `.\lazyora-windows-amd64.exe`처럼 실행합니다. DB 없이 화면만 둘러보려면 `--demo`, 버전을 확인하려면 `--version`을 붙입니다.

실행하면 접속 화면이 열립니다. Host·Port·계정·암호를 입력하고 **Service Name 또는 SID**를 선택한 뒤 접속 버튼을 누르세요. 별칭을 지정해 접속 정보를 저장할 수 있습니다. 비밀번호는 기본적으로 저장하지 않으며, **암호화 저장을 선택하면 마스터 비밀번호로 보호**합니다. [암호 관리 안내](docs/credentials.md)를 참고하세요.

## 자주 쓰는 키

| 키 | 동작 |
| --- | --- |
| F1 | 현재 화면의 도움말 |
| Alt+L | 접속 화면 |
| F9 / Ctrl+Enter | 선택한 SQL 실행 (선택이 없으면 커서 문장) |
| F5 / Ctrl+E | 스크립트 실행 / 예상 실행 계획 |
| Alt+W | 조회된 결과 내보내기 |
| Alt+C / Alt+R | 커밋 / 롤백 |
| Alt+B | 스키마 브라우저 표시·숨김 |
| Alt+U / Alt+O | 브라우저 사용자 / 객체 종류 선택 |
| F2 / Shift+F2 | 쿼리 / 결과 전체화면·복원 |
| Ctrl+C / Alt+Y | 결과 셀·선택 범위 / 한 행 복사 |
| Ctrl+Q | 종료 확인 |

전체 조작은 **F1**과 화면 하단 안내에서 확인합니다. Ctrl+Enter가 구분되지 않는 터미널에서는 F9 또는 Alt+Enter를 사용하세요. 언어·마우스 설정은 **도구 → 설정**에서 변경합니다.

F9는 커서 문장을, F5는 스크립트를 실행합니다. 조회 결과는 SELECT당 최대 1,000행이며 내보내기도 가져온 결과만 저장합니다. DML은 수동 커밋하고, **Oracle DDL은 암시적으로 커밋될 수 있습니다.** 종료할 때는 앱이 추적한 미커밋 가능 내역을 확인할 수 있습니다.

## 문서

- [사용 안내](docs/usage.md) — 접속 저장, 설정, 스키마 탐색, 전체 단축키와 트랜잭션
- [터미널 문제 해결](docs/terminal.md) — SecureCRT 키 매핑과 결과 복사
- [빌드와 검증](docs/build.md) — 소스 빌드, CentOS 호환성과 검증 범위
- [시험용 Oracle Free](test/oracle/README.md) — Docker Compose로 테스트 DB 실행
- [개발과 릴리스](docs/development.md) · [변경 이력](CHANGELOG.md) · [기여 안내](CONTRIBUTING.md)
- [사용 문의](SUPPORT.md) · [보안 제보](SECURITY.md) · [참여 규칙](CODE_OF_CONDUCT.md)

## 개발자·라이선스

이채강 (Chaekang Lee) · leechis7@gmail.com

Copyright (C) 2026 Chaekang Lee. [GNU AGPL v3.0](LICENSE) (`AGPL-3.0-only`). 의존 라이브러리에는 각 프로젝트의 라이선스가 적용됩니다.
