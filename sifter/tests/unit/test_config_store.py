from __future__ import annotations

from pathlib import Path

import pytest

from aa_sifter.config.defaults import default_application_config
from aa_sifter.config.store import (
    export_profile,
    import_profile,
    load_application_config,
    parse_application_config,
    save_application_config,
)


def test_missing_file_returns_defaults(tmp_path: Path):
    app = load_application_config(tmp_path / "absent.toml")
    assert "default" in app.profiles
    assert app.active_profile == "default"


def test_save_and_load_roundtrip(tmp_path: Path):
    path = tmp_path / "config.toml"
    app = default_application_config()
    app.profiles["default"].local.model = "custom:7b"
    app.profiles["default"].routing_policy = "local_first"
    save_application_config(app, path)
    assert path.exists()
    reloaded = load_application_config(path)
    assert reloaded.profiles["default"].local.model == "custom:7b"
    assert reloaded.profiles["default"].routing_policy == "local_first"


def test_migrate_legacy_flat_keys():
    raw = """
    version = 0
    local_model = "legacy-local:3b"
    cloud_model = "legacy-expert"
    """
    app = parse_application_config(raw)
    assert app.profiles["default"].local.model == "legacy-local:3b"
    assert app.profiles["default"].expert.model == "legacy-expert"
    assert app.version == 1


def test_export_import_no_secrets():
    app = default_application_config()
    text = export_profile(app, "default")
    assert "api_key_env" in text
    assert "api_key =" not in text
    profile = import_profile(text)
    assert profile.local.model == "qwen3.5:9b"


def test_import_rejects_secret_values():
    raw = """
    [profile]
    name = "bad"
    [profile.local]
    provider = "ollama"
    model = "x"
    api_key = "sk-secret"
    """
    with pytest.raises(ValueError):
        import_profile(raw)


def test_save_is_atomic_and_creates_parents(tmp_path: Path):
    path = tmp_path / "nested" / "config.toml"
    save_application_config(default_application_config(), path)
    assert path.exists()
