"""Deadline enforcement tests for the aa inter-module RPC v1 transport."""

from __future__ import annotations

import asyncio
from typing import Any

import pytest

from aa_sifter.app import ComputeSifter
from aa_sifter.rpc.deadline import (
    DEFAULT_TIMEOUT_MS,
    DeadlineError,
    dispatch_with_deadline,
    resolve_timeout_ms,
)
from aa_sifter.rpc.envelope import INVALID_PARAMS, UNAVAILABLE
from aa_sifter.rpc.service import SifterService
from tests.fixtures.providers import FakeProvider


class _StubHandler:
    def __init__(self, *, delay: float = 0.0, result: dict[str, Any] | None = None) -> None:
        self.delay = delay
        self.result = result if result is not None else {"jsonrpc": "2.0", "id": None, "result": {}}
        self.started = False
        self.cancelled = False

    async def handle(self, message: dict[str, Any]) -> dict[str, Any]:
        self.started = True
        try:
            if self.delay:
                await asyncio.sleep(self.delay)
        except asyncio.CancelledError:
            self.cancelled = True
            raise
        return {**self.result, "id": message.get("id")}


class _ExplodingHandler:
    async def handle(self, message: dict[str, Any]) -> dict[str, Any]:
        raise RuntimeError("boom")


# -- timeout resolution ----------------------------------------------------


def test_resolve_timeout_uses_default_when_absent() -> None:
    assert resolve_timeout_ms({}) == DEFAULT_TIMEOUT_MS


def test_resolve_timeout_reads_aa_block() -> None:
    assert resolve_timeout_ms({"aa": {"timeoutMs": 1500}}) == 1500


def test_resolve_timeout_reads_snake_case_alias() -> None:
    assert resolve_timeout_ms({"aa": {"timeout_ms": 250}}) == 250


def test_resolve_timeout_reads_top_level() -> None:
    assert resolve_timeout_ms({"timeoutMs": 90}) == 90


def test_resolve_timeout_treats_null_as_absent() -> None:
    assert resolve_timeout_ms({"aa": {"timeoutMs": None}}) == DEFAULT_TIMEOUT_MS


@pytest.mark.parametrize("value", [0, -1, "soon", True, [], {}])
def test_resolve_timeout_rejects_invalid_values(value: object) -> None:
    with pytest.raises(DeadlineError) as excinfo:
        resolve_timeout_ms({"aa": {"timeoutMs": value}})

    assert excinfo.value.code == INVALID_PARAMS


def test_deadline_error_renders_envelope() -> None:
    error = DeadlineError(INVALID_PARAMS, "bad deadline")

    assert error.as_error("rpc_1") == {
        "jsonrpc": "2.0",
        "id": "rpc_1",
        "error": {"code": INVALID_PARAMS, "message": "bad deadline"},
    }


# -- dispatch --------------------------------------------------------------


async def test_dispatch_returns_result_within_deadline() -> None:
    handler = _StubHandler()

    response = await dispatch_with_deadline(handler, {"id": "rpc_ok", "aa": {"timeoutMs": 1000}})

    assert response["id"] == "rpc_ok"
    assert "result" in response


async def test_dispatch_fails_closed_on_timeout() -> None:
    handler = _StubHandler(delay=0.5)

    response = await dispatch_with_deadline(handler, {"id": "rpc_slow", "aa": {"timeoutMs": 10}})

    assert response["error"]["code"] == UNAVAILABLE
    assert response["error"]["data"] == {"retryable": True, "timeoutMs": 10}
    assert handler.cancelled is True


async def test_dispatch_honours_explicit_deadline_seconds() -> None:
    handler = _StubHandler(delay=0.05)

    response = await dispatch_with_deadline(handler, {"id": "rpc_ok", "aa": {"timeoutMs": 250}})

    assert "result" in response


async def test_dispatch_rejects_invalid_deadline() -> None:
    handler = _StubHandler()

    response = await dispatch_with_deadline(handler, {"id": "rpc_bad", "aa": {"timeoutMs": 0}})

    assert response["error"]["code"] == INVALID_PARAMS
    assert handler.started is False


async def test_dispatch_passes_through_handler_error_envelope() -> None:
    handler = _StubHandler(result={"jsonrpc": "2.0", "id": None, "error": {"code": -32601}})

    response = await dispatch_with_deadline(handler, {"id": "rpc_err"})

    assert response["error"]["code"] == -32601


async def test_dispatch_propagates_unexpected_handler_exception() -> None:
    with pytest.raises(RuntimeError, match="boom"):
        await dispatch_with_deadline(_ExplodingHandler(), {"id": "rpc_boom"})


# -- integration: no partial state -----------------------------------------


async def test_timed_out_generate_leaves_no_idempotent_state(config) -> None:
    provider = FakeProvider(delay=0.5)
    service = SifterService(ComputeSifter(config, provider=provider))
    message = {
        "jsonrpc": "2.0",
        "id": "rpc_gen",
        "method": "sifter.generate",
        "params": {
            "tier": "local",
            "messages": [{"role": "user", "content": "hello"}],
            "idempotencyKey": "gen-timeout",
        },
        "aa": {"rpcVersion": "1.0", "timeoutMs": 10},
    }

    response = await dispatch_with_deadline(service, message)

    assert response["error"]["code"] == UNAVAILABLE
    assert service.idempotency.size == 0

    provider.delay = 0.0
    retry = await dispatch_with_deadline(service, message)

    assert "result" in retry
    assert provider.local_calls == 2
