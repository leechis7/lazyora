# Terminal troubleshooting

[한국어](../terminal.md) | **English** · [English README](../../README.en.md)

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
