"""Exercise the three result tabs in a real terminal and Oracle connection."""
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
with tempfile.TemporaryDirectory(prefix="lazyora-result-area-") as folder:
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


    try:
        wait_for("Load saved connection")
        send(b"\x1b[13;5u")
        wait_for("SAMPLE_ROWS")
        read_for(0.5)
        # F6 from SQL enters the result area; Shift+Tab reaches its tab strip.
        send(b"\x1b[17~\x1b[Z\x1b[C\x1b[C\r\t")
        output.clear()
        send(b"\x1be")  # Enable DBMS_OUTPUT while focused inside its tab.
        wait_for("DBMS_OUTPUT enabled")
        send(b"\x1b")  # Return to the grid, then F6 to SQL.
        send(b"\x1b[17~")
        send(b"\x01\x0bBEGIN DBMS_OUTPUT.PUT_LINE('lazyora_result_area_probe'); END;")
        output.clear()
        send(b"\x1b[20~")
        wait_for("affected rows:")
        # Still in the editor after execution: replace the SQL and run F5.
        send(b"\x01\x0bSELECT 451 AS RESULT_AREA_PROBE FROM dual;")
        output.clear()
        send(b"\x1b[15~")
        wait_for("Run these statements in order?")
        send(b"\t\t\r")
        wait_for("Script processed")
        send(b"\x1b[17~\x1b[Z\x1b[C\r\t")  # SQL -> grid -> tabs -> messages -> details.
        output.clear()
        send(b"\x03")
        wait_for(b"\x1b]52;c;")
        send(b"\x1bp")  # Menu shortcut reaches the same DBMS_OUTPUT tab.
        output.clear()
        send(b"\x03")
        wait_for(b"\x1b]52;c;" + base64.b64encode(b"lazyora_result_area_probe\n"))
        send(b"\x1bd")
        output.clear()
        send(b"\x03")
        assert b"\x1b]52;c;" not in output, "clearing output left retained data"
        print("Oracle PTY: result tabs, editor focus after F9/F5, and DBMS_OUTPUT enable/copy/clear passed")
    finally:
        os.kill(pid, signal.SIGTERM)
        os.waitpid(pid, 0)
        os.close(fd)
