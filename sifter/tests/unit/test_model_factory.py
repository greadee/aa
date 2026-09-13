from __future__ import annotations

import pytest

from aa_sifter.models.config import SifterConfig
from aa_sifter.models.factory import build_provider, build_registry
from aa_sifter.models.provider import ProviderError
from tests.fixtures.providers import FakeProvider


def test_build_ollama_provider():
    provider = build_provider("ollama", SifterConfig(ollama_host="http://example:1234"))
    assert provider.name == "ollama"


def test_build_deepseek_provider():
    provider = build_provider("deepseek", SifterConfig())
    assert provider.name == "deepseek"


def test_build_openai_compatible_requires_endpoint():
    config = SifterConfig(expert_provider="openai-compatible", expert_endpoint=None)
    with pytest.raises(ProviderError):
        build_provider("openai-compatible", config)


def test_build_openai_compatible_with_endpoint():
    config = SifterConfig(
        expert_provider="openai-compatible", expert_endpoint="http://localhost:8000/v1"
    )
    provider = build_provider("openai-compatible", config)
    assert provider.name == "openai-compatible"


def test_unknown_provider_raises():
    with pytest.raises(ProviderError):
        build_provider("mystery", SifterConfig())


def test_registry_with_injected_provider():
    fake = FakeProvider()
    registry = build_registry(SifterConfig(), injected=fake)
    assert registry.get("ollama") is fake


def test_registry_builds_for_configured_providers():
    registry = build_registry(SifterConfig(local_provider="ollama", expert_provider="deepseek"))
    assert registry.get("ollama").name == "ollama"
    assert registry.get("deepseek").name == "deepseek"
