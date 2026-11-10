"""Real PTY SELECT/ROWID edit, original projection reload and rollback; local DB only."""
import base64
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
with tempfile.TemporaryDirectory(prefix="lazyora-select-edit-") as folder:
    config = folder + "/settings.json"
    with open(config, "w", encoding="utf-8") as file:
        json.dump({"language": "en", "mouse": True}, file)
    pid, fd = pty.fork()
    if pid == 0:
        executable = "./bin/lazyora-linux-" + architecture
        env = dict(os.environ, TERM="xterm", TCELL_TRUECOLOR="disable", LC_ALL="C.UTF-8",
                   LAZYORA_CONFIG=config, LAZYORA_HOST="127.0.0.1",
                   LAZYORA_PORT=os.environ.get("ORACLE_PORT", "1521"), LAZYORA_USER="lazyora",
                   LAZYORA_PASSWORD=os.environ["APP_USER_PASSWORD"], LAZYORA_NAME="FREEPDB1")
        env.pop("NO_COLOR", None)  # The test explicitly verifies ANSI colors.
        os.execve(executable, [executable], env)
    fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", 32, 120, 0, 0))
    output = bytearray()

    def read_for(seconds):
        end = time.monotonic() + seconds
        while time.monotonic() < end:
            if select.select([fd], [], [], min(0.1, max(0, end-time.monotonic())))[0]:
                try:
                    data = os.read(fd, 65536)
                except OSError:
                    return
                if not data:
                    return
                output.extend(data)

    def send(data):
        os.write(fd, data)
        read_for(0.25)

    def wait_for(text):
        value = text.encode()
        end = time.monotonic() + 8
        while value not in output and time.monotonic() < end:
            read_for(0.1)
        assert value in output, "SELECT edit TUI did not reach " + repr(text)

    def copy_row():
        output.clear()
        send(b"\x1by")
        wait_for("\x1b]52;c;")
        encoded = re.findall(rb"\x1b\]52;c;([^\x07\x1b]*)", output)[-1]
        return base64.b64decode(encoded).decode()

    query = b"SELECT t.ROWID AS rid,t.title AS caption,t.amount FROM sample_rows t WHERE t.id<=2 ORDER BY t.id"

    def execute():
        send(b"\x1bOQ\x1bOQ\x01\x0b" + query)
        output.clear()
        send(b"\x1b[20~")  # F9
        wait_for("2 rows")
        send(b"\x1b[17~")  # F6: result grid.

    try:
        wait_for("Load saved connection")
        send(b"\x1bv")
        wait_for("Connection test succeeded")
        send(b"\x1b[13;5u")
        wait_for("SAMPLE_ROWS")
        read_for(0.3)
        execute()
        original = copy_row()
        assert len(original.split("\t")) == 2, "ROWID leaked into visible row copy"
        send(b"\x1b[B")
        original_second = copy_row()
        send(b"\x1b[A")
        output.clear()
        send(b"\x1bd")  # Alt+D, separate editor.
        wait_for("New value")
        send(b"\x01\x0bPTY batch title")
        output.clear()
        send(b"\x13")  # Ctrl+S stages locally.
        wait_for("Grid drafts: 1 cells")
        assert copy_row().split("\t")[0] == "PTY batch title"
        send(b"\x1b[C")  # Row 1, amount.
        send(b"12345\t")  # Stage and move to row 2, title.
        send(b"\x15PTY second\r")
        assert copy_row().split("\t")[0] == "PTY second", "Tab failed to move to next row"
        output.clear()
        send(b"\x1b[20~")  # F9 while the drafts are visible.
        wait_for("Discard all staged grid edits")
        send(b"\r")  # Cancel is the default.
        assert copy_row().split("\t")[0] == "PTY second", "cancel discarded the draft"
        send(b"\x1b[A")
        assert copy_row().split("\t") == ["PTY batch title", "12345"], "earlier row/column drafts lost"
        output.clear()
        send(b"\x1b[17~\x1b[17~")
        assert re.search(rb"\x1b\[[0-9;]*43[0-9;]*m", output), "changed cell lost ANSI yellow background"
        output.clear()
        send(b"\x1b[20~")
        wait_for("Discard all staged grid edits")
        output.clear()
        send(b"\x1b[D\r")
        wait_for("2 rows")
        assert copy_row() == original, "confirmed rerun did not discard drafts"
        send(b"\x1b[B")
        assert copy_row() == original_second, "rerun did not restore the second row"
        send(b"\x1b[A")
        send(b"Discard on rollback\r")
        output.clear()
        send(b"\x1br")
        wait_for("Rollback completed. Grid refreshed.")
        assert copy_row() == original, "rollback did not restore the grid"
        print("ROWID SELECT: three staged cells across two rows, Tab, rerun cancel/confirm, color and rollback passed")
    finally:
        os.kill(pid, signal.SIGTERM)
        os.waitpid(pid, 0)
        os.close(fd)
