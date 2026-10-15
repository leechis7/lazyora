# LazyOra

Oracle 쿼리 작성·실행을 위한 Go TUI의 첫 구현입니다. lazysql의 화면 구성을 참고하여 Oracle 전용으로 작성했습니다.

현재 앱 버전은 **0.0.1**입니다. 화면 우측하단에 표시하며 `--version`으로도 확인할 수 있습니다. F1 도움말에는 버전과 개발자·라이선스 정보를 표시합니다.

## 빌드와 실행

Go 1.24 이상과 `make`를 설치하고, 터미널에서 `go` 명령을 사용할 수 있는 개발 머신에서 실행합니다. 최초 빌드에는 의존성 다운로드를 위한 인터넷 연결이 필요합니다.

```bash
go version
make build
./bin/lazyora-linux-amd64 --demo
./bin/lazyora-linux-amd64 --version
./bin/lazyora-linux-amd64
```

Go는 공식 배포판이나 운영체제의 패키지 관리자로 설치해도 됩니다. `mise.toml`은 개발 도구 버전을 맞추려는 개발자를 위한 선택 사항입니다.

`make build`는 `bin/lazyora-linux-amd64`와 `bin/lazyora-linux-arm64`를 생성합니다. CentOS 7.4 x86_64에는 AMD64 파일을 사용합니다. ARM64 개발 머신에서는 `./bin/lazyora-linux-arm64 --demo`로 화면을 확인합니다. `make build-amd64`와 `make build-arm64`로 각각 빌드할 수 있고, `make build-native`는 현재 개발 머신의 OS·아키텍처 이름으로 생성합니다.

Linux 빌드는 `CGO_ENABLED=0`을 사용하고 AMD64는 `GOAMD64=v1`을 적용합니다. Oracle Instant Client를 설치하지 않는 단일 실행파일을 목표로 합니다. 실행 대상 서버에는 Go가 필요하지 않습니다.

CentOS 7.4 x86_64 호환성은 목표이며 아직 검증되지 않았습니다. 실제 3.10 커널의 VM 또는 서버에서 실행해야 지원을 확정할 수 있습니다. 컨테이너만으로 구형 커널 호환성을 검증할 수는 없습니다.

```bash
uname -r
uname -m
file bin/lazyora-linux-amd64
ldd bin/lazyora-linux-amd64
```

정적 실행파일에서는 `ldd`가 동적 실행파일이 아니라는 메시지를 출력할 수 있습니다.

## 연결

F2로 접속 화면을 열어 Host, Port, Username, Password를 입력합니다. 연결 유형에서 **Service Name** 또는 **SID**를 선택하고 해당 이름을 입력한 뒤 **Ctrl+Enter / Alt+C** 또는 접속 버튼 클릭으로 연결합니다. 암호는 화면에서 가리고 디스크에 저장하지 않습니다. 선택적으로 `LAZYORA_HOST`, `LAZYORA_PORT`, `LAZYORA_USER`, `LAZYORA_PASSWORD`, `LAZYORA_NAME` 환경변수로 입력 기본값을 지정할 수 있습니다. 암호를 명령행 인자로 받지 않습니다.

**별칭(Alias)**을 입력하고 하단 **저장(Ctrl+S / Alt+S)** 버튼을 누르면 호스트·포트·사용자·접속 방식·Service Name/SID를 저장합니다. 실제 DB에 연결하지 않아도 저장할 수 있습니다. 같은 별칭은 기존 정보를 갱신하고, 다른 별칭은 새 접속으로 추가합니다. 다음 실행에서 **저장된 접속(Alt+R)** 목록으로 불러올 수 있으며 **Alt+A**로 별칭 입력칸을 선택합니다. 별칭을 바꾸어 저장하면 새 항목이 추가됩니다.

접속 정보는 UI 설정 파일과 같은 폴더의 `connections.json`에 저장합니다. Linux 기본 경로는 `~/.config/lazyora/connections.json`입니다. **비밀번호는 저장하지 않으며**, 저장된 접속을 선택하면 이전 비밀번호를 지웁니다. 저장 버튼을 누를 때는 현재 입력한 비밀번호를 메모리에 유지하여 바로 접속할 수 있습니다. 암호화 비밀번호 저장은 향후 별도 설계 작업입니다.

## 언어·마우스 설정

한 실행파일에서 English와 한국어를 지원합니다. **F3** 설정 화면에서 언어와 마우스 사용 여부를 선택하고 **Ctrl+S / Alt+S** 또는 저장 버튼으로 적용합니다. Esc는 변경을 취소합니다. 언어를 바꿔도 작성한 SQL, 조회 결과와 접속 화면의 입력값을 유지합니다. Oracle에서 받은 오류와 조회 데이터는 원문을 표시합니다.

