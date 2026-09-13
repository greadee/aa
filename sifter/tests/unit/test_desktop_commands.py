from __future__ import annotations

import argparse
from pathlib import Path

import pytest

from aa_sifter.cli import desktop_commands
from aa_sifter.cli import main as cli


def _args(**overrides) -> argparse.Namespace:
    defaults = {
        "state_file": None,
        "stop": False,
        "no_browser": True,
        "headless": False,
        "foreground": False,
        "host": "127.0.0.1",
        "port": 0,
        "debug": False,
        "profile": None,
        "token": None,
    }
    defaults.update(overrides)
    return argparse.Namespace(**defaults)


def test_launch_stop_without_backend(tmp_path: Path, capsys):
    code = desktop_commands.launch(_args(stop=True, state_file=str(tmp_path / "state.json")))
    assert code == 0
    assert "No running backend" in capsys.readouterr().out


def test_stop_without_backend(tmp_path: Path, capsys):
    code = desktop_commands.stop(_args(state_file=str(tmp_path / "state.json")))
    assert code == 0
    assert "not running" in capsys.readouterr().out


def test_launch_focuses_existing_instance(monkeypatch, tmp_path: Path, capsys):
    monkeypatch.setattr(
        desktop_commands,
        "existing_instance",
        lambda path: {"url": "http://127.0.0.1:9", "token": "t", "health": {"status": "ready"}},
    )
    monkeypatch.setattr(desktop_commands, "open_browser", lambda url: True)
    code = desktop_commands.launch(_args(state_file=str(tmp_path / "state.json")))
    assert code == 0
    assert "already running" in capsys.readouterr().out


def test_launch_reports_clean_error(monkeypatch, tmp_path: Path, capsys):
    monkeypatch.setattr(desktop_commands, "existing_instance", lambda path: None)
    monkeypatch.setattr(desktop_commands, "ensure_dirs", lambda: None)

    def boom(**kwargs):
        raise desktop_commands.LauncherError(
            "Backend failed", details="see logs", options=["View Logs", "Retry"]
        )

    monkeypatch.setattr(desktop_commands, "start_backend_process", boom)
    code = desktop_commands.launch(_args(state_file=str(tmp_path / "state.json")))
    out = capsys.readouterr().out
    assert code == 1
    assert "Backend failed" in out
    assert "View Logs" in out


def test_print_issues(monkeypatch, capsys):
    class Response:
        def json(self):
            return {
                "issues": [
                    {
                        "severity": "warning",
                        "title": "Ollama unreachable",
                        "detail": "start it",
                        "options": [{"id": "start_ollama", "label": "Start Ollama"}],
                    }
                ]
            }

    monkeypatch.setattr(desktop_commands.httpx, "get", lambda *a, **k: Response())
    desktop_commands._print_issues("http://127.0.0.1:9", "t")
    out = capsys.readouterr().out
    assert "Ollama unreachable" in out
    assert "Start Ollama" in out


def test_dispatch_unknown_command(capsys):
    assert desktop_commands.dispatch("nope", _args()) == 1
    assert "Unknown desktop command" in capsys.readouterr().err


@pytest.mark.parametrize("command", ["launch", "serve", "desktop", "stop"])
def test_main_dispatches_desktop_commands(command: str, tmp_path: Path, monkeypatch):
    monkeypatch.setattr(
        desktop_commands,
        "dispatch",
        lambda cmd, args: 0 if cmd == command else 99,
    )
    argv = [command, "--state-file", str(tmp_path / "state.json")]
    assert cli.main(argv) == 0
