# Usage

[한국어](../usage.md) | **English** · [Documentation](index.md) · [Offline guide](../guide.en.md)

This guide covers v0.12.0 workflows. Its input, selection, button, grid and popup styles use **Tuivory v0.4.0**, included in the executable. Choose Dark/Light/Black-White in Tools → Settings; terminal capability independently selects 8/16/256/True Color rendering. See [UI style rules](ui-style.md).

v0.11.2 uses the published Tuivory **v0.2.0** tree. User/type selectors and live object filtering remain. On a table or view, use **Right/Space or click the disclosure arrow** to fetch columns and data types. **Left** collapses or moves to the parent; **Alt+Left/Right** pans long names. **Enter or clicking a name** opens the existing object details. Columns are fetched on expansion and cached for the current list; Ctrl+R refresh, owner changes and reconnects clear the cache. Collapse and re-expand after failure/cancellation to retry. Existing Esc cancellation and work-protection rules apply during queries.

v0.11.2 adds saved **Auto/Unicode/ASCII** selection in Tools → Settings (**Alt+D**), plus a one-run `--display-mode ascii` override. ASCII uses English UI and compatible glyphs while preserving original SQL/data, application copy/export and Korean preferences. Displayed `?` substitutes can differ from original data. Actual font coverage cannot be detected. See [modes and precedence](terminal.md).

## Help, menus and panel layout

F1 shows context/keymap help. F10 opens the main menu; Shift+F10 or right-click opens the current panel's context menu. Help also includes the offline guide, version history, diagnostics, issue reporting, keymap comparison and About. Version history comes from the embedded changelog and distinguishes unreleased work; historical release text is primarily Korean.

In guide/history views, Up/Down chooses a topic, Right/Enter enters its article, Left returns and Tab cycles sections/buttons. Focused boxes use a double border; Esc closes the dialog. Copy the current article with Ctrl+C/Alt+C. Diagnostics preview app/Go version, OS/CPU, terminal size/colors, locale and UI preferences; they exclude DB connection data, passwords, SQL and settings paths. Reports provide a URL/template and are not sent automatically.

File contains SQL file actions; SQL Tabs (Alt+G) contains tab actions/rename; Edit contains clipboard, search, line editing, completion/formatting and grid actions. Session/Execute handle connections, transactions and SQL/script/plan execution. View includes schema/result/panel actions. Long menus use grouped submenus and scroll. At the top level Left/Right changes menus; inside a submenu Left/Esc or “previous menu” returns, and Right opens the next top-level menu. Separators cannot execute. Edit uses the panel/selection active before opening the menu.

F2 maximizes/restores the SQL editor; Shift+F2 or F12 controls results. Repeat the key or use Esc to restore split layout. Esc first closes a dialog or cancels a running query. F6 switches SQL/results, including which panel is shown fullscreen. View → Maximize/restore retains menus/status/footer and differs from terminal-wide fullscreen. Help/menu overlays temporarily restore the normal layout.

Drag splitters, or use Alt+F8 / View → Panel sizing. Up/Down changes SQL height; Left/Right changes schema width; Home resets and Enter/Esc finishes. Ratios are retained per workspace during the process and survive layout restoration.

## Connections and workspaces

Startup opens the connection dialog except in demo mode. Alt+L reopens it. Enter alias, host, port, username/password and Service Name or SID. Ctrl+Enter/Alt+Enter/Alt+C or Connect connects. The alias is first; saved connections are selected separately from editable fields. Save/Ctrl+S persists profile data, not an implicit connection. Password storage is off by default; [encrypted credentials](credentials.md) are optional.

Connection Test/Alt+V uses a temporary connection to authenticate and run a basic SELECT with a 15-second limit. Esc/Cancel cancels. Inputs/saving/repeated tests are blocked while it runs; completion preserves the input and existing working session/transaction. Connecting and saving remain separate actions.

Alt+J opens connection workspaces. N adds one, Enter or Switch confirms selection, Delete closes it. Single/double mouse clicks select only; Cancel/Esc preserves the original workspace. Up to eight can be open; switching/adding is blocked during execution. SQL tabs, result/schema/output/source/bind state and transactions are isolated by workspace. Only tabs within the same workspace share its transaction. Preferences are common. Explicit vault locking clears cached keys across workspaces.

