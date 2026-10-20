# Oracle Free 시험 환경

아래 명령은 프로젝트 루트에서 실행합니다.

Docker Engine/Desktop과 Docker Compose가 설치된 개발 머신에서 실행합니다. ARM64·AMD64를 지원하는 `gvenzl/oracle-free:23.26.3-slim`을 사용하며 이미지가 CPU에 맞춰 선택됩니다. 최초 다운로드와 DB 초기화에는 수 분이 걸릴 수 있습니다. 이미지와 DB 데이터용으로 수 GB의 디스크, DB 실행용으로 약 4GB의 여유 메모리를 확보하는 편이 좋습니다.

```bash
make -C test/oracle up
make -C test/oracle status
make -C test/oracle test
```

`up`은 `.env`가 없으면 임의의 로컬 비밀번호를 생성하고, DB가 healthy 상태가 될 때까지 기다립니다. `.env`는 Git에서 제외하며 기존 파일이 있으면 덮어쓰지 않습니다. 직접 설정하려면 `.env.example`을 `.env`로 복사하고 두 비밀번호를 채웁니다. `make test`는 실제 Go 드라이버로 접속하여 샘플 데이터·한글·NULL·날짜, 스키마·13종 객체·컬럼 조회와 탐색 전후 DML 롤백을 확인합니다. 기존 자동 테스트에서는 이 실DB 테스트를 건너뜁니다.

재접속 검사도 `make test`에 포함됩니다. 전용 임시 테이블을 만들어 미커밋 변경이 있는 접속의 확인 없는 전환 차단, 잘못된 암호로 새 접속 실패 시 기존 변경 유지, 명시한 커밋·롤백 후 재접속의 실제 저장 결과를 확인하고 임시 테이블을 삭제합니다. 샘플 데이터는 변경하지 않습니다.

| 접속 항목 | 값 |
| --- | --- |
| Host | `127.0.0.1` |
| Port | `1521` (`.env`의 `ORACLE_PORT`로 변경 가능) |
| Username | `lazyora` |
| Password | 로컬 `.env`의 `APP_USER_PASSWORD` 값 |
| Connection type | `Service Name` |
| Service Name | `FREEPDB1` |

포트는 `127.0.0.1`에만 공개합니다. 앱 사용자는 PDB `FREEPDB1`에 생성됩니다. 이 사용자로 SID `FREE`에 연결하면 접속 대상이 CDB여서 같은 계정으로 인증되지 않을 수 있으므로, 우선 위 Service Name 설정을 사용합니다.

## 테스트 계정의 관리자 권한

`grants.sql`은 `lazyora` 계정에 `FREEPDB1` 내부의 `DBA` 역할과 `SYSDBA` 권한을 부여합니다. 새 DB를 초기화할 때 자동 적용합니다. 이미 생성된 테스트 DB에는 데이터를 초기화하지 않고 다음 명령으로 적용합니다.

```bash
make -C test/oracle grant-admin
```

기존 계정과 비밀번호를 그대로 사용합니다. 일반 접속에도 DBA 역할이 적용되므로 관리용 뷰를 조회할 수 있습니다. SYSDBA 권한으로 별도 접속할 때는 `AS SYSDBA` 옵션이 필요합니다. 현재 LazyOra 접속 화면에는 이 옵션 선택이 없으므로 아래 명령으로 SQL*Plus에서 확인할 수 있습니다. 비밀번호 프롬프트에는 `.env`의 `APP_USER_PASSWORD`를 입력합니다.

```bash
docker compose -f test/oracle/compose.yaml exec oracle \
  sqlplus -L lazyora@//localhost:1521/FREEPDB1 as sysdba
```

이 권한은 Compose의 로컬 시험용 PDB에 적용합니다. CDB 루트 `FREE`의 공통 관리자 계정을 만드는 것은 아닙니다.

## LazyOra 실행파일로 직접 시험

다음 명령 하나로 테스트 DB 준비, 현재 머신용 실행파일 빌드, LazyOra 실행까지 진행합니다.

```bash
make -C test/oracle run
```

