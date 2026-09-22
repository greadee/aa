"""Typed-client tests for the aa inter-module RPC v1 transport."""

from __future__ import annotations

import asyncio
import shutil
import tempfile
from collections.abc import AsyncIterator, Iterator
from pathlib import Path

import pytest

from aa_sifter.app import ComputeSifter
from aa_sifter.rpc.client import RpcClientError, RuntimeGateway, SifterClient
from aa_sifter.rpc.endpoint import SocketEndpoint
from aa_sifter.rpc.server import RpcServer
from aa_sifter.rpc.service import SifterService
from tests.fixtures.providers import FakeProvider


@pytest.fixture
def runtime_dir() -> Iterator[Path]:
    directory = Path(tempfile.mkdtemp(prefix="aa-rpc-"))
    try:
        yield directory
    finally:
        shutil.rmtree(directory, ignore_errors=True)


def _endpoint(runtime_dir: Path) -> SocketEndpoint:
    return SocketEndpoint(address=str(runtime_dir / "aa-sifter-v1.sock"), kind="unix")


def _server(config, endpoint: SocketEndpoint, provider=None, **kwargs) -> RpcServer:
    service = SifterService(ComputeSifter(config, provider=provider or FakeProvider()))
    return RpcServer(service, endpoint, **kwargs)


@pytest.fixture
async def endpoint(config, runtime_dir: Path) -> AsyncIterator[SocketEndpoint]:
    server = _server(config, _endpoint(runtime_dir))
    await server.start()
    try:
        yield server.endpoint
    finally:
        await server.stop()


async def test_client_health_round_trip(endpoint: SocketEndpoint) -> None:
    async with SifterClient(endpoint) as client:
        health = await client.health()

    assert health["available"] is True
    assert health["rpcVersion"] == "1.0"


async def test_client_route_round_trip(endpoint: SocketEndpoint) -> None:
    request = {
        "contractVersion": "1.0",
        "kind": "route_request",
        "id": "req_1",
        "prompt": "Write unit tests for a small utility function.",
    }

    async with SifterClient(endpoint) as client:
        response = await client.route(request)

    assert response["route"] in {"local", "hybrid", "cloud", "blocked"}
    assert response["requestId"] == "req_1"


async def test_client_recommend_round_trip(endpoint: SocketEndpoint) -> None:
    params = {
        "hardware": {
            "gpu_name": "NVIDIA GeForce RTX 3080",
            "gpu_vendor": "NVIDIA",
            "gpu_vram_gb": 10.0,
            "system_ram_gb": 32.0,
            "source": "manual",
        },
        "goal": "coding",
    }

    async with SifterClient(endpoint) as client:
        response = await client.recommend(params)

    assert response["recommendation"]["profile"]["local"]["model"]


async def test_client_generate_is_idempotent(config, runtime_dir: Path) -> None:
    provider = FakeProvider()
    server = _server(config, _endpoint(runtime_dir), provider)
    await server.start()
    params = {
        "tier": "local",
        "messages": [{"role": "user", "content": "hello"}],
        "idempotencyKey": "gen-1",
    }
    try:
        async with SifterClient(server.endpoint) as client:
            first = await client.generate(params)
            second = await client.generate(params)
    finally:
        await server.stop()

    assert first == second
    assert provider.local_calls == 1


async def test_client_raises_on_remote_error(endpoint: SocketEndpoint) -> None:
    async with SifterClient(endpoint) as client:
        with pytest.raises(RpcClientError) as excinfo:
            await client.request("sifter.nope", {})

    assert excinfo.value.code == -32601


async def test_client_raises_on_invalid_params(endpoint: SocketEndpoint) -> None:
    async with SifterClient(endpoint) as client:
        with pytest.raises(RpcClientError) as excinfo:
            await client.generate({"tier": "local", "messages": []})

    assert excinfo.value.code == -32602


async def test_client_surfaces_provider_error(config, runtime_dir: Path) -> None:
    server = _server(config, _endpoint(runtime_dir), FakeProvider(fail_local=True))
    await server.start()
    try:
        async with SifterClient(server.endpoint) as client:
            with pytest.raises(RpcClientError) as excinfo:
                await client.generate(
                    {
                        "tier": "local",
                        "messages": [{"role": "user", "content": "hi"}],
                    }
                )
    finally:
        await server.stop()

    assert excinfo.value.code == "aa.unavailable"
    assert excinfo.value.data["retryable"] is True


async def test_client_enforces_its_own_deadline(config, runtime_dir: Path) -> None:
    server = _server(
        config,
        _endpoint(runtime_dir),
        FakeProvider(delay=0.5),
        default_timeout_ms=5_000,
    )
    await server.start()
    try:
        async with SifterClient(server.endpoint, timeout_ms=10) as client:
            with pytest.raises(RpcClientError) as excinfo:
                await asyncio.wait_for(
                    client.generate(
                        {
                            "tier": "local",
                            "messages": [{"role": "user", "content": "hi"}],
                        }
                    ),
                    timeout=2,
                )
    finally:
        await server.stop()

    assert excinfo.value.code == "aa.unavailable"
    assert excinfo.value.data["timeoutMs"] == 10


async def test_client_requires_connection() -> None:
    endpoint = SocketEndpoint(address="/nonexistent/aa-sifter-v1.sock", kind="unix")
    client = SifterClient(endpoint)

    with pytest.raises(RuntimeError):
        await client.request("sifter.health")


async def test_client_satisfies_runtime_gateway(endpoint: SocketEndpoint) -> None:
    async with SifterClient(endpoint) as client:
        assert isinstance(client, RuntimeGateway)


async def test_client_context_manager_closes(endpoint: SocketEndpoint) -> None:
    client = SifterClient(endpoint)

    async with client:
        assert await client.health()

    with pytest.raises(RuntimeError):
        await client.request("sifter.health")
