# Oracle Free 시험 환경

아래 명령은 프로젝트 루트에서 실행합니다.

Docker Engine/Desktop과 Docker Compose가 설치된 개발 머신에서 실행합니다. ARM64·AMD64를 지원하는 `gvenzl/oracle-free:23.26.3-slim`을 사용하며 이미지가 CPU에 맞춰 선택됩니다. 최초 다운로드와 DB 초기화에는 수 분이 걸릴 수 있습니다. 이미지와 DB 데이터용으로 수 GB의 디스크, DB 실행용으로 약 4GB의 여유 메모리를 확보하는 편이 좋습니다.

```bash
make -C test/oracle up
make -C test/oracle status
make -C test/oracle test
```

`up`은 `.env`가 없으면 임의의 로컬 비밀번호를 생성하고, DB가 healthy 상태가 될 때까지 기다립니다. `.env`는 Git에서 제외하며 기존 파일이 있으면 덮어쓰지 않습니다. 직접 설정하려면 `.env.example`을 `.env`로 복사하고 두 비밀번호를 채웁니다. `make test`는 실제 Go 드라이버로 접속하여 샘플 데이터·한글·NULL·날짜와 DML 롤백을 확인합니다. 기존 자동 테스트에서는 이 실DB 테스트를 건너뜁니다.

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

`run.sh`가 로컬 `.env`를 읽어 접속 화면의 기본값을 채웁니다. 실행 후 **F2 → Ctrl+Enter 또는 Alt+C**로 접속하고 **Ctrl+Enter 또는 F4**로 편집기의 기본 조회문을 실행합니다. 접속 버튼을 마우스로 클릭해도 됩니다. **F1**은 단축키 도움말, **F3**은 언어·마우스 설정입니다. 입력 항목은 클릭하거나 Alt+H/P/U/W/T/N으로 선택할 수 있습니다. 비밀번호를 직접 복사하거나 명령행에 입력할 필요는 없습니다. DB 접속은 사용자가 접속을 실행할 때 이루어집니다.

이미 DB가 실행 중이고 빌드도 되어 있으면 다음 명령으로 실행파일만 바로 엽니다. CPU 아키텍처에 맞는 파일을 자동 선택합니다.

```bash
sh test/oracle/run.sh
```

첫 DB 초기화 때 `init.sql`로 `lazyora.sample_rows` 테이블과 샘플 3행을 생성합니다. 한글, NULL, 양수·음수 소수, DATE, TIMESTAMP를 포함합니다. TUI에서 F4로 다음 SQL을 실행해 확인합니다.

```sql
SELECT * FROM sample_rows ORDER BY id;
```

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

