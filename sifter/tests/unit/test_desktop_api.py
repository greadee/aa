from __future__ import annotations

import time
from pathlib import Path

import httpx
import pytest

from aa_sifter.desktop.service import ComputeSifterService
from tests.fixtures.desktop import RunningServer, start_server
from tests.fixtures.providers import FakeProvider, SlowProvider


@pytest.fixture
def service(isolated_config: Path) -> ComputeSifterService:
    svc = ComputeSifterService.create(
        config_path=isolated_config,
        provider=FakeProvider(),
        ollama_installed_fn=lambda: {"qwen3.5:9b"},
    )
    yield svc
    svc.close()


@pytest.fixture
def server(service: ComputeSifterService) -> RunningServer:
    running = start_server(service)
    yield running
    running.close()


def test_api_requires_token(server: RunningServer):
    with httpx.Client(base_url=server.url, timeout=5) as client:
        assert client.get("/api/health").status_code == 401


def test_index_serves_token(server: RunningServer):
    with httpx.Client(base_url=server.url, timeout=5) as client:
        response = client.get("/")
        assert response.status_code == 200
        assert server.token in response.text
        assert "aa-sifter" in response.text


def test_health_and_status(server: RunningServer):
    with server.client() as client:
        health = client.get("/api/health").json()
        assert health["status"] == "ready"
        status = client.get("/api/status").json()
        assert status["profile"]["name"] == "default"
        assert client.get("/api/models").json()["local"]["configured"]["model"]
        assert "checks" in client.get("/api/doctor").json()
        assert "usage" in client.get("/api/metrics").json()
        assert "profiles" in client.get("/api/profiles").json()


def _poll_task(client: httpx.Client, task_id: str, timeout: float = 8.0) -> dict:
    deadline = time.time() + timeout
    while time.time() < deadline:
        record = client.get(f"/api/tasks/{task_id}").json()
        if record.get("status") in {"completed", "failed", "cancelled", "stopped", "error"}:
            return record
        time.sleep(0.05)
    raise AssertionError("task did not finish")


def test_post_task_and_result(server: RunningServer):
    with server.client() as client:
        response = client.post(
            "/api/tasks", json={"prompt": "Write unit tests for a small utility function."}
        )
        assert response.status_code == 202
        task_id = response.json()["id"]
        record = _poll_task(client, task_id)
        assert record["status"] == "completed"
        assert record["result"]["route"] == "local"


def test_task_requires_prompt(server: RunningServer):
    with server.client() as client:
        assert client.post("/api/tasks", json={"prompt": "  "}).status_code == 400


def test_major_approval_over_api(server: RunningServer):
    with server.client() as client:
        task_id = client.post(
            "/api/tasks",
            json={"prompt": "Replace authentication with OAuth while preserving old sessions."},
        ).json()["id"]
        deadline = time.time() + 8
        record = {}
        while time.time() < deadline:
            record = client.get(f"/api/tasks/{task_id}").json()
            if record.get("approval_request"):
                break
            time.sleep(0.05)
        assert record["approval_request"]["category"] == "major"
        client.post(f"/api/tasks/{task_id}/approval", json={"action": "stop_task"})
        final = _poll_task(client, task_id)
        assert final["status"] in {"stopped", "failed"}


def test_cancel_over_api(isolated_config: Path):
    svc = ComputeSifterService.create(
        config_path=isolated_config,
        provider=SlowProvider(delay=0.5),
        ollama_installed_fn=lambda: {"qwen3.5:9b"},
    )
    running = start_server(svc)
    try:
        with running.client() as client:
            task_id = client.post("/api/tasks", json={"prompt": "Add logging."}).json()["id"]
            time.sleep(0.1)
            assert client.post(f"/api/tasks/{task_id}/cancel").json()["cancelled"] is True
            final = _poll_task(client, task_id)
            assert final["status"] == "cancelled"
    finally:
        running.close()


def test_profile_activation_over_api(server: RunningServer):
    with server.client() as client:
        created = client.post("/api/profiles", json={"name": "coding"})
        assert created.status_code == 201
        activated = client.post("/api/profiles/coding/activate")
        assert activated.status_code == 200
        assert activated.json()["active"] is True


def test_unknown_endpoint_404(server: RunningServer):
    with server.client() as client:
        assert client.get("/api/nope").status_code == 404


def test_post_requires_token(server: RunningServer):
    with httpx.Client(base_url=server.url, timeout=5) as client:
        assert client.post("/api/tasks", json={"prompt": "hi"}).status_code == 401


def test_set_model_endpoint(server: RunningServer):
    with server.client() as client:
        response = client.post(
            "/api/models/set",
            json={"tier": "local", "provider": "ollama", "model": "future-model:12b"},
        )
        assert response.status_code == 200
        assert response.json()["local"]["configured"]["model"] == "future-model:12b"


def test_detect_system_endpoint(server: RunningServer):
    with server.client() as client:
        response = client.post("/api/system/detect", json={"save_as": "desktop"})
        assert response.status_code == 200
        assert "operating_system" in response.json()


def test_duplicate_profile_create_is_400(server: RunningServer):
    with server.client() as client:
        assert client.post("/api/profiles", json={"name": "coding"}).status_code == 201
        assert client.post("/api/profiles", json={"name": "coding"}).status_code == 400
        assert client.post("/api/profiles", json={"name": ""}).status_code == 400


def test_missing_task_and_events_404(server: RunningServer):
    with server.client() as client:
        assert "error" in client.get("/api/tasks/does-not-exist").json()
        assert client.get("/api/tasks/does-not-exist/events?token=x").status_code == 404


def test_static_path_traversal_blocked(server: RunningServer):
    with server.client() as client:
        assert client.get("/api/../secret").status_code == 404


@pytest.mark.asyncio
async def test_sse_endpoint_streams_events(isolated_config: Path):
    svc = ComputeSifterService.create(
        config_path=isolated_config,
        provider=FakeProvider(),
        ollama_installed_fn=lambda: {"qwen3.5:9b"},
    )
    running = start_server(svc)
    try:
        task_id = running.service.submit("Write a small helper.").id
        events: list[str] = []
        with httpx.Client(base_url=running.url, timeout=10) as client:
            with client.stream(
                "GET", f"/api/tasks/{task_id}/events?token={running.token}"
            ) as response:
                for line in response.iter_lines():
                    if line.startswith("data:"):
                        events.append(line)
                    if '"event": "done"' in line or '"event":"done"' in line:
                        break
        assert any("routing" in event for event in events)
        assert any("done" in event for event in events)
    finally:
        running.close()
