"""Enforced token limits and wired budget waivers / standing-rule grants."""

from __future__ import annotations

from io import StringIO

import pytest

from aa_sifter.app import ComputeSifter
from aa_sifter.context.compression import estimate_tokens, fit_messages
from aa_sifter.decisions.classifier import DecisionClassifier
from aa_sifter.decisions.gate import DecisionGate
from aa_sifter.decisions.policy import StandingRule, StandingRuleEngine
from aa_sifter.metrics.trace import Trace
from aa_sifter.metrics.usage import UsageMetrics, UsageTracker
from aa_sifter.models.provider import Message, Tier
from aa_sifter.routing.budget import BudgetExceeded, BudgetTracker
from aa_sifter.verification.verifier import VerificationResult
from tests.fixtures.helpers import RecordingApprovalProvider
from tests.fixtures.providers import FakeProvider


def _trace() -> Trace:
    return Trace("budget-test", debug=False, sink=StringIO())


def _call_usage() -> tuple[UsageMetrics, UsageTracker]:
    return UsageMetrics(), UsageTracker()


async def test_expert_max_output_tokens_forwarded(config):
    cfg = config.model_copy(update={"cloud_allowed": True, "expert_max_output_tokens": 1234})
    provider = FakeProvider()
    sifter = ComputeSifter(cfg, provider=provider)
    usage, tracker = _call_usage()

    await sifter._call_tier(
        Tier.EXPERT,
        [Message.user("hello")],
        trace=_trace(),
        usage=usage,
        tracker=tracker,
        budget=BudgetTracker(cfg),
        purpose="cloud_execute",
    )

    assert provider.max_tokens == [1234]


async def test_local_max_output_tokens_forwarded(config):
    cfg = config.model_copy(update={"cloud_allowed": True, "local_max_output_tokens": 777})
    provider = FakeProvider()
    sifter = ComputeSifter(cfg, provider=provider)
    usage, tracker = _call_usage()

    await sifter._call_tier(
        Tier.LOCAL,
        [Message.user("hello")],
        trace=_trace(),
        usage=usage,
        tracker=tracker,
        budget=BudgetTracker(cfg),
        purpose="local_execute",
    )

    assert provider.max_tokens == [777]


def test_fit_messages_is_deterministic_and_bounded():
    messages = [Message.system("system"), Message.user("x" * 400_000)]
    first = fit_messages(messages, max_tokens=1_000)
    second = fit_messages(messages, max_tokens=1_000)
    assert [m.content for m in first] == [m.content for m in second]
    assert sum(estimate_tokens(m.content) for m in first) <= 1_000


async def test_outbound_messages_fit_context_limit(config):
    cfg = config.model_copy(
        update={"cloud_allowed": True, "local_context_limit": 4_096, "local_max_output_tokens": 256}
    )
    provider = FakeProvider()
    sifter = ComputeSifter(cfg, provider=provider)
    usage, tracker = _call_usage()

    await sifter._call_tier(
        Tier.LOCAL,
        [Message.user("y" * 200_000)],
        trace=_trace(),
        usage=usage,
        tracker=tracker,
        budget=BudgetTracker(cfg),
        purpose="local_execute",
    )

    sent = sum(estimate_tokens(m.content) for m in provider.messages[0])
    assert sent <= 4_096


def test_cost_waiver_raises_the_cloud_ceiling(config):
    strict = config.model_copy(update={"max_cloud_cost_per_task": 0.0001})
    with pytest.raises(BudgetExceeded):
        BudgetTracker(strict).check(estimated_cost=0.25)

    waived = BudgetTracker(strict, cost_waiver=0.5)
    waived.check(estimated_cost=0.25)
    assert waived.budget.max_cost == 0.5


async def test_standing_rule_cost_waiver_allows_cloud(config):
    cfg = config.model_copy(update={"cloud_allowed": True, "max_cloud_cost_per_task": 0.000001})
    rules = StandingRuleEngine(
        rules=[StandingRule(text="Cloud spending under $0.50 per task does not require approval.")]
    )
    assert rules.cost_waiver_threshold() == 0.5
    provider = FakeProvider()
    sifter = ComputeSifter(cfg, provider=provider, rules=rules)

    result = await sifter.run("Summarize this file.", policy="cloud_only")

    assert provider.cloud_calls >= 1
    assert result.status == "completed"


async def test_budget_without_waiver_blocks_cloud(config):
    cfg = config.model_copy(update={"cloud_allowed": True, "max_cloud_cost_per_task": 0.000001})
    provider = FakeProvider()
    sifter = ComputeSifter(cfg, provider=provider)

    result = await sifter.run("Summarize this file.", policy="cloud_only")

    assert result.status == "blocked"
    assert provider.cloud_calls == 0


async def test_standing_rule_grant_skips_escalation_approval(config):
    cfg = config.model_copy(update={"cloud_allowed": True})
    rules = StandingRuleEngine(
        rules=[StandingRule(text="Cloud may debug failed tests automatically.")]
    )
    assert rules.has_grant("cloud_debug_failed_tests")
    provider = FakeProvider()
    approval = RecordingApprovalProvider()
    sifter = ComputeSifter(cfg, provider=provider, rules=rules, approval_provider=approval)
    trace = _trace()
    gate = DecisionGate(
        cfg,
        classifier=sifter.classifier,
        rules=rules,
        approval_provider=approval,
        trace=trace,
        history=None,
    )
    classification = DecisionClassifier().classify(
        "Replace the persistence architecture with a new design."
    )
    usage, tracker = _call_usage()

    answer, resolved = await sifter._escalate_after_failure(
        prompt="Replace the persistence architecture with a new design.",
        answer="local attempt failed",
        verification=VerificationResult(passed=False),
        failed_attempts=2,
        classification=classification,
        human_constraints=[],
        approved_decisions=[],
        gate=gate,
        trace=trace,
        usage=usage,
        tracker=tracker,
        budget=BudgetTracker(cfg),
        approvals=[],
        allowed=True,
        autonomy_grant=True,
    )

    assert approval.requests == []
    assert resolved is True
    assert provider.cloud_calls >= 1
    assert "Expert escalation" in answer
