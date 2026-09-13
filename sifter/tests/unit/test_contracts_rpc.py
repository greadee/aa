"""Contracts bridge and RPC service tests (inter-module JSON-RPC v1)."""

from __future__ import annotations

import pytest

from aa_sifter.app import ComputeSifter
from aa_sifter.contracts import (
    ContractError,
    contracts_available,
    make_memory_record,
    make_route_response,
    validate,
)
from aa_sifter.rpc.service import SifterService
from tests.fixtures.providers import FakeProvider

pytestmark = pytest.mark.skipif(
    not contracts_available(), reason="aa_contracts bindings are not on the path"
)


def _request(prompt: str = "Write unit tests for a small utility function.") -> dict:
    return {
        "contractVersion": "1.0",
        "kind": "route_request",
        "id": "req_test_1",
        "prompt": prompt,
    }


def _service(config, provider: FakeProvider | None = None) -> SifterService:
    return SifterService(ComputeSifter(config, provider=provider or FakeProvider()))


def test_route_response_validates_against_contract(config):
    service = _service(config)

    response = service.route(_request())

    validate(response)
    assert response["requestId"] == "req_test_1"
    assert response["route"] in {"local", "hybrid", "cloud", "blocked"}
    assert response["decisionLevel"] in {"routine", "significant", "major", "critical"}
    assert isinstance(response["requiresHumanApproval"], bool)


def test_route_is_deterministic(config):
    service = _service(config)

    first = service.route(_request())
    second = service.route(_request())

    assert first["route"] == second["route"]
    assert first["decisionLevel"] == second["decisionLevel"]
    assert first["executorTier"] == second["executorTier"]
    assert first["reasons"] == second["reasons"]


def test_route_reports_human_approval_for_major_decisions(config):
    service = _service(config)

    response = service.route(
        _request("Replace authentication with OAuth while preserving old sessions.")
    )

    assert response["decisionLevel"] in {"major", "critical"}
    assert response["requiresHumanApproval"] is True


def test_health_reports_providers(config):
    health = _service(config).health()

    assert health["available"] is True
    tiers = {provider["tier"] for provider in health["providers"]}
    assert tiers == {"local", "expert"}


async def test_generate_returns_text_and_is_idempotent(config):
    provider = FakeProvider()
    service = _service(config, provider)
    params = {
        "tier": "local",
        "messages": [{"role": "user", "content": "hello"}],
        "idempotencyKey": "gen-1",
    }

    first = await service.generate(params)
    second = await service.generate(params)

    assert first["text"] == "local answer"
    assert first == second
    assert len(provider.calls) == 1


async def test_handle_envelope_roundtrip(config):
    service = _service(config)
    message = {
        "jsonrpc": "2.0",
        "id": "rpc_1",
        "method": "sifter.route",
        "params": _request(),
        "aa": {"rpcVersion": "1.0"},
    }

    response = await service.handle(message)

    assert response["jsonrpc"] == "2.0"
    assert response["id"] == "rpc_1"
    assert "result" in response


async def test_handle_rejects_unknown_rpc_major(config):
    service = _service(config)
    message = {
        "jsonrpc": "2.0",
        "id": "rpc_2",
        "method": "sifter.health",
        "params": {},
        "aa": {"rpcVersion": "2.0"},
    }

    response = await service.handle(message)

    assert response["error"]["code"] == "aa.incompatible"


async def test_handle_unknown_method(config):
    service = _service(config)

    response = await service.handle(
        {"jsonrpc": "2.0", "id": "rpc_3", "method": "sifter.nope", "params": {}}
    )

    assert response["error"]["code"] == -32601


async def test_handle_invalid_route_request(config):
    service = _service(config)
    bad = {"contractVersion": "1.0", "kind": "route_request", "id": "req_bad"}

    response = await service.handle(
        {"jsonrpc": "2.0", "id": "rpc_4", "method": "sifter.route", "params": bad}
    )

    assert response["error"]["code"] == -32602


def test_route_response_rejects_invalid_enum():
    with pytest.raises(ContractError):
        make_route_response(
            _request(),
            route="nowhere",
            decision_level="routine",
            requires_human_approval=False,
        )


def test_memory_record_candidate_validates():
    record = make_memory_record(summary="aa-sifter routed a task locally.")

    validate(record)
    assert record["lifecycle"] == "CANDIDATE"
    assert record["content"]["summary"]
