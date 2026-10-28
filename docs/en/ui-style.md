# UI style rules

[한국어](../ui-style.md) | **English** · [Documentation](index.md)

In **v0.11.0**, common styles come from **Tuivory v0.1.0**, the public Go module `github.com/leechis7/tuivory/go`. Its single `tuivory` package supplies `Styles`, `Styler`, `Choice` and state-style methods. The pinned version is in `go.mod`; see [integration and updates](tuivory-sync.md).

LazyOra's `internal/ui/style_tokens.go` adapts package values to product roles. It does not own a second palette. `theme.go` and `widgets.go` apply styles to existing widgets. Oracle execution, transactions, SQL display lexing, Toad keymaps, grid data, focus and modal input routing remain application responsibilities.

## Surfaces, components and states

| Role | Contract |
| --- | --- |
| Canvas, body, empty space | Theme canvas and body foreground |
| Lists, forms, popup cards | Panel surface, including empty rows and secondary text |
| Hints/status | Hint foreground on the declared canvas/panel |
| Text input | Light field background with dark text |
| Choices | Distinct choice background and ▾ indicator |
| Checkboxes | Field style plus shared selection state |
| Normal, form and confirmation buttons | Shared normal/focus/disabled styles |
| Focused box | Double border and contrasting emphasis; inactive boxes use single borders |
| Modal popup | Centered opaque card, padding, dimmed backdrop, blocked background input |

Paint popup cards before their children, including padding and button gaps. Background text, lines and dimming must not leak through. Menus instead paint only their own area; layout spacers must not erase the underlying workspace.

Theme and terminal capability are separate. Test **Dark/Light/Black-White × 8/16/256/True Color**. ANSI8 uses indexes 0–7, ANSI16 uses 0–15, 256-color uses explicit indexes and True Color uses RGB. Keep large surfaces neutral. The dark 8-color accent is dark yellow, dark 16-color uses cyan tones, and light uses dark blue. Actual ANSI RGB values depend on the terminal palette.

Black-White uses neutral colors at every capability. Keyword/bind underlines, italic comments, foreground/background selection and double focus borders convey state without hue. Avoid bold black foreground combinations that become gray on bright-bold terminals. The light editor uses RGB white or index 231 in richer profiles; ANSI brightness depends on the client.

## Application entry points

- Text: `s.newTextView()` with body/hint/accent roles.
- Containers: `s.newFlex()`, `s.newBox()`.
- Surface bookkeeping: `s.textSurface(view, role)`, `s.setSurface(box, role)`.
- Generic detail tables: `s.newDataTable()`.
- Lists/trees: `s.styleList()`, `s.newTreeNode()`.
- Forms/inputs/choices/checkboxes: `s.styleForm()`, `s.newInputField()`, `s.newChoice()`, `s.styleCheckbox()`.
- Buttons/modals: `s.clickable()`, `s.styleModal()`.
- Popup layout and event routing: `s.formDialog()`, `s.formPopup()`.
- Focus borders: `s.refreshBorders()`.

Theme changes in `theme_apply.go` reapply semantic roles to existing widgets, including hidden tabs. Preserve SQL/password input, selection, focus, scroll, results and pending work. Update both Box and TextView content backgrounds. Remove bookkeeping for closed widgets rather than retaining them indefinitely.

SQL syntax and result-grid header/current cell/range/NULL/edited states use Tuivory's styles with LazyOra's data. Edited cells also have a `*` row marker and status text; markers never enter copied/exported values.

## Evidence and historical material

The 12-profile comparison covers connection/settings, output/history/plan/script, objects/editors and confirmation dialogs, not just Help. Keyboard, mouse, narrow windows and real terminal palettes require appropriate checks too.

Tuivory extraction, publication and LazyOra consumption are complete. The original [proposal](../ui-style-proposal.md) and candidate CSV are archived design input, not the live palette. [The integration record](tuivory-sync.md) points to dated automated evidence and its limitations. Tuivory itself is developed in its separate repository.

## New tree integration (v0.11.0)

v0.11.0 uses public Tuivory v0.1.0 `Tree`, `TreeState` and `TreeView`. Adapt existing `NewStyles` values to body/hint/surface/selection/focus.border roles instead of mixing in the separate candidate palette. LazyOra owns Oracle metadata, identities, lazy fetching, details actions and its existing focused double borders.

In v0.11.0, LORA-78 connects the public Tuivory v0.1.0 tree ASCII option to a LazyOra display-only adapter. Character mode is independent of theme, colors and mouse. ASCII uses English UI, distinguishes single/double horizontal borders with -/=, and substitutes non-ASCII data only during drawing without changing editing/copy/export originals. Upstream source is neither changed nor copied.

## Shared implementation priority (development branch)

Prefer public Tuivory APIs for shared rendering, controls and styles. Development v0.2.0 uses `WideSafeScreen` for entire popup draws and `CellText` with its `~` marker for clipped labels. Preserve existing palettes, focused double borders and host input/commands/data ownership. Review adapters and acceptance criteria before replacing models with different product behavior; arrange missing shared capabilities in Tuivory before consumption. See [adoption/defer details](../audits/lora-76-tuivory.md).
