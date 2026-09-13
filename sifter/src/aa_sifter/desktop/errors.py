from __future__ import annotations

from collections.abc import Callable
from typing import Any

from pydantic import BaseModel, Field

from ..config.schema import ApplicationConfig, ModelTierConfig
from ..config.validation import validate_profile


class StartupIssue(BaseModel):
    kind: str
    severity: str = "warning"
    title: str
    detail: str = ""
    options: list[dict[str, Any]] = Field(default_factory=list)


def translate_startup_issues(
    app: ApplicationConfig,
    *,
    ollama_installed_fn: Callable[[], set[str] | None] | None = None,
    environ: dict[str, str] | None = None,
) -> list[StartupIssue]:
    """Turn common startup problems into user-facing, recoverable messages."""
    import os

    env = environ if environ is not None else os.environ
    installed_fn = ollama_installed_fn
    profile = app.profiles.get(app.active_profile)
    issues: list[StartupIssue] = []
    if profile is None:
        return [
            StartupIssue(
                kind="config_invalid",
                severity="error",
                title="No active profile was found.",
                detail=f"Active profile '{app.active_profile}' is not defined.",
                options=[{"id": "setup", "label": "Open Setup"}],
            )
        ]

    validation = validate_profile(profile, environ=env, require_credentials=False)
    for error in validation.errors:
        issues.append(
            StartupIssue(
                kind="config_invalid",
                severity="error",
                title="The active profile is not valid.",
                detail=error,
                options=[{"id": "setup", "label": "Open Setup"}],
            )
        )

    local = profile.local
    if local and local.enabled and local.provider == "ollama":
        installed = installed_fn() if installed_fn else None
        if installed is None:
            issues.append(
                StartupIssue(
                    kind="ollama_unreachable",
                    title="Compute Sifter could not reach Ollama.",
                    detail=(
                        "Local inference needs Ollama running at the configured host. "
                        "Cloud inference can still be used if configured."
                    ),
                    options=[
                        {"id": "start_ollama", "label": "Start Ollama"},
                        {"id": "cloud_only", "label": "Use Cloud Only"},
                        {"id": "setup", "label": "Open Setup"},
                        {"id": "details", "label": "View Details"},
                    ],
                )
            )
        elif local.model not in installed:
            issues.append(
                StartupIssue(
                    kind="model_missing",
                    title=f"Local model '{local.model}' is not installed.",
                    detail=f"Run: ollama pull {local.model}",
                    options=[
                        {"id": "install_model", "label": "Show Install Command"},
                        {"id": "setup", "label": "Change Model"},
                        {"id": "details", "label": "View Details"},
                    ],
                )
            )

    expert = profile.expert
    if (
        expert
        and expert.enabled
        and profile.preferences.cloud_allowed
        and _provider_needs_key(expert)
        and not (expert.api_key_env and env.get(expert.api_key_env))
    ):
        issues.append(
            StartupIssue(
                kind="missing_expert_key",
                title=(
                    f"{expert.provider} is configured as the expert provider, but "
                    f"{expert.api_key_env or 'its API key'} could not be found."
                ),
                detail="Local inference can still be used.",
                options=[
                    {"id": "configure_provider", "label": "Configure Provider"},
                    {"id": "local_only", "label": "Continue Local Only"},
                    {"id": "details", "label": "View Details"},
                ],
            )
        )
    return issues


def _provider_needs_key(expert: ModelTierConfig) -> bool:
    return expert.provider in {"deepseek", "openai-compatible"}
