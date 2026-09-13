from __future__ import annotations

import pytest

from aa_sifter.models.config import SifterConfig
from aa_sifter.verification.verifier import CallableVerifier
from tests.fixtures.helpers import EASY_LOCAL_TASK, MEDIUM_TASK, make_sifter


@pytest.mark.integration
@pytest.mark.asyncio
async def test_scenario_local_success(config: SifterConfig, fake_provider, store):
    """Scenario 1: easy task is handled locally with verification and no cloud."""
    aa_sifter = make_sifter(config, fake_provider, store=store)
    result = await aa_sifter.run(EASY_LOCAL_TASK)

    assert result.status == "completed", f"unexpected status: {result.status}"
    assert result.route == "local", "easy task must route local"
    assert fake_provider.cloud_calls == 0, "cloud must not be called for a local task"
    assert fake_provider.local_calls == 1, "local task should be attempted exactly once"
    assert result.usage.human_approval_requests == 0
    assert result.usage.cloud_calls == 0


@pytest.mark.integration
@pytest.mark.asyncio
async def test_medium_task_uses_objective_verification(config: SifterConfig, fake_provider, store):
    seen: list[str] = []

    def validator(text: str) -> tuple[bool, str]:
        seen.append(text)
        return True, ""

    verifier = CallableVerifier("captures-output", validator)
    aa_sifter = make_sifter(config, fake_provider, store=store, verifier=verifier)
    result = await aa_sifter.run(MEDIUM_TASK)

    assert result.status == "completed"
    assert seen, "verification must receive the model output"


@pytest.mark.integration
@pytest.mark.asyncio
async def test_failed_verification_reports_failure(config: SifterConfig, fake_provider, store):
    failing = CallableVerifier("never", lambda text: (False, "always fails"))
    aa_sifter = make_sifter(config, fake_provider, store=store, verifier=failing)
    result = await aa_sifter.run(MEDIUM_TASK, policy="local_only")

    assert result.status == "failed"
    assert fake_provider.cloud_calls == 0


@pytest.mark.integration
@pytest.mark.asyncio
async def test_history_records_local_run(config: SifterConfig, fake_provider, store):
    aa_sifter = make_sifter(config, fake_provider, store=store)
    await aa_sifter.run(EASY_LOCAL_TASK)

    runs = store.recent_runs()
    assert len(runs) == 1
    assert runs[0]["route"] == "local"
    assert runs[0]["final_success"] == 1
