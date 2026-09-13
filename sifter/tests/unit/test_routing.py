from __future__ import annotations

from aa_sifter.models.config import SifterConfig
from aa_sifter.models.provider import Tier
from aa_sifter.routing.policy import PolicyEngine, RouteKind
from aa_sifter.routing.preflight import PreflightAssessment


def _assessment(**overrides) -> PreflightAssessment:
    base = PreflightAssessment(
        complexity=0.2,
        local_success_probability=0.9,
        reasoning_difficulty=0.2,
        context_requirement=2_000,
        requires_cloud=False,
    )
    return base.model_copy(update=overrides)


def test_local_only_never_uses_cloud(config: SifterConfig):
    engine = PolicyEngine(config)
    decision = engine.decide(_assessment(requires_cloud=True), policy="local_only")
    assert decision.kind == RouteKind.LOCAL
    assert decision.uses_cloud is False


def test_cloud_only_forces_expert(config: SifterConfig):
    engine = PolicyEngine(config)
    decision = engine.decide(_assessment(), policy="cloud_only")
    assert decision.kind == RouteKind.CLOUD
    assert decision.executor_tier == Tier.EXPERT
    assert decision.forced_cloud is True


def test_adaptive_prefers_local_for_easy_task(config: SifterConfig):
    engine = PolicyEngine(config)
    decision = engine.decide(_assessment(), policy="adaptive")
    assert decision.kind == RouteKind.LOCAL


def test_adaptive_escalates_after_failure(config: SifterConfig):
    engine = PolicyEngine(config)
    decision = engine.decide(_assessment(), policy="adaptive", failed_attempts=1)
    assert decision.kind == RouteKind.CLOUD
    assert decision.uses_cloud


def test_adaptive_hybrid_when_context_pressure(config: SifterConfig):
    engine = PolicyEngine(config)
    decision = engine.decide(
        _assessment(local_success_probability=0.5, requires_cloud=True), policy="adaptive"
    )
    assert decision.kind == RouteKind.HYBRID
    assert decision.planner_tier == Tier.EXPERT
    assert decision.executor_tier == Tier.LOCAL


def test_budget_constrained_falls_back_to_local(config: SifterConfig):
    engine = PolicyEngine(config)
    decision = engine.decide(
        _assessment(requires_cloud=True), policy="budget_constrained", cloud_budget_available=False
    )
    assert decision.kind == RouteKind.LOCAL


def test_cloud_budget_unavailable_falls_back_under_adaptive(config: SifterConfig):
    engine = PolicyEngine(config)
    decision = engine.decide(
        _assessment(requires_cloud=True, local_success_probability=0.5),
        policy="adaptive",
        cloud_budget_available=False,
    )
    assert decision.kind == RouteKind.LOCAL


def test_explicit_cloud_policy_is_not_silently_downgraded(config: SifterConfig):
    engine = PolicyEngine(config)
    decision = engine.decide(
        _assessment(requires_cloud=True),
        policy="expert_first",
        cloud_budget_available=False,
    )
    assert decision.kind == RouteKind.CLOUD
    assert decision.uses_cloud is True


def test_unknown_policy_falls_back_to_adaptive(config: SifterConfig):
    engine = PolicyEngine(config)
    decision = engine.decide(_assessment(), policy="nonsense")
    assert decision.kind == RouteKind.LOCAL
