from __future__ import annotations

from aa_sifter.models.config import SifterConfig
from aa_sifter.routing.escalation import (
    EscalationAction,
    EscalationPolicy,
    EscalationRequest,
)


def _policy(config: SifterConfig) -> EscalationPolicy:
    return EscalationPolicy(config)


def test_routine_debugging_escalates_without_human(config: SifterConfig):
    outcome = _policy(config).evaluate(
        EscalationRequest(
            reason="local cannot fix a type error",
            failed_attempts=2,
            recommended_action=EscalationAction.CLOUD_DEBUG,
        )
    )
    assert outcome.escalate is True
    assert outcome.requires_human_approval is False
    assert outcome.action == EscalationAction.CLOUD_DEBUG


def test_architectural_recommendation_requires_human(config: SifterConfig):
    outcome = _policy(config).evaluate(
        EscalationRequest(
            reason="the persistence layer must be replaced",
            potential_major_decision=True,
            recommended_action=EscalationAction.CLOUD_PLAN,
        )
    )
    assert outcome.escalate is True
    assert outcome.requires_human_approval is True
    assert outcome.action == EscalationAction.USER_DECISION


def test_should_escalate_after_retry_budget(config: SifterConfig):
    policy = _policy(config)
    assert policy.should_escalate(
        failed_attempts=config.local_max_retries + 1, verification_passed=False
    )
    assert policy.should_escalate(failed_attempts=0, verification_passed=False)
    assert not policy.should_escalate(failed_attempts=0, verification_passed=True)


def test_architecture_like_detection(config: SifterConfig):
    policy = _policy(config)
    assert policy.is_architecture_like("Replace the persistence layer with a new design")
    assert policy.is_architecture_like("Change the database")
    assert not policy.is_architecture_like("Fix a typo in the parser")
