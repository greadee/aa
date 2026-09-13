from __future__ import annotations

import json
from pathlib import Path

from aa_sifter.cli import main as cli
from aa_sifter.config.store import load_application_config


def test_setup_creates_default_profile(isolated_config: Path, capsys):
    assert cli.main(["setup"]) == 0
    assert isolated_config.exists()
    app = load_application_config(isolated_config)
    assert app.active_profile == "default"
    assert app.profiles["default"].local.model == "qwen3.5:9b"
    assert app.profiles["default"].expert.provider == "deepseek"
    assert "Saved profile" in capsys.readouterr().out


def test_profile_list_and_show(isolated_config: Path, capsys):
    cli.main(["setup"])
    capsys.readouterr()
    assert cli.main(["profile", "list"]) == 0
    out = capsys.readouterr().out
    assert "default" in out
    assert cli.main(["profile", "show", "default"]) == 0
    assert "ollama / qwen3.5:9b" in capsys.readouterr().out


def test_profile_create_use_copy_delete(isolated_config: Path, capsys):
    cli.main(["setup"])
    capsys.readouterr()
    assert cli.main(["profile", "create", "coding"]) == 0
    assert cli.main(["profile", "copy", "coding", "privacy"]) == 0
    assert cli.main(["profile", "use", "privacy"]) == 0
    app = load_application_config(isolated_config)
    assert app.active_profile == "privacy"
    assert set(app.profiles) >= {"default", "coding", "privacy"}
    assert cli.main(["profile", "delete", "coding"]) == 0
    assert "coding" not in load_application_config(isolated_config).profiles


def test_profile_validate_requires_key(isolated_config: Path, monkeypatch, capsys):
    cli.main(["setup"])
    capsys.readouterr()
    monkeypatch.delenv("DEEPSEEK_API_KEY", raising=False)
    assert cli.main(["profile", "validate", "default"]) == 1
    assert "DEEPSEEK_API_KEY" in capsys.readouterr().out

    monkeypatch.setenv("DEEPSEEK_API_KEY", "test")
    assert cli.main(["profile", "validate", "default"]) == 0
    assert "PASS" in capsys.readouterr().out


def test_model_set_unknown_model_warns_but_saves(isolated_config: Path, capsys):
    cli.main(["setup"])
    capsys.readouterr()
    assert cli.main(["model", "set", "local", "ollama", "future-model:12b"]) == 0
    out = capsys.readouterr().out
    assert "WARNING" in out
    app = load_application_config(isolated_config)
    assert app.profiles["default"].local.model == "future-model:12b"


def test_recommend_saves_profile(isolated_config: Path, capsys):
    args = [
        "recommend",
        "--gpu",
        "RTX 3080",
        "--vram",
        "10",
        "--ram",
        "32",
        "--goal",
        "software development",
        "--goal",
        "software engineering agents",
        "--save",
        "coding-desktop",
    ]
    assert cli.main(args) == 0
    out = capsys.readouterr().out
    assert "Recommended configuration" in out
    app = load_application_config(isolated_config)
    assert "coding-desktop" in app.profiles
    assert app.profiles["coding-desktop"].local.model == "qwen3.5:9b"


def test_recommend_local_only_excludes_cloud(isolated_config: Path, capsys):
    assert cli.main(["recommend", "--no-cloud", "--save", "privacy", "--vram", "10"]) == 0
    capsys.readouterr()
    app = load_application_config(isolated_config)
    profile = app.profiles["privacy"]
    assert profile.expert is None
    assert profile.preferences.cloud_allowed is False
    assert profile.routing_policy == "local_only"


def test_system_configure_and_list(isolated_config: Path, capsys):
    assert (
        cli.main(
            [
                "system",
                "configure",
                "--gpu",
                "RTX 3080",
                "--vram",
                "10",
                "--ram",
                "32",
                "--save",
                "desktop",
            ]
        )
        == 0
    )
    capsys.readouterr()
    assert cli.main(["system", "list"]) == 0
    assert "desktop" in capsys.readouterr().out
    app = load_application_config(isolated_config)
    assert app.active_hardware == "desktop"
    assert app.hardware["desktop"].gpu_vram_gb == 10


def test_profile_export_import_roundtrip(isolated_config: Path, tmp_path: Path, capsys):
    cli.main(["setup"])
    export_file = tmp_path / "coding.toml"
    assert cli.main(["profile", "export", "default", "--output", str(export_file)]) == 0
    assert export_file.exists()
    assert "api_key" not in export_file.read_text(encoding="utf-8").replace("api_key_env", "")
    assert cli.main(["profile", "import", str(export_file), "--name", "imported"]) == 0
    app = load_application_config(isolated_config)
    assert "imported" in app.profiles


def test_doctor_json_healthy(isolated_config: Path, monkeypatch, capsys):
    monkeypatch.setenv("DEEPSEEK_API_KEY", "test")
    assert cli.main(["doctor", "--json"]) == 0
    payload = json.loads(capsys.readouterr().out)
    assert payload["ok"] is True
