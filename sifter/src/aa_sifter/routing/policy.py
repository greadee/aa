from __future__ import annotations

from enum import StrEnum

from pydantic import BaseModel, Field

from ..decisions.classifier import DecisionLevel
from ..metrics.trace import Trace
from ..models.config import SifterConfig
from ..models.provider import Tier
from .preflight import PreflightAssessment


class RouteKind(StrEnum):
    LOCAL = "local"
    HYBRID = "hybrid"
    CLOUD = "cloud"


class RoutingPolicy(StrEnum):
    LOCAL_ONLY = "local_only"
    CLOUD_ONLY = "cloud_only"
    LOCAL_FIRST = "local_first"
    EXPERT_FIRST = "expert_first"
    ADAPTIVE = "adaptive"
    BUDGET_CONSTRAINED = "budget_constrained"


class RouteDecision(BaseModel):
    kind: RouteKind
    planner_tier: Tier | None = None
    executor_tier: Tier = Tier.LOCAL
    review_tier: Tier | None = None
    reasons: list[str] = Field(default_factory=list)
    forced_cloud: bool = False

    @property
    def uses_cloud(self) -> bool:
        return (
            self.executor_tier == Tier.EXPERT
            or self.planner_tier == Tier.EXPERT
            or self.review_tier == Tier.EXPERT
        )


class PolicyEngine:
    """Deterministic routing policy. Not the model's decision."""

    def __init__(self, config: SifterConfig, *, trace: Trace | None = None):
        self.config = config
        self.trace = trace

    def decide(
        self,
        assessment: PreflightAssessment,
        *,
        policy: str | None = None,
        failed_attempts: int = 0,
        cloud_budget_available: bool = True,
    ) -> RouteDecision:
        selected = self._coerce(policy or self.config.routing_policy)
        local_viable = (
            assessment.local_success_probability >= self.config.local_confidence_threshold
            and assessment.context_requirement <= self.config.local_context_limit
        )
        if selected == RoutingPolicy.LOCAL_ONLY:
            decision = RouteDecision(
                kind=RouteKind.LOCAL,
                executor_tier=Tier.LOCAL,
                reasons=["policy local_only"],
            )
        elif selected == RoutingPolicy.CLOUD_ONLY:
            decision = RouteDecision(
                kind=RouteKind.CLOUD,
                planner_tier=Tier.EXPERT,
                executor_tier=Tier.EXPERT,
                review_tier=Tier.EXPERT,
                reasons=["policy cloud_only"],
                forced_cloud=True,
            )
        elif selected == RoutingPolicy.EXPERT_FIRST:
            decision = RouteDecision(
                kind=RouteKind.CLOUD,
                planner_tier=Tier.EXPERT,
                executor_tier=Tier.EXPERT,
                reasons=["policy expert_first"],
                forced_cloud=True,
            )
        elif selected == RoutingPolicy.BUDGET_CONSTRAINED:
            if not cloud_budget_available or local_viable:
                decision = RouteDecision(
                    kind=RouteKind.LOCAL,
                    executor_tier=Tier.LOCAL,
                    reasons=["policy budget_constrained; local preferred"],
                )
            else:
                decision = RouteDecision(
                    kind=RouteKind.HYBRID,
                    planner_tier=Tier.EXPERT,
                    executor_tier=Tier.LOCAL,
                    reasons=["policy budget_constrained; cloud plan only"],
                )
        elif selected == RoutingPolicy.LOCAL_FIRST:
            if local_viable and failed_attempts == 0:
                decision = RouteDecision(
                    kind=RouteKind.LOCAL,
                    executor_tier=Tier.LOCAL,
                    reasons=["policy local_first; local viable"],
                )
            else:
                decision = RouteDecision(
                    kind=RouteKind.CLOUD,
                    planner_tier=Tier.EXPERT,
                    executor_tier=Tier.EXPERT,
                    reasons=["policy local_first; escalating to expert"],
                    forced_cloud=True,
                )
        else:
            decision = self._adaptive(
                assessment,
                local_viable=local_viable,
                failed_attempts=failed_attempts,
                cloud_budget_available=cloud_budget_available,
            )
        if (
            not cloud_budget_available
            and decision.uses_cloud
            and selected
            in {
                RoutingPolicy.ADAPTIVE,
                RoutingPolicy.LOCAL_FIRST,
                RoutingPolicy.BUDGET_CONSTRAINED,
            }
        ):
            decision = RouteDecision(
                kind=RouteKind.LOCAL,
                executor_tier=Tier.LOCAL,
                reasons=["cloud budget unavailable; local fallback"],
            )
        if self.trace is not None:
            self.trace.log(
                "router",
                f"route={decision.kind.value}",
                policy=selected.value,
                local_success_probability=assessment.local_success_probability,
                executor=decision.executor_tier.value,
                reasons=decision.reasons,
            )
        return decision

    def _adaptive(
        self,
        assessment: PreflightAssessment,
        *,
        local_viable: bool,
        failed_attempts: int,
        cloud_budget_available: bool,
    ) -> RouteDecision:
        high_stakes = assessment.decision_level in {DecisionLevel.MAJOR, DecisionLevel.CRITICAL}
        if failed_attempts > 0:
            if assessment.architecture_impact and high_stakes:
                return RouteDecision(
                    kind=RouteKind.CLOUD,
                    planner_tier=Tier.EXPERT,
                    executor_tier=Tier.EXPERT,
                    review_tier=Tier.EXPERT,
                    reasons=["local failed and change is decision-significant"],
                    forced_cloud=True,
                )
            return RouteDecision(
                kind=RouteKind.CLOUD,
                planner_tier=Tier.EXPERT,
                executor_tier=Tier.EXPERT,
                reasons=[f"{failed_attempts} local failure(s); expert escalation"],
                forced_cloud=True,
            )
        if local_viable and not assessment.requires_cloud:
            return RouteDecision(
                kind=RouteKind.LOCAL,
                executor_tier=Tier.LOCAL,
                reasons=["adaptive: task within local capability"],
            )
        if assessment.reasoning_difficulty >= 0.65 or not local_viable:
            return RouteDecision(
                kind=RouteKind.HYBRID,
                planner_tier=Tier.EXPERT,
                executor_tier=Tier.LOCAL,
                review_tier=Tier.EXPERT,
                reasons=["adaptive: cloud planning/review with local execution"],
                forced_cloud=True,
            )
        return RouteDecision(
            kind=RouteKind.HYBRID,
            planner_tier=Tier.EXPERT,
            executor_tier=Tier.LOCAL,
            review_tier=Tier.EXPERT,
            reasons=["adaptive: decision requires expert planning, local execution"],
            forced_cloud=True,
        )

    @staticmethod
    def _coerce(value: str) -> RoutingPolicy:
        try:
            return RoutingPolicy(value)
        except ValueError:
            return RoutingPolicy.ADAPTIVE