초기 언어는 `LC_ALL`, `LC_MESSAGES`, `LANG` 순서로 확인하며 한국어 로캘이면 한국어, 나머지는 영어입니다. 마우스는 기본으로 켜져 있습니다. 버튼·입력칸·결과 셀을 클릭하고 휠로 스크롤할 수 있습니다. 마우스를 껐을 때도 F3으로 다시 켤 수 있습니다.

설정은 OS 사용자 설정 폴더의 `lazyora/settings.json`에 저장합니다. Linux 기본 경로는 `~/.config/lazyora/settings.json`이며 `XDG_CONFIG_HOME`을 따릅니다. `LAZYORA_CONFIG`로 파일 경로를 지정할 수도 있습니다. 언어와 마우스 설정만 저장하며 접속 정보는 같은 폴더의 별도 파일로 관리합니다. `LAZYORA_CONFIG`의 파일 이름을 `connections.json`으로 지정했다면 접속 파일 이름은 충돌을 피하도록 `connection-profiles.json`을 사용합니다. DB 비밀번호는 어느 파일에도 저장하지 않습니다. 저장에 실패하면 기존 설정을 유지하고 오류를 표시합니다.

## 테스트용 Oracle Free

Docker Compose와 환경 파일, 샘플 SQL, 실행 스크립트, 실DB 테스트는 [test/oracle](test/oracle/README.md)에 모았습니다. 프로젝트 루트에서 `make -C test/oracle up`으로 시작하고, `make -C test/oracle test`로 시험합니다.

## 조작

화면 최하단의 단축키 안내는 현재 포커스에 따라 바뀝니다. SQL 편집기에서는 `F4 / Ctrl+Enter` 실행·줄바꿈·결과창 이동을, 결과 표에서는 방향키·`PgUp/PgDn`·`Home/End` 이동을 표시합니다. 키보드나 마우스로 포커스를 바꾸면 즉시 갱신되며, 접속·설정·도움말에서는 해당 화면의 저장·닫기·스크롤 동작을 안내합니다. 하단 버튼을 클릭해도 해당 단축키와 같은 동작을 실행합니다. 전체 단축키 목록은 F1 도움말에서 확인합니다.

메인 화면의 **F5 커밋·F6 롤백** 버튼은 SQL 편집기·결과 표의 포커스에 관계없이 항상 표시하며 마우스로도 실행할 수 있습니다.

