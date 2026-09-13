from __future__ import annotations

import argparse
import atexit
import os
import signal
import sys
from pathlib import Path

import httpx

from ..config.store import default_config_path
from ..desktop.launcher import (
    LauncherError,
    clear_state,
    configure_file_logging,
    existing_instance,
    health_check,
    new_token,
    open_browser,
    read_state,
    start_backend_process,
    stop_backend,
    write_state,
)
from ..desktop.paths import desktop_state_file, ensure_dirs, log_file
from ..desktop.server import create_server, server_url
from ..desktop.service import ComputeSifterService
from ..doctor import ollama_installed

DESKTOP_COMMANDS = {"launch", "serve", "desktop", "stop"}


def _print_issues(url: str, token: str) -> None:
    try:
        response = httpx.get(
            f"{url.rstrip('/')}/api/status", headers={"X-Sifter-Token": token}, timeout=3.0
        )
        payload = response.json()
    except Exception:  # noqa: BLE001
        return
    issues = payload.get("issues") or []
    for issue in issues:
        print(f"[{issue.get('severity', 'warning')}] {issue.get('title')}")
        if issue.get("detail"):
            print(f"    {issue['detail']}")
        for option in issue.get("options", []):
            print(f"    - {option.get('label')}")


def launch(args: argparse.Namespace) -> int:
    ensure_dirs()
    state_path = Path(args.state_file) if args.state_file else desktop_state_file()

    if args.stop:
        stopped = stop_backend(state_path)
        print("Stopped Compute Sifter backend." if stopped else "No running backend found.")
        return 0

    existing = existing_instance(state_path)
    if existing is not None:
        print(f"Compute Sifter is already running at {existing['url']}")
        if not args.no_browser and not args.headless:
            open_browser(existing["url"])
        return 0

    config_path = default_config_path()
    if not config_path.exists():
        print("No Compute Sifter profile was found.")
        print("Run 'aa_sifter setup' to create one, or 'aa_sifter recommend --save default'.")

    if args.foreground:
        return _serve(args, state_path, open_ui=not args.no_browser and not args.headless)

    try:
        state = start_backend_process(
            host=args.host,
            port=args.port,
            debug=args.debug,
            state_path=state_path,
        )
    except LauncherError as exc:
        print(str(exc))
        if exc.details:
            print(f"    {exc.details}")
        for option in exc.options:
            print(f"    - {option}")
        return 1

    print(f"Compute Sifter is ready at {state['url']}")
    print(f"Backend log: {log_file('backend.log')}")
    _print_issues(state["url"], state["token"])
    if not args.no_browser and not args.headless:
        open_browser(state["url"])
    return 0


def _serve(args: argparse.Namespace, state_path: Path, *, open_ui: bool = False) -> int:
    config_path = default_config_path()
    configure_file_logging(log_file("application.log"), debug=args.debug)
    service = ComputeSifterService.create(
        config_path=config_path,
        profile_name=getattr(args, "profile", None),
        ollama_installed_fn=ollama_installed,
    )
    token = args.token or new_token()
    server = create_server(service, host=args.host, port=args.port, token=token)
    url = server_url(server)
    write_state(
        state_path,
        {
            "pid": os.getpid(),
            "url": url,
            "host": args.host,
            "port": server.server_address[1],
            "token": token,
            "started_by": "in-process" if open_ui else "serve",
        },
    )
    clear = lambda: clear_state(state_path)  # noqa: E731
    atexit.register(clear)
    atexit.register(service.close)

    def _handle_signal(signum, frame):  # noqa: ANN001, ARG001
        raise KeyboardInterrupt

    signal.signal(signal.SIGINT, _handle_signal)
    if open_ui:
        print(f"Compute Sifter is ready at {url}")
        open_browser(url)
    else:
        print(f"Compute Sifter backend listening at {url}")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()
        clear()
        service.close()
    return 0


def serve(args: argparse.Namespace) -> int:
    state_path = Path(args.state_file) if args.state_file else desktop_state_file()
    return _serve(args, state_path, open_ui=False)


def stop(args: argparse.Namespace) -> int:
    state_path = Path(args.state_file) if args.state_file else desktop_state_file()
    state = read_state(state_path)
    if state and state.get("token") and state.get("url"):
        health = health_check(state["url"], state["token"])
        if health is None:
            clear_state(state_path)
            print("Removed stale state file.")
            return 0
    stopped = stop_backend(state_path)
    print("Stopped Compute Sifter." if stopped else "Compute Sifter is not running.")
    return 0


def dispatch(command: str, args: argparse.Namespace) -> int:
    if command in {"launch", "desktop"}:
        return launch(args)
    if command == "serve":
        return serve(args)
    if command == "stop":
        return stop(args)
    print(f"Unknown desktop command: {command}", file=sys.stderr)
    return 1
