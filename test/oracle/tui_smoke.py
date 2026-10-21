"""Opt-in Linux PTY test against the local Oracle fixture; never prints passwords."""
import base64
import csv
import fcntl
import os
import platform
import pty
import re
import select
import signal
import struct
import tempfile
import termios
import time

architecture = {"aarch64": "arm64", "x86_64": "amd64"}[platform.machine()]
with tempfile.TemporaryDirectory(prefix="lazyora-tui-") as folder:
    pid, fd = pty.fork()
    if pid == 0:
        environment = dict(os.environ, TERM="xterm", TCELL_TRUECOLOR="disable", LC_ALL="C.UTF-8",
                           LAZYORA_CONFIG=folder + "/settings.json", LAZYORA_HOST="127.0.0.1",
                           LAZYORA_PORT=os.environ.get("ORACLE_PORT", "1521"), LAZYORA_USER="lazyora",
                           LAZYORA_PASSWORD=os.environ["APP_USER_PASSWORD"], LAZYORA_NAME="FREEPDB1")
        executable = "./bin/lazyora-linux-" + architecture
        os.execve(executable, [executable], environment)
    fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", 32, 120, 0, 0))
    output = bytearray()

    def read_for(seconds):
        end = time.monotonic() + seconds
        while time.monotonic() < end:
            if select.select([fd], [], [], min(0.1, max(0, end - time.monotonic())))[0]:
                try:
                    chunk = os.read(fd, 65536)
                except OSError:
                    return
                if not chunk:
                    return
                output.extend(chunk)

    def send(data):
        os.write(fd, data)
        read_for(0.2)

    def wait_for(text, seconds=8):
        value = text.encode() if isinstance(text, str) else text
        end = time.monotonic() + seconds
        while value not in output and time.monotonic() < end:
            read_for(0.1)
        if value not in output:
            raise AssertionError("TUI did not reach: " + repr(text))

    def sql(text, key=b"\x1b[20~", newline=False):
        send(b"\x1bOQ\x1bOQ")  # F2 twice: focus SQL and keep the split layout.
        send(b"\x01\x0b")  # Home and delete the current single-line query.
        send(text.encode())
        output.clear()
        if newline:
            send(b"\r")
            assert b"1 rows" not in output, "plain Enter executed SQL"
        send(key)
        wait_for("1 rows" if "UNION ALL" not in text else "2 rows")
        if newline:
            # Remove the extra newline for subsequent single-line replacements.
            send(b"\x08")

    def copied(text):
        wait_for(b"\x1b]52;c;" + base64.b64encode(text.encode()))

    def request_exit():
        output.clear()
        send(b"\x11")
        wait_for("SQL editor changes have not been saved")
        send(b"\x1b[D\r")  # Cancel -> Discard for exit only, then DB confirmation.

    try:
        wait_for("Load saved connection")  # Startup opens the login dialog.
        send(b"\x1b")
        send(b"\x1bl")  # Alt+L reopens it.
        wait_for("Load saved connection")
        send(b"\x1bn\t\x1b[C\x1b[C\x1b[D\x1b[D")  # Name -> buttons; arrows only move focus.
        send(b"\x1b[13;5u")  # CSI-u Ctrl+Enter connects.
        wait_for("SAMPLE_ROWS")
        read_for(0.3)
        output.clear()
        send(b"\x1bo\r\x1b[B\r")  # Alt+O -> View.
        wait_for("SAMPLE_ROWS_VIEW")
        output.clear()
        send(b"\x1bo\r\x1b[H\x1b[B\x1b[B\x1b[B\r")  # Function.
        wait_for("BROWSER_FUNCTION")
        output.clear()
        send(b"\x1bo\r\x1b[H\r")  # Back to Table.
        wait_for("SAMPLE_ROWS")
        send(b"\x1bu\rLAZYORA\r")  # Alt+U user chooser, keep login account.
        output.clear()
        send(b"\t\t\r")  # User -> Type -> object list -> table details.
        wait_for("CHANGED_AT")  # Wait for actual columns, not the loading title.
        send(b"\x1bd")
        wait_for("CREATE TABLE", seconds=65)
        output.clear()
        send(b"\x03")
        wait_for(b"\x1b]52;c;")
        captured = re.findall(rb"\x1b\]52;c;([^\x07\x1b]*)", output)
        assert captured and b'"SAMPLE_ROWS"' in base64.b64decode(captured[-1])
        output.clear()
        send(b"2")
        wait_for("ENABLED", seconds=15)
        output.clear()
        send(b"\x03")
        wait_for(b"\x1b]52;c;")
        captured = re.findall(rb"\x1b\]52;c;([^\x07\x1b]*)", output)
        assert captured and b"ENABLED" in base64.b64decode(captured[-1])
        output.clear()
        send(b"\t\x03")  # Detail pane.
        wait_for(b"\x1b]52;c;")
        captured = re.findall(rb"\x1b\]52;c;([^\x07\x1b]*)", output)
        detail_copy = base64.b64decode(captured[-1])
        assert b"Check condition" in detail_copy
        output.clear()
        send(b"\t\t\t\t\r")  # Selectors -> copy button preserves detail target.
        copied(detail_copy.decode())
        output.clear()
        send(b"3")
        wait_for("BROWSER_INDEX", seconds=15)
        send(b"\x1b")  # Definition -> object.
        send(b"\x1b")  # Object -> browser.
        print("Read-only DDL/constraints/indexes, section switching and raw copy: passed")
        print("Startup login, Alt+L, Alt+U user and Alt+O object types: passed")
        send(b"\x1bp")
        wait_for("DBMS_OUTPUT disabled")
        send(b"\x1be")
        wait_for("DBMS_OUTPUT enabled")
        send(b"\x1b")
        send(b"\x1bOQ\x1bOQ\x01\x0b")
        send("BEGIN DBMS_OUTPUT.PUT_LINE('PTY output 한글 [red]'); END;".encode())
        output.clear()
        send(b"\x1b[20~")
        wait_for("Statement completed")
        send(b"\x1bp")
        send(b"\x03")
        copied("PTY output 한글 [red]\n")
        send(b"\x1bd")
        wait_for("0 retained lines")
        send(b"\x1be")
        wait_for("DBMS_OUTPUT disabled")
        send(b"\x1b")
        send(b"\x1br")
        wait_for("Rollback completed.")
        print("DBMS_OUTPUT enable, F9 capture, raw Unicode copy, clear, disable and rollback: passed")
        # Routine execution through the menu uses metadata, explicit input,
        # a default-cancel confirmation, and the normal raw-value grid copy.
        send(b"\x1bOQ\x1bOQ\x01\x0bBROWSER_FUNCTION")
        output.clear()
        send(b"\x1bx\x1b[F\r")  # Execute menu -> last item: routine execution.
        wait_for("P_VALUE IN NUMBER")
        send(b"\r")  # Select standalone function signature.
        send(b"\r\x1b[H\r\t42")  # Default selector -> Value, input 42.
        output.clear()
        send(b"\x1b[20~")
        wait_for("Execute this routine")
        send(b"\r")  # Default is Cancel, no execution.
        output.clear()
        send(b"\x1b[20~\x1b[D\r")
        wait_for("Routine results")
        send(b"\r")  # Scalar outputs -> result grid.
        output.clear()
        send(b"\x1b[C\x1b[C\x1b[C\x03")
        copied("42")
        # Separate Escape events: adjacent ESC bytes may decode as Alt+Esc.
        for _ in range(4):
            send(b"\x1b")  # Grid -> outputs -> inputs -> routines -> main.
        send(b"\x1br")
        wait_for("Rollback completed.")
        print("Routine metadata, value input, default-cancel, F9 run, output grid and raw copy: passed")
        send(b"\x1bOQ\x1bOQ\x01\x0bSELECT * FROM sam")
        output.clear()
        send(b"\x14")  # Ctrl+T: Toad Code Insight.
        wait_for('"SAMPLE_ROWS"')
        send(b"\x1b")  # Cancel must preserve the partial word.
        output.clear()
        send(b"\x14")
        wait_for('"SAMPLE_ROWS"')
        send(b"\r WHERE id=1")
        output.clear()
        send(b"\x1b[20~")
        wait_for("1 rows")
        send(b"\x01\x0bSELECT s.ti FROM sample_rows s\x01" + b"\x1b[C" * 11)
        output.clear()
        send(b"\x14")
        wait_for('"TITLE"')
        send(b"\r\x1b[F WHERE id=1")
        output.clear()
        send(b"\x1b[20~")
        wait_for("1 rows")
        print("Ctrl+T object/alias column completion, cancel and quoted replacement: passed")
        original_format_sql = "SELECT 11 AS ELEVEN,22 AS TWENTYTWO FROM dual"
        send(b"\x1bOQ\x1bOQ\x01\x0b" + original_format_sql.encode())
        output.clear()
        send(b"\x1be\x1b[F\r")  # Edit menu -> Format (Shift shortcut fallback).
        wait_for("Code formatted")
        output.clear()
        send(b"\x1b[20~")
        wait_for("1 rows")
        send(b"\x1a")  # Ctrl+Z restores one original line in one edit.
        output.clear()
        send(b"\x0c\x03")  # Ctrl+L select all, Ctrl+C raw SQL copy.
        copied(original_format_sql)
        send(b"\x1b[C")  # End selection without changing text.
        print("SQL formatter menu, formatted F9 execution and one-step undo: passed")
        sql("SELECT 11 AS ELEVEN FROM dual", b"\x1b[13;5u", newline=True)
        sql("SELECT 12 AS TWELVE FROM dual", b"\x1b[27;5;13~")
        sql("SELECT 13 AS THIRTEEN FROM dual", b"\x1b\r")
        print("Ctrl+Enter CSI-u/modifyOtherKeys and Alt+Enter: passed; plain Enter stays newline")
        output.clear()
        send(b"\x1b[19~")  # F8 SQL history.
        wait_for("SQL history")
        send(b"\x1b")
        output.clear()
        send(b"\x1b[18~")  # F7 clear SQL; default cancels.
        wait_for("Clear all SQL editor text?")
        send(b"\x1b[D\x1b[C")  # Left Proceed, Right Cancel; no activation.
        send(b"\r")
        output.clear()
        send(b"\x06")  # Ctrl+F literal search.
        wait_for("case sensitive")
        send(b"THIRTEEN\r")
        wait_for("Match at byte")
        send(b"\x1b[17~\x1b[17~")  # F6 result -> SQL.
        output.clear()
        send(b"\x1b[21~")  # F10 context menu, not focus movement.
        wait_for("Context menu")
        send(b"\x1b")
        print("Toad F7 confirmation, F8 history, Ctrl+F, F6 and F10: passed")
        sql("SELECT '한글 [red]' AS name, CAST(NULL AS VARCHAR2(10)) AS note FROM dual")
        send(b"\x1b[1;2Q")  # Shift+F2 fullscreen results.
        output.clear()
        send(b"\x1by")
        copied("한글 [red]\tNULL")
        output.clear()
        send(b"\x03")
        copied("한글 [red]")
        send(b"\x1b[1;2Q")  # Shift+F2 restores split.
        sql("SELECT 11 AS a,22 AS b FROM dual UNION ALL SELECT 33,44 FROM dual")
        send(b"\x1b[1;2Q")
        read_for(0.2)
        # Maximized results: header at y=1, data rows y=2/3 (zero based).
        send(b"\x1b[<0;2;3M")
        send(b"\x1b[<32;5;4M")
        send(b"\x1b[<0;5;4m")
        output.clear()
        send(b"\x03")
        copied("11\t22\n33\t44")
        send(b"\x1be")  # Edit menu.
        wait_for("Copy preview")
        send(b"\x1b[H" + b"\x1b[B" * 8 + b"\r")  # Copy preview after the editor actions.
        wait_for("Mouse reporting is off")
        send(b"\x1b")
        send(b"\x1b[1;2Q")  # Shift+F2 restores split.
        print("OSC 52 row/cell/drag range, fallback preview, Shift+F2 fullscreen and restore: passed")
        send(b"\x1bOQ\x1bOQ\x01\x0b")
        send(b"LAZYORA.SAMPLE_ROWS")
        output.clear()
        send(b"\x1bOS")  # F4 describe, never executes SQL.
        wait_for("NUMBER(10,0)")
        wait_for("Comment")
        send(b"\x1b")
        sql("SELECT 21 AS script_value FROM dual")
        send(b"\x1bx")
        send(b"\x1b[B\r")  # F5 via Execute menu, review first.
        wait_for("Run these statements in order?")
        send(b"\t\t\r")  # Cancel -> SQL preview -> Execute.
        wait_for("Script results")
        send(b"\x1b")
        output.clear()
        send(b"\x05")  # Ctrl+E expected plan; never executes source SQL.
        wait_for("Estimated execution plan")
        send(b"\x1b")
        send(b"\x1bOQ\x1bOQ\x01\x0b")
        send(b"SELECT :n AS bound_value FROM dual")
        output.clear()
        send(b"\x1b[20~")
        wait_for("IN bind values")
        send(b"\t42\x1b[13;5u")
        wait_for("1 rows")
        output.clear()
        send(b"\x1bw")  # Export loaded results without submitting SQL.
        wait_for("Export results")
        send(b"\r\x1b[B\x1b[B\r\t\x01\x0b")  # XLSX, then Path.
        export_path = folder + "/result.xlsx"
        send(export_path.encode())
        send(b"\x13")
        wait_for("Exported 1 loaded rows")
        with open(export_path, "rb") as exported:
            assert exported.read(2) == b"PK", "XLSX export was not a ZIP workbook"
        print("F5 review/results, Ctrl+E plan, IN bind input and XLSX export: passed")
        send(b"\x1bOQ\x1bOQ\x01\x0b")
        send(b"SELECT level AS n FROM dual CONNECT BY level <= 2507 ORDER BY level")
        output.clear()
        send(b"\x1b[20~")
        wait_for("limited to 1000 rows")
        output.clear()
        send(b"\x1bw")
        wait_for("Export results")
        send(b"\t\x01\x0b")  # CSV remains the default; edit Path.
        full_path = folder + "/full.csv"
        send(full_path.encode())
        send(b"\t\r\x1b[F\r")  # Scope -> last entry: Full DB result.
        send(b"\x13")
        wait_for("Run this original SELECT again")
        send(b"\r")  # The default must be Cancel.
        assert not os.path.exists(full_path), "default requery action wrote a file"
        output.clear()
        send(b"\x13")
        wait_for("Run this original SELECT again")
        send(b"\x1b[D\r")  # Explicitly accept the reviewed SQL.
        wait_for("Exported 2507 rows from the full DB query")
        with open(full_path, encoding="utf-8-sig", newline="") as exported:
            rows = list(csv.reader(exported))
        assert len(rows) == 2508 and rows[-1] == ["2507"], "full export was truncated"
        print("Full DB result selection, default Cancel, explicit requery and 2507-row CSV: passed")

        send(b"\x1bOQ\x1bOQ\x01\x0b")
        send(b"BEGIN DBMS_SESSION.SLEEP(3); END;")
        output.clear()
        send(b"\x1b[20~")
        wait_for("Running Run")
        send(b"\x1bOP")  # F1 while the real Oracle call is still running.
        wait_for("Help — F1 / Esc closes")
        output.clear()
        send(b"\x1b")  # Close help, leaving the Oracle call running.
        wait_for("Statement completed")
        assert b"Error:" not in output, "closing help cancelled the Oracle call"
        print("Running Oracle SQL survives F1 help and Escape close: passed")

        send(b"\x1bOQ\x1bOQ\x01\x0b")
        send(b"UPDATE sample_rows SET title=title WHERE id=1")
        output.clear()
        send(b"\x1b[20~")
        wait_for("affected rows: 1")
        output.clear()
        send(b"\x1bl\x1b[13;5u")
        wait_for("Uncommitted changes remain")
        send(b"\r")  # Default cancels and restores the connection form.
        output.clear()
        send(b"\x1b[13;5u")
        wait_for("Uncommitted changes remain")
        send(b"\x1b")  # Give a standalone Esc time to cancel the modal.
        send(b"\x1bw\x01\x0b")  # Then edit the target password.
        send(b"invalid-test-password")
        output.clear()
        send(b"\x1b[13;5u")
        wait_for("Uncommitted changes remain")
        send(b"\x1b[D\x1b[D\r")  # Commit choice; failed target must not commit.
        wait_for("Error:")  # go-ora may report a protocol/auth error instead of ORA-01017.
        output.clear()
        send(b"\x1bl\x1b[13;5u")  # Reopen with the valid environment password.
        wait_for("Uncommitted changes remain")  # Existing work survived failure.
        send(b"\x1b[D\r")  # Explicit rollback and connect.
        wait_for("SAMPLE_ROWS")
        print("Reconnect default Cancel, failed target preserves pending work, explicit rollback: passed")

        send(b"\x1bOQ\x1bOQ\x01\x0b")
        send(b"UPDATE sample_rows SET title=title WHERE id=1")
        output.clear()
        send(b"\x1b[20~")
        wait_for("affected rows: 1")
        output.clear()
        request_exit()
        wait_for("Tracked statements: 1")
        send(b"\x1bv")  # Exit > View pending work.
        wait_for("Client execution log")
        output.clear()
        send(b"\x03")
        copied("UPDATE sample_rows SET title=title WHERE id=1")
        send(b"\x1b")  # Back to exit dialog; default remains Keep working.
        send(b"\r")
        output.clear()
        send(b"\x1br")
        wait_for("Rollback completed")
        output.clear()
        request_exit()
        # tcell sends screen differences; the unchanged suffix may not be sent
        # again after the intermediate unsaved-file modal.
        wait_for("No uncommitted")
        send(b"\x1b")
        send(b"\x1b[20~")
        wait_for("affected rows: 1")
        output.clear()
        send(b"\x1bc")
        wait_for("Commit completed")
        output.clear()
        request_exit()
        wait_for("Exit LazyOra?")
        send(b"\r")  # Default keeps working.
        output.clear()
        request_exit()
        wait_for("Exit LazyOra?")
        send(b"\x1b[D\r")
        deadline = time.monotonic() + 5
        while time.monotonic() < deadline:
            done, status = os.waitpid(pid, os.WNOHANG)
            if done:
                assert os.waitstatus_to_exitcode(status) == 0
                pid = None
                break
            read_for(0.1)
            time.sleep(0.05)
        assert pid is None, "TUI did not exit"
        print("F4 describe, pending SQL details/copy, commit/rollback clearing and confirmed exit: passed")
    finally:
        if pid is not None:
            os.kill(pid, signal.SIGTERM)
            os.waitpid(pid, 0)
        os.close(fd)
