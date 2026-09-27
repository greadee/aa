from __future__ import annotations

import argparse
import json
from pathlib import Path

import pytest

from aa_inference.app import InferenceResult
from aa_inference.cli import main as cli
from aa_inference.metrics.usage import UsageMetrics


@pytest.fixture(autouse=True)
def _isolated_env(tmp_path: Path, monkeypatch: pytest.MonkeyPatch):
    monkeypatch.setenv("INFERENCE_DATABASE_PATH", str(tmp_path / "cli.db"))
    monkeypatch.setenv("INFERENCE_CONFIG", str(tmp_path / "config.toml"))
    monkeypatch.setenv("INFERENCE_NON_INTERACTIVE", "true")
    for var in ("LOCAL_MODEL", "CLOUD_MODEL", "DEEPSEEK_API_KEY"):
        monkeypatch.delenv(var, raising=False)


def test_config_command(capsys: pytest.CaptureFixture[str]):
    assert cli.main(["config"]) == 0
    payload = json.loads(capsys.readouterr().out)
    assert payload["local_model"]


def test_stats_command(capsys: pytest.CaptureFixture[str]):
    assert cli.main(["stats"]) == 0
    payload = json.loads(capsys.readouterr().out)
    assert "runs" in payload


def test_doctor_json_command(capsys: pytest.CaptureFixture[str]):
    assert cli.main(["doctor", "--json"]) == 0
    payload = json.loads(capsys.readouterr().out)
    assert payload["ok"] is True


def test_rules_add_list_remove(capsys: pytest.CaptureFixture[str]):
    assert cli.main(["rules", "add", "SQLite is the approved persistence technology."]) == 0
    capsys.readouterr()
    assert cli.main(["rules", "list"]) == 0
    listing = capsys.readouterr().out
    assert "SQLite is the approved" in listing
    assert cli.main(["rules", "remove", "1"]) == 0
    assert cli.main(["rules", "list"]) == 0
    assert "No standing rules" in capsys.readouterr().out


def test_no_query_prints_help(capsys: pytest.CaptureFixture[str]):
    assert cli.main([]) == 1
    assert "aa-inference" in capsys.readouterr().out


def test_load_context_inline_and_file(tmp_path: Path):
    inline = argparse.Namespace(context="inline", context_file=None)
    assert cli._load_context(inline) == "inline"
    path = tmp_path / "context.txt"
    path.write_text("from file", encoding="utf-8")
    from_file = argparse.Namespace(context=None, context_file=str(path))
    assert cli._load_context(from_file) == "from file"


def test_load_context_rejects_both():
    both = argparse.Namespace(context="a", context_file="b")
    with pytest.raises(SystemExit):
        cli._load_context(both)


def test_print_result_plain_and_json(capsys: pytest.CaptureFixture[str]):
    result = InferenceResult(trace_id="t", status="completed", answer="the answer")
    cli._print_result(result, argparse.Namespace(json=False, debug=False))
    assert "the answer" in capsys.readouterr().out

    cli._print_result(result, argparse.Namespace(json=True, debug=False))
    payload = json.loads(capsys.readouterr().out)
    assert payload["answer"] == "the answer"


def test_print_result_blocked(capsys: pytest.CaptureFixture[str]):
    result = InferenceResult(
        trace_id="t",
        status="stopped",
        answer="blocked",
        blocked_reason="awaiting human decision",
        usage=UsageMetrics(),
    )
    cli._print_result(result, argparse.Namespace(json=False, debug=False))
    out = capsys.readouterr().out
    assert "[status: stopped]" in out
    assert "awaiting human decision" in out
