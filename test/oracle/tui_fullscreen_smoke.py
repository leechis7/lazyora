"""PTY Shift+F2 protocol regression; demo mode, no Oracle required."""
import fcntl
import json
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
executable = os.path.abspath("bin/lazyora-linux-" + architecture)
protocols = (b"\x1b[26~", b"\x1b[1;2Q", b"\x1b[12;2~", b"\x1b[24~")

for term in ("linux", "ansi", "xterm", "xterm-256color"):
    with tempfile.TemporaryDirectory(prefix="lazyora-fullscreen-") as folder:
        with open(folder + "/settings.json", "w", encoding="utf-8") as config:
            json.dump({"language": "en", "mouse": True}, config)
        pid, fd = pty.fork()
        if pid == 0:
            environment = dict(os.environ, TERM=term, TCELL_TRUECOLOR="disable",
                               LC_ALL="C.UTF-8", LAZYORA_CONFIG=folder + "/settings.json")
            os.execve(executable, [executable, "--demo"], environment)
        fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", 40, 120, 0, 0))
        output = bytearray()

        def read_for(seconds):
            deadline = time.monotonic() + seconds
            while time.monotonic() < deadline:
                if select.select([fd], [], [], min(0.1, max(0, deadline-time.monotonic())))[0]:
                    try:
                        data = os.read(fd, 65536)
                    except OSError:
                        return
                    if not data:
                        return
                    output.extend(data)

        def send(data):
            os.write(fd, data)
            read_for(0.3)

        def assert_fullscreen(expected):
            output.clear()
            send(b"\x1b[11~")  # F1: current-screen help snapshots fullscreen state.
            assert (b"Restore split layout" in output) == expected, (
                "unexpected fullscreen state for TERM=" + term + ": " + repr(bytes(output)[-400:]))
            assert b"Ctrl+C" in output, "result grid did not receive focus"
            send(b"\x1b")  # Closing help must return to the same layout.

        try:
            read_for(0.8)
            output.clear()
            send(b"\x1b[28~")  # F15: legacy Shift+F3 opens Find if no search exists.
            assert b"case sensitive" in output, "legacy Shift+F3 failed for " + term
            send(b"\x1b")
            output.clear()
            send(b"\x1b[22q")  # F22: higher function-key input, aliased to Shift+F10.
            assert b"Context menu" in output, "legacy Shift+F10 failed for " + term
            send(b"\x1b")
            for protocol in protocols:
                send(protocol)
                assert_fullscreen(True)
                send(protocol)
                assert_fullscreen(False)
                send(protocol)
                send(b"\x1b")  # Esc restores split as well.
                assert_fullscreen(False)
            output.clear()
            send(b"\x11")
            assert b"Exit LazyOra?" in output, "exit confirmation did not open"
            send(b"\t\r")
            deadline = time.monotonic() + 8
            while time.monotonic() < deadline:
                child, status = os.waitpid(pid, os.WNOHANG)
                if child:
                    assert os.waitstatus_to_exitcode(status) == 0
                    pid = None
                    break
                read_for(0.1)
            assert pid is None, "fullscreen demo did not exit"
            print(term + ": legacy Shift+F3/F10 and modified/legacy Shift+F2 and F12, toggle, Esc and focus passed")
        finally:
            if pid is not None:
                try:
                    os.kill(pid, signal.SIGTERM)
                except ProcessLookupError:
                    pass
                os.waitpid(pid, 0)
            os.close(fd)
