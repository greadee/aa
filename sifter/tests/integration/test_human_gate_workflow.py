from __future__ import annotations

import pytest

from aa_sifter.decisions.approval import ApprovalAction, HumanDecision
from aa_sifter.decisions.classifier import DecisionLevel
from aa_sifter.decisions.policy import StandingRule, StandingRuleEngine
from aa_sifter.models.config import SifterConfig
from tests.fixtures.helpers import (
    CRITICAL_TASK,
    DATABASE_SWAP_TASK,
    MAJOR_ARCHITECTURE_TASK,
    MAJOR_AUTH_TASK,
    ScriptedApprovalProvider,
    make_sifter,
)


@pytest.mark.integration
@pytest.mark.asyncio
async def test_major_decision_does_not_call_cloud_without_human_approval(
    config: SifterConfig, fake_provider, store
):
    """Highest-priority regression: major decisions stop before cloud reasoning."""
    aa_sifter = make_sifter(config, fake_provider, store=store)
    result = await aa_sifter.run(MAJOR_AUTH_TASK)

    assert result.status == "stopped", f"expected human stop, got {result.status}"
    assert result.decision_level == DecisionLevel.MAJOR
    assert result.requires_human_approval is True
    assert fake_provider.cloud_calls == 0, "DeepSeek must NOT be called before approval"
    assert fake_provider.local_calls == 0


@pytest.mark.integration
@pytest.mark.asyncio
async def test_critical_decision_blocks_before_cloud(config: SifterConfig, fake_provider, store):
    aa_sifter = make_sifter(config, fake_provider, store=store)
    result = await aa_sifter.run(CRITICAL_TASK)

    assert result.decision_level == DecisionLevel.CRITICAL
    assert result.status == "stopped"
    assert fake_provider.cloud_calls == 0


@pytest.mark.integration
@pytest.mark.asyncio
async def test_major_decision_two_stage_approval(config: SifterConfig, fake_provider, store):
    approver = ScriptedApprovalProvider(
        [
            HumanDecision(
                action=ApprovalAction.ALLOW_EXPERT_ANALYSIS,
                constraints=["Do not replace SQLite"],
            ),
            HumanDecision(
                action=ApprovalAction.APPROVE,
                constraints=["Keep backwards compatibility"],
            ),
        ]
    )
    aa_sifter = make_sifter(config, fake_provider, approval_provider=approver, store=store)
    result = await aa_sifter.run(MAJOR_AUTH_TASK)

    assert result.status == "completed"
    assert fake_provider.cloud_calls >= 1
    assert len(approver.requests) == 2, "analysis and implementation need separate approvals"
    assert approver.requests[0].cloud_reasoning_invoked is False
    assert approver.requests[1].cloud_reasoning_invoked is True
    assert any(
        "Do not replace SQLite" in message.content
        for batch in fake_provider.messages
        for message in batch
    ), "human constraints must propagate downstream"


@pytest.mark.integration
@pytest.mark.asyncio
async def test_expert_analysis_does_not_imply_implementation_approval(
    config: SifterConfig, fake_provider, store
):
    approver = ScriptedApprovalProvider(
        [
            HumanDecision(action=ApprovalAction.ALLOW_EXPERT_ANALYSIS),
            HumanDecision(action=ApprovalAction.STOP),
        ]
    )
    aa_sifter = make_sifter(config, fake_provider, approval_provider=approver, store=store)
    result = await aa_sifter.run(MAJOR_ARCHITECTURE_TASK)

    assert result.status == "stopped"
    assert fake_provider.cloud_calls == 1, "only the analysis call is permitted"
    assert fake_provider.local_calls == 0, "implementation must not start without approval"


@pytest.mark.integration
@pytest.mark.asyncio
async def test_standing_rule_conflict_requires_approval(config: SifterConfig, fake_provider, store):
    rules = StandingRuleEngine(
        store=store,
        rules=[StandingRule(text="SQLite is the approved persistence technology.")],
    )
    aa_sifter = make_sifter(config, fake_provider, store=store, rules=rules)
    result = await aa_sifter.run(DATABASE_SWAP_TASK)

    assert result.requires_human_approval is True
    assert result.status == "stopped"
    assert fake_provider.cloud_calls == 0
