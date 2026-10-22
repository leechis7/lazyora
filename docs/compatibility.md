# 호환성·장애 회귀 검증

검증일: 2026-10-22. 대상: v0.7.0 이후 개발 변경(LORA-51·52).
성공한 빌드, 모의 터미널 검사, 실제 터미널·DB 시험을 구분합니다.

로컬 결과: `go test -race ./...`, `go vet ./...`, Linux ARM64 기본 빌드 통과.
공통 DB 검사와 전체 Free 통합 검사(약 55.9초), 실제 실행파일 PTY 3종 모두 통과했습니다.
PTY는 기본 실행·출력·객체·페이지·내보내기·재접속, 셀/편집/테마, 접속 작업 공간 격리를 포함합니다.
시험 후 `docker compose down`으로 컨테이너를 종료했으며 DB 볼륨은 보존했습니다.

## 환경별 상태

| 대상 | 이번 검증 | 남은 확인 |
| --- | --- | --- |
| Linux ARM64 / Oracle Free 26ai | 실DB 통합 검사, 실제 실행파일 PTY, 접속별 트랜잭션 격리 | 사용 중인 실제 터미널의 클립보드 수신 |
| Oracle 10g | 외부 서버용 공통 회귀 검사 준비 | 실제 서버의 Service Name/SID·문자셋·SQL 기능 시험 |
| Oracle 19c | 외부 서버용 공통 회귀 검사 준비 | 실제 서버 시험 |
| CentOS 7.4 / SecureCRT | 과거 사용자 확인: 접속·커서·8색 표시 | 이번 신규 기능 재시험, 정확한 OS·터미널 버전 기록 |
| Windows / macOS | CI에 모의 UI·내보내기 오류 검사 추가 | 이번 통합 CI 실행 및 실제 TUI·키·마우스·클립보드·DB 시험 |
| 6종 배포 대상 | 기존 CI의 교차 빌드 및 Windows 버전 명령 | 이번 변경 통합 CI 실행 |

Linux 컨테이너는 호스트 커널을 사용하므로 CentOS 7.4 커널 시험을 대신하지 않습니다.
Oracle Free의 PDB Service Name 시험은 10g/19c 또는 SID 시험을 대신하지 않습니다.

## 실DB 공통 검사

로컬 Free가 실행 중이면 다음을 사용합니다. DB의 시작·종료는 별도로 합니다.

```sh
make -C test/oracle up
make -C test/oracle compat-test
make -C test/oracle test
make -C test/oracle down
```

`compat-test`는 Free 전용 표본·관리 뷰·`DBMS_SESSION.SLEEP`·12c 이후 SQL에 의존하지 않습니다.
접속한 서버 버전·클라이언트 OS/CPU·접속 방식을 기록하고 다음을 검사합니다.

- 30바이트 미만 고유 이름의 시험 테이블 생성·정리
- 한글·38자리 NUMBER·NULL 바인드·CLOB, PL/SQL과 DBMS_OUTPUT
- 객체·컬럼 조회와 동일 커서에서 2,507행 읽기
- 다른 연결에서 미커밋 변경이 보이지 않음, 명시적 COMMIT/ROLLBACK
- 메타데이터·출력·페이지 조회가 미커밋 변경을 보존함

외부 Oracle 10g/19c는 **별도 시험 계정**을 준비합니다. 접속·CREATE TABLE·테이블스페이스 할당량과
DBMS_OUTPUT 실행 권한이 필요합니다. 검사에서 생성한 테이블만 삭제하며 기존 객체를 바꾸지 않습니다.
DDL에는 암시적 커밋이 있으므로 평소 작업 중인 접속을 재사용하지 않습니다.
한글을 표현할 수 있는 DB 문자셋이 필요합니다. 이 검사는 모든 SQL 기능의 완전한 인증 시험이 아닙니다.

Linux/macOS Bash 예시(암호는 기록되지 않는 프롬프트에서 입력):

```bash
export LAZYORA_COMPAT_HOST=db-test.example
export LAZYORA_COMPAT_PORT=1521
export LAZYORA_COMPAT_USER=lazyora_test
export LAZYORA_COMPAT_MODE=service  # SID 시험은 sid
export LAZYORA_COMPAT_NAME=ORCL
read -r -s -p 'DB password: ' LAZYORA_COMPAT_PASSWORD; echo
export LAZYORA_COMPAT_PASSWORD
sh test/oracle/compat-test.sh --external
unset LAZYORA_COMPAT_PASSWORD
```

Windows에서는 같은 `LAZYORA_COMPAT_*` 환경 변수와 `LAZYORA_COMPAT=1`을 설정하고
`go test ./test/oracle -run '^TestOracleCompatibilityIntegration$' -v -count=1 -timeout=3m`을 실행합니다.
Service Name과 SID는 각 방식으로 계정이 인증되는 올바른 대상을 별도로 지정하여 각각 기록합니다.
외부 모드에서는 로컬 `.env`를 읽거나 Docker를 조작하지 않습니다.

## 장애·대형 데이터 검사

`make -C test/oracle test`에 다음 검사를 포함합니다.

