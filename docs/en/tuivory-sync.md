# Tuivory integration and updates

[한국어](../tuivory-sync.md) | **English** · [UI style rules](ui-style.md)

## LORA-78: capability detection and ASCII display (2026-10-28, unreleased)

Keep public module v0.1.0, tag go/v0.1.0, commit c034a00b1db92ee7ad4abd1ea5956907f1c1a124. Connect TreeView.ASCII to the effective mode. LazyOra displayLayout/asciiScreen substitutes compatible glyphs only while drawing existing tview screens. Data, language preferences and terminal I/O protocols remain application-owned; do not copy or develop upstream sources here. Wire the unused Tuivory Panel/Split ASCII options when those components are adopted in their own issues.

CharacterSet and CanDisplay provide estimates, not real font coverage. [Terminal guidance](terminal.md) describes saved/manual modes, CLI precedence and original-data preservation. Full race/vet/ARM64 build, 144 simulated screen combinations and three local PTY runs passed. See the [bilingual validation record](../audits/lora-78-ascii.md) and Jira review comment for original-data preservation and real remote-font/live-DB limits.


## Current state

**LazyOra v0.10.2 ships Tuivory v0.0.1**, module `github.com/leechis7/tuivory/go`, release tag `go/v0.0.1`, source commit `cf06d95f4334f65436197ae4f12faf35783f5b7a`. It uses the single public `tuivory` package. TUIS is no longer a dependency; earlier TUIS entries in the Korean record document the migration history.

The Go module lives under the upstream repository's `go/` directory. Current development consumers specify `@v0.1.0` (released LazyOra v0.10.2 used `@v0.0.1`); the repository tag includes the `go/` prefix. Do not import upstream `internal` packages or replace the module with a local checkout.

| LazyOra area | Tuivory responsibility |
| --- | --- |
| `style_tokens.go`, `theme.go`, `widgets.go` | Themes/profiles, surfaces, forms, lists, checkboxes, editor and borders |
| `choice.go` | Public Choice component, Form attribute isolation and wide-glyph overlay handling |
| `grid.go`, `grid_inline.go` | Grid palettes, header/current/NULL/edited/range styles and decoration colors |
| `sql_colors.go` | Style for tokens classified by the application lexer |
| Comparison, tabs, toolbar, popup, splitter | Corresponding component/state styles |

The old lowercase `themeColors`/`gridColors` fields are value adapters, not independent palettes. The app retains Oracle execution/transactions, SQL lexer, Toad keymaps, data identities, clipboard/export values, widget lifecycle, focus and popup input blocking. Theme reapplication preserves existing widgets and hidden-tab state instead of rebuilding them.

Both sides use compatible Go/tcell/tview requirements. The main module's tview replacement applies to dependencies too, preserving LazyOra's SQL-highlighting TextArea extension. Tuivory uses AGPL-3.0-only, as does LazyOra; retain dependency notices.

## Update procedure

1. Check the published upstream release, exact source commit, public API and dependency/license changes.
2. Record differences in the related LORA issue. Develop Tuivory itself in its separate project.
3. Update with `go get github.com/leechis7/tuivory/go@<version>`, then `go mod tidy`.
4. Adjust application adapters rather than copying shared style implementation or changing Oracle behavior incidentally.
5. Check all 12 theme/profile combinations, focus/selection/input/popup states and theme round trips. Check SQL/password values, scroll, pending-edit markers and copying. Run appropriate regression/race/vet and default Linux ARM64 build checks.
6. Record the version, affected areas, evidence and remaining real-terminal checks in both language guides. Commit/publish/release only within the user's authorization.

## LORA-72: public v0.1.0 tree (2026-10-28, unreleased)

The development branch pins `github.com/leechis7/tuivory/go v0.1.0`, tag [go/v0.1.0](https://github.com/leechis7/tuivory/releases/tag/go/v0.1.0), source `c034a00b1db92ee7ad4abd1ea5956907f1c1a124`. Module Origin and checksums were verified; no source copying or local replacement is used. Published LazyOra v0.10.2 still contains v0.0.1.

`browser_tree.go` delegates rendering/navigation/scroll to public Tree/TreeState/TreeView while preserving Oracle identities and asynchronous job references. `browser_columns.go` fetches table/view columns on expansion and caches them within the current list. Filtering/type changes retain cached branches; refresh, owner changes and reconnects replace them. Late responses require the original node instance to remain in the current cache. Preserve selectors, filtering, click/Enter details and focused double borders.

Adapt the public NewStyles semantic values to tree palette roles rather than mixing in the separate draft candidate palette. Defer Tab/Panel/Split/Breadcrumb/Status adoption to LORA-73–76; this task does not complete those issues.

Validation passed: existing UI regression, six tree-specific tests (including 12 theme/color combinations), 5,000-object navigation, full `go test -race ./...`, `go vet ./...`, and Linux ARM64 `make build`. Checked 93 local documentation links and eight sections in each embedded guide before adding this audit link. No fresh remote-terminal or live-database connection test, remote CI, publishing or release was performed. See the [bilingual validation record](../audits/lora-72-tree.md).

## Evidence and migration history

The public v0.0.1 implementation was compared with the previous TUIS styles/widgets. Nine corresponding runtime files match after package/import/comment-name normalization; structure and the public entry point changed. The token draft format changed too, but LazyOra does not consume that candidate JSON directly.

Fresh verification after migration passed full race/vet and ARM64 build. The 3,216 simulated frames matched the original dev baseline after normalizing only random test-directory text; 1,608 theme-roundtrip pairs were unchanged. Captured role/SQL style values matched. The reference-palette contrast analysis is not a guarantee for every customized terminal.

See [dated verification (Korean)](../audits/lora-70.md), [machine-readable comparison](../audits/lora-70-tuivory-comparison.json) and [v0.10.2 release](https://github.com/leechis7/lazyora/releases/tag/v0.10.2). TUIS pseudo-version/v0.1.0 checks remain historical evidence, not the current dependency.
