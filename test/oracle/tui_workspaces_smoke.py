"""Opt-in Linux PTY test against the local Oracle fixture; never prints passwords."""
import base64
import csv
import fcntl
import json
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
            plain = re.sub(rb"\x1b\[[0-?]*[ -/]*[@-~]", b"", bytes(output))
            raise AssertionError("TUI did not reach: " + repr(text) + "; last display=" + repr(plain[-700:]))

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


    def login(alias, environment):
        wait_for("Load saved connection")
        send(b"\x1ba\x01\x0b" + alias.encode())
        send(b"\x1be\r" + b"\x1b[B" * environment + b"\r")
        output.clear()
        send(b"\x1b[13;5u")
        wait_for("SAMPLE_ROWS")
        read_for(0.3)

    def statement(text, expected):
        send(b"\x1bOQ\x1bOQ")
        send(b"\x0c" + text.encode())
        output.clear()
        send(b"\x1b[20~")
        wait_for(expected)

    def check_value(text, expected):
        statement(text, "1 rows")
        send(b"\x1b[1;2Q")
        output.clear()
        send(b"\x03")
        copied(expected)
        send(b"\x1b[1;2Q")

    def close_first():
        output.clear()
        send(b"\x1bj")
        wait_for("Connection workspaces")
        send(b"\x1b[H\x1b[3~")
        wait_for("Save, discard for this action")
        send(b"\x1b[D\r")
        wait_for("Commit and close")

    try:
        marker_a = "LAZYORA_WS_A_" + str(os.getpid())
        marker_b = "LAZYORA_WS_B_" + str(os.getpid())
        login("DB-A", 3)  # Production label is explicit, not a DB permission.
        statement("UPDATE sample_rows SET title='" + marker_a + "' WHERE id=1", "affected rows: 1")
        output.clear()
        send(b"\x1bj")
        wait_for("Connection workspaces")
        send(b"n")
        login("DB-B", 1)
        check_value("SELECT CASE WHEN title='" + marker_a + "' THEN 'LEAK' ELSE 'ISOLATED' END AS mark FROM sample_rows WHERE id=1", "ISOLATED")
        statement("UPDATE sample_rows SET title='" + marker_b + "' WHERE id=2", "affected rows: 1")
        output.clear()
        send(b"\x1bj")
        wait_for("Connection workspaces")
        send(b"\x1b[H\r")
        check_value("SELECT CASE WHEN title='" + marker_a + "' THEN 'FIRST_OWN' ELSE 'LOST' END AS mark FROM sample_rows WHERE id=1", "FIRST_OWN")
        check_value("SELECT CASE WHEN title='" + marker_b + "' THEN 'LEAK' ELSE 'ISOLATED' END AS mark FROM sample_rows WHERE id=2", "ISOLATED")
        print("Two live sessions: independent SQL/results and uncommitted row visibility: passed")
        close_first()
        send(b"\r")  # Cancel is the default: retain both connections.
        check_value("SELECT CASE WHEN title='" + marker_a + "' THEN 'FIRST_OWN' ELSE 'LOST' END AS mark FROM sample_rows WHERE id=1", "FIRST_OWN")
        close_first()
        output.clear()
        send(b"\x1br")  # Roll back and close DB-A only.
        read_for(0.7)
        check_value("SELECT CASE WHEN title='" + marker_a + "' THEN 'LEAK' ELSE 'CLEAN' END AS mark FROM sample_rows WHERE id=1", "CLEAN")
        check_value("SELECT CASE WHEN title='" + marker_b + "' THEN 'SECOND_OWN' ELSE 'LOST' END AS mark FROM sample_rows WHERE id=2", "SECOND_OWN")
        output.clear()
        send(b"\x1br")
        wait_for("Rollback completed")
        check_value("SELECT CASE WHEN title='" + marker_b + "' THEN 'LEAK' ELSE 'CLEAN' END AS mark FROM sample_rows WHERE id=2", "CLEAN")
        print("Workspace close: default Cancel, targeted rollback and surviving transaction: passed")
        output.clear()
        send(b"\x11")
        wait_for("Save, discard for this action")
        send(b"\x1b[D\r")
        wait_for("Exit LazyOra?")
        send(b"\x1b[D\r")
        end = time.monotonic() + 8
        while time.monotonic() < end:
            child, status = os.waitpid(pid, os.WNOHANG)
            if child:
                assert os.waitstatus_to_exitcode(status) == 0, "workspace app exit failed"
                pid = None
                break
            read_for(0.1)
        assert pid is None, "workspace app did not close all sessions"
        print("Final workspace shutdown after both rollbacks: passed")
    finally:
        if pid is not None:
            try:
                os.kill(pid, signal.SIGTERM)
            except ProcessLookupError:
                pass
            os.waitpid(pid, 0)
        os.close(fd)