Connection labels and Development/Test/Production markers help identify the target; these are user classifications, not database privileges. `[TX]` marks pending work and `!` an uncertain transaction outcome. Closing a workspace checks its unsaved SQL/source and pending transaction. Ctrl+Q checks all workspaces in order; cancellation preserves remaining workspaces.

## SQL files and tabs

Ctrl+N creates a tab, Ctrl+O opens a file, Ctrl+S saves and Ctrl+Shift+S saves as. Ctrl+W closes; closing the last tab keeps an empty tab. Ctrl+Tab/Ctrl+Shift+Tab or Alt+PgDn/Alt+PgUp cycles tabs; mouse selection works too. The SQL Tabs menu can rename **fileless** tabs with 1–40 characters. Once saved to a file, its filename takes precedence. Running SQL locks tab changes to keep result ownership clear.

Each tab retains text, cursor, selection, scroll and results. A `*` marks unsaved SQL. Save/Discard/Cancel protects changes before replacement/close/exit, with Cancel as default. A canceled/failed save preserves editing state. **Saving/discarding a file does not commit/rollback the database.**

Files support UTF-8 up to 16 MiB, preserve BOM and LF/CRLF, and normalize mixed line endings to the first style when edited. Unchanged saves preserve original bytes. Invalid UTF-8, NULs and symbolic links are rejected. Overwrite/external modification checks and a `<filename>.lazyora.lock` protect app saves, but cannot lock every external editor. Stop any writer before removing a stale lock. Parents must exist; save failures preserve the text.

## Execute SQL, scripts and binds

F9/Ctrl+Enter/Alt+Enter/Shift+F9 executes the selection, otherwise the cursor statement/PLSQL unit. Plain Enter inserts a newline. SQL uses semicolons; PL/SQL/stored-unit boundaries use `/` on its own line (the final slash may be omitted). Separators inside quotes/comments/q/nq literals are ignored; incomplete quoting/comments/parentheses are rejected. F7 clears SQL after confirmation (undo can restore it); F8 opens the process-local history of up to 100 submitted statements. Loading history does not execute it.

F5 executes selected text or the whole editor as a script, after reviewing the original script and transaction warning. It is limited to 100 statements and 60 seconds, stops at the first error/cancellation, and does not automatically roll back earlier successful DML. Basic COMMIT/ROLLBACK inside the script act on the shared session. ROLLBACK TO, SQL*Plus commands (SET/SPOOL/PROMPT/@), and `&` substitution are unsupported. Unsupported input is checked before submission.

**DBMS_OUTPUT collection is supported for F9 and F5** when enabled beforehand; it is collected after each completed statement, not live. Script history shows original line positions, status/counts and SQL. Each script SELECT retains up to 1,000 rows with truncation marked; the last SELECT is shown in the main grid and another script result can be selected explicitly.

F9/F5 `:name`/`:1` inputs support at most 32 distinct case-insensitive **IN** binds: string, NUMBER, DATE (`YYYY-MM-DD`), TIMESTAMP (`YYYY-MM-DD HH:MM:SS[.fraction]`) and NULL. Comments/strings/quoted identifiers/q literals and `:=` do not create binds. Values are passed as driver arguments, not interpolated SQL. NUMBER preserves Oracle precision; NULL ignores the value field. Quoted bind names and OUT/IN OUT/REF CURSOR output binds in this execution path are unsupported. The separate routine runner has its own argument support.

Recent bind values are held only in memory for identical SQL in the same tab. Clear remembered values clears them across tabs; reconnect/tab-close/exit also clears relevant values. They are not put in settings, SQL history or pending-SQL text; server errors/results may still contain data.

## Transactions and warnings

Alt+C commits and Alt+R rolls back the current workspace. Session → Pending work shows tracked statements, not row-level diffs or a full server audit. Other tabs in that workspace share the transaction; other workspaces do not.

Potentially dangerous DDL/DML shows an execution warning with Cancel as default. **Oracle DDL and PL/SQL-internal COMMIT can commit independently of the app's tracking.** Exit checks unsaved files/source separately from database changes, and still asks for exit confirmation when no pending work is tracked. Cancel/Esc keeps the app open.

Treat an uncertain result or failed transaction recovery as uncertain, not success. Further work can be blocked to avoid compounding it; verify actual DB state from another connection. Canceling a running operation is not the same as rolling back its earlier effects.

## Schema, objects and source

