from __future__ import annotations

from aa_sifter.decisions.policy import (
    StandingRule,
    StandingRuleEngine,
    derive_rule_metadata,
)


def test_derive_metadata_from_approved_technology():
    metadata = derive_rule_metadata("SQLite is the approved persistence technology.")
    assert "sqlite" in metadata["forbid_terms"]


def test_derive_metadata_grants():
    metadata = derive_rule_metadata("DeepSeek may debug failed tests automatically.")
    assert "cloud_debug_failed_tests" in metadata["grants"]


def test_conflict_when_proposal_uses_alternative():
    engine = StandingRuleEngine(
        rules=[StandingRule(text="SQLite is the approved persistence technology.")]
    )
    assert engine.detect_conflicts("Migrate to PostgreSQL.")
    assert engine.detect_conflicts("Use SQLite for storage.") == []


def test_auth_conflict_detected():
    engine = StandingRuleEngine(
        rules=[StandingRule(text="Never change authentication architecture without asking.")]
    )
    assert engine.detect_conflicts("Replace the auth provider with OAuth.")


def test_cost_waiver_threshold_parsed():
    engine = StandingRuleEngine(
        rules=[StandingRule(text="Cloud spending under $0.50 per task does not require approval.")]
    )
    assert engine.cost_waiver_threshold() == 0.5


def test_has_grant():
    engine = StandingRuleEngine(
        rules=[StandingRule(text="DeepSeek may debug failed tests automatically.")]
    )
    assert engine.has_grant("cloud_debug_failed_tests")
    assert not engine.has_grant("cloud_auto_escalation")


def test_add_rule_persists_and_deactivates(store):
    engine = StandingRuleEngine(store=store)
    rule = engine.add_rule("Do not introduce another frontend framework.")
    assert rule.id is not None
    assert engine.rules

    engine2 = StandingRuleEngine(store=store)
    assert engine2.rules
    assert store.deactivate_standing_rule(rule.id) is True
    engine2.reload()
    assert engine2.rules == []
