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


    try:
        wait_for("Load saved connection")
        send(b"\x1b[13;5u")
        wait_for("SAMPLE_ROWS")
        read_for(0.3)
        output.clear()
        # Tools -> Settings, then Alt+T selects Theme. Keep Light for all
        # subsequent real-DB LOB/save/edit/rollback flows in this PTY.
        output.clear()
        send(b"\x1bt\r")
        wait_for("Ctrl+S saves")
        send(b"\x1bt\r\x1b[B\r")
        output.clear()
        send(b"\x13")
        wait_for("Settings saved.")
        with open(folder + "/settings.json", encoding="utf-8") as config:
            assert json.load(config)["theme"] == "light", "theme was not persisted"
        print("Settings theme shortcut, Light switch and persistence: passed")
        send(b"\x1bh\x1b[F\r")
        wait_for("AGPL-3.0-only")
        send(b"\x03")
        wait_for(b"\x1b]52;c;")
        captured = re.findall(rb"\x1b\]52;c;([^\x07\x1b]*)", output)
        assert captured and b"https://github.com/leechis7/lazyora" in base64.b64decode(captured[-1])
        send(b"\x1b")
        print("About menu, shared metadata and raw copy: passed")
        sql("SELECT TO_BLOB(HEXTORAW('0001FF')) AS payload FROM dual")
        send(b"\x1b[1;2Q\r")
        wait_for("Cell details")
        output.clear()
        send(b"\x03")
        copied("0001ff")
        path = folder + "/payload.bin"
        send(b"\t\x01\x0b" + path.encode())
        output.clear()
        send(b"\x13")
        wait_for("Saved 3 bytes")
        with open(path, "rb") as saved:
            assert saved.read() == b"\x00\x01\xff", "saved hex dump instead of binary"
        send(b"\x1b")
        send(b"\x1b[1;2Q")
        print("BLOB detail, raw hex copy and original binary file save: passed")
        send(b"\x1bu\rLAZYORA\r\t\t")
        output.clear()
        send(b"\x1b[21~\x1b[F\r")
        wait_for("Editable table loaded")
        send(b"\x1b[C")  # TITLE, after ID.
        output.clear()
        send(b"\x1b[21~\x1b[H" + b"\x1b[B" * 16 + b"\r")
        wait_for("New value")
        send(b"\x01\x0bPTY reviewed edit")
        output.clear()
        send(b"\x13")
        wait_for("Apply one bound UPDATE")
        send(b"\r")  # default cancel
        output.clear()
        send(b"\x13")
        wait_for("Apply one bound UPDATE")
        send(b"\x1b[D\r")
        wait_for("One cell update applied")
        output.clear()
        send(b"\x1br")
        wait_for("Rollback completed")
        print("Safe table edit menu, default-cancel preview, apply/reload and rollback: passed")
    finally:
        if pid is not None:
            os.kill(pid, signal.SIGTERM)
            os.waitpid(pid, 0)
        os.close(fd)
