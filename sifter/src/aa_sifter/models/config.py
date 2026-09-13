from __future__ import annotations

import os
from pathlib import Path
from typing import Any

from dotenv import load_dotenv
from pydantic import BaseModel, model_validator

from ..model_defaults import (
    DEFAULT_EXPERT_API_KEY_ENV,
    DEFAULT_EXPERT_CONTEXT,
    DEFAULT_EXPERT_INPUT_COST,
    DEFAULT_EXPERT_MODEL,
    DEFAULT_EXPERT_OUTPUT_COST,
    DEFAULT_EXPERT_PROVIDER,
    DEFAULT_LOCAL_CONTEXT,
    DEFAULT_LOCAL_MODEL,
    DEFAULT_LOCAL_PROVIDER,
    DEFAULT_ROUTING_POLICY,
    DEFAULT_TEMPERATURE,
)
from .provider import Tier


def _env(name: str, default: str | None = None, prefix: str = "SIFTER_") -> str | None:
    if prefix and f"{prefix}{name}" in os.environ:
        return os.environ[f"{prefix}{name}"]
    return os.environ.get(name, default)


def _env_bool(name: str, default: bool) -> bool:
    raw = _env(name)
    if raw is None:
        return default
    return raw.strip().lower() in {"1", "true", "yes", "on"}


def _env_int(name: str, default: int) -> int:
    raw = _env(name)
    if raw is None or raw.strip() == "":
        return default
    try:
        return int(raw)
    except ValueError:
        return default


def _env_float(name: str, default: float) -> float:
    raw = _env(name)
    if raw is None or raw.strip() == "":
        return default
    try:
        return float(raw)
    except ValueError:
        return default


class ModelConfig(BaseModel):
    name: str
    provider: str = "ollama"
    tier: Tier = Tier.LOCAL
    context_limit: int = 64_000
    input_cost_per_mtok: float = 0.0
    output_cost_per_mtok: float = 0.0
    supports_structured: bool = True
    timeout_seconds: float = 300.0

    def estimate_cost(self, input_tokens: int, output_tokens: int) -> float:
        return (
            input_tokens / 1_000_000 * self.input_cost_per_mtok
            + output_tokens / 1_000_000 * self.output_cost_per_mtok
        )


