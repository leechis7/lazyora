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

최신 환경별 상태와 재현 명령, 장애·대형 데이터 검사 및 실제 터미널 확인표는
[호환성·장애 회귀 검증](compatibility.md)에 정리합니다.

검증 기록은 아래의 각 시점과 환경을 기준으로 합니다. v0.1.0 릴리스의 검증 요약은 [릴리스 안내](releases/v0.1.0.md#다운로드와-검증)를 참고하세요.

검증 환경: Go 1.26.9, Linux ARM64. 접속 문자열·트랜잭션·결과 행 제한 테스트와 tcell 모의 터미널에서 데모 화면·키 입력 테스트가 통과했습니다. 설정 저장·재읽기·저장 실패 처리, 언어 변경 시 입력 보존, 접속·설정 단축키, 마우스 버튼·입력칸·도움말 스크롤과 마우스 끄기·다시 켜기도 자동 검사합니다. `go vet`과 CGO 없는 Linux AMD64/ARM64 빌드도 통과했습니다. AMD64 파일은 `file`과 `readelf`로 정적 링크를 확인했고, ARM64 파일은 실제 터미널에서 데모 시작·종료를 확인했습니다.

2026-10-15에 Docker Oracle Free 26ai (`23.26.3-slim`)에서 Service Name 접속, 샘플 데이터와 한글·NULL·NUMBER·DATE·TIMESTAMP 조회, DML 실행 후 롤백을 확인했습니다. 기존 `go-ora/v2 v2.9.0`은 같은 계정으로 인증이 실패하여 `go-ora/v3 v3.0.1`로 변경했습니다. 이 드라이버는 Go 1.24 이상을 필요로 하며 이번 검증에는 Go 1.26.9를 사용했습니다.

Oracle 10g 접속 시험에서 `network.newAcceptPacketFromData`의 `slice bounds out of range [41:32]` 패닉이 보고되어, 짧은 TNS ACCEPT 응답의 길이를 확인하는 [업스트림 수정](https://github.com/sijms/go-ora/commit/9197bc12b6fea25726fa028e48c731468cd05b18)을 포함한 `go-ora/v3 v3.0.2-0.20260731135026-9197bc12b6fe`로 고정했습니다. 이 패킷 파싱 수정과 Oracle 10g 전체 접속·쿼리 지원 검증은 별개이며, 실제 10g 서버에서 재시험이 필요합니다.

실제 SID 접속, COMMIT·PL/SQL·취소·재연결 동작의 전체 검증과 Oracle 버전별 접속·조회 검증은 아직 남아 있습니다. CentOS 시험 완료 판단은 사용자 확인에 근거하며, Oracle 10g의 상세 SQL·트랜잭션 성공 결과를 별도로 수집한 것은 아닙니다.

```bash
go test -race ./...
go vet ./...
```

실DB에서는 Service Name/SID 각각의 연결, `SELECT sysdate FROM dual`, 한글·NULL·NUMBER·DATE 결과, DML 후 COMMIT/ROLLBACK, PL/SQL, 실행 취소와 재연결을 확인해야 합니다. 트랜잭션 검증은 테스트 스키마에서 수행합니다.

### v0.5.0 검증 (2026-10-20)

Linux ARM64·Go 1.26.9에서 전체 `go test -race ./...`, `go vet ./...`, 기본 ARM64 빌드를 검증했습니다. SQL UTF-8/BOM/줄바꿈·저장 오류, 다중 탭·미저장 종료, 커서 문장 구분, 스크립트 오류/취소 중단, 바인드 타입/NULL, 예상 계획, CSV/TSV/XLSX/INSERT와 원본 파일 보존을 자동 검사합니다.

Docker Oracle Free에서 SQL/PLSQL 순차 실행·명시적 COMMIT/ROLLBACK·다중 SELECT, 38자리 NUMBER·한글·날짜/소수초/NULL 바인드, 예상 계획의 DELETE 미실행·기존 DML 롤백 보존, INSERT 내보내기→재입력의 원본 결과 비교를 확인했습니다. 실제 ARM64 실행파일의 Linux PTY 시험도 접속·키/마우스·복사·전체화면·F5·Ctrl+E·바인드·XLSX 저장·미저장 확인→커밋/롤백/종료까지 통과했습니다.

이번 기능의 실제 CentOS 7.4·Oracle 10g·SID 접속·Windows/macOS TUI는 재시험하지 않았습니다. GitHub CI에서는 6종 교차 빌드와 Windows AMD64 버전 명령을 검사합니다. 자세한 지원 한계는 [v0.5.0 안내](releases/v0.5.0.md)를 참고하세요.

## SQL 분석 성능 개선 (LORA-39, 2026-10-21)

문장 시작 행을 계산할 때 원문 앞부분을 반복해서 세던 작업을 줄별 순차 탐색으로 변경했습니다. 긴 한 줄의 여러 나눗셈 `/`도 해당 줄이 PL/SQL 구분 줄인지 한 번만 판정합니다. UTF-8 바이트 범위와 행 번호, CRLF·주석·인용 문자열·PL/SQL 구분 규칙은 유지합니다.

Linux ARM64·Go 1.26.9에서 다음 벤치마크를 변경 전후 같은 조건으로 3회씩 실행한 중앙값입니다. 다른 머신의 성능을 보장하는 수치는 아닙니다.

```bash
go test ./internal/oracle -run '^$' -bench 'Benchmark(Statements|AtStatement)$' -benchtime=1x -count=3
```

| 입력 | 변경 전 | 변경 후 |
| --- | ---: | ---: |
| SELECT 2만 문장, 0.4MB | 213.225ms | 19.964ms |
| SELECT 4만 문장, 0.8MB | 806.815ms | 37.458ms |
| SELECT 8만 문장, 1.6MB | 3157.930ms | 83.045ms |
| 한 줄의 나눗셈 2만 개 | 628.853ms | 4.587ms |
| 한 줄의 나눗셈 4만 개 | 2506.816ms | 11.315ms |
| 8만 문장에서 커서 위치 선택 | 3164.235ms | 85.455ms |

행 인덱스용 별도 배열은 추가하지 않습니다. 8만 문장의 누적 메모리 할당은 변경 전후 약 88.9MB로 비슷하며 이는 최대 RSS가 아닙니다. 커서 SQL 선택은 여전히 UI 입력 처리 중 파일 전체를 분석하므로 더 큰 파일·느린 CPU의 지연이나 토큰화 메모리까지 없애는 변경은 아닙니다.

## 개발 정보

앱 버전과 개발자·라이선스 정보의 원본은 `internal/appinfo/info.go`입니다. 일반 `go build`·`go run`과 Makefile 빌드가 동일한 버전을 사용합니다. 버전 변경 시 이 파일과 README의 현재 버전을 함께 갱신합니다.

브랜치·PR·릴리스 절차는 [개발과 릴리스](development.md)를 참고하세요.

### v0.5.1 검증 (2026-10-20): DB 전체 결과 내보내기

- 화면 조회 한도는 1,000행으로 유지하며, 전체 결과는 원 SQL·IN 바인드 재조회 후 행별로 파일에 기록합니다. 전체 저장에는 앱의 64 MiB·60초 제한을 적용하지 않습니다.
- `make test check build`의 race 테스트·vet·Linux ARM64 빌드 통과. CSV·TSV·INSERT·XLSX 2,507행, 64 MiB 초과 순차 출력, 취소·읽기/쓰기/커서 종료 오류·외부 변경 시 기존 파일 보존을 자동 검사했습니다. Excel 시트 분할은 축소한 시트 한도에서 헤더 유무·정확한 경계·다중 시트 관계와 누락 여부를 검사했습니다.
- Docker Oracle Free에서 실제 2,507행과 최초 바인드 값 보존을 확인하고, 1,500행에서 취소한 뒤 기존 파일·미커밋 DML을 보존하여 명시적 롤백이 가능한지 확인했습니다.
- Linux ARM64 실제 실행파일의 PTY 시험에서 전체 범위 선택·기본 취소·명시적 재조회 확인·2,507행 CSV 저장까지 통과했습니다. 기존 접속·실행·복사·계획·바인드·XLSX·트랜잭션 종료 시험도 통과했습니다.

대용량 내보내기는 공통 256 KiB 출력 버퍼를 추가해 네 형식의 작은 파일 쓰기를 모읍니다. 10만·100만 행의 파일 저장/Oracle Free 조회 포함 저장 시간, 파일 전 행 수, 최대 RSS를 확인했습니다. 측정 조건과 수치는 [성능 측정 기록](../test/oracle/benchmarks/export-2026-10-20.md)에 있습니다. 별도 성능 시험을 기본 테스트에서 제외하며, 각 실행의 결과는 환경에 따라 달라집니다.

### 다음 버전: 결과 추가 가져오기·정렬·필터 (2026-10-21)

- F9 SELECT를 같은 커서에서 1,000행씩 읽으며 10,000행 또는 추정 64 MiB 보관 상한을 적용합니다. 화면 정렬·필터는 가져온 행만 대상으로 하고 DB 전체 내보내기는 원 SQL 재조회로 구분합니다.
- Oracle Free 실DB에서 2,507행을 1,000/1,000/507행으로 이어 읽고, 첫 페이지 뒤 다른 세션의 UPDATE/COMMIT에도 최초 조회 시점이 유지됨을 확인했습니다. 중간 메타데이터 조회·작업 컨텍스트 취소 이후에도 이어 읽습니다.
- 고정된 드라이버가 타입 이름을 생략하는 NUMBER는 정밀도·스케일 메타데이터로 구분합니다. 문자열 `00123`을 숫자로 추측 변환하지 않으며 38자리 수와 숫자 NULL의 타입을 보존합니다.
- 느린 서버 함수로 다음 페이지의 네트워크 읽기를 지연시킨 후 취소하는 시험을 추가했습니다. 200ms 제한 후 취소 응답 처리까지 약 711ms를 측정했고, 커서 해제·기존 결과·미커밋 DML 보존을 확인했습니다. 이는 이 시험 환경의 관찰값이며 취소가 정확히 제한 시각에 끝난다는 보장은 아닙니다.
- 8색 Linux ARM64 PTY에서 같은 커서의 1,000→2,000→2,507행 가져오기, 실제 숫자 내림차순, 필터·초기화·원문 복사와 전체 DB 내보내기를 확인했습니다. Oracle 10g/19c·Windows/macOS 실제 실행은 별도 검증 대상입니다.

### 대형 셀·안전 편집·About 개발 검증

Oracle Free에서 256 Ki 단위를 넘는 한글·이모지 CLOB과 0/FF 포함 BLOB을 미리보기·추가 읽기·개별 전체 스트림 저장·전체 재조회로 비교했습니다. NULL/빈 값과 실행별 참조 만료, 실패/취소 시 기존 파일 보존을 검사했습니다. 국가 문자 LOB 타입 보고·XLSX/INSERT 한계는 사용 문서에 명시합니다.

일반 테이블 편집은 ROWID와 모든 원래 값을 비교하며 다른 컬럼의 동시 변경·PK 없는 중복 행·NULL·NUMBER·DATE/TIMESTAMP·RAW·제약 오류·커밋/롤백을 검사했습니다. 드라이버의 잠금 대기 취소가 지연돼 NOWAIT 사전 잠금으로 보완했고, 타 세션이 잠근 행을 즉시 거부하는 것을 Oracle Free의 race 통합 검사로 확인했습니다. 복구 실패 시 불확실 상태와 커밋/추가 실행 차단은 오류 주입 단위 검사로 확인합니다. 실제 구형 Oracle·Windows/macOS TUI 확대 시험(LORA-52)은 사용자 요청으로 연기했습니다.

NVARCHAR2·NCHAR은 드라이버의 국가 문자 전용 바인드로 확장해 한글·이모지 원문과 고정 길이 공백을 검증했습니다. About·셀 바이너리 저장·안전 편집의 기본 취소/반영/롤백은 Linux ARM64의 8색 실제 PTY 검사를 통과했습니다.
