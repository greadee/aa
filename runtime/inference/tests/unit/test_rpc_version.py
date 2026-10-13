"""Version negotiation at the inference RPC boundary (aa inter-module RPC v1)."""

from __future__ import annotations

from aa_inference.app import ComputeInference
from aa_inference.rpc.envelope import is_compatible
from aa_inference.rpc.service import InferenceService
from tests.fixtures.providers import FakeProvider


def _service(config) -> InferenceService:
    return InferenceService(ComputeInference(config, provider=FakeProvider()))


def test_absent_versions_are_compatible() -> None:
    assert is_compatible(None) is True
    assert is_compatible({}) is True


def test_rpc_minor_is_compatible_major_is_not() -> None:
    assert is_compatible({"rpcVersion": "1.7"}) is True
    assert is_compatible({"rpcVersion": "2.0"}) is False


def test_contract_minor_is_compatible_major_is_not() -> None:
    assert is_compatible({"rpcVersion": "1.0", "contractVersion": "2.4"}) is True
    assert is_compatible({"rpcVersion": "1.0", "contractVersion": "1.0"}) is False


async def test_handle_rejects_unsupported_contract_major(config) -> None:
    service = _service(config)
    response = await service.handle(
        {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "inference.health",
            "aa": {"rpcVersion": "1.0", "contractVersion": "1.0"},
        }
    )
    assert response["error"]["code"] == "aa.incompatible"


def test_health_advertises_capabilities(config) -> None:
    service = _service(config)
    health = service.health()
    assert health["rpcVersion"] == "1.0"
    assert str(health["contractVersion"]).startswith("2")
    assert set(health["capabilities"]) >= {"route", "generate", "health", "recommend"}
