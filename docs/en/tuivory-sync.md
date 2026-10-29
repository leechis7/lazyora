# Tuivory integration and updates

[한국어](../tuivory-sync.md) | **English** · [UI style rules](ui-style.md)

## LORA-76: component/application boundary correction (2026-10-29, review)

Tuivory is a TUI component library. Prefer public APIs for reusable control rendering, internal focus/selection/input, style/options and glyph-overlap protection. LazyOra owns actual connection/settings/help screen composition, application layout/sizing/order, cross-component focus, global shortcuts and data/action binding. Host UI composition is normal, not an unfinished migration. Withdraw the earlier proposal to transfer all host screen UI; see the [corrected ownership table and sources](../audits/lora-76-tuivory.md).

Corrected the TVR-7 description and overly broad request comment in Korean/English. Keep [TVR-17](https://leechis7.atlassian.net/browse/TVR-17), its single simple grid with options/presets and both comparison images. Avoid duplicated shared controls without delegating whole application screens. This changes review records, not application code or published assets.

## LORA-76: omit duplicate column-leaf icons (2026-10-29, development)

Retain public Tuivory Go v0.2.0 / go/v0.2.0 / 987d1a6786033a6bf063235b5e7fea7b565f0af0. TreeView already supplies a leaf marker (Unicode · / ASCII .); leave public TreeNode.Icon/ASCIIIcon empty instead of adding LazyOra's · / C column icon. Recover two cells per column while preserving the default marker, indentation, identity, selection and details actions. Do not copy or change the renderer. Any future option to hide the built-in marker belongs to Tuivory before consumption. Published v0.11.2 assets remain unchanged.

## LORA-76: shared UI via Tuivory v0.2.0 (2026-10-29, v0.11.2)

- Source: [public go/v0.2.0 release](https://github.com/leechis7/tuivory/releases/tag/go/v0.2.0), commit `987d1a6786033a6bf063235b5e7fea7b565f0af0`. Verified module Origin and Sum `h1:q+NJW7/inCfX1cbGLIlEihF8mo/i86cGHozvl9VL6Gw=`. Go 1.24, tcell/tview and AGPL-3.0-only requirements are compatible. No local module replacement or upstream source copying.
- Adopted: draw the whole shared popup (backdrop/card/shadow/child borders) through public `WideSafeScreen`. Remove local `popupFill`; use `CellText` for clipped labels, with its common `~` omission marker. Retain LORA-79 physical-output regressions.
- Host adapter: the new drawing API uses `Put`; route Put/SetCell/PutStr methods through the existing display-only ASCII conversion. Preserve originals, cell widths, copying and preferences without reimplementing wide-glyph repairs.
- Reviewed: existing style/widget/tree APIs remain. Style values are unchanged; Choice now uses common overlap repair. Do not bulk-replace application state with new Form/FocusStack/Menu/Grid/Tab/Panel/Split/Status models. See [adoption/defer review](../audits/lora-76-tuivory.md) for behavior differences and follow-up issues.
- Policy: solve shared UI defects through public Tuivory APIs; arrange missing capabilities in the separate Tuivory project before consumption. Keep product Oracle/command/data adapters distinct. This ships in v0.11.2; existing v0.11.1 binaries remain unchanged.

## LORA-79: wide-glyph modal boundaries (2026-10-28, v0.11.1)

Retain the public `github.com/leechis7/tuivory/go v0.1.0` dependency, tag `go/v0.1.0`, source `c034a00b1db92ee7ad4abd1ea5956907f1c1a124`; rechecked Go module Origin. The open Tuivory Choice already protects wide-glyph boundaries. This defect is in LazyOra's own popup card/shadow composition. Continue importing the public package without upstream edits, source copying or a local replacement.

`popupFill` in `form_popup.go` clears both cells of a background glyph cut by either edge of a card/shadow row, preserving intact outside glyphs/styles. Previous checks inspected logical cells and single-cell backgrounds, missing skipped physical output. New regression tests compare the actual buffer after `Show()`, covering Korean/English, three dialogs, 12 theme/color combinations, resizing, Korean/CJK/emoji/combining marks, background restoration and preserved SQL/results/focus. See the [bilingual verification record](../audits/lora-79-popup.md). This fix ships in LazyOra v0.11.1; existing v0.11.0 assets are unchanged. A general public popup/overlay boundary-protection API belongs to the separate Tuivory project and can be adopted in a later LazyOra issue once available.

## LORA-78: capability detection and ASCII display (2026-10-28, v0.11.0)

Keep public module v0.1.0, tag go/v0.1.0, commit c034a00b1db92ee7ad4abd1ea5956907f1c1a124. Connect TreeView.ASCII to the effective mode. LazyOra displayLayout/asciiScreen substitutes compatible glyphs only while drawing existing tview screens. Data, language preferences and terminal I/O protocols remain application-owned; do not copy or develop upstream sources here. Wire the unused Tuivory Panel/Split ASCII options when those components are adopted in their own issues.

CharacterSet and CanDisplay provide estimates, not real font coverage. [Terminal guidance](terminal.md) describes saved/manual modes, CLI precedence and original-data preservation. Full race/vet/ARM64 build, 144 simulated screen combinations and three local PTY runs passed. See the [bilingual validation record](../audits/lora-78-ascii.md) and Jira review comment for original-data preservation and real remote-font/live-DB limits.


## Current state

**LazyOra v0.11.2 ships Tuivory v0.2.0**, module `github.com/leechis7/tuivory/go`, release tag `go/v0.2.0`, source commit `987d1a6786033a6bf063235b5e7fea7b565f0af0`. It uses the single public `tuivory` package. TUIS is no longer a dependency; earlier TUIS entries in the Korean record document the migration history.

The Go module lives under the upstream repository's `go/` directory. Current consumers specify `@v0.2.0` (released LazyOra v0.10.2 used `@v0.0.1`); the repository tag includes the `go/` prefix. Do not import upstream `internal` packages or replace the module with a local checkout.

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

## LORA-72: public v0.1.0 tree (2026-10-28, v0.11.0)

LazyOra v0.11.0 pins `github.com/leechis7/tuivory/go v0.1.0`, tag [go/v0.1.0](https://github.com/leechis7/tuivory/releases/tag/go/v0.1.0), source `c034a00b1db92ee7ad4abd1ea5956907f1c1a124`. Module Origin and checksums were verified; no source copying or local replacement is used. LazyOra v0.10.2 contained v0.0.1.

`browser_tree.go` delegates rendering/navigation/scroll to public Tree/TreeState/TreeView while preserving Oracle identities and asynchronous job references. `browser_columns.go` fetches table/view columns on expansion and caches them within the current list. Filtering/type changes retain cached branches; refresh, owner changes and reconnects replace them. Late responses require the original node instance to remain in the current cache. Preserve selectors, filtering, click/Enter details and focused double borders.

Adapt the public NewStyles semantic values to tree palette roles rather than mixing in the separate draft candidate palette. Defer Tab/Panel/Split/Breadcrumb/Status adoption to LORA-73–76; this task does not complete those issues.

Validation passed: existing UI regression, six tree-specific tests (including 12 theme/color combinations), 5,000-object navigation, full `go test -race ./...`, `go vet ./...`, and Linux ARM64 `make build`. Checked 93 local documentation links and eight sections in each embedded guide before adding this audit link. No fresh remote-terminal or live-database connection test, remote CI, publishing or release was performed. See the [bilingual validation record](../audits/lora-72-tree.md).

## Evidence and migration history

The public v0.0.1 implementation was compared with the previous TUIS styles/widgets. Nine corresponding runtime files match after package/import/comment-name normalization; structure and the public entry point changed. The token draft format changed too, but LazyOra does not consume that candidate JSON directly.

Fresh verification after migration passed full race/vet and ARM64 build. The 3,216 simulated frames matched the original dev baseline after normalizing only random test-directory text; 1,608 theme-roundtrip pairs were unchanged. Captured role/SQL style values matched. The reference-palette contrast analysis is not a guarantee for every customized terminal.

See [dated verification (Korean)](../audits/lora-70.md), [machine-readable comparison](../audits/lora-70-tuivory-comparison.json) and [v0.10.2 release](https://github.com/leechis7/lazyora/releases/tag/v0.10.2). TUIS pseudo-version/v0.1.0 checks remain historical evidence, not the current dependency.
