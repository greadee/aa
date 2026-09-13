from __future__ import annotations

from pathlib import Path
from typing import Any

import pytest

from aa_sifter.history.sqlite import SqliteHistoryStore
from aa_sifter.models.config import SifterConfig
from tests.fixtures.providers import FakeProvider


def pytest_collection_modifyitems(config: Any, items: list[Any]) -> None:
    """Auto-mark tests placed under tests/unit so `pytest -m unit` works."""
    for item in items:
        if "unit" in Path(str(item.fspath)).parts:
            item.add_marker(pytest.mark.unit)


@pytest.fixture
def config(tmp_path: Path) -> SifterConfig:
    return SifterConfig.from_env(
        dotenv_path=tmp_path / "missing.env",
        database_path=str(tmp_path / "aa_sifter.db"),
        non_interactive=True,
        debug=False,
        local_model="qwen3.5:9b",
        cloud_model="deepseek-v4.1-flash:cloud",
        max_cloud_cost_per_task=1.0,
    )


@pytest.fixture
def store(config: SifterConfig) -> SqliteHistoryStore:
    history = SqliteHistoryStore(config.resolved_database_path)
    yield history
    history.close()


@pytest.fixture
def fake_provider() -> FakeProvider:
    return FakeProvider()


_CONFIG_ENV_VARS = (
    "LOCAL_MODEL",
    "CLOUD_MODEL",
    "LOCAL_PROVIDER",
    "EXPERT_PROVIDER",
    "EXPERT_ENDPOINT",
    "EXPERT_API_KEY_ENV",
    "SIFTER_LOCAL_MODEL",
    "SIFTER_CLOUD_MODEL",
    "DEEPSEEK_API_KEY",
    "OPENAI_API_KEY",
    "CLOUD_ALLOWED",
    "SIFTER_CONFIG",
)


@pytest.fixture
def isolated_config(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Path:
    """Isolate the config file and database, and clear legacy env vars."""
    config_path = tmp_path / "config.toml"
    monkeypatch.setenv("SIFTER_CONFIG", str(config_path))
    monkeypatch.setenv("SIFTER_DATABASE_PATH", str(tmp_path / "aa_sifter.db"))
    for var in _CONFIG_ENV_VARS:
        if var != "SIFTER_CONFIG":
            monkeypatch.delenv(var, raising=False)
    return config_path
