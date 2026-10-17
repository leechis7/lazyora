"""Opt-in Linux PTY test against the local Oracle fixture; never prints passwords."""
import base64
import fcntl
import os
import platform
import pty
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
        send(b"\x1b[17~")  # F6 focuses SQL, does not execute rollback.
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

    try:
        wait_for("F2")
        send(b"\x1bOQ")
        wait_for("Load saved connection")
        send(b"\x1b[13;5u")  # CSI-u Ctrl+Enter connects.
        wait_for("SAMPLE_ROWS")
        read_for(0.3)
        sql("SELECT 11 AS ELEVEN FROM dual", b"\x1b[13;5u", newline=True)
        sql("SELECT 12 AS TWELVE FROM dual", b"\x1b[27;5;13~")
        sql("SELECT 13 AS THIRTEEN FROM dual", b"\x1b\r")
        print("Ctrl+Enter CSI-u/modifyOtherKeys and Alt+Enter: passed; plain Enter stays newline")
        sql("SELECT '한글 [red]' AS name, CAST(NULL AS VARCHAR2(10)) AS note FROM dual")
        send(b"\x1b[21~")  # F10 results.
        send(b"\x1b[23~")  # F11 maximize with chrome.
        send(b"\x1b[23~")  # F11 restore split.
        send(b"\x1b[23;5~")  # Ctrl+F11 fullscreen.
        output.clear()
        send(b"\x1by")
        copied("한글 [red]\tNULL")
        output.clear()
        send(b"\x03")
        copied("한글 [red]")
        send(b"\x1b[23;5~")
        sql("SELECT 11 AS a,22 AS b FROM dual UNION ALL SELECT 33,44 FROM dual")
        send(b"\x1b[21~\x1b[23;5~")
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
        send(b"\x1b[B\x1b[B\r")
        wait_for("Mouse reporting is off")
        send(b"\x1b")
        send(b"\x1b[23;5~")
        print("OSC 52 row/cell/drag range, fallback preview, maximize and restore: passed")
        send(b"\x1b[17~\x01\x0b")
        send(b"LAZYORA.SAMPLE_ROWS")
        output.clear()
        send(b"\x1bOS")  # F4 describe, never executes SQL.
        wait_for("NUMBER(10,0)")
        wait_for("Comment")
        send(b"\x1b")
        send(b"\x1bx")
        send(b"\x1b[B\r")  # Script is disabled.
        wait_for("not supported yet")
        send(b"\x1b[17~\x01\x0b")
        send(b"UPDATE sample_rows SET title=title WHERE id=1")
        output.clear()
        send(b"\x1b[20~")
        wait_for("affected rows: 1")
        output.clear()
        send(b"\x1br")
        wait_for("Rollback completed")
        send(b"\x1b[20~")
        wait_for("affected rows: 1")
        output.clear()
        send(b"\x1bc")
        wait_for("Commit completed")
        output.clear()
        send(b"\x11")
        wait_for("Exit LazyOra?")
        send(b"\r")  # Default keeps working.
        output.clear()
        send(b"\x11")
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
        print("F4 describe, menus, Alt+C/Alt+R transactions and confirmed exit: passed")
    finally:
        if pid is not None:
            os.kill(pid, signal.SIGTERM)
            os.waitpid(pid, 0)
        os.close(fd)
