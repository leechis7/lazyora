# Terminal troubleshooting

[한국어](../terminal.md) | **English** · [English README](../../README.en.md)

## Character display mode (v0.11.2)

In v0.11.2, choose **Auto (default) / Unicode / ASCII** under **Tools → Settings → Character display (Alt+D)** and Save. Apply saved changes immediately to every connection workspace and retain them across launches. Canceling or failing to save preserves the active preferences. The settings note and Help → Diagnostics show the effective mode and reason.

- Auto checks the actual output character set and tcell’s encoding/terminfo estimates for required line/icon glyphs. Color count or `TERM=ansi` alone does not force ASCII. UTF-8 support cannot establish actual font coverage.
- Unicode is a manual display selection. It does not change terminal encoding/fonts or guarantee unsupported glyphs.
- ASCII uses English UI with ASCII lines/icons/markers: focused double horizontal borders become `=`, ordinary lines `-`, and vertical lines `|`. Auto also uses English when Korean cannot be encoded. Saved Korean preferences remain intact and are restored in a supported environment or when returning to Unicode/Auto where supported.

Override only the current invocation with the commands below. CLI overrides take precedence without being written into preferences. Saving another selection during an overridden run affects future launches, while the current CLI override remains effective.

```sh
./bin/lazyora-linux-arm64 --demo --display-mode ascii
./bin/lazyora-linux-arm64 --display-mode auto
./bin/lazyora-linux-arm64 --display-mode unicode
```

The optional `display_mode` setting accepts `auto`, `unicode`, or `ascii`; old files without it use Auto. SQL, object identities, database values, edit targets, files and application copy/export values are never converted. Non-ASCII data glyphs appear as `?` with their original cell width in ASCII rendering. Terminal-native selection copies those displayed substitutes; use **application copy commands or file export** to retrieve original values. Mouse, colors and keyboard protocols remain independent.

ASCII mode still requires a terminal supporting basic cursor positioning and screen updates. Character substitution does not add TUI capabilities to an unsupported TERM or a plain output-only terminal.

This feature is available since v0.11.2.


## Ctrl+Enter inserts a newline

Use **F9 or Alt+Enter** to execute SQL. Plain Enter inserts a newline. In the connection dialog, Ctrl+Enter, Alt+Enter or Alt+C connects.

The bytes sent by the terminal client matter. If Enter and Ctrl+Enter both send CR (`0d`), the application cannot distinguish them. Changing the server's `TERM` alone does not change what the client sends.

For SecureCRT, edit the current session's **Options → Session Options → Terminal → Emulation → Mapped Keys**. Map Ctrl+Enter to **Send String**:

```text
\e[13;5u
```

This sends CSI-u Ctrl+Enter. The xterm modifyOtherKeys sequence `\e[27;5;13~` is also supported. Apply the mapping to the session you are testing and verify both normal Enter and SQL execution. See the [detailed mapping notes and sources (Korean)](../terminal.md).

## Shift+function keys and MobaXterm

With some `TERM=linux` clients, Shift+F2 arrives as `Esc[26~`, which the key library calls F14. LazyOra also accepts this as Shift+F2. Unmodified F13–F24 are treated as Shift+F1–F12 where an action is assigned. This does not add actions for unassigned combinations.

Some MobaXterm configurations send Shift+F2 as `Esc[24~`, the same input as ordinary F12. **F12 and Shift+F2 both maximize/restore results**; F2 controls the SQL editor. Esc restores the layout when no higher-priority dialog or running query needs it. The app cannot recover modifier information the client did not send.

## Themes and color profiles

v0.11.2 uses Tuivory v0.2.0 for shared styling. Dark/Light/Black-White themes are independent from the 8/16/256/True Color profile reported by the terminal. Reference-palette contrast checks do not guarantee the appearance of a customized ANSI palette. See [UI style rules](ui-style.md).

## Color, mouse and missing TERM

On Unix, an absent or empty `TERM` makes LazyOra use `ansi` internally. It does not rewrite a configured `TERM` or the parent shell environment. Windows keeps its native console selection.

Color count and mouse capability are separate. `TERM=ansi` does not advertise mouse support. To test 8-color rendering with mouse input in an xterm-compatible terminal, run:

```sh
TERM=xterm TCELL_TRUECOLOR=disable ./bin/lazyora-linux-arm64
```

Use the executable for your architecture. In SecureCRT, also set the session's Terminal → Emulation to Xterm. A truthful `TERM` setting describes existing client capabilities; it cannot add them. In **Help → Environment diagnostics**, check TERM, Colors, Terminal mouse support and the app's Mouse setting.

## Result copying

Focus the result grid. **Ctrl+C** copies the current cell or selected range; **Alt+Y** copies the whole row. Use Shift+arrows or drag the mouse to select a rectangular range, then copy. Esc clears the range.

Copied values come from the original data, before display truncation and color markup. Cells use tabs and rows use newlines; tabs, newlines and quotes inside values are preserved with TSV quoting. NULL is copied as `NULL`.

LazyOra requests clipboard delivery through tcell's OSC 52 support. The terminal, SSH multiplexer or security policy may block or limit it. There is no clipboard-success acknowledgment, so sending a request does not prove that the desktop clipboard received it.

For reports, include the terminal application and emulation settings, app version, OS/CPU, `TERM`, `locale charmap` and `tput colors`. Remove credentials, private hostnames and business data before sharing output. See [Support](support.md).


## Keyboard panel sizing and exit reports

Use Alt+F8 or View → Panel sizing. Up/Down adjusts SQL/result height; Left/Right adjusts schema/right-panel width. Home restores defaults; Enter/Esc finishes. The menu provides the same action when the terminal does not deliver Alt+F8.

An `ESC [ 13;28;13;0;0;1 _`-like string after exit is a Win32 key-release input report. Unix TTY startup no longer requests that mode; CSI-u/modifyOtherKeys remain available. Native Windows console selection is unchanged.

If OSC 52 copying is unavailable, copy inside the app, then use Edit → View copied content. Mouse reporting is temporarily disabled there so the terminal can select text. Use arrows/PgUp/PgDn and the terminal's own copy shortcut; Esc restores the prior mouse setting and focus. Long lines may require horizontal scrolling or copying smaller TSV ranges.