class SifterConfig(BaseModel):
    ollama_host: str = "http://localhost:11434"

    local_provider: str = DEFAULT_LOCAL_PROVIDER
    local_endpoint: str | None = None
    local_api_key_env: str | None = None
    local_model: str = DEFAULT_LOCAL_MODEL
    local_context_limit: int = DEFAULT_LOCAL_CONTEXT
    local_confidence_threshold: float = 0.72
    local_max_retries: int = 2
    local_max_output_tokens: int | None = None
    local_temperature: float | None = DEFAULT_TEMPERATURE

    expert_provider: str = DEFAULT_EXPERT_PROVIDER
    expert_endpoint: str | None = None
    expert_api_key_env: str | None = DEFAULT_EXPERT_API_KEY_ENV
    cloud_model: str = DEFAULT_EXPERT_MODEL
    cloud_context_limit: int = DEFAULT_EXPERT_CONTEXT
    cloud_input_cost_per_mtok: float = DEFAULT_EXPERT_INPUT_COST
    cloud_output_cost_per_mtok: float = DEFAULT_EXPERT_OUTPUT_COST
    expert_max_output_tokens: int | None = None
    expert_temperature: float | None = DEFAULT_TEMPERATURE

    cloud_allowed: bool = True
    prefer_local: bool = True
    cost_sensitive: bool = True
    privacy_sensitive: bool = True

    config_profile: str | None = None

    routing_policy: str = DEFAULT_ROUTING_POLICY

    max_cloud_calls_per_task: int = 5
    max_cloud_tokens_per_task: int = 200_000
    max_cloud_cost_per_task: float = 2.0
    max_parallel_cloud_calls: int = 2
    max_parallel_local_calls: int = 1

    request_timeout_seconds: float = 300.0

    human_approval_for_major_decisions: bool = True
    human_approval_for_critical_decisions: bool = True
    human_approval_for_significant_decisions: bool = False

    non_interactive: bool = False
    redact_secrets: bool = True
    debug: bool = False

    database_path: str = "~/.aa_sifter/aa_sifter.db"

    model_config = {"extra": "ignore", "validate_assignment": True}

    @model_validator(mode="after")
    def _enforce_safety(self) -> SifterConfig:
        # Critical approval is never silently disableable.
        if not self.human_approval_for_critical_decisions:
            object.__setattr__(self, "human_approval_for_critical_decisions", True)
        if not self.human_approval_for_major_decisions:
            object.__setattr__(self, "human_approval_for_major_decisions", True)
        if self.local_context_limit <= 0:
            object.__setattr__(self, "local_context_limit", 64_000)
        if self.max_cloud_cost_per_task < 0:
            object.__setattr__(self, "max_cloud_cost_per_task", 0.0)
        return self

    @property
    def resolved_database_path(self) -> Path:
        return Path(os.path.expanduser(self.database_path))

    def local_model_config(self) -> ModelConfig:
        return ModelConfig(
            name=self.local_model,
            provider=self.local_provider,
            tier=Tier.LOCAL,
            context_limit=self.local_context_limit,
            timeout_seconds=self.request_timeout_seconds,
        )

    def cloud_model_config(self) -> ModelConfig:
        return ModelConfig(
            name=self.cloud_model,
            provider=self.expert_provider,
            tier=Tier.EXPERT,
            context_limit=self.cloud_context_limit,
            input_cost_per_mtok=self.cloud_input_cost_per_mtok,
            output_cost_per_mtok=self.cloud_output_cost_per_mtok,
            timeout_seconds=self.request_timeout_seconds,
        )

    @classmethod
    def from_env(cls, dotenv_path: str | Path | None = None, **overrides: Any) -> SifterConfig:
        if dotenv_path is not None:
            load_dotenv(dotenv_path)
        else:
            load_dotenv()
        values: dict[str, Any] = {
            "ollama_host": _env("OLLAMA_HOST", "http://localhost:11434", prefix=""),
            "local_provider": _env("LOCAL_PROVIDER", DEFAULT_LOCAL_PROVIDER),
            "local_endpoint": _env("LOCAL_ENDPOINT", None),
            "local_api_key_env": _env("LOCAL_API_KEY_ENV", None),
            "local_model": _env("LOCAL_MODEL", DEFAULT_LOCAL_MODEL),
            "local_context_limit": _env_int("LOCAL_CONTEXT_LIMIT", DEFAULT_LOCAL_CONTEXT),
            "local_confidence_threshold": _env_float("LOCAL_CONFIDENCE_THRESHOLD", 0.72),
            "local_max_retries": _env_int("LOCAL_MAX_RETRIES", 2),
            "local_max_output_tokens": _env_int("LOCAL_MAX_OUTPUT_TOKENS", 0) or None,
            "local_temperature": _env_float("LOCAL_TEMPERATURE", DEFAULT_TEMPERATURE),
            "expert_provider": _env("EXPERT_PROVIDER", DEFAULT_EXPERT_PROVIDER),
            "expert_endpoint": _env("EXPERT_ENDPOINT", None),
            "expert_api_key_env": _env("EXPERT_API_KEY_ENV", DEFAULT_EXPERT_API_KEY_ENV),
            "cloud_model": _env("CLOUD_MODEL", DEFAULT_EXPERT_MODEL),
            "cloud_context_limit": _env_int("CLOUD_CONTEXT_LIMIT", DEFAULT_EXPERT_CONTEXT),
            "cloud_input_cost_per_mtok": _env_float(
                "CLOUD_INPUT_COST_PER_MTOK", DEFAULT_EXPERT_INPUT_COST
            ),
            "cloud_output_cost_per_mtok": _env_float(
                "CLOUD_OUTPUT_COST_PER_MTOK", DEFAULT_EXPERT_OUTPUT_COST
            ),
            "expert_max_output_tokens": _env_int("EXPERT_MAX_OUTPUT_TOKENS", 0) or None,
            "expert_temperature": _env_float("EXPERT_TEMPERATURE", 0.2),
            "cloud_allowed": _env_bool("CLOUD_ALLOWED", True),
            "prefer_local": _env_bool("PREFER_LOCAL", True),
            "cost_sensitive": _env_bool("COST_SENSITIVE", True),
            "privacy_sensitive": _env_bool("PRIVACY_SENSITIVE", True),
            "routing_policy": _env("ROUTING_POLICY", "adaptive"),
            "max_cloud_calls_per_task": _env_int("MAX_CLOUD_CALLS_PER_TASK", 5),
            "max_cloud_tokens_per_task": _env_int("MAX_CLOUD_TOKENS_PER_TASK", 200_000),
            "max_cloud_cost_per_task": _env_float("MAX_CLOUD_COST_PER_TASK", 2.0),
            "max_parallel_cloud_calls": _env_int("MAX_PARALLEL_CLOUD_CALLS", 2),
            "max_parallel_local_calls": _env_int("MAX_PARALLEL_LOCAL_CALLS", 1),
            "request_timeout_seconds": _env_float("REQUEST_TIMEOUT_SECONDS", 300.0),
            "human_approval_for_major_decisions": _env_bool(
                "HUMAN_APPROVAL_FOR_MAJOR_DECISIONS", True
            ),
            "human_approval_for_critical_decisions": _env_bool(
                "HUMAN_APPROVAL_FOR_CRITICAL_DECISIONS", True
            ),
            "human_approval_for_significant_decisions": _env_bool(
                "HUMAN_APPROVAL_FOR_SIGNIFICANT_DECISIONS", False
            ),
            "non_interactive": _env_bool("NON_INTERACTIVE", False),
            "redact_secrets": _env_bool("REDACT_SECRETS", True),
            "debug": _env_bool("DEBUG", False),
            "database_path": _env("DATABASE_PATH", "~/.aa_sifter/aa_sifter.db"),
        }
        values.update({k: v for k, v in overrides.items() if v is not None})
        return cls(**values)
