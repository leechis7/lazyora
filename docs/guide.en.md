# User guide

## Connect and run your first query

The connection dialog opens at startup; Alt+L opens it again.
Enter an alias, host, port, user, password, and Service Name or SID, then choose Connect.
Light input fields accept text; fields marked ▾ are choices operated with arrows/Enter or the mouse.

In the SQL editor, enter this statement and press F9 or Ctrl+Enter:

```sql
SELECT sysdate AS server_time FROM dual;
```

Enter inserts a newline. Use F9 or Alt+Enter if your terminal cannot send Ctrl+Enter.
F1 lists controls for the current screen and selected editor keymap.
Use Up/Down to select a contents item, Right/Enter to enter its article, and Left to return.
F10 opens/closes the main menu. Shift+F10 or right-click opens the current panel’s context menu.
Navigate menus with Up/Down, Home/End and PgUp/PgDn; Enter invokes/opens a child group. Disabled items are skipped. Left/Esc returns to the parent; top-level Esc restores the caller; Right opens the next top-level menu.
v0.12.0 uses Tuivory v0.4.0 menu single borders, surface-aware text and gray/Dim disabled cues. Disabled entries cannot be focused/activated; eight-color Dim brightness is terminal-dependent.
Source-editor Shift+F10 actions appear in a popup menu; use arrows/Enter and Esc to return. Shadow clicks cannot operate the page behind.
Tab cycles panels; the focused box has a double border and a distinct highlight color.
F2 makes the SQL editor fullscreen; Shift+F2 or F12 makes the result grid fullscreen. Repeat the key or press Esc to restore split.
Esc closes an open dialog first, or cancels a running query before restoring the layout.

## SQL files, tabs and editing

Ctrl+N opens a tab, Ctrl+O opens a file, and Ctrl+S saves. Use File for Save As.
An asterisk in the tab title means unsaved SQL. Saving a file does not commit the database.
F9 executes the selection, or the statement at the cursor when nothing is selected.
F5 executes a script in order and stops at the first error or cancellation.
Separate PL/SQL blocks with a standalone / line. SQL*Plus commands are unsupported.

Tuivory v0.2.0 supplies common theme, input, choice, button, grid and syntax styles. It is included in the executable; no separate installation is needed. Dark/Light/Black-White themes and 8/16/256/True Color terminal profiles are distinct.

Tools → Settings selects Toad (default), Vim, Notepad++, VS Code or IntelliJ IDEA keymaps, language, Dark/Light/Black-White theme and mouse support.
Help → Editor keymap comparison, or Alt+K in F1 help, compares the assigned keys. Arrows navigate; full selected-cell text appears below. Enter switches table/detail; scroll detail with arrows or PgUp/PgDn. Tab cycles table/detail/Close. Ctrl+C copies the table and Esc returns.
Check F1 → Editor keymaps for completion, formatting and editing shortcuts.

## Commit, rollback and exit

Tabs within one connection share its transaction. DML uses manual commit.
Alt+C commits and Alt+R rolls back. Session → View pending work shows the in-memory SQL journal.
The journal is not a row-level database diff.
Oracle DDL and explicit COMMIT inside PL/SQL may commit changes. Review execution warnings; Cancel is the default.
Ctrl+Q checks unsaved files and pending changes separately. Escape cancels exit.
An uncertain outcome needs verification through a new connection; never assume a failed commit did nothing.

## Connections and schema browsing

v0.11.2 uses the published Tuivory **v0.2.0** tree. User/type selectors and live object filtering remain. On a table or view, use **Right/Space or click the disclosure arrow** to fetch columns and data types. **Left** collapses or moves to the parent; **Alt+Left/Right** pans long names. **Enter or clicking a name** opens the existing object details. Columns are fetched on expansion and cached for the current list; Ctrl+R refresh, owner changes and reconnects clear the cache. Collapse and re-expand after failure/cancellation to retry. Existing Esc cancellation and work-protection rules apply during queries.

Alt+J opens workspaces: N adds a connection, Enter switches, Delete asks before closing one.
SQL tabs, results and transactions are separate between connections. Check the environment, account and database labels.
Closing one workspace preserves the other connections' transactions.

