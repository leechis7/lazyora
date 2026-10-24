# Support and security reports

[한국어 지원 안내](../../SUPPORT.md) · [한국어 보안 안내](../../SECURITY.md) · [English README](../../README.en.md)

## Help and issue reports

Start with the [user guide](../guide.en.md), [terminal troubleshooting](terminal.md), [build notes](build.md) and [credential management](credentials.md). F1 in the app lists controls for the current screen and editor keymap. The Help menu also contains environment diagnostics and version history.

For bugs, feature proposals and usage questions, use [GitHub Issues](https://github.com/leechis7/lazyora/issues). Korean and English reports are welcome. Include:

- LazyOra version (`--version`), OS/kernel and CPU architecture.
- Oracle version and Service Name or SID connection mode.
- Steps to reproduce, expected behavior and the actual error.
- For display/input issues: terminal client and emulation, `TERM`, `locale charmap` and `tput colors`.

Remove passwords, private connection details and real business data before sharing logs or screenshots. LazyOra is under active development; response times and support for every environment are not guaranteed. Historical release notes and detailed audit records are primarily in Korean; see [GitHub Releases](https://github.com/leechis7/lazyora/releases) for version-specific notes.

## Security reports

Do not post vulnerability details or credentials in public issues or PRs. Email **leechis7@gmail.com**, with `[LazyOra 보안]` (LazyOra security) in the subject. Include the affected version/environment, expected impact, reproduction steps, a minimal example with secrets removed, and a reply address.

Email is not guaranteed to be end-to-end encrypted. Do not send passwords or production data. Coordinate public disclosure with the maintainer while a fix and guidance are prepared.

Security fixes prioritize current development and subsequent releases. The project does not guarantee long-term support for every older version or a response deadline. Connection profiles contain no database passwords; optional encrypted passwords live in a separate vault. SQL text, Oracle errors and query results can still contain sensitive information.
