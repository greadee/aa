"""Smoke tests for the installed package and CLI.

These run with no providers and no network (the Ollama check in `doctor` is
optional and never fails the command).
"""

from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parents[2]


def _run(*args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [sys.executable, "-m", "aa_sifter.cli.main", *args],
        cwd=ROOT,
        capture_output=True,
        text=True,
        timeout=120,
    )


@pytest.mark.smoke
def test_package_imports():
    import aa_sifter

    assert aa_sifter.__version__


@pytest.mark.smoke
def test_cli_help_runs():
    result = _run("--help")
    assert result.returncode == 0, result.stderr
    assert "aa-sifter" in result.stdout


@pytest.mark.smoke
def test_cli_config_outputs_json():
    result = _run("config")
    assert result.returncode == 0, result.stderr
    payload = json.loads(result.stdout)
    assert payload["local_model"]
    assert payload["cloud_model"]


@pytest.mark.smoke
def test_cli_doctor_is_healthy_without_providers():
    result = _run("doctor")
    assert result.returncode == 0, result.stdout + result.stderr
    assert "doctor: healthy" in result.stdout


@pytest.mark.smoke
def test_cli_doctor_json():
    result = _run("doctor", "--json")
    assert result.returncode == 0, result.stderr
    payload = json.loads(result.stdout)
    assert payload["ok"] is True
    names = {check["name"] for check in payload["checks"]}
    assert {"profile", "configuration"} <= names
