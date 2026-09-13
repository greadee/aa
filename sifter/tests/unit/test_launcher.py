from __future__ import annotations

import os
import socket
import time
from pathlib import Path

import pytest

from aa_sifter.desktop import launcher
from aa_sifter.desktop.service import ComputeSifterService
from tests.fixtures.desktop import RunningServer, start_server
from tests.fixtures.providers import FakeProvider


@pytest.fixture
def server(isolated_config: Path) -> RunningServer:
    svc = ComputeSifterService.create(
        config_path=isolated_config,
        provider=FakeProvider(),
        ollama_installed_fn=lambda: {"qwen3.5:9b"},
    )
    running = start_server(svc)
    yield running
    running.close()


def test_choose_port(monkeypatch):
    assert launcher.choose_port(preferred=0) == 0
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        free_port = sock.getsockname()[1]
    assert launcher.choose_port(preferred=free_port) == free_port


def test_port_available_false_when_bound(monkeypatch):
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        sock.listen()
        port = sock.getsockname()[1]
        assert launcher.port_available(port) is False


def test_state_roundtrip(tmp_path: Path):
    path = tmp_path / "state.json"
    launcher.write_state(path, {"url": "http://127.0.0.1:1", "token": "abc"})
    assert launcher.read_state(path)["token"] == "abc"
    launcher.clear_state(path)
    assert launcher.read_state(path) is None


def test_health_check_rejects_bad_token(server: RunningServer):
    assert launcher.health_check(server.url, "wrong") is None
    health = launcher.health_check(server.url, server.token)
    assert health is not None and health["status"] == "ready"


def test_existing_instance(server: RunningServer, tmp_path: Path):
    path = tmp_path / "state.json"
    launcher.write_state(path, {"url": server.url, "token": server.token})
    found = launcher.existing_instance(path)
    assert found is not None
    assert found["health"]["status"] == "ready"


def test_existing_instance_ignores_stale(server: RunningServer, tmp_path: Path):
    path = tmp_path / "state.json"
    launcher.write_state(path, {"url": "http://127.0.0.1:1", "token": "abc"})
    assert launcher.existing_instance(path) is None


def test_stop_backend_shuts_down_owned_server(server: RunningServer, tmp_path: Path):
    path = tmp_path / "state.json"
    launcher.write_state(path, {"url": server.url, "token": server.token, "pid": None})
    assert launcher.stop_backend(path) is True
    assert launcher.read_state(path) is None
    time.sleep(0.3)
    assert launcher.health_check(server.url, server.token) is None


def test_start_backend_process_reports_clean_error(monkeypatch, tmp_path: Path):
    monkeypatch.setattr(launcher, "ensure_dirs", lambda: None)
    monkeypatch.setattr(launcher, "spawn_backend", lambda **kwargs: object())
    monkeypatch.setattr(launcher, "wait_for_state", lambda path, timeout: None)
    with pytest.raises(launcher.LauncherError) as exc:
        launcher.start_backend_process(state_path=tmp_path / "state.json", port=0)
    assert "did not become ready" in str(exc.value)
    assert exc.value.options


def test_process_alive_reports_current_process():
    assert launcher._process_alive(os.getpid()) is True


def test_process_alive_reports_dead_process():
    assert launcher._process_alive(999999) is False