Alt+B toggles the browser, Alt+U selects the user (initially the connected account), and Alt+O selects object type. Tables/views, functions/procedures/packages and other supported object categories appear by type. Alt+I immediately filters object names within the loaded user/type without a DB requery; Down/Enter enters the list and Esc clears. User/reconnect changes reset search; type changes retain it. Workspaces keep their own browser state.

Ctrl+R refreshes schema/object lists or the current detail panel. Enter opens object/column details; F4 describes the selected/cursor object. In Toad mode Ctrl+D opens named-object lookup; editor keymaps can reassign Ctrl+D. Tab includes visible browser user/type/list controls; F6 switches SQL/results. Hidden browser controls are excluded from focus traversal.

Table/view details include column comments. Program objects offer source and argument views, source editing and compilation, plus compile errors. Source apply/compile are **DDL**, with confirmation and implicit-commit implications; they are not read-only lookup. Object detail offers DDL (Alt+D), constraints (Alt+C), indexes (Alt+I), privileges/dependencies and relations as available. In DDL/constraints/index views, 1/2/3 switches views and Ctrl+R refreshes. Copy actions use the displayed original DDL/detail/TSV, not rewritten SQL. Metadata availability depends on privileges/version.

Execute → Run procedure/function is distinct from a step debugger: LazyOra has **no breakpoint/step debugger**. The runner supports its documented scalar IN/OUT/IN OUT arguments and REF CURSOR results; up to four cursors, up to 1,000 fetched rows each. Routine export uses fetched results and does not rerun the routine. Record/collection/LOB/timezone-specific argument types are not generally supported. See the [debugger design record](plsql-debugger.md) for planned, unimplemented work.

## Editor keymaps, search and completion

Settings offers Toad (default), Vim, Notepad++, VS Code and IntelliJ IDEA. The empty/legacy `editor_keymap` value means Toad; other stored values are `vim`, `notepad++`, `vscode`, `intellij`. These provide subsets of familiar Windows/Linux editing behavior, not complete IDE emulation. App-wide execute/commit/rollback/exit/fullscreen keys take priority.

F1 → Editor keymaps explains the active mode, including Vim NORMAL/INSERT. Help → Editor keymap comparison, or Alt+K inside F1's editor-keymap help, compares actual assignments. `*` marks the current map; “Menu” means an action without a binding and “—” means unsupported. Arrow/PgUp/PgDn navigates, the lower area shows the full binding, Enter switches table/detail, Tab cycles regions, Ctrl+C copies TSV and Esc returns. It is informational, not an execution menu.

Search is case-sensitive literal text, wraps at the end and supports next/previous with F3/Shift+F3; Ctrl+F finds and Ctrl+R replaces in normal editor use. Ctrl+H is not used for replace because it collides with terminal Backspace. Completion/formatting/line commands vary by keymap: Toad uses Ctrl+T completion and Ctrl+Shift+F formatting, while Notepad++/VS Code/IDEA provide Ctrl+Space completion. Use menus if the terminal cannot send modifiers.

Completion uses available metadata and SQL context, not a full Oracle parser. CTE/derived-column inference, synonym/DB-link/package-member resolution and PL/SQL local symbol analysis are limited or unsupported; unavailable metadata is reported. Formatting is not SQL validation. macOS Command combinations, multiple cursors and full IDE/project features are not implied. Syntax-check Ctrl+F9 and debugger-continue F11 are reserved/unsupported, not DB execution shortcuts.

## Results, large cells and editing

F9 SELECT fetches from the same cursor in batches of 1,000, retaining at most 10,000 rows or an estimated 64 MiB. Fetch Next continues that cursor. Loaded-row sort/filter does not change server WHERE/ORDER BY, fetch missing rows or process the entire DB result. Numeric DB types retain precision; NULL sorts last; string/date ordering can differ from Oracle NLS. New pages reapply the tab's sort/filter. Reset restores fetched order.

Ctrl+C copies the active cell/range; Alt+Y copies the row. Shift+arrows or mouse drag selects a rectangle; Esc clears it. Copy uses original values before display clipping/markup, with TSV quoting and `NULL` for nulls. A successful OSC 52 request does not prove receipt by the desktop clipboard; use Edit → View copied content for terminal-native selection when necessary.

Read-only Enter opens cell details for long values/NULL/BLOB, with full-value reading and individual file saving where supported. LOB references belong to their execution/session and can expire; streamed full saves differ from clipped previews. XLSX/INSERT have format/type limits; do not infer that a preview contains the full value.

