"""Security tests: every cloud-bound request is redacted at one chokepoint.

The inference's promise is that no secret reaches a cloud provider. These tests
exercise the shared chokepoint in ``ComputeInference._call_tier`` / ``generate``
and the direct structured-call paths (preflight, decomposition).
"""

from __future__ import annotations

from io import StringIO

import pytest

from aa_inference.app import ComputeInference
from aa_inference.context.handoff import SecretRedactor
from aa_inference.metrics.trace import Trace
from aa_inference.metrics.usage import UsageMetrics, UsageTracker
from aa_inference.models.provider import Message, ProviderError, Tier
from aa_inference.routing.budget import BudgetTracker
from aa_inference.routing.preflight import PreflightAssessor
from tests.fixtures.providers import FakeProvider

SECRET = "password = hunter2secret"


def _contents(provider: FakeProvider) -> str:
    return "\n".join(message.content for call in provider.messages for message in call)


def _trace() -> Trace:
    return Trace("redaction-test", debug=False, sink=StringIO())


async def test_expert_call_tier_redacts_secret(config):
    cfg = config.model_copy(update={"cloud_allowed": True, "redact_secrets": True})
    provider = FakeProvider()
    inference = ComputeInference(cfg, provider=provider)
    trace = _trace()

    await inference._call_tier(
        Tier.EXPERT,
        [Message.user(SECRET)],
        trace=trace,
        usage=UsageMetrics(),
        tracker=UsageTracker(),
        budget=BudgetTracker(cfg),
        purpose="cloud_execute",
    )

    sent = _contents(provider)
    assert "hunter2secret" not in sent
    assert "[REDACTED:" in sent
    assert any(event["component"] == "security" for event in trace.events)


async def test_expert_call_tier_redacts_context_appended_to_message(config):
    cfg = config.model_copy(update={"cloud_allowed": True, "redact_secrets": True})
    provider = FakeProvider()
    inference = ComputeInference(cfg, provider=provider)

    message = Message.user("Summarize this config.\n" + "API_KEY=sk-" + "a" * 24)
    await inference._call_tier(
        Tier.EXPERT,
        [message],
        trace=_trace(),
        usage=UsageMetrics(),
        tracker=UsageTracker(),
        budget=BudgetTracker(cfg),
        purpose="cloud_execute",
    )

    sent = _contents(provider)
    assert "sk-aaaaaaaaaaaaaaaaaaaaaaaa" not in sent


async def test_generate_expert_redacts_secret(config):
    cfg = config.model_copy(update={"cloud_allowed": True, "redact_secrets": True})
    provider = FakeProvider()
    inference = ComputeInference(cfg, provider=provider)

    await inference.generate([Message.user(SECRET)], tier="expert")

    assert "hunter2secret" not in _contents(provider)


async def test_generate_expert_blocked_when_cloud_disabled(config):
    cfg = config.model_copy(update={"cloud_allowed": False, "redact_secrets": True})
    provider = FakeProvider()
    inference = ComputeInference(cfg, provider=provider)

    with pytest.raises(ProviderError):
        await inference.generate([Message.user("do something")], tier="expert")
    assert provider.cloud_calls == 0


async def test_local_call_is_not_redacted(config):
    cfg = config.model_copy(update={"cloud_allowed": True, "redact_secrets": True})
    provider = FakeProvider()
    inference = ComputeInference(cfg, provider=provider)

    await inference._call_tier(
        Tier.LOCAL,
        [Message.user(SECRET)],
        trace=_trace(),
        usage=UsageMetrics(),
        tracker=UsageTracker(),
        budget=BudgetTracker(cfg),
        purpose="local_execute",
    )

    assert "hunter2secret" in _contents(provider)


async def test_preflight_model_assessment_redacts_secret(config):
    provider = FakeProvider(structured={"decision_level": "routine"})
    assessor = PreflightAssessor(config.model_copy(update={"redact_secrets": True}))

    await assessor.assess_with_model(provider, config.local_model_config(), SECRET)

    assert "hunter2secret" not in _contents(provider)


async def test_decomposition_redacts_secret(config):
    from aa_inference.tasks.decomposition import Decomposer

    provider = FakeProvider(structured={"tasks": [{"task": "do it"}]})
    decomposer = Decomposer()

    await decomposer.decompose(SECRET, provider, config.local_model_config())

    assert "hunter2secret" not in _contents(provider)


def test_redactor_disabled_leaves_text_untouched():
    redactor = SecretRedactor(enabled=False)
    result = redactor.redact(SECRET)
    assert result.text == SECRET
