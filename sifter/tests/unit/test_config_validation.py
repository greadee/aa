from __future__ import annotations

from aa_sifter.catalog import ModelCatalog
from aa_sifter.config.defaults import default_profile
from aa_sifter.config.schema import HardwareProfile, ModelTierConfig, UserPreferenceConfig
from aa_sifter.config.validation import validate_profile


def test_default_profile_without_key_is_an_error():
    result = validate_profile(default_profile(), catalog=ModelCatalog(), environ={})
    assert result.ok is False
    assert any("DEEPSEEK_API_KEY" in error for error in result.errors)


def test_missing_key_is_warning_when_credentials_not_required():
    result = validate_profile(
        default_profile(), catalog=ModelCatalog(), environ={}, require_credentials=False
    )
    assert result.ok is True
    assert any("DEEPSEEK_API_KEY" in warning for warning in result.warnings)


def test_with_key_passes():
    result = validate_profile(
        default_profile(), catalog=ModelCatalog(), environ={"DEEPSEEK_API_KEY": "x"}
    )
    assert result.ok is True


def test_unknown_model_is_warning_not_error():
    profile = default_profile()
    profile.local.model = "future-model:12b"
    result = validate_profile(profile, catalog=ModelCatalog(), environ={"DEEPSEEK_API_KEY": "x"})
    assert result.ok is True
    assert any("no catalog metadata" in warning for warning in result.warnings)


def test_negative_context_is_error():
    profile = default_profile()
    profile.local.context_limit = -1
    result = validate_profile(profile, catalog=ModelCatalog(), environ={"DEEPSEEK_API_KEY": "x"})
    assert any("context_limit" in error for error in result.errors)


def test_unknown_routing_policy_is_error():
    profile = default_profile()
    profile.routing_policy = "warp_speed"
    result = validate_profile(profile, catalog=ModelCatalog(), environ={"DEEPSEEK_API_KEY": "x"})
    assert any("routing policy" in error for error in result.errors)


def test_local_only_profile_does_not_require_expert_key():
    profile = default_profile()
    profile.expert = None
    profile.preferences = UserPreferenceConfig(cloud_allowed=False)
    result = validate_profile(profile, catalog=ModelCatalog(), environ={})
    assert result.ok is True


def test_vram_warning_when_model_too_large():
    profile = default_profile()
    profile.local = ModelTierConfig(provider="ollama", model="qwen3.5:32b", context_limit=8192)
    hardware = HardwareProfile(gpu_name="RTX 3080", gpu_vram_gb=10.0)
    result = validate_profile(
        profile,
        catalog=ModelCatalog(),
        hardware=hardware,
        environ={"DEEPSEEK_API_KEY": "x"},
    )
    assert any("VRAM" in warning for warning in result.warnings)