Open a table explicitly for editing, or execute a validated single-table SELECT containing **actual ROWID and original columns**:

```sql
SELECT t.ROWID, t.* FROM scott.emp t WHERE t.deptno = 10 ORDER BY t.empno;
```

Validated ROWID is hidden in the grid, row/range copy and fetched-result export, but retained internally. Full DB export repeats the original SQL and can include ROWID. Joins/views/synonyms/DB links/CTEs/subqueries/DISTINCT/aggregates/calculations/ROWIDTOCHAR or unverifiable origins remain read-only. Supported origins are permanent heap tables and direct columns/`t.*`; omitted schema uses CURRENT_SCHEMA. LOB and unsupported types remain read-only.

Enter/double-click edits an editable cell; typing replaces its value. Enter previews before UPDATE, Esc cancels the draft, Ctrl+U clears, Alt+Delete requests NULL, and Alt+D opens the separate editor. The **change-preview** default is Proceed; Cancel returns to the draft. This differs from the safe Cancel default on general dangerous-SQL warnings. Applied cells have a changed background and `*` status text until commit/rollback.

Edits use ROWID and original fetched editable-column values to detect conflicts, not every unfetched/read-only column. Lock conflicts and failed recovery are distinct from success. After commit/rollback the current grid requeries its original SQL/binds or explicit edit-table source, preserving sort/filter/widths and a feasible selection while clearing edit markers. Paging restarts at the first page; other SQL tabs' old results need rerunning. A failed refresh is reported separately from a successful transaction. The current editor text is never executed implicitly for this refresh; exit-time transaction actions proceed to exit.

## Export, plans and DBMS_OUTPUT

Alt+W saves CSV/TSV/XLSX/INSERT. **Fetched results** export retained data. **All DB results** reruns the original SELECT with its original IN binds and streams rows without the grid row cap; it can observe newer data. It does not rerun routines/scripts as a substitute. Large output uses buffered streaming; cancellation/errors preserve an existing destination. INSERT export creates SQL text and never executes it. Choose appropriate formats for LOB/type/Excel limits.

Ctrl+E shows the **estimated** plan, not actual runtime statistics and not execution of the target DML. Plan objects/privileges and Oracle version can limit availability.

Alt+P opens DBMS_OUTPUT; enable it with Alt+E in that view or the Session menu **before** F9/F5. Completed statements collect output, including an attempt after SQL errors. Server buffering is 1,000,000 bytes; collection is capped at 2,000 lines/256 KiB per statement and separately time-limited to three seconds. The display retains the latest 8,000 lines/1 MiB; omitted output is marked and remaining server output is discarded to prevent mixing into later statements. Older Oracle line/buffer limits still apply.

Ctrl+C copies retained output, Alt+D clears the display and Esc closes the view without canceling SQL. Output control/collection does not commit/rollback or add pending SQL entries. Reconnect disables output and resets history. SQL errors and output-collection errors are reported separately; failed buffer cleanup can block the next SQL to avoid mixing output.

For password storage, terminal key mappings and environment-specific evidence, see [Credentials](credentials.md), [Terminal troubleshooting](terminal.md) and [Compatibility](compatibility.md).

### Menus and result grid in v0.12.0

Current sources use the common Tuivory v0.4.0 menu/grid. Navigate menus with Up/Down, Home/End and PgUp/PgDn; Enter invokes a command or enters a child group. Disabled entries are skipped and cannot execute by click. Left/Esc or Back returns to the parent; top-level Esc restores the calling panel. Right retains the existing next-top-level-menu policy.

Drag a column divider in the grid header to resize; existing width menus/keyboard commands remain. The current-row marker is `>` and long cells/menus use `~` for clipping. Enter/double-click editing, Shift+arrows/mouse ranges, row-number row selection, Ctrl+C/Alt+Y original-value copying, F10 menu and F12 result maximization remain. Edited-cell backgrounds and status `*` indicate uncommitted work; row-number `*` is omitted. These changes are included in v0.12.0.

In v0.4.0 the shared package renders menu single borders and clipped titles. Disabled entries use gray/Dim without an ordinary prefix and cannot be focused/activated. Eight-color Dim brightness depends on the terminal. Ordinary information text uses the actual card/panel background.