`run.sh`로 실행한 LazyOra를 종료하면 **이 시험 환경의 `docker compose down`도 자동 실행**해 Oracle 컨테이너의 메모리를 반환합니다. 앱이 오류로 종료돼도 정리를 시도하며, DB 데이터 볼륨은 삭제하지 않습니다. 다시 시험하려면 `make -C test/oracle run`으로 시작하세요. 이 환경을 함께 사용하는 다른 시험 연결도 종료됩니다.

`run.sh`가 로컬 `.env`를 읽어 접속 화면의 기본값을 채웁니다. 실행 시 먼저 열린 접속 화면에서 **Ctrl+Enter 또는 Alt+C**로 접속하고 **Ctrl+Enter·Alt+Enter 또는 F9**로 편집기의 기본 조회문을 실행합니다. 접속 버튼을 마우스로 클릭해도 됩니다. 접속 화면은 Alt+L로 다시 엽니다. **F1**은 단축키 도움말, **도구 → 설정**은 언어·마우스 설정입니다. 입력 항목은 클릭하거나 Alt+H/P/U/W/T/N으로 선택할 수 있습니다. 비밀번호를 직접 복사하거나 명령행에 입력할 필요는 없습니다. DB 접속은 사용자가 접속을 실행할 때 이루어집니다.

미커밋 변경이 남아 있는 상태에서 Alt+L로 다른 접속을 시도하면 **커밋 후 접속 / 롤백 후 접속 / 취소**를 확인합니다. 기본 Enter와 Esc는 취소이며 접속 입력을 유지합니다. 새 대상 연결을 먼저 확인하므로 암호가 틀리거나 접속 중 취소되면 기존 트랜잭션은 유지됩니다. 커밋·롤백 자체가 실패하면 전환하지 않습니다. 이후 재접속에는 **DB 반영 결과를 알 수 없음**을 경고하며, 명시적으로 기존 연결을 끊기로 선택해야 복구합니다. 이 경우 SQL을 다시 실행하기 전에 DB에서 반영 결과를 확인해야 합니다.

이미 DB가 실행 중이고 빌드도 되어 있으면 다음 명령으로 실행파일만 바로 엽니다. CPU 아키텍처에 맞는 파일을 자동 선택합니다.

```bash
sh test/oracle/run.sh
```

첫 DB 초기화 때 `init.sql`로 `lazyora.sample_rows` 테이블과 샘플 3행을 생성합니다. 한글, NULL, 양수·음수 소수, DATE, TIMESTAMP를 포함합니다. TUI에서 F9로 다음 SQL을 실행해 확인합니다.

```sql
SELECT * FROM sample_rows ORDER BY id;
```

`browser.sql`은 뷰·함수·프로시저·패키지/본문·시퀀스·동의어·인덱스·트리거·타입/본문·구체화 뷰 표본을 생성합니다. 구체화 뷰 소유자에게 필요한 `CREATE TABLE` 권한은 시험 계정에 직접 부여합니다. 새 DB에는 자동 적용합니다. 기존 DB에서는 아래 명령을 한 번 실행합니다. 기존 테이블과 행은 유지합니다.

```bash
make -C test/oracle browser-fixture
```

접속 후 Tab 또는 클릭으로 왼쪽 탐색기에 이동하여 위쪽 사용자가 `LAZYORA`인지 확인한 뒤 객체 종류 `테이블`의 `SAMPLE_ROWS` 또는 `뷰`의 `SAMPLE_ROWS_VIEW`를 Enter/클릭으로 열면 컬럼 상세를 확인할 수 있습니다. 보기 → 스키마는 스키마 창 표시·숨김 토글입니다. 함수·패키지 등은 기본 정보와 상태를 보여줍니다. Ctrl+R로 새로고침하고 Esc로 돌아옵니다. SQL로 UPDATE 후 객체를 탐색하고 Alt+R로 롤백하면 탐색이 미완료 트랜잭션을 유지하는지도 직접 확인할 수 있습니다.

상태 확인, 로그 보기와 중지:

```bash
make -C test/oracle status
make -C test/oracle logs
make -C test/oracle down
```

