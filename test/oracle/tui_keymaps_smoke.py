"""Actual terminal editor preset key decoding; demo mode, no database required."""
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

arch = {"aarch64": "arm64", "x86_64": "amd64"}[platform.machine()]
for keymap in ("vscode", "intellij"):
    with tempfile.TemporaryDirectory(prefix="lazyora-keymap-") as folder:
        config = folder + "/settings.json"
        with open(config, "w", encoding="utf-8") as file:
            json.dump({"language": "en", "mouse": True, "editor_keymap": keymap}, file)
        pid, fd = pty.fork()
        if pid == 0:
            executable = "./bin/lazyora-linux-" + arch
            env = dict(os.environ, TERM="xterm", TCELL_TRUECOLOR="disable", LC_ALL="C.UTF-8", LAZYORA_CONFIG=config)
            os.execve(executable, [executable, "--demo"], env)
        fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", 32, 120, 0, 0))
        output = bytearray()

        def read_for(seconds):
            end = time.monotonic() + seconds
            while time.monotonic() < end:
                if select.select([fd], [], [], 0.05)[0]:
                    try:
                        data = os.read(fd, 65536)
                    except OSError:
                        return
                    if not data:
                        return
                    output.extend(data)

        def send(data):
            os.write(fd, data)
            read_for(0.15)

        def contents():
            output.clear()
            send(b"\x01\x03")  # Ctrl+A/C, editor preset select/copy.
            end = time.monotonic() + 3
            while b"\x1b]52;c;" not in output and time.monotonic() < end:
                read_for(0.1)
            matches = re.findall(rb"\x1b\]52;c;([^\x07\x1b]*)", output)
            assert matches, "missing preset copy " + keymap
            text = base64.b64decode(matches[-1]).decode()
            send(b"\x1b[1;5F")  # Ctrl+End clears selection, keeps editing focus.
            return text

        try:
            read_for(0.8)
            send(b"\x01first\rsecond")
            assert contents() == "first\nsecond"
            if keymap == "intellij":
                send(b"\x04")  # Ctrl+D duplicates line, never quick describe.
                assert contents() == "first\nsecond\nsecond"
                send(b"\x1a")
                assert contents() == "first\nsecond"
                send(b"\x19")  # Ctrl+Y deletes line, never redo in IDEA.
                assert contents() == "first"
                send(b"\x1a")
                assert contents() == "first\nsecond"
            else:
                send(b"\x1b[1;5H\x1b[1;3B")  # Ctrl+Home, Alt+Down.
                value = contents()
                assert value == "second\nfirst", repr(value)
                send(b"\x1b[1;4B")  # Alt+Shift+Down duplicates line.
                assert contents() == "second\nfirst\nfirst"
                send(b"\x1a")
                value = contents()
                assert value == "second\nfirst", repr(value)
            send(b"\x1b[122;6u")  # CSI-u Ctrl+Shift+Z redo.
            expected = "first" if keymap == "intellij" else "second\nfirst\nfirst"
            assert contents() == expected, "modified redo failed " + keymap
            print(keymap + ": native preset select/copy, duplicate/move/delete, undo and modified redo passed")
        finally:
            os.kill(pid, signal.SIGTERM)
            os.waitpid(pid, 0)
            os.close(fd)
