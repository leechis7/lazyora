# Oracle Free test environment

[한국어](../../test/oracle/README.md) | **English** · [Compatibility](compatibility.md)

Keep Compose files, environment files, SQL fixtures, scripts and real-database tests in `test/oracle/`. Run the commands below from the repository root. Docker Engine/Desktop and Compose are required; the pinned `gvenzl/oracle-free:23.26.3-slim` image supports the configured AMD64/ARM64 test setup. Database initialization/download takes time and needs substantial memory/disk space.

## Start and test

```sh
make -C test/oracle up
make -C test/oracle status
make -C test/oracle test
make -C test/oracle down
```

`up` creates an ignored `.env` with random local passwords when absent, preserves an existing file and waits for health. To configure manually, copy `.env.example` and set both passwords. Real DB checks are opt-in; ordinary unit tests skip them. `down` stops containers and retains the volume.

| Connection field | Value |
| --- | --- |
| Host | `127.0.0.1` |
| Port | `1521`, unless `ORACLE_PORT` overrides it |
| User | `lazyora` |
| Password | Generated/local `APP_USER_PASSWORD` in `.env` |
| Mode | Service Name |
| Name | `FREEPDB1` |

The port is bound to localhost. The app user is created in the PDB; SID `FREE` targets the CDB and is not interchangeable with `FREEPDB1` for this user. Use Service Name first.

## Interactive testing and cleanup

```sh
make -C test/oracle run
```

This explicitly builds the **native** executable, starts/reuses Oracle Free if needed, waits for health and runs LazyOra. It differs from the normal `make build` Linux ARM64 default. On app exit, the launcher asks in Korean whether to stop Oracle Free (`[Y/n]`). Enter/Y releases container memory using Compose down; n keeps it running. Noninteractive/signal exits clean up automatically. Volumes remain; stopping containers also disconnects other tests sharing this environment. The launcher is for Linux/macOS.

```sh
make -C test/oracle compat-test
make -C test/oracle tui-test
```

`compat-test` runs the shared DB checks. `tui-test` builds natively and performs real executable PTY tests; it is not ordinary cross-platform simulated rendering. Read the script's setup and use a dedicated test environment. External Oracle 10g/19c commands and privileges are in the [detailed compatibility record (Korean)](../compatibility.md); no full support is implied by the harness existing.

The DB-free `python3 test/oracle/tui_terminal_smoke.py` checks missing/empty TERM and terminal startup/exit using the demo binary. `tui_fullscreen_smoke.py` checks fullscreen and function-key encodings. Prepare the expected Linux ARM64 binary with `make build` in the project's local environment. These do not certify actual remote clients.

## Privileges and fixtures

The test initialization grants the local PDB app user **DBA and SYSDBA**. For an existing volume, `make -C test/oracle grant-admin` applies `grants.sql` without resetting data. This is a local test privilege choice, not a recommendation for a production account or a CDB common administrator. LazyOra's connection form does not expose an AS SYSDBA selector; use SQL*Plus explicitly when testing that mode.

`make -C test/oracle browser-fixture` loads the browser fixture. `export-bench` is an explicit performance test, separate from normal tests. Historical timings remain in [the dated benchmark record](../../test/oracle/benchmarks/export-2026-10-20.md) and are not new measurements.

Do not commit `.env`, credentials or real data. Tests create/clean their own temporary objects and exercise DML/DDL, paging, output, reconnection, transaction isolation and supported types. DDL can commit implicitly; never point these tests at an unrelated working transaction. See the Korean environment README for the full fixture and manual SQL procedures.
