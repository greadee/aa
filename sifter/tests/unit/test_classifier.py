from __future__ import annotations

from aa_sifter.decisions.classifier import DecisionClassifier, DecisionLevel


def test_easy_testing_task_is_routine():
    classifier = DecisionClassifier()
    result = classifier.classify("Write unit tests for this small utility function.")
    assert result.level == DecisionLevel.ROUTINE
    assert result.requires_human_approval is False


def test_debugging_task_is_not_major():
    classifier = DecisionClassifier()
    result = classifier.classify(
        "Find why these existing integration tests fail after the latest change."
    )
    assert result.level in {DecisionLevel.ROUTINE, DecisionLevel.SIGNIFICANT}


def test_oauth_replacement_is_major():
    classifier = DecisionClassifier()
    result = classifier.classify("Replace authentication with OAuth while preserving old sessions.")
    assert result.level == DecisionLevel.MAJOR
    assert result.requires_human_approval is True
    assert result.signals.auth_change is True


def test_persistence_swap_is_major():
    classifier = DecisionClassifier()
    result = classifier.classify("Replace SQLite with PostgreSQL for the whole project.")
    assert result.level == DecisionLevel.MAJOR
    assert result.signals.database_change or result.signals.persistence_change


def test_destructive_operation_is_critical():
    classifier = DecisionClassifier()
    result = classifier.classify("Drop table users and delete from the production database.")
    assert result.level == DecisionLevel.CRITICAL
    assert result.signals.destructive is True


def test_model_hint_can_raise_but_not_lower():
    classifier = DecisionClassifier()
    raised = classifier.classify("Add a small helper.", model_level=DecisionLevel.MAJOR)
    assert raised.level == DecisionLevel.MAJOR
    lowered = classifier.classify(
        "Replace the persistence architecture.", model_level=DecisionLevel.ROUTINE
    )
    assert lowered.level == DecisionLevel.MAJOR
    assert lowered.model_assisted is True
