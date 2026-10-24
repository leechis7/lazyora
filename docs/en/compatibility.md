# Compatibility and verification scope

[한국어](../compatibility.md) | **English** · [Build](build.md)

## Current summary: v0.10.2, 2026-10-24

Tuivory v0.0.1 integration passed full race/vet, local Linux ARM64 build, six-target Release builds, Windows CLI execution and Windows/macOS simulated UI/settings checks. Subsequent documentation main/dev CI also passed. The UI comparison covered 3,216 simulated frames across 12 theme/profile combinations, 67 scenes, both languages and fresh/roundtrip states. Only random test-path text was normalized; no captured visual differences remained.

These are not new real-database or remote-terminal tests. The dated 2026-10-22 record remains the source for Oracle Free and Linux PTY evidence. References to a pending integration CI in that historical table describe the old date; later CI has passed.

| Environment | Evidence | Remaining distinction |
| --- | --- | --- |
| Linux ARM64 / Oracle Free | Earlier real DB integration and executable PTY checks | Actual client clipboard delivery |
| Oracle 10g/19c | Shared external-server test harness; short ACCEPT packet fix for the reported 10g panic | Server/version-specific SQL, transaction, charset and Service Name/SID tests |
| CentOS 7.4 / SecureCRT | Earlier user confirmation of connection/cursor/8-color behavior | Every new feature and exact OS/client version matrix |
| Windows/macOS | CI simulated UI/settings/export checks; Windows AMD64 CLI | Real TUI keys/mouse/clipboard/database sessions |
| Six binary targets | CI/Release cross-builds | Cross-build is not full runtime certification |

Linux containers share the host kernel. Oracle Free PDB Service Name testing cannot substitute for older Oracle or SID tests. Published compatibility claims should distinguish mock tests, simulated rendering, actual PTY execution and real DB results.

## Opt-in real-database checks

```sh
make -C test/oracle up
make -C test/oracle compat-test
make -C test/oracle test
make -C test/oracle down
```

`compat-test` avoids Free-specific fixtures/admin views and newer SQL assumptions. It uses uniquely named test tables and checks Unicode, precise numbers, NULL binds, PL/SQL/DBMS_OUTPUT, objects/columns, 2,507-row cursor paging and transaction isolation/preservation.

Use a dedicated test account for external 10g/19c, with connection, CREATE TABLE, tablespace quota and DBMS_OUTPUT access as needed. DDL can commit implicitly; never reuse a session with unrelated pending work. See [Oracle testing](oracle-testing.md) and the [dated detailed commands (Korean)](../compatibility.md).

The historical record also covers error injection, lost connections, large results/LOBs and real-terminal checklists. Those are environment-specific observations, not guarantees that cancellation, clipboard delivery or every SQL feature works identically everywhere.
