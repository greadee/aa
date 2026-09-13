from __future__ import annotations

import pytest

from aa_sifter.decisions.approval import (
    ApprovalAction,
    ApprovalOption,
    ApprovalRequest,
    AutoApprovalProvider,
    ConsoleApprovalProvider,
    parse_decision_input,
    render_request,
)
from aa_sifter.decisions.classifier import DecisionLevel

_OPTIONS = [
    ApprovalOption(key="A", label="Keep current architecture"),
    ApprovalOption(key="B", label="Allow DeepSeek to analyze these options"),
    ApprovalOption(key="C", label="Give different instructions"),
    ApprovalOption(key="D", label="Stop this work"),
]


def _request(category=DecisionLevel.MAJOR) -> ApprovalRequest:
    return ApprovalRequest(
        category=category,
        issue="Persistence architecture may need to change",
        why_it_matters="Affects scheduler, memory and history.",
        current_approach="SQLite with direct repository access.",
        options=_OPTIONS,
        recommended_action="Ask DeepSeek to compare A/B/C.",
    )


def test_render_makes_cloud_status_explicit():
    text = render_request(_request())
    assert "MAJOR DECISION REQUIRED" in text
    assert "Cloud reasoning has NOT been invoked yet." in text
    assert "[D] Stop this work" in text


def test_parse_option_b_allows_expert_analysis():
    decision = parse_decision_input("B", _OPTIONS)
    assert decision.action == ApprovalAction.ALLOW_EXPERT_ANALYSIS
    assert decision.authorized


def test_parse_stop_is_not_authorized():
    decision = parse_decision_input("d", _OPTIONS)
    assert decision.action == ApprovalAction.STOP
    assert not decision.authorized


def test_parse_constraints():
    decision = parse_decision_input("constraint: Do not replace SQLite; Keep it fast", _OPTIONS)
    assert decision.action == ApprovalAction.APPROVE
    assert "Do not replace SQLite" in decision.constraints


@pytest.mark.asyncio
async def test_auto_provider_defaults_safe_for_major():
    provider = AutoApprovalProvider()
    decision = await provider.request(_request(DecisionLevel.MAJOR))
    assert decision.action == ApprovalAction.STOP
    assert not decision.authorized


@pytest.mark.asyncio
async def test_auto_provider_never_auto_approves_critical():
    provider = AutoApprovalProvider(
        major=None,
    )
    decision = await provider.request(_request(DecisionLevel.CRITICAL))
    assert not decision.authorized


@pytest.mark.asyncio
async def test_console_provider_uses_input():
    provider = ConsoleApprovalProvider(
        input_fn=lambda _: "B",
        output_fn=lambda _: None,
    )
    decision = await provider.request(_request())
    assert decision.action == ApprovalAction.ALLOW_EXPERT_ANALYSIS


@pytest.mark.asyncio
async def test_critical_override_still_requires_explicit_choice():
    provider = AutoApprovalProvider()
    decision = await provider.request(_request(DecisionLevel.CRITICAL))
    assert decision.action == ApprovalAction.STOP
