# Documentation index

[한국어 (primary)](../index.md) | **English**

Use the guides below for current behavior. The app-version source is `internal/appinfo/info.go`, dependency versions are in `go.mod`, and actual published binaries are in [GitHub Releases](https://github.com/leechis7/lazyora/releases). **v0.11.2 uses Tuivory v0.2.0 for shared UI styles.** Earlier TUIS references and independent-project plans are migration/design history, not current dependencies or incomplete extraction work.

## Current guidance

| Area | English | 한국어 |
| --- | --- | --- |
| Introduction, installation, common keys | [README](../../README.en.md) | [README](../../README.md) |
| Detailed feature usage | [Usage](usage.md) | [사용 안내](../usage.md) |
| Guide embedded in the executable | [Offline guide](../guide.en.md) | [내장 가이드](../guide.ko.md) |
| Terminal, color, mouse and copying | [Terminal](terminal.md) | [터미널](../terminal.md) |
| Saved passwords and recovery | [Credentials](credentials.md) | [암호 관리](../credentials.md) |
| Source builds and verification | [Build](build.md) | [빌드](../build.md) |
| PRs, documentation publishing, releases | [Development](development.md) | [개발](../development.md) |
| Shared UI rules and app adapters | [UI styles](ui-style.md) | [UI 규칙](../ui-style.md) |
| Tuivory module/API/update procedure | [Integration](tuivory-sync.md) | [적용 기록](../tuivory-sync.md) |
| Compatibility and evidence | [Compatibility](compatibility.md) | [호환성](../compatibility.md) |
| Local Oracle Free/real DB tests | [Oracle testing](oracle-testing.md) | [시험 환경](../../test/oracle/README.md) |
| Contributions | [Contributing](contributing.md) | [기여](../../CONTRIBUTING.md) |
| Support and private security reports | [Support/security](support.md) | [지원](../../SUPPORT.md) · [보안](../../SECURITY.md) |
| Community rules | [Code of conduct](../../CODE_OF_CONDUCT.en.md) | [행동 규칙](../../CODE_OF_CONDUCT.md) |
| v0.11.2 release description | [English](releases/v0.11.2.md) | [한국어](../releases/v0.11.2.md) |
| v0.11.1 release description | [English](releases/v0.11.1.md) | [한국어](../releases/v0.11.1.md) |
| v0.11.0 release description | [English](releases/v0.11.0.md) | [한국어](../releases/v0.11.0.md) |
| v0.10.2 release description | [English](releases/v0.10.2.md) | [한국어](../releases/v0.10.2.md) |

The [bilingual review record](../audits/lora-71-docs.md) and [machine-readable inventory](../audits/lora-71-docs-inventory.json) describe this refresh.

## Historical evidence and proposals

The [changelog](../../CHANGELOG.md) and `docs/releases/` describe their respective versions, not a single current feature list. Later changes go in Unreleased. Unchanged historical releases are not bulk-translated.

The [LORA-69 survey](../audits/lora-69.md), [implementation](../audits/lora-69-implementation.md), [LORA-70 migration evidence](../audits/lora-70.md) and their HTML/CSV/JSON outputs preserve specific runs. “Not published” or “CI pending” in a dated record describes that time; it does not override later deployment evidence.

The [original style proposal](../ui-style-proposal.md), [principles](../tui-style-guide/principles.md) and [project plan](../tui-style-guide/project-plan.md) preserve design origins. The candidate CSV is not the live Tuivory palette. The [debugger design record](plsql-debugger.md) describes an unimplemented step debugger, distinct from source editing/compilation and routine execution.

The [2026-10-20 export benchmark](../../test/oracle/benchmarks/export-2026-10-20.md) remains a dated measurement; this documentation refresh did not rerun it. Upstream `third_party/tview/` documents/licenses remain dependency provenance; [local fork notes](../../third_party/tview/README.lazyora.md) describe the application extension.

## Maintenance

Write/update current guidance in Korean and English together. Align commands, shortcuts, versions, protection limits and the distinction between mock/simulated/cross-build and real environment tests. Editing an embedded-guide source affects the next executable build, not already-published binaries. Documentation publication alone does not recreate an existing version tag/release.
