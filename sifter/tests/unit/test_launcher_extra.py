from __future__ import annotations

import logging
from pathlib import Path

from aa_sifter.desktop import launcher


def test_new_token_is_unique():
    assert launcher.new_token() != launcher.new_token()


def test_read_state_invalid_json(tmp_path: Path):
    path = tmp_path / "state.json"
    path.write_text("{not json", encoding="utf-8")
    assert launcher.read_state(path) is None


def test_wait_for_state(tmp_path: Path):
    path = tmp_path / "state.json"
    launcher.write_state(path, {"url": "http://127.0.0.1:1"})
    assert launcher.wait_for_state(path, timeout=1)["url"].endswith(":1")


def test_start_backend_process_success(monkeypatch, tmp_path: Path):
    monkeypatch.setattr(launcher, "ensure_dirs", lambda: None)

    def fake_spawn(**kwargs):
        launcher.write_state(kwargs["state_path"], {"url": "http://127.0.0.1:9", "token": "t"})

        class Process:
            pid = 4321

        return Process()

    monkeypatch.setattr(launcher, "spawn_backend", fake_spawn)
    monkeypatch.setattr(
        launcher, "health_check", lambda url, token, timeout=1.5: {"status": "ready"}
    )
    state = launcher.start_backend_process(state_path=tmp_path / "state.json", port=0)
    assert state["owned"] is True
    assert state["pid"] == 4321
    assert state["health"]["status"] == "ready"


def test_stop_backend_force_with_dead_pid(tmp_path: Path):
    path = tmp_path / "state.json"
    launcher.write_state(path, {"url": "http://127.0.0.1:1", "token": "t", "pid": 999999})
    assert launcher.stop_backend(path, force=True) is True
    assert launcher.read_state(path) is None


def test_open_browser_uses_webbrowser(monkeypatch):
    calls: list[str] = []
    monkeypatch.setattr(launcher.webbrowser, "open", lambda url: calls.append(url) or True)
    assert launcher.open_browser("http://127.0.0.1:9") is True
    assert calls == ["http://127.0.0.1:9"]


def test_configure_file_logging(tmp_path: Path):
    logger = logging.getLogger("aa_sifter")
    target = tmp_path / "logs" / "application.log"
    launcher.configure_file_logging(target, debug=True)
    handlers = [h for h in logger.handlers if isinstance(h, logging.FileHandler)]
    assert handlers
    for handler in handlers:
        logger.removeHandler(handler)
        handler.close()


def test_stop_backend_without_state_returns_false(tmp_path: Path):
    assert launcher.stop_backend(tmp_path / "missing.json") is False
