from __future__ import annotations

import pytest

from aa_sifter.decisions.approval import (
    ApprovalAction,
    ApprovalRequest,
    AutoApprovalProvider,
    HumanDecision,
)
from aa_sifter.decisions.classifier import DecisionLevel
from aa_sifter.decisions.gate import DecisionGate, Proposal
from aa_sifter.decisions.policy import StandingRule, StandingRuleEngine
from aa_sifter.models.config import SifterConfig


@pytest.mark.asyncio
async def test_major_proposal_requires_approval(config: SifterConfig):
    gate = DecisionGate(config)
    result = await gate.classify("Replace authentication with OAuth.")
    assert result.level == DecisionLevel.MAJOR
    assert result.requires_human_approval is True


@pytest.mark.asyncio
async def test_routine_proposal_needs_no_approval(config: SifterConfig):
    gate = DecisionGate(config)
    result = await gate.classify("Add logging to the parser.")
    assert result.requires_human_approval is False


@pytest.mark.asyncio
async def test_standing_rule_conflict_forces_major(config: SifterConfig):
    rules = StandingRuleEngine(
        rules=[StandingRule(text="SQLite is the approved persistence technology.")]
    )
    gate = DecisionGate(config, rules=rules)
    result = await gate.classify("Migrate the project to PostgreSQL.")
    assert result.level == DecisionLevel.MAJOR
    assert result.conflicts_with_rules
    assert result.requires_human_approval is True


@pytest.mark.asyncio
async def test_changing_previously_approved_decision_is_major(config: SifterConfig):
    rules = StandingRuleEngine(
        rules=[StandingRule(text="Never change authentication architecture without asking.")]
    )
    gate = DecisionGate(config, rules=rules)
    result = await gate.classify("Replace the auth provider with a new OAuth service.")
    assert result.level == DecisionLevel.MAJOR
    assert result.conflicts_with_rules


@pytest.mark.asyncio
async def test_proposal_flags_raise_level(config: SifterConfig):
    gate = DecisionGate(config)
    result = await gate.classify(Proposal(description="Small edit", security_sensitive=True))
    assert result.level == DecisionLevel.MAJOR


@pytest.mark.asyncio
async def test_significant_approval_can_be_configured(config: SifterConfig):
    config.human_approval_for_significant_decisions = True
    gate = DecisionGate(config)
    result = await gate.classify("Refactor several related files.")
    assert result.level == DecisionLevel.SIGNIFICANT
    assert result.requires_human_approval is True


@pytest.mark.asyncio
async def test_gate_records_approval(config: SifterConfig, store):
    from aa_sifter.decisions.approval import ApprovalRequest
    from aa_sifter.metrics.trace import Trace

    provider = AutoApprovalProvider(
        major=HumanDecision(action=ApprovalAction.ALLOW_EXPERT_ANALYSIS)
    )
    trace = Trace(history=store, show_trace=False)
    gate = DecisionGate(config, approval_provider=provider, trace=trace, history=store)
    result = await gate.classify("Replace authentication with OAuth.")
    decision = await gate.require_approval(ApprovalRequest(category=result.level, issue="test"))
    assert decision.authorized
    assert provider.requests
    approvals = store._conn.execute("SELECT COUNT(*) FROM approvals").fetchone()[0]
    assert approvals == 1


@pytest.mark.asyncio
async def test_critical_approval_cannot_be_disabled(config: SifterConfig):
    config.human_approval_for_critical_decisions = False
    assert config.human_approval_for_critical_decisions is True
    gate = DecisionGate(config)
    result = await gate.classify("Drop table users in production.")
    assert result.level == DecisionLevel.CRITICAL
    assert result.requires_human_approval is True


@pytest.mark.asyncio
async def test_rejection_stops_action(config: SifterConfig):
    provider = AutoApprovalProvider(major=HumanDecision(action=ApprovalAction.REJECT))
    gate = DecisionGate(config, approval_provider=provider)
    decision = await gate.require_approval(
        ApprovalRequest(category=DecisionLevel.MAJOR, issue="risky")
    )
    assert decision.authorized is False
    assert decision.action == ApprovalAction.REJECT


@pytest.mark.asyncio
async def test_modification_is_preserved(config: SifterConfig):
    provider = AutoApprovalProvider(
        major=HumanDecision(
            action=ApprovalAction.MODIFY, note="use a repository abstraction instead"
        )
    )
    gate = DecisionGate(config, approval_provider=provider)
    decision = await gate.require_approval(
        ApprovalRequest(category=DecisionLevel.MAJOR, issue="persistence")
    )
    assert decision.authorized is False
    assert "repository abstraction" in decision.note


@pytest.mark.asyncio
async def test_constraints_propagate_from_gate(config: SifterConfig):
    provider = AutoApprovalProvider(
        major=HumanDecision(
            action=ApprovalAction.ALLOW_EXPERT_ANALYSIS,
            constraints=["Do not replace SQLite"],
        )
    )
    gate = DecisionGate(config, approval_provider=provider)
    decision = await gate.require_approval(
        ApprovalRequest(category=DecisionLevel.MAJOR, issue="persistence")
    )
    assert "Do not replace SQLite" in decision.constraints


@pytest.mark.asyncio
async def test_gate_without_provider_defaults_to_stop(config: SifterConfig):
    gate = DecisionGate(config)
    decision = await gate.require_approval(
        ApprovalRequest(category=DecisionLevel.MAJOR, issue="persistence")
    )
    assert decision.authorized is False
    assert decision.action == ApprovalAction.STOP
