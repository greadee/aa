"""Durable idempotency tests for the aa inter-module RPC v1 transport."""

from __future__ import annotations

from pathlib import Path

import pytest

from aa_sifter.app import ComputeSifter
from aa_sifter.rpc.envelope import CONFLICT
from aa_sifter.rpc.errors import ConflictError
from aa_sifter.rpc.idempotency import (
    DEFAULT_WINDOW_SECONDS,
    IdempotencyStore,
    request_fingerprint,
)
from aa_sifter.rpc.service import SifterService
from tests.fixtures.providers import FakeProvider


class _Clock:
    def __init__(self, now: float = 1_000.0) -> None:
        self.now = now

    def __call__(self) -> float:
        return self.now


# -- store -----------------------------------------------------------------


def test_default_window_is_bounded() -> None:
    assert DEFAULT_WINDOW_SECONDS > 0


def test_window_must_be_positive() -> None:
    with pytest.raises(ValueError):
        IdempotencyStore(window_seconds=0)


def test_fingerprint_is_order_independent() -> None:
    assert request_fingerprint({"a": 1, "b": 2}) == request_fingerprint({"b": 2, "a": 1})


def test_fingerprint_tracks_content() -> None:
    assert request_fingerprint({"a": 1}) != request_fingerprint({"a": 2})


def test_remember_and_resolve_same_request() -> None:
    store = IdempotencyStore()

    store.remember("k", "fp", {"text": "cached"})

    assert store.size == 1
    assert store.resolve("k", "fp") == {"text": "cached"}
    assert store.get("k") == {"text": "cached"}


def test_resolve_unknown_key_returns_none() -> None:
    assert IdempotencyStore().resolve("missing", "fp") is None


def test_resolve_conflicting_request_raises() -> None:
    store = IdempotencyStore()
    store.remember("k", "fp-a", {"text": "cached"})

    with pytest.raises(ConflictError) as excinfo:
        store.resolve("k", "fp-b")

    assert excinfo.value.code == CONFLICT
    assert excinfo.value.data == {"retryable": False, "idempotencyKey": "k"}


def test_expired_record_is_evicted() -> None:
    clock = _Clock()
    store = IdempotencyStore(window_seconds=10, clock=clock)
    store.remember("k", "fp", {"text": "cached"})

    clock.now += 11

    assert store.resolve("k", "fp") is None
    assert store.size == 0


def test_store_is_durable_across_instances(tmp_path: Path) -> None:
    path = tmp_path / "idempotency.jsonl"
    first = IdempotencyStore(path)
    first.remember("k", "fp", {"text": "cached"})

    reopened = IdempotencyStore(path)

    assert reopened.resolve("k", "fp") == {"text": "cached"}


def test_reopened_store_drops_expired_records(tmp_path: Path) -> None:
    path = tmp_path / "idempotency.jsonl"
    clock = _Clock()
    IdempotencyStore(path, window_seconds=10, clock=clock).remember("k", "fp", {"text": "x"})

    clock.now += 11

    assert IdempotencyStore(path, window_seconds=10, clock=clock).size == 0


def test_prune_compacts_durable_log(tmp_path: Path) -> None:
    clock = _Clock()
    path = tmp_path / "idempotency.jsonl"
    store = IdempotencyStore(path, window_seconds=10, clock=clock)
    store.remember("a", "fp", {"text": "x"})
    store.remember("b", "fp", {"text": "y"})
    assert len(path.read_text(encoding="utf-8").splitlines()) == 2

    clock.now += 11
    removed = store.prune()

    assert removed == 2
    assert store.size == 0
    assert path.read_text(encoding="utf-8") == ""


def test_store_ignores_malformed_lines(tmp_path: Path) -> None:
    path = tmp_path / "idempotency.jsonl"
    path.write_text('not json\n{"key": "k"}\n\n', encoding="utf-8")

    assert IdempotencyStore(path).size == 0


# -- service integration ---------------------------------------------------


def _service(config, provider=None, idempotency=None) -> SifterService:
    return SifterService(
        ComputeSifter(config, provider=provider or FakeProvider()),
        idempotency=idempotency,
    )


def _params(key: str, text: str = "hello") -> dict:
    return {
        "tier": "local",
        "messages": [{"role": "user", "content": text}],
        "idempotencyKey": key,
    }


async def test_generate_replay_does_not_call_provider_again(config) -> None:
    provider = FakeProvider()
    service = _service(config, provider)

    first = await service.generate(_params("gen-1"))
    second = await service.generate(_params("gen-1"))

    assert first == second
    assert provider.local_calls == 1
    assert service.idempotency.size == 1


async def test_generate_conflicts_on_reused_key_with_different_request(config) -> None:
    provider = FakeProvider()
    service = _service(config, provider)
    await service.generate(_params("gen-1", "hello"))

    response = await service.handle(
        {
            "jsonrpc": "2.0",
            "id": "rpc_conflict",
            "method": "sifter.generate",
            "params": _params("gen-1", "something else"),
        }
    )

    assert response["error"]["code"] == CONFLICT
    assert provider.local_calls == 1


async def test_generate_uses_durable_store(config, tmp_path: Path) -> None:
    path = tmp_path / "idempotency.jsonl"
    provider = FakeProvider()
    await _service(config, provider, IdempotencyStore(path)).generate(_params("gen-1"))

    fresh_provider = FakeProvider()
    replayed = await _service(config, fresh_provider, IdempotencyStore(path)).generate(
        _params("gen-1")
    )

    assert replayed["text"] == "local answer"
    assert fresh_provider.local_calls == 0


async def test_expired_key_executes_again(config) -> None:
    clock = _Clock()
    store = IdempotencyStore(window_seconds=10, clock=clock)
    provider = FakeProvider()
    service = _service(config, provider, store)
    await service.generate(_params("gen-1"))

    clock.now += 11
    await service.generate(_params("gen-1"))

    assert provider.local_calls == 2
