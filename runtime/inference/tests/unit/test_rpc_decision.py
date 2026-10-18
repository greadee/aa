"""The RPC execution path consumes the allocator's decision; it never re-decides."""

from __future__ import annotations

from aa_inference.app import ComputeInference
from aa_inference.rpc.envelope import BUDGET_EXCEEDED, INVALID_PARAMS
from aa_inference.rpc.service import InferenceService
from tests.fixtures.providers import FakeProvider


def _service(config, provider=None) -> InferenceService:
    return InferenceService(ComputeInference(config, provider=provider or FakeProvider()))


def _message(provider: FakeProvider, decision=None) -> dict:
    params = {
        "messages": [{"role": "user", "content": "hello"}],
        "idempotencyKey": "dec-1",
    }
    if decision is not None:
        params["routingDecision"] = decision
    return {"jsonrpc": "2.0", "id": "rpc_dec", "method": "inference.generate", "params": params}


async def test_generate_requires_a_decision(config) -> None:
    provider = FakeProvider()
    response = await _service(config, provider).handle(_message(provider))
    assert response["error"]["code"] == INVALID_PARAMS
    assert provider.local_calls == 0


async def test_generate_rejects_a_skewed_decision(config) -> None:
    provider = FakeProvider()
    response = await _service(config, provider).handle(
        _message(provider, {"locality": "cloud", "tier": "local"})
    )
    assert response["error"]["code"] == INVALID_PARAMS
    assert provider.local_calls == 0


async def test_blocked_decision_never_calls_a_provider(config) -> None:
    provider = FakeProvider()
    response = await _service(config, provider).handle(
        _message(
            provider,
            {"locality": "cloud", "tier": "expert", "blocked": True, "reason": "budget"},
        )
    )
    assert response["error"]["code"] == BUDGET_EXCEEDED
    assert provider.cloud_calls == 0
    assert provider.local_calls == 0
