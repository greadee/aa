from __future__ import annotations

from pathlib import Path

import pytest

from aa_sifter.config import profiles as profile_ops
from aa_sifter.config.defaults import default_application_config


def _app():
    return default_application_config()


def test_get_and_list_profiles():
    app = _app()
    profile = profile_ops.get_profile(app)
    assert profile.name == "default"
    summaries = profile_ops.list_profiles(app)
    assert summaries[0]["active"] is True
    with pytest.raises(profile_ops.ProfileError):
        profile_ops.get_profile(app, "missing")


def test_create_copy_delete_profile(tmp_path: Path):
    app = _app()
    path = tmp_path / "config.toml"
    created = profile_ops.create_profile(app, "coding", path=path)
    assert created.name == "coding"
    assert path.exists()
    with pytest.raises(profile_ops.ProfileError):
        profile_ops.create_profile(app, "coding", path=path)
    profile_ops.copy_profile(app, "coding", "privacy", force=True, path=path)
    assert "privacy" in app.profiles

    active = profile_ops.delete_profile(app, "privacy", path=path)
    assert "privacy" not in app.profiles
    assert active == app.active_profile


def test_delete_last_profile_rejected(tmp_path: Path):
    app = _app()
    with pytest.raises(profile_ops.ProfileError):
        profile_ops.delete_profile(app, "default", path=tmp_path / "c.toml")


def test_activate_profile(tmp_path: Path):
    app = _app()
    profile_ops.create_profile(app, "coding", path=tmp_path / "c.toml")
    profile_ops.activate_profile(app, "coding", path=tmp_path / "c.toml")
    assert app.active_profile == "coding"


def test_set_tier_updates_fields(tmp_path: Path):
    app = _app()
    path = tmp_path / "c.toml"
    profile_ops.set_tier(
        app,
        tier="local",
        provider="ollama",
        model="custom:7b",
        context_limit=16384,
        path=path,
    )
    assert app.profiles["default"].local.model == "custom:7b"
    assert app.profiles["default"].local.context_limit == 16384

    profile_ops.set_tier(
        app,
        tier="expert",
        provider="openai-compatible",
        model="my-model",
        endpoint="http://localhost:8000/v1",
        api_key_env="MY_KEY",
        path=path,
    )
    assert app.profiles["default"].expert.provider == "openai-compatible"
    assert app.profiles["default"].expert.endpoint == "http://localhost:8000/v1"

    with pytest.raises(profile_ops.ProfileError):
        profile_ops.set_tier(app, tier="bogus", provider="x", model="y", path=path)


def test_update_profile_fields(tmp_path: Path):
    app = _app()
    path = tmp_path / "c.toml"
    profile_ops.update_profile(
        app,
        local_model="local:1b",
        expert_model="expert:1b",
        routing="local_first",
        cloud=False,
        local_first=True,
        context_limit=8192,
        path=path,
    )
    profile = app.profiles["default"]
    assert profile.local.model == "local:1b"
    assert profile.expert.model == "expert:1b"
    assert profile.routing_policy == "local_first"
    assert profile.preferences.cloud_allowed is False
