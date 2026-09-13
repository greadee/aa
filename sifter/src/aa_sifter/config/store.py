from __future__ import annotations

import os
import tomllib
from pathlib import Path
from typing import Any

from .defaults import default_application_config, default_profile
from .schema import CONFIG_VERSION, ApplicationConfig, ComputeProfile
from .toml import dumps as dumps_toml

_SECRET_KEYS = {"api_key", "apikey", "token", "password", "secret", "credentials"}


def default_config_path() -> Path:
    override = os.environ.get("SIFTER_CONFIG") or os.environ.get("AA_SIFTER_CONFIG")
    if override:
        return Path(os.path.expanduser(override))
    return Path.home() / ".aa_sifter" / "config.toml"


def load_application_config(path: str | Path | None = None) -> ApplicationConfig:
    config_path = Path(path) if path is not None else default_config_path()
    if not config_path.exists():
        return default_application_config()
    raw = config_path.read_bytes()
    return parse_application_config(raw)


def parse_application_config(raw: bytes | str) -> ApplicationConfig:
    data = tomllib.loads(raw if isinstance(raw, str) else raw.decode("utf-8"))
    data, _ = migrate(data)
    config = ApplicationConfig.model_validate(data)
    if "default" not in config.profiles:
        config.profiles["default"] = default_profile()
    if config.active_profile not in config.profiles:
        config.active_profile = "default"
    return config


def migrate(data: dict[str, Any]) -> tuple[dict[str, Any], list[str]]:
    """Upgrade older config shapes. Returns (data, warnings)."""
    warnings: list[str] = []
    version = int(data.get("version", 0) or 0)
    if version > CONFIG_VERSION:
        warnings.append(
            f"config version {version} is newer than supported {CONFIG_VERSION}; "
            "unknown fields are ignored"
        )
    if "profiles" not in data or not data["profiles"]:
        profile = default_profile()
        # Legacy flat keys that predate the profile system.
        local_model = data.pop("local_model", None)
        expert_model = data.pop("expert_model", data.pop("cloud_model", None))
        if local_model and profile.local is not None:
            profile.local.model = str(local_model)
            warnings.append(f"migrated legacy local_model='{local_model}' into the default profile")
        if expert_model and profile.expert is not None:
            profile.expert.model = str(expert_model)
            warnings.append(
                f"migrated legacy expert_model='{expert_model}' into the default profile"
            )
        data["profiles"] = {"default": profile.model_dump(exclude_none=True)}
        if version:
            warnings.append(f"migrated config from version {version} to {CONFIG_VERSION}")
    data["version"] = CONFIG_VERSION
    return data, warnings


def save_application_config(config: ApplicationConfig, path: str | Path | None = None) -> Path:
    config_path = Path(path) if path is not None else default_config_path()
    config_path.parent.mkdir(parents=True, exist_ok=True)
    payload = config.model_dump(exclude_none=True)
    config_path.write_text(dumps_toml(payload), encoding="utf-8")
    return config_path


def export_profile(config: ApplicationConfig, name: str) -> str:
    if name not in config.profiles:
        raise KeyError(f"profile '{name}' does not exist")
    profile = config.profiles[name]
    payload = profile.model_dump(exclude_none=True)
    _assert_no_secrets(payload)
    return dumps_toml({"profile": payload})


def import_profile(text: str) -> ComputeProfile:
    data = tomllib.loads(text)
    payload = data.get("profile", data)
    _assert_no_secrets(payload)
    return ComputeProfile.model_validate(payload)


def _assert_no_secrets(payload: Any, path: str = "") -> None:
    if isinstance(payload, dict):
        for key, value in payload.items():
            if key.lower() in _SECRET_KEYS:
                raise ValueError(
                    f"refusing to handle profile containing secret-like key '{path}{key}'; "
                    "store secrets in environment variables and reference them with api_key_env"
                )
            _assert_no_secrets(value, f"{path}{key}.")
    elif isinstance(payload, list):
        for item in payload:
            _assert_no_secrets(item, path)
