# Build and verification

[한국어](../build.md) | **English** · [English README](../../README.en.md)

To run a prebuilt release, follow the README. This page covers source builds and the scope of verification.

## Build from source

Install Go **1.24 or newer** and `make`; run these commands in the repository root. The first build needs network access to download dependencies. You can install Go normally; the version in `mise.toml` is an optional way to match the development toolchain.

```sh
go version
make build
./bin/lazyora-linux-arm64 --demo
./bin/lazyora-linux-arm64 --version
```

**`make build` produces Linux ARM64 only**, the project's local development default. If your machine uses another OS or architecture, choose its target explicitly:

| Target | Output |
| --- | --- |
| `make build-amd64` | `bin/lazyora-linux-amd64` |
| `make build-arm64` | `bin/lazyora-linux-arm64` |
| `make build-windows-amd64` | `bin/lazyora-windows-amd64.exe` |
| `make build-windows-arm64` | `bin/lazyora-windows-arm64.exe` |
| `make build-darwin-amd64` | `bin/lazyora-darwin-amd64` |
| `make build-darwin-arm64` | `bin/lazyora-darwin-arm64` |
| `make build-native` | Executable matching the development machine |

GitHub CI and Release use `make build-all` for all six targets. Builds disable CGO; AMD64 builds use `GOAMD64=v1`. The target machine needs neither Go nor Oracle Instant Client. The app uses go-ora and a local tview TextArea extension for SQL highlighting; common styles come from the pinned Tuivory module in `go.mod`.

## Local checks

```sh
go test -race ./...
go vet ./...
```

These run unit, database-mock and simulated-terminal checks. Real database tests are opt-in and are not implied by a passing ordinary `go test` run.

For the local Docker Oracle Free environment:

```sh
make -C test/oracle up
make -C test/oracle test
make -C test/oracle down
```

Docker Engine/Desktop and Compose are required. `up` creates an ignored local `.env` with random passwords when needed and waits for a healthy database. `down` stops containers while retaining the database volume. For interactive binary testing, `make -C test/oracle run` starts the database if needed and asks whether to stop it when the app exits. Use the generated `APP_USER_PASSWORD`, user `lazyora`, host `127.0.0.1`, port `1521` unless overridden, and Service Name `FREEPDB1`.

This environment grants its test user administrative privileges inside the test PDB. Keep it as a disposable local test environment. Detailed setup and test commands are in [test/oracle/README.md (Korean)](../../test/oracle/README.md).

## Compatibility and evidence

A successful cross-build is not proof of complete interactive support. GitHub CI checks all six builds, Windows AMD64 CLI execution, and simulated UI/settings checks on Windows and macOS. Actual keyboard, clipboard, mouse and database behavior still depends on the client terminal and environment. macOS release binaries are not code-signed or notarized.

CentOS 7.4/SecureCRT connection, cursor and 8-color issues were previously tested by the user. This does not certify every subsequent feature on that environment. Linux containers share the host kernel and cannot validate an older CentOS kernel. Oracle Free Service Name tests do not establish full Oracle 10g/19c or SID coverage. The driver contains the short TNS ACCEPT packet fix prompted by the reported Oracle 10g panic; broader 10g behavior requires server testing.

See [detailed compatibility records (Korean)](../compatibility.md), [build history (Korean)](../build.md) and [Tuivory migration verification (Korean)](../audits/lora-70.md) for dated evidence and remaining checks.

For contributions, use the workflow in [Contributing](contributing.md). Maintainers update `internal/appinfo/info.go`, both README versions and the changelog when preparing a release.
