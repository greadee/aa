from __future__ import annotations

import pytest

from aa_sifter.decisions.classifier import DecisionLevel
from aa_sifter.models.config import SifterConfig
from aa_sifter.routing.preflight import PreflightAssessor
from tests.fixtures.providers import FakeProvider, MalformedProvider


def test_easy_task_assessment_is_local(config: SifterConfig):
    assessor = PreflightAssessor(config)
    assessment = assessor.assess("Write unit tests for a small utility function.")

    assert assessment.task_type == "testing"
    assert assessment.decision_level == DecisionLevel.ROUTINE
    assert assessment.requires_human_approval is False
    assert assessment.local_success_probability >= config.local_confidence_threshold
    assert assessment.requires_cloud is False
    assert assessment.recommended_route == "local"


def test_large_context_requires_cloud(config: SifterConfig):
    assessor = PreflightAssessor(config)
    large_context = "x" * (config.local_context_limit * 4 + 100)
    assessment = assessor.assess("Summarize this.", large_context)

    assert assessment.context_requirement > config.local_context_limit
    assert assessment.requires_cloud is True
    assert assessment.recommended_route == "cloud"


def test_prior_failures_lower_local_success_probability(config: SifterConfig):
    assessor = PreflightAssessor(config)
    baseline = assessor.assess("Fix the failing tests.")
    degraded = assessor.assess("Fix the failing tests.", failed_attempts=3)
    assert degraded.local_success_probability < baseline.local_success_probability


def test_major_task_is_flagged(config: SifterConfig):
    assessor = PreflightAssessor(config)
    assessment = assessor.assess("Replace authentication with OAuth.")
    assert assessment.decision_level == DecisionLevel.MAJOR
    assert assessment.architecture_impact is True or assessment.security_sensitive is True


@pytest.mark.asyncio
async def test_model_assessment_can_raise_level(config: SifterConfig):
    assessor = PreflightAssessor(config)
    provider = FakeProvider(
        structured={
            "task_type": "implementation",
            "complexity": 0.9,
            "local_success_probability": 0.2,
            "decision_level": "major",
            "reason": "touches multiple subsystems",
        }
    )
    base = assessor.assess("Add a helper function.")
    merged = await assessor.assess_with_model(
        provider, config.local_model_config(), "Add a helper function.", base=base
    )

    assert merged.model_assisted is True
    assert merged.decision_level == DecisionLevel.MAJOR
    assert merged.local_success_probability <= base.local_success_probability


@pytest.mark.asyncio
async def test_malformed_model_assessment_falls_back(config: SifterConfig):
    assessor = PreflightAssessor(config)
    provider = MalformedProvider()
    base = assessor.assess("Add a helper function.")
    merged = await assessor.assess_with_model(
        provider, config.local_model_config(), "Add a helper function.", base=base
    )
    assert merged == base
