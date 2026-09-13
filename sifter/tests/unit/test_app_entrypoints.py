from __future__ import annotations

import pytest

from aa_sifter.models.config import SifterConfig
from aa_sifter.models.provider import Message, Tier
from tests.fixtures.helpers import make_sifter


@pytest.mark.asyncio
async def test_generate_local_entrypoint(config: SifterConfig, fake_provider, store):
    aa_sifter = make_sifter(config, fake_provider, store=store)
    result = await aa_sifter.generate([Message.user("hi")], tier="local")
    assert result.tier == Tier.LOCAL
    assert fake_provider.local_calls == 1
    assert fake_provider.cloud_calls == 0


@pytest.mark.asyncio
async def test_generate_expert_entrypoint(config: SifterConfig, fake_provider, store):
    aa_sifter = make_sifter(config, fake_provider, store=store)
    result = await aa_sifter.generate([Message.user("hi")], tier="expert")
    assert result.tier == Tier.EXPERT
    assert fake_provider.cloud_calls == 1


@pytest.mark.asyncio
async def test_as_inference_layer_and_aclose(config: SifterConfig, fake_provider, store):
    aa_sifter = make_sifter(config, fake_provider, store=store)
    assert aa_sifter.as_inference_layer() is aa_sifter
    await aa_sifter.aclose()


def test_usage_as_dict(config: SifterConfig):
    from aa_sifter.metrics.usage import UsageMetrics

    metrics = UsageMetrics(local_calls=2, cloud_calls=1)
    payload = metrics.as_dict()
    assert payload["local_calls"] == 2
    assert payload["cloud_calls"] == 1
