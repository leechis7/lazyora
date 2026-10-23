"""Linux PTY help/resources smoke test; --demo, no database or repository docs needed."""
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
import subprocess
import tempfile
import termios
import time

architecture = {"aarch64": "arm64", "x86_64": "amd64"}[platform.machine()]
executable = os.path.abspath("bin/lazyora-linux-" + architecture)
version = re.search(r"\d+\.\d+\.\d+", subprocess.check_output([executable, "--version"]).decode()).group()

for language, theme, term in [("en", "dark", "xterm"), ("ko", "light", "xterm-256color"), ("en", "blackwhite", "xterm"), ("ko", "blackwhite", "xterm-256color")]:
    with tempfile.TemporaryDirectory(prefix="lazyora-help-") as folder:
        with open(folder + "/settings.json", "w", encoding="utf-8") as config:
            json.dump({"language": language, "theme": theme, "mouse": True}, config)
        pid, fd = pty.fork()
        if pid == 0:
            environment = dict(os.environ, TERM=term, TCELL_TRUECOLOR="disable", LC_ALL="C.UTF-8",
                               LAZYORA_CONFIG=folder + "/settings.json",
                               LAZYORA_PASSWORD="must-not-leak-password", LAZYORA_USER="must-not-leak-user")
            os.chdir(folder)  # Verify documents come from the executable.
            os.execve(executable, [executable, "--demo"], environment)
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

        def wait_for(text):
            value = text.encode() if isinstance(text, str) else text
            end = time.monotonic() + 8
            while value not in output and time.monotonic() < end:
                read_for(0.1)
            assert value in output, "help TUI did not reach " + repr(text)

        def menu(index):
            output.clear()
            send(b"\x1bh\x1b[H" + b"\x1b[B" * index + b"\r")

        def copy():
            output.clear()
            send(b"\x03")
            wait_for(b"\x1b]52;c;")
            encoded = re.findall(rb"\x1b\]52;c;([^\x07\x1b]*)", output)[-1]
            value = base64.b64decode(encoded).decode()
            assert "must-not-leak" not in value
            return value

        try:
            read_for(0.8)
            # Shift+F10 opens the context menu; F10 switches to the main menu.
            # Cross every top-level menu, including grouped Edit and View,
            # without Right unexpectedly entering a subgroup.
            send(b"\x1b[21;2~\x1b[21~" + b"\x1b[C" * 7 + b"\x1b[H\x1b[B\r")
            guide = copy()
            assert "SELECT sysdate" in guide and ("Connect and run" if language == "en" else "처음 접속") in guide
            # Contents receives initial focus; no preliminary Tab is needed.
            send(b"\x1b[B")
            assert ("SQL files, tabs" if language == "en" else "SQL 파일·탭") in copy()
            send(b"\x1b[C\x1b[D\x1b[A")
            assert "SELECT sysdate" in copy()
            # Use real SGR press/release bytes over the schema selectors
            # behind the top contents rows, then return to the first topic.
            send(b"\x1b[<0;14;4M\x1b[<0;14;4m")
            assert ("SQL files, tabs" if language == "en" else "SQL 파일·탭") in copy()
            send(b"\x1b[<0;14;3M\x1b[<0;14;3m")
            assert "SELECT sysdate" in copy()
            send(b"\x1bOP")  # F1 nested help must return to the guide.
            send(b"\x1b")
            assert "SELECT sysdate" in copy()
            send(b"\x1b")
            menu(2)
            assert "[" + version + "]" in copy()
            send(b"\x1b[<0;14;3M\x1b[<0;14;3m")
            assert ("Unreleased (development)" if language == "en" else "개발 중 (미배포)") in copy()
            send(b"\x1b[<0;14;4M\x1b[<0;14;4m")
            assert "[" + version + "]" in copy()
            send(b"\x1b")
            menu(3)
            wait_for("OS/CPU:")
            diagnostics = copy()
            assert "TERM: " + term in diagnostics and "Language: " + language in diagnostics
            send(b"\x1b")
            menu(4)
            wait_for("/issues")
            assert "/issues" in copy() and "OS/CPU:" in copy()
            send(b"\x1b")
            menu(5)
            comparison = copy()
            assert "Toad" in comparison and "IntelliJ IDEA" in comparison and "Ctrl+Space" in comparison
            send(b"\r\x1b[F\r")  # Enter detail, scroll to end, return to table.
            assert copy() == comparison
            send(b"\x1b")
            menu(6)
            wait_for("AGPL-3.0-only")
            assert "Chaekang Lee" in copy()
            # Close the centered About card by mouse (including xterm 8-color).
            send(b"\x1b[<0;60;25M\x1b[<0;60;25m")
            output.clear()
            send(b"\x11")
            send(b"\x1b[D\r")
            end = time.monotonic() + 8
            while time.monotonic() < end:
                child, status = os.waitpid(pid, os.WNOHANG)
                if child:
                    assert os.waitstatus_to_exitcode(status) == 0
                    pid = None
                    break
                read_for(0.1)
            assert pid is None, "help demo did not exit"
            print("F10 main menu, Shift+F10 context menu, offline help, release history, safe diagnostics/report copy, nested F1 and About: " + language + "/" + theme + "/" + term + " passed")
        finally:
            if pid is not None:
                try:
                    os.kill(pid, signal.SIGTERM)
                except ProcessLookupError:
                    pass
                os.waitpid(pid, 0)
            os.close(fd)
