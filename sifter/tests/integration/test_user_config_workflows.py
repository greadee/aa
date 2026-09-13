from __future__ import annotations

from pathlib import Path

import pytest

from aa_sifter.app import ComputeSifter
from aa_sifter.catalog import ModelCatalog
from aa_sifter.cli import main as cli
from aa_sifter.config.defaults import default_application_config
from aa_sifter.config.resolve import load_runtime_config
from aa_sifter.config.schema import ModelTierConfig, UserGoal
from aa_sifter.config.store import load_application_config, save_application_config
from aa_sifter.config.validation import validate_profile
from aa_sifter.recommend.recommender import recommend_profile
from tests.fixtures.helpers import EASY_LOCAL_TASK, MAJOR_AUTH_TASK, make_sifter
from tests.fixtures.providers import FakeProvider


@pytest.mark.integration
def test_scenario_a_new_user_setup(isolated_config: Path, capsys):
    """No config -> setup -> profile saved -> runtime loads it."""
    assert cli.main(["setup"]) == 0
    capsys.readouterr()

    app = load_application_config(isolated_config)
    assert app.active_profile == "default"

    config, _ = load_runtime_config()
    assert config.local_model == "qwen3.5:9b"
    assert config.cloud_model == "deepseek-chat"
    assert config.expert_provider == "deepseek"
    assert config.routing_policy == "adaptive"


@pytest.mark.integration
def test_scenario_b_manual_expert_profile(isolated_config: Path):
    """Advanced user sets arbitrary provider/model pairs and runtime uses them."""
    app = default_application_config()
    app.profiles["default"].local = ModelTierConfig(
        provider="ollama", model="custom-local:7b", context_limit=16384
    )
    app.profiles["default"].expert = ModelTierConfig(
        provider="openai-compatible",
        model="custom-expert",
        endpoint="http://localhost:8000/v1",
        api_key_env="CUSTOM_KEY",
    )
    save_application_config(app, isolated_config)

    config, _ = load_runtime_config()
    assert config.local_model == "custom-local:7b"
    assert config.expert_provider == "openai-compatible"
    assert config.cloud_model == "custom-expert"

    fake = FakeProvider()
    aa_sifter = ComputeSifter(config, provider=fake, history=None)
    assert aa_sifter.local_config().name == "custom-local:7b"
    assert aa_sifter.cloud_config().name == "custom-expert"


@pytest.mark.integration
@pytest.mark.asyncio
async def test_scenario_c_unknown_model_accepted(isolated_config: Path, store):
    app = default_application_config()
    app.profiles["default"].local = ModelTierConfig(provider="ollama", model="future-model:12b")
    save_application_config(app, isolated_config)
    config, _ = load_runtime_config()

    result = validate_profile(
        app.profiles["default"], catalog=ModelCatalog(), environ={"DEEPSEEK_API_KEY": "x"}
    )
    assert result.ok is True
    assert any("no catalog metadata" in warning for warning in result.warnings)

    aa_sifter = make_sifter(config, FakeProvider(), store=store)
    run = await aa_sifter.run(EASY_LOCAL_TASK)
    assert run.status == "completed"


@pytest.mark.integration
def test_scenario_d_missing_secret_is_clear_error(isolated_config: Path):
    app = default_application_config()
    save_application_config(app, isolated_config)
    result = validate_profile(app.profiles["default"], environ={})
    assert result.ok is False
    assert any("DEEPSEEK_API_KEY" in error for error in result.errors)

    # Runtime resolution still works for local-only use and does not crash.
    config, _ = load_runtime_config()
    fake = FakeProvider()
    aa_sifter = ComputeSifter(config, provider=fake, history=None)
    assert aa_sifter is not None


@pytest.mark.integration
@pytest.mark.asyncio
async def test_scenario_e_local_only_never_calls_cloud(isolated_config: Path, store):
    goal = UserGoal(
        workload=["software development"],
        cloud_allowed=False,
        privacy_priority=1.0,
    )
    rec = recommend_profile(hardware=None, goal=goal, catalog=ModelCatalog())
    assert rec.profile.expert is None
    assert rec.profile.routing_policy == "local_only"

    app = default_application_config()
    app.profiles["privacy"] = rec.profile
    app.profiles["privacy"].name = "privacy"
    app.active_profile = "privacy"
    save_application_config(app, isolated_config)

    config, _ = load_runtime_config()
    assert config.cloud_allowed is False
    assert config.routing_policy == "local_only"

    provider = FakeProvider()
    aa_sifter = make_sifter(config, provider, store=store)
    result = await aa_sifter.run(EASY_LOCAL_TASK)
    assert result.status == "completed"
    assert provider.cloud_calls == 0

    major = await aa_sifter.run(MAJOR_AUTH_TASK)
    assert major.status in {"stopped", "failed"}
    assert provider.cloud_calls == 0
