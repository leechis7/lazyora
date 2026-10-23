"""Unix PTY startup without TERM; demo mode, no database required."""
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

for term in (None, "", "ansi", "xterm"):
    with tempfile.TemporaryDirectory(prefix="lazyora-terminal-") as folder:
        with open(folder + "/settings.json", "w", encoding="utf-8") as config:
            json.dump({"language": "en", "mouse": True}, config)
        pid, fd = pty.fork()
        if pid == 0:
            environment = dict(os.environ, LC_ALL="C.UTF-8", TCELL_TRUECOLOR="disable",
                               LAZYORA_CONFIG=folder + "/settings.json")
            if term is None:
                environment.pop("TERM", None)
            else:
                environment["TERM"] = term
            os.execve(executable, [executable, "--demo"], environment)
        fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", 32, 120, 0, 0))
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

        try:
            read_for(0.8)
            assert b"\x1b[?9001h" not in output, "Win32 release reports enabled for TERM=" + repr(term)
            output.clear()
            os.write(fd, b"\x11")  # Ctrl+Q opens the confirmation dialog.
            deadline = time.monotonic() + 8
            while b"Exit LazyOra?" not in output and time.monotonic() < deadline:
                read_for(0.1)
            assert b"Exit LazyOra?" in output, "startup/keyboard failed for TERM=" + repr(term)
            os.write(fd, b"\t\r")  # Safe default is Keep working; Tab selects Exit.
            deadline = time.monotonic() + 8
            while time.monotonic() < deadline:
                child, status = os.waitpid(pid, os.WNOHANG)
                if child:
                    assert os.waitstatus_to_exitcode(status) == 0
                    pid = None
                    break
                read_for(0.1)
            assert pid is None, "confirmed exit failed for TERM=" + repr(term)
            print("TERM=" + repr(term) + ": demo startup, confirmation and clean exit passed")
        finally:
            if pid is not None:
                try:
                    os.kill(pid, signal.SIGTERM)
                except ProcessLookupError:
                    pass
                os.waitpid(pid, 0)
            os.close(fd)
