"""Server-loop tests for the aa inter-module RPC v1 transport."""

from __future__ import annotations

import asyncio
import contextlib
import shutil
import tempfile
from collections import deque
from collections.abc import Iterator
from pathlib import Path

import pytest

from aa_sifter.app import ComputeSifter
from aa_sifter.rpc.endpoint import SocketEndpoint
from aa_sifter.rpc.framing import FrameReader, encode_frame
from aa_sifter.rpc.listener import Ownership
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


def _service(config, provider=None) -> SifterService:
    return SifterService(ComputeSifter(config, provider=provider or FakeProvider()))


class _Client:
    def __init__(self, reader: asyncio.StreamReader, writer: asyncio.StreamWriter) -> None:
        self.reader = reader
        self.writer = writer
        self._framing = FrameReader()
        self._queue: deque[dict] = deque()

    @classmethod
    async def connect(cls, endpoint: SocketEndpoint) -> _Client:
        reader, writer = await asyncio.open_unix_connection(endpoint.address)
        return cls(reader, writer)

    async def send(self, message: dict) -> None:
        self.writer.write(encode_frame(message))
        await self.writer.drain()

    async def send_raw(self, data: bytes) -> None:
        self.writer.write(data)
        await self.writer.drain()

    async def recv(self) -> dict:
        while not self._queue:
            chunk = await self.reader.read(65536)
            if not chunk:
                raise EOFError
            self._queue.extend(self._framing.feed(chunk))
        return self._queue.popleft()

    async def close(self) -> None:
        self.writer.close()
        with contextlib.suppress(OSError):
            await self.writer.wait_closed()


def _hardware() -> dict:
    return {
        "gpu_name": "NVIDIA GeForce RTX 3080",
        "gpu_vendor": "NVIDIA",
        "gpu_vram_gb": 10.0,
        "system_ram_gb": 32.0,
        "source": "manual",
    }


async def test_server_round_trips_health(config, runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    server = RpcServer(_service(config), endpoint)
    assert await server.start() is Ownership.OWNED
    assert server.ready is True
    client = await _Client.connect(endpoint)
    try:
        await client.send({"jsonrpc": "2.0", "id": "h", "method": "sifter.health", "params": {}})

        response = await client.recv()

        assert response["id"] == "h"
        assert response["result"]["available"] is True
    finally:
        await client.close()
        await server.stop()


async def test_server_round_trips_route(config, runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    server = RpcServer(_service(config), endpoint)
    await server.start()
    client = await _Client.connect(endpoint)
    try:
        await client.send(
            {
                "jsonrpc": "2.0",
                "id": "r",
                "method": "sifter.route",
                "params": {
                    "contractVersion": "1.0",
                    "kind": "route_request",
                    "id": "req_1",
                    "prompt": "Write unit tests for a small utility function.",
                },
            }
        )

        response = await client.recv()

        assert response["result"]["route"] in {"local", "hybrid", "cloud", "blocked"}
    finally:
        await client.close()
        await server.stop()


async def test_server_round_trips_recommend(config, runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    server = RpcServer(_service(config), endpoint)
    await server.start()
    client = await _Client.connect(endpoint)
    try:
        await client.send(
            {
                "jsonrpc": "2.0",
                "id": "rec",
                "method": "sifter.recommend",
                "params": {"hardware": _hardware(), "goal": "coding"},
            }
        )

        response = await client.recv()

        assert response["result"]["recommendation"]["profile"]["local"]["model"]
    finally:
        await client.close()
        await server.stop()


async def test_server_returns_method_not_found(config, runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    server = RpcServer(_service(config), endpoint)
    await server.start()
    client = await _Client.connect(endpoint)
    try:
        await client.send({"jsonrpc": "2.0", "id": "x", "method": "sifter.nope", "params": {}})

        response = await client.recv()

        assert response["error"]["code"] == -32601
    finally:
        await client.close()
        await server.stop()


async def test_server_reports_parse_error_and_closes(config, runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    server = RpcServer(_service(config), endpoint)
    await server.start()
    client = await _Client.connect(endpoint)
    try:
        await client.send_raw(b"{not json}\n")

        response = await client.recv()

        assert response["error"]["code"] == -32700
        with pytest.raises(EOFError):
            await client.recv()
    finally:
        await client.close()
        await server.stop()


async def test_server_enforces_deadline(config, runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    server = RpcServer(_service(config, FakeProvider(delay=0.5)), endpoint)
    await server.start()
    client = await _Client.connect(endpoint)
    try:
        await client.send(
            {
                "jsonrpc": "2.0",
                "id": "g",
                "method": "sifter.generate",
                "params": {
                    "tier": "local",
                    "messages": [{"role": "user", "content": "hi"}],
                    "idempotencyKey": "gen-1",
                },
                "aa": {"rpcVersion": "1.0", "timeoutMs": 10},
            }
        )

        response = await client.recv()

        assert response["error"]["code"] == "aa.unavailable"
        assert response["error"]["data"]["retryable"] is True
    finally:
        await client.close()
        await server.stop()


async def test_server_ready_and_releases_socket_on_stop(config, runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    server = RpcServer(_service(config), endpoint)
    await server.start()

    assert server.ready is True
    assert Path(endpoint.address).exists()

    await server.stop()

    assert server.ready is False
    assert not Path(endpoint.address).exists()


async def test_server_attaches_to_live_owner(config, runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    first = RpcServer(_service(config), endpoint)
    second = RpcServer(_service(config), endpoint)
    try:
        assert await first.start() is Ownership.OWNED
        assert await second.start() is Ownership.ATTACHED

        assert second.ready is False
    finally:
        await second.stop()
        await first.stop()


async def test_server_graceful_shutdown_closes_connections(config, runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    server = RpcServer(_service(config), endpoint, grace_seconds=0.05)
    await server.start()
    client = await _Client.connect(endpoint)
    try:
        await server.stop()

        with pytest.raises(EOFError):
            await client.recv()
    finally:
        await client.close()


async def test_server_stop_is_idempotent(config, runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    server = RpcServer(_service(config), endpoint)
    await server.start()

    await server.stop()
    await server.stop()

    assert server.ready is False
