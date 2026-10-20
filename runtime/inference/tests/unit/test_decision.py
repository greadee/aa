from __future__ import annotations

import pytest

from aa_inference.contracts import ContractError
from aa_inference.decision import RoutingBlockedError, parse_routing_decision
from aa_inference.models.provider import Tier


def test_parses_local_and_cloud_decisions():
    local = parse_routing_decision({"locality": "local", "tier": "local"})
    assert local.provider_tier == Tier.LOCAL
    cloud = parse_routing_decision({"locality": "cloud", "tier": "expert", "estimatedCostUsd": 0.2})
    assert cloud.provider_tier == Tier.EXPERT
    assert cloud.estimated_cost_usd == 0.2


@pytest.mark.parametrize(
    "payload",
    [
        None,
        {},
        {"locality": "local", "tier": "expert"},
        {"locality": "cloud", "tier": "local"},
        {"locality": "orbital", "tier": "local"},
        {"locality": "local", "tier": "local", "decisionLevel": "nonsense"},
    ],
)
def test_missing_or_skewed_decisions_fail_closed(payload):
    with pytest.raises(ContractError):
        parse_routing_decision(payload)


def test_blocked_decision_keeps_reason():
    decision = parse_routing_decision(
        {"locality": "cloud", "tier": "expert", "blocked": True, "reason": "budget"}
    )
    assert decision.blocked
    assert RoutingBlockedError(decision).decision.reason == "budget"
