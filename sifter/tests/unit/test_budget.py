from __future__ import annotations

import pytest

from aa_sifter.models.config import SifterConfig
from aa_sifter.models.provider import GenerationResult, Tier
from aa_sifter.routing.budget import BudgetExceeded, BudgetTracker


def test_budget_exceeds_calls(config: SifterConfig):
    tracker = BudgetTracker(config)
    tracker.budget.max_calls = 1
    tracker.check()
    tracker.record(
        GenerationResult(text="x", model="m", tier=Tier.EXPERT, input_tokens=10, output_tokens=10),
        config.cloud_model_config(),
    )
    with pytest.raises(BudgetExceeded) as exc:
        tracker.check()
    assert exc.value.kind == "max_cloud_calls"


def test_budget_exceeds_cost(config: SifterConfig):
    tracker = BudgetTracker(config, max_cost=0.000001)
    tracker.record(
        GenerationResult(
            text="x", model="m", tier=Tier.EXPERT, input_tokens=100_000, output_tokens=100_000
        ),
        config.cloud_model_config(),
    )
    with pytest.raises(BudgetExceeded) as exc:
        tracker.check()
    assert exc.value.kind == "max_cloud_cost"


def test_cost_waiver_threshold_applies():
    from aa_sifter.routing.budget import BudgetTracker as BT

    class _Cfg:
        max_cloud_calls_per_task = 5
        max_cloud_tokens_per_task = 1000
        max_cloud_cost_per_task = 10.0
        max_parallel_cloud_calls = 2

    tracker = BT(_Cfg())  # type: ignore[arg-type]
    assert tracker.cost_waiver(0.5) is True
    assert tracker.needs_increase_approval(0.4, waiver=0.5) is False


def test_remaining_values():
    from aa_sifter.routing.budget import Budget

    budget = Budget(max_calls=3, max_cost=1.0)
    budget.calls = 1
    budget.cost = 0.25
    assert (3 - budget.calls) == 2
    assert budget.would_exceed(estimated_cost=0.8) == "max_cloud_cost"


def test_budget_exceeds_tokens(config: SifterConfig):
    tracker = BudgetTracker(config)
    tracker.budget.max_tokens = 10
    tracker.record(
        GenerationResult(text="x", model="m", tier=Tier.EXPERT, input_tokens=100, output_tokens=0),
        config.cloud_model_config(),
    )
    with pytest.raises(BudgetExceeded) as exc:
        tracker.check()
    assert exc.value.kind == "max_cloud_tokens"


def test_budget_exhausted_before_first_call():
    from aa_sifter.routing.budget import Budget

    budget = Budget(max_calls=0, max_cost=1.0)
    assert budget.would_exceed() == "max_cloud_calls"


def test_budget_exhausted_midway_preserves_prior_usage(config: SifterConfig):
    tracker = BudgetTracker(config, max_cost=0.00002)
    tracker.check()
    tracker.record(
        GenerationResult(text="x", model="m", tier=Tier.EXPERT, input_tokens=100, output_tokens=50),
        config.cloud_model_config(),
    )
    assert tracker.budget.calls == 1
    with pytest.raises(BudgetExceeded) as exc:
        tracker.check()
    assert exc.value.kind == "max_cloud_cost"
    assert exc.value.used > 0


def test_parallel_limit_is_configured(config: SifterConfig):
    tracker = BudgetTracker(config)
    assert tracker.budget.max_parallel == config.max_parallel_cloud_calls


def test_remaining_helpers(config: SifterConfig):
    tracker = BudgetTracker(config)
    assert tracker.remaining_calls == config.max_cloud_calls_per_task
    assert tracker.remaining_cost == config.max_cloud_cost_per_task
    tracker.record(
        GenerationResult(text="x", model="m", tier=Tier.EXPERT, input_tokens=10, output_tokens=10),
        config.cloud_model_config(),
    )
    assert tracker.remaining_calls == config.max_cloud_calls_per_task - 1
    assert tracker.remaining_cost < config.max_cloud_cost_per_task
