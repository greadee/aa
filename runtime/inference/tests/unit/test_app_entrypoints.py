from __future__ import annotations

import pytest

from aa_inference.models.config import InferenceConfig
from aa_inference.models.provider import Message, Tier
from tests.fixtures.helpers import make_inference


@pytest.mark.asyncio
async def test_generate_local_entrypoint(config: InferenceConfig, fake_provider, store):
    aa_inference = make_inference(config, fake_provider, store=store)
    result = await aa_inference.generate([Message.user("hi")], tier="local")
    assert result.tier == Tier.LOCAL
    assert fake_provider.local_calls == 1
    assert fake_provider.cloud_calls == 0


@pytest.mark.asyncio
async def test_generate_expert_entrypoint(config: InferenceConfig, fake_provider, store):
    aa_inference = make_inference(config, fake_provider, store=store)
    result = await aa_inference.generate([Message.user("hi")], tier="expert")
    assert result.tier == Tier.EXPERT
    assert fake_provider.cloud_calls == 1


@pytest.mark.asyncio
async def test_as_inference_layer_and_aclose(config: InferenceConfig, fake_provider, store):
    aa_inference = make_inference(config, fake_provider, store=store)
    assert aa_inference.as_inference_layer() is aa_inference
    await aa_inference.aclose()


def test_usage_as_dict(config: InferenceConfig):
    from aa_inference.metrics.usage import UsageMetrics

    metrics = UsageMetrics(local_calls=2, cloud_calls=1)
    payload = metrics.as_dict()
    assert payload["local_calls"] == 2
    assert payload["cloud_calls"] == 1
