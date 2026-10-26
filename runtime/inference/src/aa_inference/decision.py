"""The allocator's routing decision, consumed by inference execution.

Inference does not decide locality, tier, or budget. The control-plane
allocator produces a ``RoutingDecision`` and passes it over the RPC boundary;
this module validates it and maps it onto the provider tier. Missing or skewed
decisions fail closed.
"""

from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, ValidationError

from .contracts import ContractError
from .models.provider import Tier

_DECISION_LEVELS = frozenset({"routine", "significant", "major", "critical"})


class RoutingBlockedError(RuntimeError):
    """The allocator's decision blocks execution."""

    def __init__(self, decision: RoutingDecision):
        super().__init__(f"routing decision blocked execution: {decision.reason or 'blocked'}")
        self.decision = decision


class RoutingDecision(BaseModel):
    """The allocator's routing/budget outcome."""

    model_config = ConfigDict(populate_by_name=True)

    locality: Literal["local", "cloud"]
    tier: Literal["local", "expert"]
    blocked: bool = False
    reason: str = ""
    estimated_cost_usd: float | None = Field(default=None, alias="estimatedCostUsd")
    decision_level: str = Field(default="routine", alias="decisionLevel")
    requires_human_approval: bool = Field(default=False, alias="requiresHumanApproval")

    @property
    def provider_tier(self) -> Tier:
        return Tier.EXPERT if self.tier == "expert" else Tier.LOCAL

    def validate_consistency(self) -> None:
        if self.locality == "local" and self.tier != "local":
            raise ContractError("routingDecision: local locality requires the local tier")
        if self.locality == "cloud" and self.tier != "expert":
            raise ContractError("routingDecision: cloud locality requires the expert tier")
        if self.estimated_cost_usd is not None and self.estimated_cost_usd < 0:
            raise ContractError("routingDecision: estimatedCostUsd must be >= 0")
        if self.decision_level not in _DECISION_LEVELS:
            raise ContractError(f"routingDecision: unknown decisionLevel {self.decision_level!r}")


def parse_routing_decision(payload: Any) -> RoutingDecision:
    """Validate a decision from the RPC boundary. Absent or skewed input fails closed."""
    if not isinstance(payload, dict) or not payload:
        raise ContractError("routingDecision: is required")
    try:
        decision = RoutingDecision.model_validate(payload)
    except ValidationError as exc:
        detail = exc.errors()[0]["msg"] if exc.errors() else "invalid"
        raise ContractError(f"routingDecision: {detail}") from exc
    decision.validate_consistency()
    return decision
