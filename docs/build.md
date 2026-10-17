# 빌드와 검증

배포된 실행파일로 시작하려면 [README](../README.md)를 참고하세요. 이 문서는 소스 빌드와 호환성·검증 기록을 다룹니다.

## 빌드와 실행

Go 1.24 이상과 `make`를 설치하고, 터미널에서 `go` 명령을 사용할 수 있는 개발 머신에서 실행합니다. 아래 명령은 프로젝트 루트에서 실행합니다. 최초 빌드에는 의존성 다운로드를 위한 인터넷 연결이 필요합니다.

```bash
go version
make build
./bin/lazyora-linux-arm64 --demo
./bin/lazyora-linux-arm64 --version
./bin/lazyora-linux-arm64
```

Go는 공식 배포판이나 운영체제의 패키지 관리자로 설치해도 됩니다. `mise.toml`은 개발 도구 버전을 맞추려는 개발자를 위한 선택 사항입니다.

`make build`는 로컬 개발용 **Linux ARM64** 실행파일 `bin/lazyora-linux-arm64` 하나만 생성합니다. 다른 OS·아키텍처는 필요할 때 명시적으로 빌드합니다.

| 명령 | 생성 파일 |
| --- | --- |
| `make build-amd64` | `bin/lazyora-linux-amd64` |
| `make build-windows-amd64` | `bin/lazyora-windows-amd64.exe` |
| `make build-windows-arm64` | `bin/lazyora-windows-arm64.exe` |
| `make build-darwin-amd64` | `bin/lazyora-darwin-amd64` |
| `make build-darwin-arm64` | `bin/lazyora-darwin-arm64` |
| `make build-native` | 현재 개발 머신의 OS·아키텍처 파일 |

GitHub CI·Release는 `make build-all`로 전체 지원 대상(Linux/Windows/macOS AMD64·ARM64)을 빌드합니다.

Windows에서는 사용하는 CPU에 맞는 `.exe`를 PowerShell·Windows Terminal 등 콘솔에서 실행합니다. 예: `.\bin\lazyora-windows-amd64.exe --demo`. Windows에 Go나 Oracle Instant Client를 별도로 설치하지 않고 빌드된 파일로 실행합니다. GitHub CI에서 Windows CLI 실행을 확인하며, 실Windows TUI·DB 접속 검증은 별도입니다.

`make build-linux`, `make build-windows`, `make build-darwin`으로 OS별 빌드도 가능합니다. CentOS 7.4 x86_64에는 AMD64 파일을 사용합니다. ARM64 개발 머신에서는 `./bin/lazyora-linux-arm64 --demo`로 화면을 확인합니다. `make build-amd64`와 `make build-arm64`로 각각 빌드할 수 있고, `make build-native`는 현재 개발 머신의 OS·아키텍처 이름으로 생성합니다.

Linux 빌드는 `CGO_ENABLED=0`을 사용하고 AMD64는 `GOAMD64=v1`을 적용합니다. Oracle Instant Client를 설치하지 않는 단일 실행파일을 목표로 합니다. 실행 대상 서버에는 Go가 필요하지 않습니다.

CentOS 7.4 서버의 사용자 시험에서 발견한 Oracle 접속 패킷·커서·색상 호환 문제를 수정했으며, 사용자가 이번 CentOS 시험 완료를 확인했습니다. Ubuntu ARM에서도 8색 모드 표시를 확인했습니다. 정확한 커널·로캘·터미널 버전별 전체 검증 결과를 수집한 것은 아니며, 컨테이너만으로 구형 커널 호환성을 판정하지 않습니다.

```bash
uname -r
uname -m
file bin/lazyora-linux-amd64
ldd bin/lazyora-linux-amd64
```

정적 실행파일에서는 `ldd`가 동적 실행파일이 아니라는 메시지를 출력할 수 있습니다.

