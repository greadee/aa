from __future__ import annotations

import os
from pathlib import Path
from typing import Any

from ..catalog.catalog import ModelCatalog
from ..models.config import SifterConfig
from .schema import ApplicationConfig, ComputeProfile, ModelTierConfig
from .store import default_config_path, load_application_config

# Precedence (highest first): CLI override > active profile > environment > defaults.
PRECEDENCE = "cli_override > active_profile > environment > defaults"


def resolve_config(
    app: ApplicationConfig,
    *,
    profile_name: str | None = None,
    overrides: dict[str, Any] | None = None,
    dotenv_path: str | Path | None = None,
    catalog: ModelCatalog | None = None,
    apply_profile: bool = True,
) -> tuple[SifterConfig, list[str]]:
    """Resolve an application config into the runtime SifterConfig."""
    base = SifterConfig.from_env(dotenv_path=dotenv_path)
    warnings: list[str] = []
    name = profile_name or app.active_profile
    profile = app.profiles.get(name) if apply_profile else None
    if profile is not None:
        base = apply_profile_to_config(base, profile, catalog=catalog, warnings=warnings)
        _warn_legacy_env(profile, warnings)
    if overrides:
        base = base.model_copy(update={k: v for k, v in overrides.items() if v is not None})
    base = base.model_copy(update={"config_profile": name if profile is not None else None})
    if not base.cloud_allowed:
        # Local-only profiles must never route to a paid cloud provider.
        base = base.model_copy(update={"routing_policy": _local_only_policy(base.routing_policy)})
    return base, warnings


def load_runtime_config(
    *,
    profile_name: str | None = None,
    overrides: dict[str, Any] | None = None,
    path: str | Path | None = None,
    dotenv_path: str | Path | None = None,
    catalog: ModelCatalog | None = None,
) -> tuple[SifterConfig, list[str]]:
    config_path = Path(path) if path is not None else default_config_path()
    app = load_application_config(config_path)
    # With no profile file, legacy environment variables remain authoritative.
    return resolve_config(
        app,
        profile_name=profile_name,
        overrides=overrides,
        dotenv_path=dotenv_path,
        catalog=catalog,
        apply_profile=config_path.exists(),
    )


def apply_profile_to_config(
    base: SifterConfig,
    profile: ComputeProfile,
    *,
    catalog: ModelCatalog | None = None,
    warnings: list[str] | None = None,
) -> SifterConfig:
    warnings = warnings if warnings is not None else []
    updates: dict[str, Any] = {
        "routing_policy": profile.routing_policy,
        "prefer_local": profile.preferences.local_first,
        "cost_sensitive": profile.preferences.cost_sensitive,
        "privacy_sensitive": profile.preferences.privacy_sensitive,
    }

    local = profile.local
    if local is not None and local.enabled:
        updates.update(_tier_updates(local, prefix="local"))
    elif local is not None:
        updates["local_provider"] = local.provider

    expert = profile.expert
    if expert is not None and expert.enabled:
        updates.update(_tier_updates(expert, prefix="expert"))
        updates["cloud_allowed"] = True
        catalog_entry = catalog.get(expert.provider, expert.model) if catalog else None
        if expert.input_cost_per_mtok is None and catalog_entry and catalog_entry.cost:
            updates["cloud_input_cost_per_mtok"] = catalog_entry.cost.input_per_mtok
        if expert.output_cost_per_mtok is None and catalog_entry and catalog_entry.cost:
            updates["cloud_output_cost_per_mtok"] = catalog_entry.cost.output_per_mtok
    else:
        updates["cloud_allowed"] = False

    if not profile.preferences.cloud_allowed:
        updates["cloud_allowed"] = False
    if profile.goal is not None and not profile.goal.cloud_allowed:
        updates["cloud_allowed"] = False

    budget = profile.budget
    updates.update(
        {
            "max_cloud_calls_per_task": budget.max_cloud_calls_per_task,
            "max_cloud_tokens_per_task": budget.max_cloud_tokens_per_task,
            "max_cloud_cost_per_task": budget.max_cloud_cost_per_task,
            "max_parallel_cloud_calls": budget.max_parallel_cloud_calls,
            "max_parallel_local_calls": budget.max_parallel_local_calls,
        }
    )

    approval = profile.approval
    updates.update(
        {
            "human_approval_for_major_decisions": approval.major,
            "human_approval_for_critical_decisions": approval.critical,
            "human_approval_for_significant_decisions": approval.significant,
            "non_interactive": approval.non_interactive,
        }
    )
    if not approval.major or not approval.critical:
        warnings.append("major/critical approval is mandatory and was re-enabled")
    return base.model_copy(update=updates)


def _tier_updates(tier: ModelTierConfig, *, prefix: str) -> dict[str, Any]:
    updates: dict[str, Any] = {}
    if prefix == "local":
        updates["local_provider"] = tier.provider
        updates["local_model"] = tier.model
        if tier.context_limit is not None:
            updates["local_context_limit"] = tier.context_limit
        if tier.endpoint is not None:
            updates["local_endpoint"] = tier.endpoint
        if tier.api_key_env is not None:
            updates["local_api_key_env"] = tier.api_key_env
        if tier.max_output_tokens is not None:
            updates["local_max_output_tokens"] = tier.max_output_tokens
        if tier.temperature is not None:
            updates["local_temperature"] = tier.temperature
    else:
        updates["expert_provider"] = tier.provider
        updates["cloud_model"] = tier.model
        if tier.context_limit is not None:
            updates["cloud_context_limit"] = tier.context_limit
        if tier.endpoint is not None:
            updates["expert_endpoint"] = tier.endpoint
        if tier.api_key_env is not None:
            updates["expert_api_key_env"] = tier.api_key_env
        if tier.max_output_tokens is not None:
            updates["expert_max_output_tokens"] = tier.max_output_tokens
        if tier.temperature is not None:
            updates["expert_temperature"] = tier.temperature
        if tier.input_cost_per_mtok is not None:
            updates["cloud_input_cost_per_mtok"] = tier.input_cost_per_mtok
        if tier.output_cost_per_mtok is not None:
            updates["cloud_output_cost_per_mtok"] = tier.output_cost_per_mtok
    return updates


def _local_only_policy(policy: str) -> str:
    return "local_only" if policy != "local_only" else policy


def _warn_legacy_env(profile: ComputeProfile, warnings: list[str]) -> None:
    checks = [
        (
            "LOCAL_MODEL",
            os.environ.get("SIFTER_LOCAL_MODEL") or os.environ.get("LOCAL_MODEL"),
            profile.local.model if profile.local else None,
        ),
        (
            "CLOUD_MODEL",
            os.environ.get("SIFTER_CLOUD_MODEL") or os.environ.get("CLOUD_MODEL"),
            profile.expert.model if profile.expert else None,
        ),
    ]
    for name, env_value, profile_value in checks:
        if env_value and profile_value and env_value != profile_value:
            warnings.append(
                f"environment {name}='{env_value}' is overridden by profile "
                f"'{profile.name}' ('{profile_value}'); precedence is {PRECEDENCE}"
            )


def config_path_for_display() -> str:
    return str(default_config_path())
