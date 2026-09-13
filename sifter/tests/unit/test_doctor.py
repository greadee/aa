from __future__ import annotations

from pathlib import Path

from aa_sifter import doctor
from aa_sifter.config.defaults import default_application_config
from aa_sifter.config.schema import ModelTierConfig


class _FakeResponse:
    def __init__(self, status_code: int, payload: dict | None = None):
        self.status_code = status_code
        self._payload = payload or {}

    def json(self) -> dict:
        return self._payload


def test_ollama_installed_parses_tags(monkeypatch):
    monkeypatch.setattr(
        doctor.httpx,
        "get",
        lambda url, timeout: _FakeResponse(200, {"models": [{"name": "qwen3.5:9b"}]}),
    )
    assert doctor.ollama_installed() == {"qwen3.5:9b"}


def test_ollama_installed_handles_failure(monkeypatch):
    def boom(*args, **kwargs):
        raise RuntimeError("no server")

    monkeypatch.setattr(doctor.httpx, "get", boom)
    assert doctor.ollama_installed() is None


def test_ollama_installed_non_200(monkeypatch):
    monkeypatch.setattr(doctor.httpx, "get", lambda url, timeout: _FakeResponse(500))
    assert doctor.ollama_installed() is None


def test_live_expert_check_ollama(monkeypatch):
    monkeypatch.setattr(doctor.httpx, "get", lambda url, timeout: _FakeResponse(200))
    ok, detail = doctor.live_expert_check(ModelTierConfig(provider="ollama", model="qwen3.5:9b"))
    assert ok is True
    assert "Ollama" in detail


def test_live_expert_check_openai_compatible(monkeypatch):
    monkeypatch.setattr(doctor.httpx, "post", lambda *a, **k: _FakeResponse(200))
    ok, detail = doctor.live_expert_check(
        ModelTierConfig(
            provider="deepseek",
            model="deepseek-chat",
            endpoint="https://api.deepseek.com/v1",
            api_key_env="DEEPSEEK_API_KEY",
        ),
        environ={"DEEPSEEK_API_KEY": "x"},
    )
    assert ok is True
    assert detail == "reachable"


def test_live_expert_check_api_error(monkeypatch):
    monkeypatch.setattr(doctor.httpx, "post", lambda *a, **k: _FakeResponse(401))
    ok, detail = doctor.live_expert_check(
        ModelTierConfig(provider="deepseek", model="deepseek-chat"),
        environ={"DEEPSEEK_API_KEY": "x"},
    )
    assert ok is False
    assert "401" in detail


def test_build_report_ollama_unreachable(tmp_path: Path):
    app = default_application_config()
    report = doctor.build_doctor_report(
        app,
        config_path=tmp_path / "config.toml",
        ollama_installed_fn=lambda: None,
        environ={},
    )
    names = {check["name"] for check in report["checks"]}
    assert "profile" in names
    assert "configuration" in names
    assert report["ok"] is True  # unreachable ollama is optional


def test_build_report_live_path(tmp_path: Path):
    app = default_application_config()
    report = doctor.build_doctor_report(
        app,
        config_path=tmp_path / "config.toml",
        live=True,
        ollama_installed_fn=lambda: {"qwen3.5:9b"},
        live_check_fn=lambda expert: (True, "reachable"),
        environ={"DEEPSEEK_API_KEY": "x"},
    )
    connectivity = next(c for c in report["checks"] if c["name"] == "expert connectivity")
    assert connectivity["ok"] is True


def test_build_report_missing_profile(tmp_path: Path):
    app = default_application_config()
    app.profiles = {}
    report = doctor.build_doctor_report(app, config_path=tmp_path / "c.toml")
    assert report["ok"] is False