기본 색상은 [Nord 팔레트](https://www.nordtheme.com/docs/colors-and-palettes/)를 참고했습니다. SQL·폼 입력칸은 흰색 계열 바탕과 어두운 글자, 버튼은 청회색, 선택·포커스는 청록색으로 구분합니다. 하단 단축키 영역은 별도 청회색 배경으로 표시합니다.

| 키 | 동작 |
| --- | --- |
| F1 | 단축키 도움말 열기 / 닫기 |
| Esc / Alt+X | 도움말·접속·설정 화면 닫기 |
| F2 | 연결 / 재연결 |
| F3 | 언어·마우스 설정 |
| Ctrl+Enter / F4 | 편집기 내용 실행 |
| F5 | COMMIT |
| F6 | ROLLBACK |
| F8 | 실행 취소 요청 |
| F9 / F10 | 편집기 / 결과 표로 바로 이동 |
| Tab / Shift+Tab | 편집기 / 결과 표 이동 |
| Ctrl+Enter / Alt+C | 접속 화면에서 연결 |
| Alt+H/P/U/W/T/N | 접속 화면의 호스트/포트/계정/비밀번호/접속 방식/이름 선택 |
| Alt+R / Alt+A | 접속 화면의 저장된 접속 / 별칭 선택 |
| Ctrl+S / Alt+S | 접속 화면에서는 접속 정보 저장, 설정 화면에서는 설정 저장 |
| Alt+L / Alt+M | 설정의 언어 / 마우스 항목 선택 |
| Enter | 편집기에서 줄바꿈 |
| Ctrl+Q / Ctrl+C | 종료, 미완료 트랜잭션 롤백 |

Ctrl+Enter는 터미널이 Ctrl 조합을 구분해 전달할 때 동작합니다. 일반 Enter와 같은 입력으로 전달되는 환경에서는 F4를 사용합니다. 도움말은 접속 화면에서도 열 수 있으며 F1 또는 Esc로 닫으면 이전 화면과 입력 위치로 돌아갑니다.

SELECT와 WITH 조회는 결과 표로 표시합니다. 결과는 최대 1,000행까지 가져옵니다. DML은 수동 커밋 방식입니다. **Oracle DDL은 암시적 COMMIT을 발생시킬 수 있습니다.** PL/SQL 프로시저 내부 COMMIT도 Oracle의 동작을 따릅니다. 재연결에 성공하면 이전 연결을 롤백하고 닫습니다. 실행은 백그라운드 작업으로 처리하며 60초 제한과 취소 요청을 적용합니다. 드라이버·네트워크 상태에 따른 취소 동작은 실DB 검증이 필요합니다.

한 번에 SQL 한 문장 또는 `BEGIN`/`DECLARE` PL/SQL 블록 하나를 실행합니다. 일반 SQL 끝의 세미콜론은 제거하고 PL/SQL 내부 세미콜론은 유지합니다. PL/SQL 끝의 별도 줄 `/`는 제거합니다. SQL*Plus 스크립트, 여러 문장 일괄 실행, CREATE PROCEDURE 스크립트, 바인드 변수 입력, REF CURSOR/DBMS_OUTPUT 출력은 아직 지원하지 않습니다. 현재 LOB 표시와 대형 단일 셀 메모리 사용량도 추가 검증이 필요합니다.

## 검증

검증 환경: Go 1.26.9, Linux ARM64. 접속 문자열·트랜잭션·결과 행 제한 테스트와 tcell 모의 터미널에서 데모 화면·키 입력 테스트가 통과했습니다. 설정 저장·재읽기·저장 실패 처리, 언어 변경 시 입력 보존, 접속·설정 단축키, 마우스 버튼·입력칸·도움말 스크롤과 마우스 끄기·다시 켜기도 자동 검사합니다. `go vet`과 CGO 없는 Linux AMD64/ARM64 빌드도 통과했습니다. AMD64 파일은 `file`과 `readelf`로 정적 링크를 확인했고, ARM64 파일은 실제 터미널에서 데모 시작·종료를 확인했습니다.

2026-10-15에 Docker Oracle Free 26ai (`23.26.3-slim`)에서 Service Name 접속, 샘플 데이터와 한글·NULL·NUMBER·DATE·TIMESTAMP 조회, DML 실행 후 롤백을 확인했습니다. 기존 `go-ora/v2 v2.9.0`은 같은 계정으로 인증이 실패하여 `go-ora/v3 v3.0.1`로 변경했습니다. 이 드라이버는 Go 1.24 이상을 필요로 하며 이번 검증에는 Go 1.26.9를 사용했습니다.

Oracle 10g 접속 시험에서 `network.newAcceptPacketFromData`의 `slice bounds out of range [41:32]` 패닉이 보고되어, 짧은 TNS ACCEPT 응답의 길이를 확인하는 [업스트림 수정](https://github.com/sijms/go-ora/commit/9197bc12b6fea25726fa028e48c731468cd05b18)을 포함한 `go-ora/v3 v3.0.2-0.20260731135026-9197bc12b6fe`로 고정했습니다. 이 패킷 파싱 수정과 Oracle 10g 전체 접속·쿼리 지원 검증은 별개이며, 실제 10g 서버에서 재시험이 필요합니다.

실제 SID 접속, COMMIT·PL/SQL·취소·재연결 동작의 전체 검증, 구형 Oracle 버전 및 CentOS 7.4 실행은 아직 남아 있습니다.

```bash
go test -race ./...
go vet ./...
```

실DB에서는 Service Name/SID 각각의 연결, `SELECT sysdate FROM dual`, 한글·NULL·NUMBER·DATE 결과, DML 후 COMMIT/ROLLBACK, PL/SQL, 실행 취소와 재연결을 확인해야 합니다. 트랜잭션 검증은 테스트 스키마에서 수행합니다.

## 다음 단계

1. 실제 Oracle 버전별 접속·쿼리·취소·트랜잭션 검증
2. CentOS 7.4 VM에서 실행 및 터미널 검증
3. 스키마·테이블·뷰 탐색, SQL 이력, 접속 비밀번호 암호화 저장 방식 검토
4. 바인드 변수, CSV 내보내기, 객체 DDL·실행 계획

참고: [lazysql](https://github.com/jorgerojas26/lazysql), [go-ora](https://github.com/sijms/go-ora).

## 개발자·라이선스

개발자: 이채강 (Chaekang Lee), leechis7@gmail.com

Copyright (C) 2026 Chaekang Lee. LazyOra는 **GNU Affero General Public License v3.0 (AGPL-3.0-only)**로 배포합니다. 전문은 [LICENSE](LICENSE)를 참조하세요. 의존 라이브러리에는 각 프로젝트의 라이선스가 적용됩니다.

앱 버전과 개발자·라이선스 정보의 원본은 `internal/appinfo/info.go`입니다. 일반 `go build`·`go run`과 Makefile 빌드가 동일한 버전을 사용합니다. 버전 변경 시 이 파일과 README의 현재 버전을 함께 갱신합니다.