macOS 실행파일은 v0.1.1부터 포함합니다. Intel Mac은 `lazyora-darwin-amd64`, Apple Silicon Mac은 `lazyora-darwin-arm64`를 사용하며, `chmod +x` 후 터미널에서 실행합니다. GitHub에서 CGO 없이 교차 빌드하며, 코드 서명·공증과 실Mac TUI·DB 접속 검증은 포함하지 않습니다.

## 테스트용 Oracle Free

Docker Compose와 환경 파일, 샘플 SQL, 실행 스크립트, 실DB 테스트는 [test/oracle](../test/oracle/README.md)에 모았습니다. 프로젝트 루트에서 `make -C test/oracle up`으로 시작하고, `make -C test/oracle test`로 시험합니다.

## 검증 기록과 범위

다음은 기존 검증 기록이며, 이 문서 정리 작업에서 테스트를 다시 실행한 것은 아닙니다. v0.1.0 릴리스의 검증 요약은 [릴리스 안내](releases/v0.1.0.md#다운로드와-검증)를 참고하세요.

검증 환경: Go 1.26.9, Linux ARM64. 접속 문자열·트랜잭션·결과 행 제한 테스트와 tcell 모의 터미널에서 데모 화면·키 입력 테스트가 통과했습니다. 설정 저장·재읽기·저장 실패 처리, 언어 변경 시 입력 보존, 접속·설정 단축키, 마우스 버튼·입력칸·도움말 스크롤과 마우스 끄기·다시 켜기도 자동 검사합니다. `go vet`과 CGO 없는 Linux AMD64/ARM64 빌드도 통과했습니다. AMD64 파일은 `file`과 `readelf`로 정적 링크를 확인했고, ARM64 파일은 실제 터미널에서 데모 시작·종료를 확인했습니다.

2026-10-15에 Docker Oracle Free 26ai (`23.26.3-slim`)에서 Service Name 접속, 샘플 데이터와 한글·NULL·NUMBER·DATE·TIMESTAMP 조회, DML 실행 후 롤백을 확인했습니다. 기존 `go-ora/v2 v2.9.0`은 같은 계정으로 인증이 실패하여 `go-ora/v3 v3.0.1`로 변경했습니다. 이 드라이버는 Go 1.24 이상을 필요로 하며 이번 검증에는 Go 1.26.9를 사용했습니다.

Oracle 10g 접속 시험에서 `network.newAcceptPacketFromData`의 `slice bounds out of range [41:32]` 패닉이 보고되어, 짧은 TNS ACCEPT 응답의 길이를 확인하는 [업스트림 수정](https://github.com/sijms/go-ora/commit/9197bc12b6fea25726fa028e48c731468cd05b18)을 포함한 `go-ora/v3 v3.0.2-0.20260731135026-9197bc12b6fe`로 고정했습니다. 이 패킷 파싱 수정과 Oracle 10g 전체 접속·쿼리 지원 검증은 별개이며, 실제 10g 서버에서 재시험이 필요합니다.

실제 SID 접속, COMMIT·PL/SQL·취소·재연결 동작의 전체 검증과 Oracle 버전별 접속·조회 검증은 아직 남아 있습니다. CentOS 시험 완료 판단은 사용자 확인에 근거하며, Oracle 10g의 상세 SQL·트랜잭션 성공 결과를 별도로 수집한 것은 아닙니다.

```bash
go test -race ./...
go vet ./...
```

실DB에서는 Service Name/SID 각각의 연결, `SELECT sysdate FROM dual`, 한글·NULL·NUMBER·DATE 결과, DML 후 COMMIT/ROLLBACK, PL/SQL, 실행 취소와 재연결을 확인해야 합니다. 트랜잭션 검증은 테스트 스키마에서 수행합니다.

## 개발 정보

앱 버전과 개발자·라이선스 정보의 원본은 `internal/appinfo/info.go`입니다. 일반 `go build`·`go run`과 Makefile 빌드가 동일한 버전을 사용합니다. 버전 변경 시 이 파일과 README의 현재 버전을 함께 갱신합니다.

브랜치·PR·릴리스 절차는 [개발과 릴리스](development.md)를 참고하세요.
