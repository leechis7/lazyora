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

    query = b"SELECT t.ROWID AS rid,t.title AS caption FROM sample_rows t WHERE t.id=1"

    def execute():
        send(b"\x1bOQ\x1bOQ\x01\x0b" + query)
        output.clear()
        send(b"\x1b[20~")  # F9
        wait_for("1 rows")
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
        assert len(original.split("\t")) == 1, "ROWID leaked into visible row copy"
        output.clear()
        send(b"\x1bd")  # Alt+D opens the separate cell editor.
        wait_for("New value")
        send(b"\x01\x0bPTY ROWID SELECT edit")
        output.clear()
        send(b"\x13")
        wait_for("Apply one bound UPDATE")
        send(b"\x1b[C\r")  # Explicit Cancel returns to the form.
        output.clear()
        send(b"\x13")
        wait_for("Apply one bound UPDATE")
        send(b"\r")  # Proceed is selected by default.
        wait_for("One cell update applied")
        changed = copy_row().split("\t")
        assert len(changed) == 1 and changed[0] == "PTY ROWID SELECT edit", "reload lost original projection or value"
        # The original popup remains available; now edit inside the grid.
        output.clear()
        send(b"\r")
        wait_for("Enter Review")
        send(b"\x15INLINE discarded")  # Ctrl+U clears InputField text.
        send(b"\x1b")  # Esc cancels input without applying SQL.
        read_for(0.15)
        assert copy_row().split("\t")[0] == "PTY ROWID SELECT edit", "Esc applied an inline draft"
        output.clear()
        send(b"INLINE ROWID value")  # Printable text starts replacement.
        wait_for("Enter Review")
        output.clear()
        send(b"\r")
        wait_for("Apply one bound UPDATE")
        send(b"\x1b[C\r")  # Explicit Cancel returns to the inline draft.
        output.clear()
        send(b"\r")
        wait_for("Apply one bound UPDATE")
        send(b"\r")  # Proceed is selected by default.
        wait_for("One cell update applied")
        assert copy_row().split("\t")[0] == "INLINE ROWID value", "inline edit or original-query reload failed"
        # Select the changed cell: its distinct 8-color background survives focus.
        output.clear()
        send(b"\x1b[17~\x1b[17~")  # Switch focus away and back to repaint.
        assert re.search(rb"\x1b\[[0-9;]*43[0-9;]*m", output), "changed cell lost ANSI yellow background"
        output.clear()
        send(b"\x1br")
        wait_for("Rollback completed. Grid refreshed.")
        assert copy_row() == original, "rollback did not automatically restore the grid"
        output.clear()
        send(b"\x1bc")
        wait_for("Commit completed. Grid refreshed.")
        assert copy_row() == original, "commit refresh changed the current result"
        print("ROWID SELECT: popup and inline editing, confirmation/cancel, cell color, original query reload and automatic rollback/commit refresh passed")
    finally:
        os.kill(pid, signal.SIGTERM)
        os.waitpid(pid, 0)
        os.close(fd)
