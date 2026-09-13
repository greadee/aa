from __future__ import annotations

import asyncio
from pathlib import Path

import pytest

from aa_sifter.desktop.service import ComputeSifterService
from tests.fixtures.providers import FakeProvider, SlowProvider


def _service(config_path: Path, provider=None, **kwargs) -> ComputeSifterService:
    return ComputeSifterService.create(
        config_path=config_path,
        provider=provider or FakeProvider(),
        ollama_installed_fn=lambda: {"qwen3.5:9b"},
        **kwargs,
    )


async def _wait_for(predicate, timeout: float = 8.0) -> None:
    deadline = asyncio.get_event_loop().time() + timeout
    while asyncio.get_event_loop().time() < deadline:
        if predicate():
            return
        await asyncio.sleep(0.02)
    raise AssertionError("condition was not met in time")


@pytest.mark.asyncio
async def test_health_and_status(isolated_config: Path):
    service = _service(isolated_config)
    try:
        health = service.health()
        assert health["status"] == "ready"
        assert health["profile"] == "default"
        assert health["local_provider"] == "ready"
        status = service.status()
        assert status["profile"]["name"] == "default"
        assert status["models"]["local"]["configured"]["model"] == "qwen3.5:9b"
    finally:
        service.close()


@pytest.mark.asyncio
async def test_submit_local_task_streams_events(isolated_config: Path):
    provider = FakeProvider()
    service = _service(isolated_config, provider)
    try:
        record = service.submit("Write unit tests for this small utility function.")
        record = await service.wait(record.id)
        assert record is not None
        assert record.status == "completed"
        assert record.route == "local"
        assert provider.cloud_calls == 0
        event_names = [event["event"] for event in record.events]
        assert "routing" in event_names
        assert "complete" in event_names
        assert record.result is not None and record.result["answer"]
    finally:
        service.close()


@pytest.mark.asyncio
async def test_major_decision_waits_for_desktop_approval(isolated_config: Path):
    provider = FakeProvider()
    service = _service(isolated_config, provider)
    try:
        record = service.submit("Replace authentication with OAuth while preserving old sessions.")
        await _wait_for(lambda: service.get(record.id).status == "waiting_approval")
        pending = service.get(record.id)
        assert pending.approval_request is not None
        assert pending.approval_request["category"] == "major"
        assert provider.cloud_calls == 0, "cloud must not run before approval"

        assert service.resolve_approval(record.id, action="allow_expert_analysis")
        await _wait_for(lambda: provider.cloud_calls >= 1)
        await _wait_for(lambda: service.get(record.id).approval_request is not None)
        assert service.resolve_approval(record.id, action="approve")
        record = await service.wait(record.id)
        assert record.status == "completed"
    finally:
        service.close()


@pytest.mark.asyncio
async def test_major_decision_rejection_stops_task(isolated_config: Path):
    provider = FakeProvider()
    service = _service(isolated_config, provider)
    try:
        record = service.submit("Replace the persistence architecture with a new design.")
        await _wait_for(lambda: service.get(record.id).status == "waiting_approval")
        assert service.resolve_approval(record.id, action="stop_task")
        record = await service.wait(record.id)
        assert record.status in {"stopped", "failed"}
        assert provider.cloud_calls == 0
    finally:
        service.close()


@pytest.mark.asyncio
async def test_cancellation_of_running_task(isolated_config: Path):
    service = _service(isolated_config, SlowProvider(delay=0.5))
    try:
        record = service.submit("Add logging to the parser.")
        await _wait_for(lambda: service.get(record.id).status == "running")
        assert service.cancel(record.id)
        record = await service.wait(record.id)
        assert record.status == "cancelled"
    finally:
        service.close()


@pytest.mark.asyncio
async def test_profile_management(isolated_config: Path):
    service = _service(isolated_config)
    try:
        assert any(p["name"] == "default" for p in service.profiles())
        created = service.create_profile("coding")
        assert created["name"] == "coding"
        activated = service.activate_profile("coding")
        assert activated["active"] is True
        assert service.app_config.active_profile == "coding"
        active = service.delete_profile("coding")
        assert active == "default"
    finally:
        service.close()


@pytest.mark.asyncio
async def test_startup_issues_ollama_and_missing_key(isolated_config: Path, monkeypatch):
    monkeypatch.delenv("DEEPSEEK_API_KEY", raising=False)
    service = ComputeSifterService.create(
        config_path=isolated_config,
        provider=FakeProvider(),
        ollama_installed_fn=lambda: None,
    )
    try:
        kinds = {issue["kind"] for issue in service.startup_issues()}
        assert "ollama_unreachable" in kinds
        assert "missing_expert_key" in kinds
    finally:
        service.close()


@pytest.mark.asyncio
async def test_doctor_and_metrics(isolated_config: Path):
    service = _service(isolated_config)
    try:
        report = service.doctor(live=False)
        assert "checks" in report and isinstance(report["ok"], bool)
        metrics = service.metrics()
        assert metrics["tasks"] == 0
        assert metrics["usage"]["cloud_calls"] == 0
    finally:
        service.close()