Choose a user and object type above the schema list. Alt+U selects User, Alt+O Type, and Alt+B toggles the browser.
Type in Alt+I object search to filter names immediately within the selected user/type. Down/Enter enters the list; Esc clears the search.
Enter opens details; F4 describes the object at the cursor. Applying or compiling source is DDL.
Execute → Run procedure/function supports routine arguments. Step debugging is not implemented.

## Browse, copy and inspect results

F6 switches SQL/results. Alt+F8 or View opens panel sizing: Up/Down changes SQL height, Left/Right changes schema width, Home resets, and Enter/Esc finishes. Splitters can also be dragged with the mouse. Use arrows and PgUp/PgDn in the result grid.
Drag a header column divider to resize. The current row uses `>`, edited cells use a distinct background and status `*`, and common width clipping uses `~`.
Click, drag or Shift+arrows selects cells; Ctrl+C copies and Alt+Y copies a row.
View/Shift+F10 provides the next 1,000 rows, local sorting/filtering and column widths.
One F9 query retains at most 10,000 rows / estimated 64 MiB; loaded rows are not the full database result.
For read-only cells, Enter opens cell details for long values, NULL and binary content, including original-value file saving.
Editing supports explicit browser table loads and verified single-table ROWID SELECT results; commit remains manual.
Editable cells: Enter/double-click edits the existing value; typing replaces it. Enter reviews before applying, Esc cancels input, Ctrl+U clears, Alt+Delete sets NULL. Alt+D / Edit > Edit cell in dialog opens the separate editor; review defaults to Proceed and Cancel returns to the draft; the context menu opens cell details. Applied cells have a distinct background until commit/rollback. Successful commit/rollback automatically refreshes the current replayable result using its original query/binds, preserving sorting, filter and widths. Reload failure is reported separately from transaction completion.
If clipboard delivery fails, use Edit → Copy preview and your terminal's selection/copy command.

Execute `SELECT t.ROWID, t.* FROM table_name t WHERE ...` with F9 to edit verified original columns. The verified physical ROWID is hidden in the grid and retained internally; commit or roll back explicitly. Joins, expressions and ambiguous results stay read-only.

## Export, explain plans and output

Alt+W exports CSV, TSV, XLSX or INSERT SQL.
Distinguish loaded results from Full DB result (requery), which repeats the original SELECT and original binds.
Data can change between queries. Escape cancels; failure/cancellation preserves an existing destination.
INSERT export creates a file only; it never executes the generated SQL.
Ctrl+E shows an estimated plan. Alt+P opens DBMS_OUTPUT; enable it before running F9/F5.
Output is collected after the statement completes, not streamed live.

## Saved connections and the master password

Save stores the alias and connection metadata. Password saving is off by default.
Optional encrypted saving uses a master password of at least 8 characters; use Tools → Passwords to manage it.
You can still connect by typing the database password if you forget the master password.
The master password cannot be recovered. Resetting its vault deletes the saved database passwords.

## Terminal issues and support

Use Xterm emulation in SecureCRT. Missing TERM uses ANSI; TERM=ansi has no mouse protocol. Use TERM=xterm TCELL_TRUECOLOR=disable to test 8 colors with mouse, and check Help → Diagnostics.
If Ctrl+Enter arrives as plain Enter, use F9/Alt+Enter or configure your terminal's key mapping.
OSC 52 delivery depends on terminal/SSH settings and clipboard permissions.
Help → View/copy diagnostics previews public environment details.
Help → Report an issue / suggest a feature provides the URL and report template. Remove real credentials, SQL and results.
Release history marks unreleased development separately. About is the last Help menu item.

Tools → Settings → Character display (Alt+D) saves Auto/Unicode/ASCII (v0.11.2). --display-mode ascii overrides only this run. ASCII uses English UI and ? substitutes for non-ASCII data glyphs, preserving original SQL/data/application copy/export and Korean preferences. Settings/Diagnostics show the effective mode and reason. Actual font coverage cannot be detected.
