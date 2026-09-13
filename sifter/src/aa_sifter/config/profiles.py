from __future__ import annotations

from pathlib import Path

from .schema import ApplicationConfig, ComputeProfile, ModelTierConfig
from .store import save_application_config


class ProfileError(Exception):
    pass


def get_profile(app: ApplicationConfig, name: str | None = None) -> ComputeProfile:
    selected = name or app.active_profile
    profile = app.profiles.get(selected)
    if profile is None:
        raise ProfileError(f"profile '{selected}' does not exist")
    return profile


def profile_summary(name: str, profile: ComputeProfile, active: str) -> dict:
    return {
        "name": name,
        "active": name == active,
        "local": profile.local.model_dump(exclude_none=True) if profile.local else None,
        "expert": profile.expert.model_dump(exclude_none=True) if profile.expert else None,
        "local_fallbacks": [t.model_dump(exclude_none=True) for t in profile.local_fallbacks],
        "routing_policy": profile.routing_policy,
        "preferences": profile.preferences.model_dump(),
        "budget": profile.budget.model_dump(),
        "approval": profile.approval.model_dump(),
        "goal": profile.goal.model_dump() if profile.goal else None,
        "hardware": profile.hardware.model_dump() if profile.hardware else None,
    }


def list_profiles(app: ApplicationConfig) -> list[dict]:
    return [
        profile_summary(name, profile, app.active_profile) for name, profile in app.profiles.items()
    ]


def activate_profile(
    app: ApplicationConfig, name: str, *, save: bool = True, path: Path | None = None
) -> ComputeProfile:
    profile = get_profile(app, name)
    app.active_profile = name
    if save:
        save_application_config(app, path)
    return profile


def create_profile(
    app: ApplicationConfig,
    name: str,
    *,
    source: str | None = None,
    force: bool = False,
    save: bool = True,
    path: Path | None = None,
) -> ComputeProfile:
    if name in app.profiles and not force:
        raise ProfileError(f"profile '{name}' already exists")
    source_name = source or app.active_profile
    base = get_profile(app, source_name).model_copy(deep=True)
    base.name = name
    app.profiles[name] = base
    if save:
        save_application_config(app, path)
    return base


def copy_profile(
    app: ApplicationConfig,
    source: str,
    target: str,
    *,
    force: bool = False,
    save: bool = True,
    path: Path | None = None,
) -> ComputeProfile:
    return create_profile(app, target, source=source, force=force, save=save, path=path)


def delete_profile(
    app: ApplicationConfig, name: str, *, save: bool = True, path: Path | None = None
) -> str:
    get_profile(app, name)
    if len(app.profiles) == 1:
        raise ProfileError("cannot delete the last profile")
    del app.profiles[name]
    if app.active_profile == name:
        app.active_profile = next(iter(app.profiles))
    if save:
        save_application_config(app, path)
    return app.active_profile


def set_tier(
    app: ApplicationConfig,
    *,
    tier: str,
    provider: str,
    model: str,
    profile_name: str | None = None,
    context_limit: int | None = None,
    max_output_tokens: int | None = None,
    endpoint: str | None = None,
    api_key_env: str | None = None,
    save: bool = True,
    path: Path | None = None,
) -> ComputeProfile:
    profile = get_profile(app, profile_name)
    if tier not in {"local", "expert"}:
        raise ProfileError(f"unknown tier '{tier}'")
    if tier == "local":
        if profile.local is None:
            profile.local = ModelTierConfig(provider=provider, model=model)
        else:
            profile.local.provider = provider
            profile.local.model = model
        target = profile.local
    else:
        if profile.expert is None:
            profile.expert = ModelTierConfig(provider=provider, model=model)
        else:
            profile.expert.provider = provider
            profile.expert.model = model
        target = profile.expert
    if context_limit is not None:
        target.context_limit = context_limit
    if max_output_tokens is not None:
        target.max_output_tokens = max_output_tokens
    if endpoint:
        target.endpoint = endpoint
    if api_key_env:
        target.api_key_env = api_key_env
    if save:
        save_application_config(app, path)
    return profile


def update_profile(
    app: ApplicationConfig,
    *,
    profile_name: str | None = None,
    local_model: str | None = None,
    expert_model: str | None = None,
    routing: str | None = None,
    cloud: bool | None = None,
    local_first: bool | None = None,
    context_limit: int | None = None,
    save: bool = True,
    path: Path | None = None,
) -> ComputeProfile:
    profile = get_profile(app, profile_name)
    changed = False
    if local_model:
        if profile.local is None:
            profile.local = ModelTierConfig(provider="ollama", model=local_model)
        else:
            profile.local.model = local_model
        changed = True
    if expert_model:
        if profile.expert is None:
            profile.expert = ModelTierConfig(provider="deepseek", model=expert_model)
        else:
            profile.expert.model = expert_model
        changed = True
    if routing:
        profile.routing_policy = routing
        changed = True
    if cloud is not None:
        profile.preferences.cloud_allowed = cloud
        if profile.expert is not None:
            profile.expert.enabled = cloud
        changed = True
    if local_first is not None:
        profile.preferences.local_first = local_first
        changed = True
    if context_limit is not None and profile.local is not None:
        profile.local.context_limit = context_limit
        changed = True
    if changed and save:
        save_application_config(app, path)
    return profile