- **실제 접속 단절:** 시험 전용 접속에서 INSERT 후 COMMIT/ROLLBACK 직전에, 별도 관리 접속의
  `ALTER SYSTEM DISCONNECT SESSION ... IMMEDIATE`로 해당 시험 세션만 종료합니다.
  클라이언트가 반영 결과를 불확실 상태로 유지하고 반복 처리·추가 SQL을 차단하는지,
  별도 접속에서 미커밋 INSERT가 보이지 않는지, 새 연결을 열 수 있는지 검사합니다.
  인스턴스·다른 접속은 종료하지 않습니다. 로컬 Free 계정에 기존 부여한 관리 권한이 필요합니다.
  [Oracle의 세션 종료 동작](https://docs.oracle.com/en/database/oracle/oracle-database/19/sqlrf/ALTER-SYSTEM.html)을 사용한 시험으로,
  COMMIT이 서버에 도착한 후 응답만 유실되는 상황이나 응답 없는 방화벽 단절은 검증하지 않습니다.
  단순 TCP 중계 프록시는 이 환경에서 Oracle 인증이 완료되지 않아 회귀 시험에서 제외했습니다.
- **넓은 결과:** 2,000행 × 16컬럼 × 셀당 4,000바이트를 조회하여 실제 보관 추정량이
  64 MiB를 넘기 전에 페이지 커서를 닫고 이미 가져온 값을 보존하는지 검사합니다.
- **기존 회귀:** 대형 Unicode CLOB/BLOB 원문·전체 스트림 저장, 잠금 충돌·안전 편집,
  느린 네트워크 가져오기 취소, 전체 내보내기 취소와 미커밋 DML 보존을 함께 실행합니다.

`go test ./internal/exportdata`는 CSV·TSV·INSERT·XLSX 임시 파일을 일부 기록한 다음
`ENOSPC`를 주입하여 원본 파일·파일 정보 보존과 임시 파일·잠금 정리를 확인합니다.
실제 디스크를 채우는 시험은 아니며, 파일시스템의 전원 장애 내구성이나 Sync 실패까지 입증하지 않습니다.

64 MiB는 **F9 페이지 조회 실행 하나의 보관 추정 한도**입니다. 전체 프로세스 RSS의 상한이 아닙니다.
드라이버 수신 버퍼·일시 할당·UI·기존 탭과 접속 작업 공간의 결과는 추가 메모리를 사용합니다.
여러 탭의 결과 합산을 제한하는 전역 메모리 예산은 아직 없습니다.
F5 스크립트의 각 SELECT는 1,000행 한도이며, 스크립트 전체 결과에 대한 합산 바이트 제한은 없습니다.
넓은 SELECT를 여러 개 포함한 대형 스크립트는 이 페이지 조회 시험의 보장 범위 밖입니다.

2026-10-22 Linux ARM64·Go 1.26.9·Oracle Free 26ai의 단독 넓은 결과 검사에서
486행을 보관하고 커서를 닫았습니다. 보관 추정량은 67,029,120바이트,
문자열 원문은 31,104,000바이트, GC 후 Go 힙 증가 관측값은 32,573,296바이트였습니다.
누적 할당은 131,840,736바이트, 조회·검사·GC를 포함한 구간은 약 1.039초였습니다.
이는 한 번의 관측값으로 최대 RSS나 다른 환경의 지연·메모리를 보장하지 않습니다.

대형 LOB는 4,096단위 미리보기, 확장 미리보기 256 Ki단위, 전체 결과 내보내기의 단일 LOB는
2 Mi단위 제한을 사용합니다. 개별 셀 저장은 스트리밍합니다.

## 실제 터미널 수동 확인표

Windows Terminal/PowerShell, macOS Terminal/iTerm2, CentOS/SecureCRT 각각에서 기록합니다.
환경 수집에는 `--version`, OS·CPU, 터미널 버전, 로캘, TERM·색상 수를 포함합니다.
DB 암호·마스터 암호는 로그나 화면 캡처에 포함하지 않습니다.

| 동작 | 확인할 결과 |
| --- | --- |
| 시작·Alt+L 접속 | 한글 입력·표시, 입력/선택 구분, 기본 취소 |
| Ctrl+Enter·Alt+Enter·F9 | 터미널이 전달하는 키별 SQL 실행, 안 되는 키는 실제 매핑 기록 |
| 화살표·Tab·Alt 단축키·마우스 | 포커스·버튼 선택·팝업 조작, 작은 창에서 항목 접근 |
| SQL 선택·원문/행/범위 복사 | OSC 52 지원 여부와 PC 클립보드의 실제 값 비교 |
| F2·Shift+F2·색상·크기 변경 | 창 복구, 8/16색·256색·true color, 커서 잔상 없음 |
| Alt+J 두 접속 전환 | 접속 표시, 탭·결과·트랜잭션 격리, 개별 닫기 |
| CSV/TSV/XLSX/INSERT·셀 저장 | 경로·한글·NULL·원본 파일 보존, 실제 저장 파일 확인 |
| 취소·재접속·종료 | 미커밋 경고와 기본 취소, 명시한 커밋·롤백 결과 별도 조회 |

모의 터미널과 Linux PTY가 통과해도 위 운영체제의 실제 클립보드·키 매핑은 미검증으로 남습니다.
