from __future__ import annotations

from pathlib import Path

from aa_sifter.config.defaults import default_application_config
from aa_sifter.config.resolve import load_runtime_config, resolve_config
from aa_sifter.config.store import save_application_config


def test_environment_used_when_no_config_file(tmp_path: Path, monkeypatch):
    monkeypatch.setenv("SIFTER_CONFIG", str(tmp_path / "absent.toml"))
    monkeypatch.setenv("LOCAL_MODEL", "env-local:1b")
    config, warnings = load_runtime_config(path=tmp_path / "absent.toml")
    assert config.local_model == "env-local:1b"
    assert warnings == []


def test_profile_overrides_environment(tmp_path: Path, monkeypatch):
    path = tmp_path / "config.toml"
    app = default_application_config()
    app.profiles["default"].local.model = "profile-local:7b"
    save_application_config(app, path)
    monkeypatch.setenv("LOCAL_MODEL", "env-local:1b")
    config, warnings = load_runtime_config(path=path)
    assert config.local_model == "profile-local:7b"
    assert any("LOCAL_MODEL" in warning for warning in warnings)


def test_cli_override_beats_profile(tmp_path: Path):
    path = tmp_path / "config.toml"
    app = default_application_config()
    app.profiles["default"].local.model = "profile-local:7b"
    save_application_config(app, path)
    config, _ = load_runtime_config(path=path, overrides={"local_model": "cli-local:2b"})
    assert config.local_model == "cli-local:2b"


def test_cloud_allowed_false_forces_local_only(tmp_path: Path):
    path = tmp_path / "config.toml"
    app = default_application_config()
    app.profiles["default"].preferences.cloud_allowed = False
    save_application_config(app, path)
    config, _ = load_runtime_config(path=path)
    assert config.cloud_allowed is False
    assert config.routing_policy == "local_only"


def test_unused_overrides_none_are_ignored():
    app = default_application_config()
    config, _ = resolve_config(app, overrides={"local_model": None})
    assert config.local_model == "qwen3.5:9b"