`down`은 데이터를 보존합니다. 다시 `up`하면 기존 DB를 사용합니다. `.env`의 비밀번호 변경이나 초기화 SQL 변경은 기존 DB에 자동 반영되지 않습니다. 로그 보기에서 Ctrl+C는 로그 추적만 종료합니다.

DB 콘솔은 `make -C test/oracle shell`로 열고 `.env`의 앱 비밀번호를 프롬프트에 입력합니다. LazyOra TUI로 시험하려면 위 `run` 명령을 사용합니다.

**테스트 DB 데이터를 모두 버리고 처음부터 생성**하려면 다음을 실행합니다. `down -v`는 이 Compose 프로젝트의 Oracle 데이터 볼륨을 삭제합니다. `.env`는 남습니다.

```bash
docker compose -f test/oracle/compose.yaml down -v
make -C test/oracle up
```

참고: [Oracle Free 이미지 및 초기화 동작](https://github.com/gvenzl/oci-oracle-free). 이 DB로 최신 Oracle에서의 동작을 검증할 수 있으며, 구형 Oracle 버전이나 CentOS 7.4 커널 호환성 검증은 별도로 진행합니다.


소스·컴파일 검사는 함수·프로시저·패키지/타입 및 본문·트리거의 소스 조회와 재컴파일, 인자·반환값·오버로드 조회, 오류가 있는 함수의 INVALID 상태·오류 행/열과 수정 후 VALID 복구를 실제 DB에서 확인합니다. `browser-fixture`는 이 검사용 객체를 생성·갱신합니다.


실행파일의 실제 키 바이트·메뉴·범위 복사·전체화면·객체 조회·F5 스크립트·실행 계획·바인드 입력·XLSX 저장·2,507행 전체 CSV 재조회 저장·기본 취소·재접속 실패 시 미커밋 내역 보존·명시적 롤백 후 접속·미저장 확인과 트랜잭션 종료를 Linux PTY로 시험하려면 Python 3가 있는 환경에서 실행합니다. DB가 실행 중이고 browser-fixture가 적용돼 있어야 합니다. 비밀번호는 .env에서 읽으며 출력하지 않습니다.

PTY 시험은 Oracle Free의 `DBMS_SESSION.SLEEP(3)` 실행 중 F1 도움말을 열고 Esc로 닫아도 SQL이 취소되지 않고 완료되는지 확인합니다. 이 시험용 호출은 Oracle Free에서 검증하며 구형 Oracle 지원 여부를 의미하지 않습니다.

```bash
make -C test/oracle tui-test
```

Ctrl+Enter는 CSI-u와 modifyOtherKeys 두 형식, Alt+Enter를 검사합니다. 이것이 SecureCRT 실기 키 매핑 확인을 대신하는 것은 아닙니다. 복사는 터미널에 출력한 OSC 52 페이로드를 확인하며 PC 클립보드의 실제 수신 여부는 터미널에서 확인해야 합니다.

## 대용량 내보내기 성능 측정

`make -C test/oracle export-bench`로 CSV·TSV·INSERT·XLSX의 10만·100만 행을 측정합니다. DB 없는 파일 저장과 Oracle Free 조회 포함 저장을 별도 프로세스로 실행하고, 처리 시간·파일 크기·최대 RSS·Go 힙을 기록한 뒤 실제 파일의 전체 행 수를 검사합니다. 매번 수백 MiB의 임시 파일을 기록하며 종료 후 제거합니다. 일반 `make test`에서는 이 시험을 실행하지 않습니다.

측정 조건·실측 결과·부분 실행 방법은 [대용량 내보내기 측정](benchmarks/export-2026-10-20.md)을 참고하세요.


DBMS_OUTPUT 검사는 출력 켜기·끄기, 한글·빈 줄·30KB 행, 문장별 F5 출력, 최대 행 수와 초과분 정리, SQL 오류 전 출력 보존 및 제어·수집 전후 미커밋 DML의 격리를 검사합니다. 실제 TUI 검사에도 Alt+P 출력 창·켜기·복사·지우기·끄기를 포함합니다. 시험 후 메모리를 반환하려면 `make -C test/oracle down`을 실행하세요.
