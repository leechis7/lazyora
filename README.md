# LazyOra

Oracle 쿼리 작성·실행을 위한 Go TUI의 첫 구현입니다. lazysql의 화면 구성을 참고하여 Oracle 전용으로 작성했습니다.

## 빌드와 실행

mise와 인터넷 연결이 있는 개발 머신에서:

```bash
mise install
mise exec -- go mod tidy
mise exec -- make build
./bin/lazyora --demo
./bin/lazyora
```

ARM64 개발 머신에서는 `mise exec -- make build-native` 후 `./bin/lazyora-native --demo`로 화면을 확인합니다. `bin/lazyora`는 CentOS 배포용 Linux AMD64 실행파일입니다.

`make build`는 `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1`로 빌드합니다. Oracle Instant Client를 설치하지 않는 단일 실행파일을 목표로 합니다. 실행 대상 서버에는 Go가 필요하지 않습니다.

CentOS 7.4 x86_64 호환성은 목표이며 아직 검증되지 않았습니다. 실제 3.10 커널의 VM 또는 서버에서 실행해야 지원을 확정할 수 있습니다. 컨테이너만으로 구형 커널 호환성을 검증할 수는 없습니다.

```bash
uname -r
uname -m
file bin/lazyora
ldd bin/lazyora
```

정적 실행파일에서는 `ldd`가 동적 실행파일이 아니라는 메시지를 출력할 수 있습니다.

## 연결

F2로 접속 화면을 열어 Host, Port, Username, Password를 입력합니다. 연결 유형에서 **Service Name** 또는 **SID**를 선택하고 해당 이름을 입력합니다. 암호는 화면에서 가리고 디스크에 저장하지 않습니다. 선택적으로 `LAZYORA_HOST`, `LAZYORA_PORT`, `LAZYORA_USER`, `LAZYORA_PASSWORD`, `LAZYORA_NAME` 환경변수로 입력 기본값을 지정할 수 있습니다. 암호를 명령행 인자로 받지 않습니다.

## 조작

| 키 | 동작 |
| --- | --- |
| F2 | 연결 / 재연결 |
| F4 | 편집기 내용 실행 |
| F5 | COMMIT |
| F6 | ROLLBACK |
| F8 | 실행 취소 요청 |
| Tab | 편집기 / 결과 표 이동 |
| Ctrl+Q / Ctrl+C | 종료, 미완료 트랜잭션 롤백 |

SELECT와 WITH 조회는 결과 표로 표시합니다. 결과는 최대 1,000행까지 가져옵니다. DML은 수동 커밋 방식입니다. **Oracle DDL은 암시적 COMMIT을 발생시킬 수 있습니다.** PL/SQL 프로시저 내부 COMMIT도 Oracle의 동작을 따릅니다. 재연결에 성공하면 이전 연결을 롤백하고 닫습니다. 실행은 백그라운드 작업으로 처리하며 60초 제한과 취소 요청을 적용합니다. 드라이버·네트워크 상태에 따른 취소 동작은 실DB 검증이 필요합니다.

한 번에 SQL 한 문장 또는 `BEGIN`/`DECLARE` PL/SQL 블록 하나를 실행합니다. 일반 SQL 끝의 세미콜론은 제거하고 PL/SQL 내부 세미콜론은 유지합니다. PL/SQL 끝의 별도 줄 `/`는 제거합니다. SQL*Plus 스크립트, 여러 문장 일괄 실행, CREATE PROCEDURE 스크립트, 바인드 변수 입력, REF CURSOR/DBMS_OUTPUT 출력은 아직 지원하지 않습니다. 현재 LOB 표시와 대형 단일 셀 메모리 사용량도 추가 검증이 필요합니다.

## 검증

개발 환경: mise Go 1.26.9, Linux ARM64. 접속 문자열·트랜잭션·결과 행 제한 테스트와 tcell 모의 터미널에서 데모 화면·키 입력 테스트가 통과했습니다. `go vet`과 CGO 없는 Linux AMD64/ARM64 빌드도 통과했습니다. AMD64 파일은 `file`과 `readelf`로 정적 링크를 확인했고, ARM64 파일은 `TERM=xterm-256color` 터미널에서 데모 시작·종료를 확인했습니다. 실제 Oracle 접속과 CentOS 7.4 실행은 아직 검증하지 않았습니다.

```bash
mise exec -- go test -race ./...
mise exec -- go vet ./...
```

실DB에서는 Service Name/SID 각각의 연결, `SELECT sysdate FROM dual`, 한글·NULL·NUMBER·DATE 결과, DML 후 COMMIT/ROLLBACK, PL/SQL, 실행 취소와 재연결을 확인해야 합니다. 트랜잭션 검증은 테스트 스키마에서 수행합니다.

## 다음 단계

1. 실제 Oracle 버전별 접속·쿼리·취소·트랜잭션 검증
2. CentOS 7.4 VM에서 실행 및 터미널 검증
3. 접속 프로필, 스키마·테이블·뷰 탐색, SQL 이력
4. 바인드 변수, CSV 내보내기, 객체 DDL·실행 계획

참고: [lazysql](https://github.com/jorgerojas26/lazysql), [go-ora](https://github.com/sijms/go-ora).
