"""Shipped reference profile built on the shared model defaults."""

from __future__ import annotations

from ..model_defaults import (
    DEFAULT_EXPERT_API_KEY_ENV,
    DEFAULT_EXPERT_CONTEXT,
    DEFAULT_EXPERT_ENDPOINT,
    DEFAULT_EXPERT_INPUT_COST,
    DEFAULT_EXPERT_MAX_OUTPUT,
    DEFAULT_EXPERT_MODEL,
    DEFAULT_EXPERT_OUTPUT_COST,
    DEFAULT_EXPERT_PROVIDER,
    DEFAULT_LOCAL_CONTEXT,
    DEFAULT_LOCAL_MAX_OUTPUT,
    DEFAULT_LOCAL_MODEL,
    DEFAULT_LOCAL_PROVIDER,
    DEFAULT_ROUTING_POLICY,
    DEFAULT_TEMPERATURE,
)
from .schema import (
    ApplicationConfig,
    ApprovalConfig,
    BudgetConfig,
    ComputeProfile,
    ModelTierConfig,
    UserPreferenceConfig,
)


def default_local_tier() -> ModelTierConfig:
    return ModelTierConfig(
        provider=DEFAULT_LOCAL_PROVIDER,
        model=DEFAULT_LOCAL_MODEL,
        context_limit=DEFAULT_LOCAL_CONTEXT,
        max_output_tokens=DEFAULT_LOCAL_MAX_OUTPUT,
        temperature=DEFAULT_TEMPERATURE,
    )


def default_expert_tier() -> ModelTierConfig:
    return ModelTierConfig(
        provider=DEFAULT_EXPERT_PROVIDER,
        model=DEFAULT_EXPERT_MODEL,
        endpoint=DEFAULT_EXPERT_ENDPOINT,
        api_key_env=DEFAULT_EXPERT_API_KEY_ENV,
        context_limit=DEFAULT_EXPERT_CONTEXT,
        max_output_tokens=DEFAULT_EXPERT_MAX_OUTPUT,
        temperature=DEFAULT_TEMPERATURE,
        input_cost_per_mtok=DEFAULT_EXPERT_INPUT_COST,
        output_cost_per_mtok=DEFAULT_EXPERT_OUTPUT_COST,
    )


def default_profile(name: str = "default") -> ComputeProfile:
    return ComputeProfile(
        name=name,
        local=default_local_tier(),
        expert=default_expert_tier(),
        routing_policy=DEFAULT_ROUTING_POLICY,
        budget=BudgetConfig(),
        approval=ApprovalConfig(),
        preferences=UserPreferenceConfig(
            local_first=True,
            cost_sensitive=True,
            privacy_sensitive=True,
            cloud_allowed=True,
        ),
    )


def default_application_config() -> ApplicationConfig:
    return ApplicationConfig(
        active_profile="default",
        profiles={"default": default_profile()},
    )
