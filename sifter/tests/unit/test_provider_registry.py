from __future__ import annotations

import pytest

from aa_sifter.models.provider import ModelRegistry, ProviderError
from tests.fixtures.providers import FakeProvider


def test_cloud_model_prefers_ollama_provider():
    primary = FakeProvider()
    other = FakeProvider()
    registry = ModelRegistry({"ollama": primary, "other": other})
    assert registry.provider_for("deepseek-v4.1-flash:cloud") is primary


def test_single_provider_is_default():
    provider = FakeProvider()
    registry = ModelRegistry({"ollama": provider})
    assert registry.provider_for("some-unknown-model") is provider


def test_default_named_provider_used():
    primary = FakeProvider()
    other = FakeProvider()
    registry = ModelRegistry({"ollama": primary, "other": other})
    assert registry.provider_for("x", default="other") is other


def test_missing_provider_raises():
    registry = ModelRegistry({"ollama": FakeProvider(), "other": FakeProvider()})
    with pytest.raises(ProviderError):
        registry.provider_for("x", default="missing")


def test_get_missing_raises():
    registry = ModelRegistry({"ollama": FakeProvider()})
    with pytest.raises(ProviderError):
        registry.get("nope")


@pytest.mark.asyncio
async def test_register_and_aclose():
    closed: list[str] = []

    class Tracking(FakeProvider):
        async def aclose(self) -> None:
            closed.append("closed")

    registry = ModelRegistry({})
    tracker = Tracking()
    registry.register(tracker)
    assert registry.get("ollama") is tracker
    await registry.aclose()
    assert closed == ["closed"]
