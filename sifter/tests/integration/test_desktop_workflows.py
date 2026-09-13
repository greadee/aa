from __future__ import annotations

import time
from pathlib import Path

import httpx
import pytest

from aa_sifter.desktop.service import ComputeSifterService
from tests.fixtures.desktop import start_server
from tests.fixtures.providers import FakeProvider, SlowProvider


def _service(config_path: Path, provider=None, **kwargs) -> ComputeSifterService:
    return ComputeSifterService.create(
        config_path=config_path,
        provider=provider or FakeProvider(),
        ollama_installed_fn=kwargs.pop("ollama_installed_fn", lambda: {"qwen3.5:9b"}),
        **kwargs,
    )


def _poll(client: httpx.Client, task_id: str, timeout: float = 8.0) -> dict:
    deadline = time.time() + timeout
    while time.time() < deadline:
        record = client.get(f"/api/tasks/{task_id}").json()
        if record.get("status") in {"completed", "failed", "cancelled", "stopped", "error"}:
            return record
        time.sleep(0.05)
    raise AssertionError("task did not finish")


@pytest.mark.integration
def test_scenario_a_successful_startup(isolated_config: Path):
    running = start_server(_service(isolated_config))
    try:
        with running.client() as client:
            health = client.get("/api/health").json()
            assert health["status"] == "ready"
            assert health["local_provider"] == "ready"
            task_id = client.post(
                "/api/tasks", json={"prompt": "Write unit tests for a small utility function."}
            ).json()["id"]
            record = _poll(client, task_id)
            assert record["status"] == "completed"
            assert record["result"]["route"] == "local"
    finally:
        running.close()


@pytest.mark.integration
def test_scenario_b_ollama_missing(isolated_config: Path):
    service = _service(isolated_config, ollama_installed_fn=lambda: None)
    running = start_server(service)
    try:
        with running.client() as client:
            health = client.get("/api/health").json()
            assert health["status"] == "ready"  # backend still usable
            issues = client.get("/api/status").json()["issues"]
            assert any(issue["kind"] == "ollama_unreachable" for issue in issues)
            # A recoverable error is surfaced, no crash, and fake routing still works.
            task_id = client.post(
                "/api/tasks", json={"prompt": "Add logging to the parser."}
            ).json()["id"]
            assert _poll(client, task_id)["status"] == "completed"
    finally:
        running.close()


@pytest.mark.integration
def test_scenario_c_missing_cloud_credential(isolated_config: Path, monkeypatch):
    monkeypatch.delenv("DEEPSEEK_API_KEY", raising=False)
    running = start_server(_service(isolated_config))
    try:
        with running.client() as client:
            status = client.get("/api/status").json()
            assert status["providers"]["expert"] == "missing_credentials"
            assert any(issue["kind"] == "missing_expert_key" for issue in status["issues"])
            task_id = client.post(
                "/api/tasks", json={"prompt": "Write unit tests for a small utility function."}
            ).json()["id"]
            record = _poll(client, task_id)
            assert record["status"] == "completed"
            assert record["result"]["usage"]["cloud_calls"] == 0
    finally:
        running.close()


@pytest.mark.integration
def test_scenario_d_major_approval_via_desktop(isolated_config: Path):
    provider = FakeProvider()
    running = start_server(_service(isolated_config, provider))
    try:
        with running.client() as client:
            task_id = client.post(
                "/api/tasks",
                json={"prompt": "Replace authentication with OAuth while preserving old sessions."},
            ).json()["id"]
            deadline = time.time() + 8
            record: dict = {}
            while time.time() < deadline:
                record = client.get(f"/api/tasks/{task_id}").json()
                if record.get("approval_request"):
                    break
                time.sleep(0.05)
            assert record["approval_request"]["category"] == "major"
            assert provider.cloud_calls == 0, "DeepSeek must not run before approval"
            client.post(f"/api/tasks/{task_id}/approval", json={"action": "allow_expert_analysis"})
            time.sleep(0.2)
            client.post(f"/api/tasks/{task_id}/approval", json={"action": "approve"})
            final = _poll(client, task_id)
            assert provider.cloud_calls >= 1
            assert final["status"] == "completed"
    finally:
        running.close()


@pytest.mark.integration
def test_scenario_e_reject_major_decision(isolated_config: Path):
    provider = FakeProvider()
    running = start_server(_service(isolated_config, provider))
    try:
        with running.client() as client:
            task_id = client.post(
                "/api/tasks",
                json={"prompt": "Replace the persistence architecture with a new design."},
            ).json()["id"]
            deadline = time.time() + 8
            while time.time() < deadline:
                record = client.get(f"/api/tasks/{task_id}").json()
                if record.get("approval_request"):
                    break
                time.sleep(0.05)
            client.post(f"/api/tasks/{task_id}/approval", json={"action": "stop_task"})
            final = _poll(client, task_id)
            assert final["status"] in {"stopped", "failed"}
            assert provider.cloud_calls == 0
    finally:
        running.close()


@pytest.mark.integration
def test_scenario_f_task_cancellation(isolated_config: Path):
    running = start_server(_service(isolated_config, SlowProvider(delay=0.5)))
    try:
        with running.client() as client:
            task_id = client.post("/api/tasks", json={"prompt": "Add logging."}).json()["id"]
            time.sleep(0.1)
            assert client.post(f"/api/tasks/{task_id}/cancel").json()["cancelled"] is True
            assert _poll(client, task_id)["status"] == "cancelled"
    finally:
        running.close()


@pytest.mark.integration
def test_scenario_g_backend_restart(isolated_config: Path):
    first = start_server(_service(isolated_config))
    try:
        with first.client() as client:
            assert client.get("/api/health").json()["status"] == "ready"
    finally:
        first.close()

    second = start_server(_service(isolated_config))
    try:
        with second.client() as client:
            assert client.get("/api/health").json()["status"] == "ready"
    finally:
        second.close()
