from __future__ import annotations

import pytest

from aa_sifter.models.config import SifterConfig
from aa_sifter.verification.verifier import CallableVerifier
from tests.fixtures.helpers import HARD_DEBUGGING_TASK, MEDIUM_TASK, make_sifter


@pytest.mark.integration
@pytest.mark.asyncio
async def test_scenario_budget_exhaustion_stops_further_cloud_calls(config: SifterConfig, store):
    config.max_cloud_cost_per_task = 0.00002
    provider = _provider()
    failing = CallableVerifier("never", lambda text: (False, "always fails"))
    aa_sifter = make_sifter(config, provider, store=store, verifier=failing)
    result = await aa_sifter.run(HARD_DEBUGGING_TASK, policy="cloud_only")

    assert provider.cloud_calls == 1, "only the affordable cloud call may run"
    assert "budget" in result.answer.lower() or "skipped" in result.answer.lower()
    assert result.status == "failed"


@pytest.mark.integration
@pytest.mark.asyncio
async def test_budget_exhausted_before_first_cloud_call(config: SifterConfig, store):
    config.max_cloud_calls_per_task = 0
    provider = _provider()
    aa_sifter = make_sifter(config, provider, store=store)
    result = await aa_sifter.run(MEDIUM_TASK, policy="cloud_only")

    assert provider.cloud_calls == 0, "provider must never be invoked when the budget is empty"
    assert result.status == "blocked"


@pytest.mark.integration
@pytest.mark.asyncio
async def test_no_budget_falls_back_to_local(config: SifterConfig, store):
    config.max_cloud_calls_per_task = 0
    provider = _provider()
    aa_sifter = make_sifter(config, provider, store=store)
    result = await aa_sifter.run(MEDIUM_TASK, policy="budget_constrained")

    assert provider.cloud_calls == 0
    assert provider.local_calls >= 1
    assert result.route == "local"
    assert result.status == "completed"


def _provider():
    from tests.fixtures.providers import FakeProvider

    return FakeProvider()
