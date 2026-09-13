from __future__ import annotations

import json
from pathlib import Path

from aa_sifter.cli import config_commands
from aa_sifter.cli import main as cli
from aa_sifter.config.schema import HardwareProfile
from aa_sifter.config.store import load_application_config


def _setup(capsys) -> None:
    assert cli.main(["setup"]) == 0
    capsys.readouterr()


def test_profile_error_paths(isolated_config: Path, capsys):
    _setup(capsys)
    assert cli.main(["profile", "show", "missing"]) == 1
    assert cli.main(["profile", "use", "missing"]) == 1
    assert cli.main(["profile", "delete", "default"]) == 1  # cannot delete the last
    assert cli.main(["profile", "create", "coding", "--source", "missing"]) == 1
    assert cli.main(["profile", "export", "missing"]) == 1
    assert cli.main(["profile", "import", "missing.toml"]) == 1
    assert cli.main(["profile", "bad-action"]) == 1


def test_profile_copy_conflict_and_force(isolated_config: Path, capsys):
    _setup(capsys)
    assert cli.main(["profile", "copy", "default", "coding"]) == 0
    assert cli.main(["profile", "copy", "default", "coding"]) == 1
    assert cli.main(["profile", "copy", "default", "coding", "--force"]) == 0


def test_profile_edit_flags(isolated_config: Path, capsys):
    _setup(capsys)
    assert (
        cli.main(
            [
                "profile",
                "edit",
                "default",
                "--local-model",
                "custom:7b",
                "--routing",
                "local_first",
                "--context-limit",
                "16384",
            ]
        )
        == 0
    )
    app = load_application_config(isolated_config)
    assert app.profiles["default"].local.model == "custom:7b"
    assert app.profiles["default"].routing_policy == "local_first"


def test_profile_edit_without_flags_prints_path(isolated_config: Path, capsys):
    _setup(capsys)
    assert cli.main(["profile", "edit", "default"]) == 0
    assert "Profile file" in capsys.readouterr().out


def test_model_list_and_show(isolated_config: Path, capsys, monkeypatch):
    _setup(capsys)
    monkeypatch.setattr(config_commands, "_ollama_installed", lambda: {"qwen3.5:9b"})
    assert cli.main(["model", "list"]) == 0
    assert "qwen3.5:9b" in capsys.readouterr().out
    assert cli.main(["model", "list", "--tier", "local"]) == 0
    assert cli.main(["model", "list", "--tier", "expert"]) == 0
    assert cli.main(["model", "show"]) == 0
    assert cli.main(["model", "bad-action"]) == 1


def test_model_set_expert_options(isolated_config: Path, capsys):
    _setup(capsys)
    assert (
        cli.main(
            [
                "model",
                "set",
                "expert",
                "openai-compatible",
                "gpt-4o-mini",
                "--endpoint",
                "http://localhost:8000/v1",
                "--api-key-env",
                "OPENAI_API_KEY",
            ]
        )
        == 0
    )
    app = load_application_config(isolated_config)
    assert app.profiles["default"].expert.provider == "openai-compatible"
    assert app.profiles["default"].expert.endpoint == "http://localhost:8000/v1"
    assert cli.main(["model", "set", "bogus", "ollama", "x"]) == 1
    assert cli.main(["model", "set", "local", "ollama"]) == 1


def test_system_detect_save(isolated_config: Path, capsys, monkeypatch):
    _setup(capsys)
    monkeypatch.setattr(
        config_commands,
        "detect_hardware",
        lambda: HardwareProfile(
            gpu_name="NVIDIA RTX 3080", gpu_vram_gb=10.0, system_ram_gb=32.0, source="detected"
        ),
    )
    assert cli.main(["system", "detect", "--save", "desktop"]) == 0
    assert "RTX 3080" in capsys.readouterr().out
    app = load_application_config(isolated_config)
    assert app.active_hardware == "desktop"
    assert cli.main(["system", "show"]) == 0
    assert cli.main(["system", "use", "desktop"]) == 0
    assert cli.main(["system", "use", "missing"]) == 1
    assert cli.main(["system", "bad-action"]) == 1


def test_system_show_without_profile(isolated_config: Path, capsys):
    assert cli.main(["system", "show"]) == 0
    assert "No active hardware profile" in capsys.readouterr().out


def test_recommend_json(isolated_config: Path, capsys):
    assert cli.main(["recommend", "--vram", "10", "--ram", "32", "--json"]) == 0
    payload = json.loads(capsys.readouterr().out)
    assert payload["profile"]["local"]["model"] == "qwen3.5:9b"


def test_setup_with_flags_and_yes(isolated_config: Path, capsys):
    assert (
        cli.main(
            [
                "setup",
                "--yes",
                "--gpu",
                "RTX 3080",
                "--vram",
                "10",
                "--ram",
                "32",
                "--goal",
                "software development",
                "--name",
                "desktop",
            ]
        )
        == 0
    )
    app = load_application_config(isolated_config)
    assert app.active_profile == "desktop"


def test_doctor_live_with_patched_check(isolated_config: Path, capsys, monkeypatch):
    _setup(capsys)
    monkeypatch.setattr(config_commands, "_ollama_installed", lambda: {"qwen3.5:9b"})
    monkeypatch.setattr(config_commands, "_live_expert_check", lambda expert: (True, "reachable"))
    assert cli.main(["doctor", "--live"]) == 0
    out = capsys.readouterr().out
    assert "doctor: healthy" in out
    assert "expert connectivity" in out
