from __future__ import annotations

import os
from collections.abc import Mapping

from pydantic import BaseModel, Field

from ..catalog.catalog import ModelCatalog
from .schema import ComputeProfile, HardwareProfile

KNOWN_PROVIDERS = {"ollama", "deepseek", "openai-compatible"}
KEYED_PROVIDERS = {"deepseek", "openai-compatible"}
KNOWN_POLICIES = {
    "local_only",
    "cloud_only",
    "local_first",
    "expert_first",
    "adaptive",
    "budget_constrained",
}


class ValidationResult(BaseModel):
    errors: list[str] = Field(default_factory=list)
    warnings: list[str] = Field(default_factory=list)

    @property
    def ok(self) -> bool:
        return not self.errors


def validate_profile(
    profile: ComputeProfile,
    *,
    catalog: ModelCatalog | None = None,
    hardware: HardwareProfile | None = None,
    environ: Mapping[str, str] | None = None,
    require_credentials: bool = True,
) -> ValidationResult:
    env = environ if environ is not None else os.environ
    catalog = catalog or ModelCatalog()
    result = ValidationResult()

    if profile.routing_policy not in KNOWN_POLICIES:
        result.errors.append(
            f"unknown routing policy '{profile.routing_policy}'; "
            f"expected one of {sorted(KNOWN_POLICIES)}"
        )

    if profile.local is None or not profile.local.enabled:
        result.errors.append("no local model tier is configured")
    else:
        _check_tier(
            profile.local,
            label="local",
            catalog=catalog,
            hardware=hardware,
            env=env,
            result=result,
            require_credentials=require_credentials,
        )

    cloud_allowed = profile.preferences.cloud_allowed and (
        profile.goal is None or profile.goal.cloud_allowed
    )
    if cloud_allowed and profile.expert is not None and profile.expert.enabled:
        _check_tier(
            profile.expert,
            label="expert",
            catalog=catalog,
            hardware=hardware,
            env=env,
            result=result,
            require_credentials=require_credentials,
        )

    if profile.budget.max_cloud_cost_per_task < 0:
        result.errors.append("max_cloud_cost_per_task cannot be negative")
    if profile.budget.max_cloud_calls_per_task < 0:
        result.errors.append("max_cloud_calls_per_task cannot be negative")
    if profile.budget.max_cloud_tokens_per_task < 0:
        result.errors.append("max_cloud_tokens_per_task cannot be negative")
    return result


def _check_tier(
    model_tier,
    *,
    label: str,
    catalog: ModelCatalog,
    hardware: HardwareProfile | None,
    env: Mapping[str, str],
    result: ValidationResult,
    require_credentials: bool = True,
) -> None:
    if not model_tier.provider:
        result.errors.append(f"{label} provider is missing")
    if not model_tier.model:
        result.errors.append(f"{label} model is missing")
    if model_tier.provider and model_tier.provider not in KNOWN_PROVIDERS:
        result.warnings.append(
            f"{label} provider '{model_tier.provider}' is not a built-in provider; "
            "ensure it is registered at runtime"
        )
    if model_tier.context_limit is not None and model_tier.context_limit <= 0:
        result.errors.append(f"{label} context_limit must be positive")

    if model_tier.provider in KEYED_PROVIDERS:
        env_name = model_tier.api_key_env
        if not env_name:
            result.errors.append(
                f"{label} provider '{model_tier.provider}' requires api_key_env to be set"
            )
        elif not env.get(env_name):
            message = f"{env_name} is required for the {label} provider but is not configured"
            if require_credentials:
                result.errors.append(message)
            else:
                result.warnings.append(message)

    metadata = catalog.get(model_tier.provider, model_tier.model)
    if metadata is None:
        result.warnings.append(
            f"{label} model '{model_tier.provider}/{model_tier.model}' has no catalog metadata "
            "(configuration is still allowed)"
        )
        return

    if (
        hardware is not None
        and metadata.minimum_vram_gb is not None
        and hardware.gpu_vram_gb is not None
        and metadata.minimum_vram_gb > hardware.gpu_vram_gb
    ):
        result.warnings.append(
            f"model '{metadata.display_name}' needs ~{metadata.minimum_vram_gb} GB VRAM but "
            f"only {hardware.gpu_vram_gb} GB is available; expect CPU offload"
        )
    if (
        hardware is not None
        and metadata.minimum_ram_gb is not None
        and hardware.system_ram_gb is not None
        and metadata.minimum_ram_gb > hardware.system_ram_gb
    ):
        result.warnings.append(
            f"model '{metadata.display_name}' wants ~{metadata.minimum_ram_gb} GB RAM but "
            f"{hardware.system_ram_gb} GB is available"
        )
