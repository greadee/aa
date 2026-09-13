from __future__ import annotations

import json
import logging
import os
import secrets
import socket
import subprocess
import sys
import time
import webbrowser
from pathlib import Path

import httpx

from .paths import desktop_state_file, ensure_dirs, log_file
from .service import ComputeSifterService


def new_token() -> str:
    return secrets.token_urlsafe(24)


def preferred_port() -> int:
    raw = os.environ.get("SIFTER_DESKTOP_PORT")
    if raw and raw.isdigit():
        return int(raw)
    return 8765


def port_available(port: int, host: str = "127.0.0.1") -> bool:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        try:
            sock.bind((host, port))
        except OSError:
            return False
    return True


def choose_port(preferred: int | None = None, host: str = "127.0.0.1") -> int:
    """Return an available local port, or 0 to request an ephemeral port."""
    candidate = preferred if preferred is not None else preferred_port()
    if candidate == 0:
        return 0
    if port_available(candidate, host):
        return candidate
    for offset in range(1, 25):
        if port_available(candidate + offset, host):
            return candidate + offset
    return 0


def write_state(path: Path, payload: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, indent=2), encoding="utf-8")


def read_state(path: Path | None = None) -> dict | None:
    path = path or desktop_state_file()
    if not path.exists():
        return None
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (json.JSONDecodeError, OSError):
        return None


def clear_state(path: Path | None = None) -> None:
    path = path or desktop_state_file()
    try:
        path.unlink()
    except FileNotFoundError:
        pass


def health_check(url: str, token: str, timeout: float = 1.5) -> dict | None:
    try:
        response = httpx.get(
            f"{url.rstrip('/')}/api/health",
            headers={"X-Sifter-Token": token},
            timeout=timeout,
        )
    except Exception:  # noqa: BLE001 - health checks are best-effort
        return None
    if response.status_code != 200:
        return None
    try:
        return response.json()
    except json.JSONDecodeError:
        return None


def existing_instance(state_path: Path | None = None) -> dict | None:
    state = read_state(state_path)
    if not state:
        return None
    url = state.get("url")
    token = state.get("token")
    if not url or not token:
        return None
    health = health_check(url, token)
    if health is None:
        return None
    return {**state, "health": health}


def open_browser(url: str) -> bool:
    try:
        return webbrowser.open(url)
    except Exception:  # noqa: BLE001
        return False


def wait_for_state(state_path: Path, timeout: float = 20.0) -> dict | None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        state = read_state(state_path)
        if state and state.get("url"):
            return state
        time.sleep(0.2)
    return None


def wait_for_health(url: str, token: str, timeout: float = 20.0) -> dict | None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        health = health_check(url, token)
        if health is not None:
            return health
        time.sleep(0.25)
    return None


def spawn_backend(
    *,
    state_path: Path,
    port: int,
    host: str = "127.0.0.1",
    debug: bool = False,
    log_path: Path | None = None,
) -> subprocess.Popen:
    ensure_dirs()
    log_path = log_path or log_file("backend.log")
    log_path.parent.mkdir(parents=True, exist_ok=True)
    command = [
        sys.executable,
        "-m",
        "aa_sifter.cli.main",
        "serve",
        "--port",
        str(port),
        "--host",
        host,
        "--state-file",
        str(state_path),
    ]
    if debug:
        command.append("--debug")
    kwargs: dict = {
        "stdout": open(log_path, "ab"),
        "stderr": subprocess.STDOUT,
        "stdin": subprocess.DEVNULL,
        "close_fds": True,
    }
    if sys.platform == "win32":
        kwargs["creationflags"] = getattr(subprocess, "DETACHED_PROCESS", 0) | getattr(
            subprocess, "CREATE_NEW_PROCESS_GROUP", 0
        )
    else:
        kwargs["start_new_session"] = True
    return subprocess.Popen(command, **kwargs)  # noqa: SIM115 - handle kept for ownership


def configure_file_logging(path: Path, debug: bool = False) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    root = logging.getLogger("aa_sifter")
    if not any(isinstance(handler, logging.FileHandler) for handler in root.handlers):
        handler = logging.FileHandler(path, encoding="utf-8")
        handler.setFormatter(logging.Formatter("%(asctime)s %(levelname)s %(name)s: %(message)s"))
        root.addHandler(handler)
        root.setLevel(logging.DEBUG if debug else logging.INFO)


class LauncherError(RuntimeError):
    def __init__(self, message: str, *, details: str = "", options: list[str] | None = None):
        super().__init__(message)
        self.details = details
        self.options = options or []


def start_backend_process(
    *,
    host: str = "127.0.0.1",
    port: int | None = None,
    debug: bool = False,
    state_path: Path | None = None,
) -> dict:
    """Start a detached backend and return its state once healthy."""
    ensure_dirs()
    state_path = state_path or desktop_state_file()
    clear_state(state_path)
    chosen = choose_port(port, host)
    process = spawn_backend(state_path=state_path, port=chosen, host=host, debug=debug)
    state = wait_for_state(state_path, timeout=25.0)
    if state is None:
        raise LauncherError(
            "Compute Sifter backend did not become ready.",
            details=f"Check the log at {log_file('backend.log')}",
            options=["View Logs", "Retry"],
        )
    health = wait_for_health(state["url"], state["token"], timeout=20.0)
    if health is None:
        raise LauncherError(
            "Compute Sifter backend started but failed its health check.",
            details=f"pid={process.pid} url={state.get('url')}",
            options=["View Logs", "Retry"],
        )
    state["health"] = health
    state["pid"] = process.pid
    state["owned"] = True
    return state


def stop_backend(state_path: Path | None = None, *, force: bool = False) -> bool:
    """Stop a backend owned by Compute Sifter. Never touches unrelated processes."""
    state_path = state_path or desktop_state_file()
    state = read_state(state_path)
    if not state:
        return False
    url, token, pid = state.get("url"), state.get("token"), state.get("pid")
    if url and token:
        try:
            httpx.post(
                f"{url.rstrip('/')}/api/shutdown",
                headers={"X-Sifter-Token": token},
                timeout=3.0,
            )
        except Exception:  # noqa: BLE001
            pass
    if force and pid and _process_alive(pid):
        try:
            os.kill(pid, 9)
        except Exception:  # noqa: BLE001
            pass
    clear_state(state_path)
    return True


def _process_alive(pid: int) -> bool:
    if pid <= 0:
        return False
    try:
        import psutil

        return psutil.pid_exists(pid)
    except Exception:
        pass
    if sys.platform == "win32":
        import ctypes

        process_query_limited_information = 0x1000
        still_active = 259
        kernel32 = ctypes.windll.kernel32
        handle = kernel32.OpenProcess(process_query_limited_information, False, pid)
        if not handle:
            return False
        try:
            exit_code = ctypes.c_ulong()
            ok = kernel32.GetExitCodeProcess(handle, ctypes.byref(exit_code))
            return bool(ok) and exit_code.value == still_active
        finally:
            kernel32.CloseHandle(handle)
    try:
        os.kill(pid, 0)
        return True
    except OSError:
        return False


def build_service(
    *,
    config_path: Path | None = None,
    profile_name: str | None = None,
    provider=None,
    debug: bool = False,
) -> ComputeSifterService:
    return ComputeSifterService.create(
        config_path=config_path,
        profile_name=profile_name,
        provider=provider,
    )
