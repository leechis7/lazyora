# User guide

## Connect and run your first query

The connection dialog opens at startup; Alt+L opens it again.
Enter an alias, host, port, user, password, and Service Name or SID, then choose Connect.
White fields accept text; fields marked ▾ are choices operated with arrows/Enter or the mouse.

In the SQL editor, enter this statement and press F9 or Ctrl+Enter:

```sql
SELECT sysdate AS server_time FROM dual;
```

Enter inserts a newline. Use F9 or Alt+Enter if your terminal cannot send Ctrl+Enter.
F1 lists controls for the current screen and selected editor keymap.
Use Up/Down to select a contents item, Right/Enter to enter its article, and Left to return.
F10 opens/closes the main menu. Shift+F10 or right-click opens the current panel’s context menu.
Tab cycles panels; the focused box has a double border and a distinct highlight color.
F2 makes the SQL editor fullscreen; Shift+F2 or F12 makes the result grid fullscreen. Repeat the key or press Esc to restore split.
Esc closes an open dialog first, or cancels a running query before restoring the layout.

## SQL files, tabs and editing

Ctrl+N opens a tab, Ctrl+O opens a file, and Ctrl+S saves. Use File for Save As.
An asterisk in the tab title means unsaved SQL. Saving a file does not commit the database.
F9 executes the selection, or the statement at the cursor when nothing is selected.
F5 executes a script in order and stops at the first error or cancellation.
Separate PL/SQL blocks with a standalone / line. SQL*Plus commands are unsupported.

Tools → Settings selects Toad (default), Vim, Notepad++, VS Code or IntelliJ IDEA keymaps, language, Dark/Light/Black-White theme and mouse support.
Check F1 → Editor keymaps for completion, formatting and editing shortcuts.

## Commit, rollback and exit

Tabs within one connection share its transaction. DML uses manual commit.
Alt+C commits and Alt+R rolls back. Session → View pending work shows the in-memory SQL journal.
The journal is not a row-level database diff.
Oracle DDL and explicit COMMIT inside PL/SQL may commit changes. Review execution warnings; Cancel is the default.
Ctrl+Q checks unsaved files and pending changes separately. Escape cancels exit.
An uncertain outcome needs verification through a new connection; never assume a failed commit did nothing.

## Connections and schema browsing

Alt+J opens workspaces: N adds a connection, Enter switches, Delete asks before closing one.
SQL tabs, results and transactions are separate between connections. Check the environment, account and database labels.
Closing one workspace preserves the other connections' transactions.

Choose a user and object type above the schema list. Alt+U selects User, Alt+O Type, and Alt+B toggles the browser.
Enter opens details; F4 describes the object at the cursor. Applying or compiling source is DDL.
Execute → Run procedure/function supports routine arguments. Step debugging is not implemented.

## Browse, copy and inspect results

F6 switches SQL/results. Use arrows and PgUp/PgDn in the result grid.
Click, drag or Shift+arrows selects cells; Ctrl+C copies and Alt+Y copies a row.
View/Shift+F10 provides the next 1,000 rows, local sorting/filtering and column widths.
One F9 query retains at most 10,000 rows / estimated 64 MiB; loaded rows are not the full database result.
For read-only cells, Enter opens cell details for long values, NULL and binary content, including original-value file saving.
Editing supports explicit browser table loads and verified single-table ROWID SELECT results; commit remains manual.
Editable cells: Enter/double-click edits the existing value; typing replaces it. Enter reviews before applying, Esc cancels input, Ctrl+U clears, Alt+Delete sets NULL. Edit > Edit cell in dialog remains available; the context menu opens cell details. Applied cells have a distinct background until commit/rollback; requery after rollback.
If clipboard delivery fails, use Edit → Copy preview and your terminal's selection/copy command.

Execute `SELECT t.ROWID, t.* FROM table_name t WHERE ...` with F9 to edit verified original columns. ROWID stays read-only; commit or roll back explicitly. Joins, expressions and ambiguous results stay read-only.

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
