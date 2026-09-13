from __future__ import annotations

import pytest

from aa_sifter.models.config import SifterConfig
from aa_sifter.models.provider import ProviderError
from tests.fixtures.helpers import HARD_DEBUGGING_TASK, make_sifter
from tests.fixtures.providers import FakeProvider, ScriptedProvider


@pytest.mark.integration
@pytest.mark.asyncio
async def test_scenario_local_retry_then_cloud_escalation(config: SifterConfig, store):
    provider = FakeProvider(fail_local=True, fail_local_times=100)
    aa_sifter = make_sifter(config, provider, store=store)
    result = await aa_sifter.run(HARD_DEBUGGING_TASK)

    assert result.status == "completed", f"escalation should recover, got {result.status}"
    assert provider.local_calls == config.local_max_retries + 1, (
        "local retry budget must be respected"
    )
    assert provider.cloud_calls >= 1, "expert escalation must occur after local failures"
    assert result.usage.escalations >= 1
    assert result.usage.human_approval_requests == 0, "routine debugging needs no human approval"


@pytest.mark.integration
@pytest.mark.asyncio
async def test_local_failure_within_retry_budget_stays_local(config: SifterConfig, store):
    provider = FakeProvider(fail_local=True, fail_local_times=1)
    aa_sifter = make_sifter(config, provider, store=store)
    result = await aa_sifter.run(HARD_DEBUGGING_TASK)

    assert result.status == "completed"
    assert provider.local_calls == 2, "one failure then one successful retry"
    assert provider.cloud_calls == 0, "no escalation when the retry succeeds"


@pytest.mark.integration
@pytest.mark.asyncio
async def test_local_timeout_escalates(config: SifterConfig, store):
    provider = FakeProvider(timeout_local=True)
    aa_sifter = make_sifter(config, provider, store=store)
    result = await aa_sifter.run(HARD_DEBUGGING_TASK)

    assert result.status == "completed"
    assert provider.cloud_calls >= 1


@pytest.mark.integration
@pytest.mark.asyncio
async def test_cloud_failure_is_graceful(config: SifterConfig, store):
    provider = FakeProvider(fail_local=True, fail_local_times=100, timeout_cloud=True)
    aa_sifter = make_sifter(config, provider, store=store)
    result = await aa_sifter.run(HARD_DEBUGGING_TASK)

    assert result.status == "error"
    assert result.blocked_reason
    assert "timed out" in result.blocked_reason.lower()


@pytest.mark.integration
@pytest.mark.asyncio
async def test_scripted_cloud_recovery_preserves_local_context(config: SifterConfig, store):
    provider = ScriptedProvider(
        local_outcomes=[ProviderError("local failed", retryable=True)] * 3,
        cloud_outcomes=["the correct diagnosis"],
    )
    aa_sifter = make_sifter(config, provider, store=store)
    result = await aa_sifter.run(HARD_DEBUGGING_TASK)

    assert result.status == "completed"
    assert "the correct diagnosis" in result.answer
    assert provider.cloud_calls == 1


@pytest.mark.integration
@pytest.mark.asyncio
async def test_local_only_policy_never_touches_cloud(config: SifterConfig, store):
    provider = FakeProvider(fail_local=True, fail_local_times=100)
    aa_sifter = make_sifter(config, provider, store=store)
    result = await aa_sifter.run(HARD_DEBUGGING_TASK, policy="local_only")

    assert provider.cloud_calls == 0
    assert result.status == "failed"


@pytest.mark.integration
@pytest.mark.asyncio
async def test_local_failure_implying_architecture_change_asks_user(config: SifterConfig, store):
    """A local failure that implies a redesign must not auto-escalate to cloud."""
    from aa_sifter.decisions.approval import ApprovalAction, HumanDecision
    from tests.fixtures.helpers import MAJOR_ARCHITECTURE_TASK, ScriptedApprovalProvider

    provider = FakeProvider(fail_local=True, fail_local_times=100)
    approver = ScriptedApprovalProvider(
        [
            HumanDecision(action=ApprovalAction.CONTINUE_CURRENT),
            HumanDecision(action=ApprovalAction.STOP),
        ]
    )
    aa_sifter = make_sifter(config, provider, approval_provider=approver, store=store)
    result = await aa_sifter.run(MAJOR_ARCHITECTURE_TASK)

    assert result.status == "failed"
    assert provider.cloud_calls == 0
    assert "architectural escalation requires human approval" in result.answer
    assert len(approver.requests) == 2, "analysis and escalation each need a decision"


@pytest.mark.integration
@pytest.mark.asyncio
async def test_user_may_permit_architectural_escalation(config: SifterConfig, store):
    from aa_sifter.decisions.approval import ApprovalAction, HumanDecision
    from tests.fixtures.helpers import MAJOR_ARCHITECTURE_TASK, ScriptedApprovalProvider

    provider = FakeProvider(fail_local=True, fail_local_times=100)
    approver = ScriptedApprovalProvider(
        [
            HumanDecision(action=ApprovalAction.CONTINUE_CURRENT),
            HumanDecision(action=ApprovalAction.ALLOW_EXPERT_ANALYSIS),
        ]
    )
    aa_sifter = make_sifter(config, provider, approval_provider=approver, store=store)
    result = await aa_sifter.run(MAJOR_ARCHITECTURE_TASK)

    assert result.status == "completed"
    assert provider.cloud_calls >= 1
