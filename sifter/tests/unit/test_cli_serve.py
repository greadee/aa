"""``aa-sifter serve`` entry-point tests."""

from __future__ import annotations

import asyncio
import shutil
import tempfile
from collections.abc import Iterator
from pathlib import Path

import pytest

from aa_sifter.app import ComputeSifter
from aa_sifter.cli.main import _serve_options, build_parser
from aa_sifter.cli.serve_commands import (
    ServeOptions,
    build_server,
    default_idempotency_path,
    resolve_serve_endpoint,
    serve,
)
from aa_sifter.rpc.client import SifterClient
from aa_sifter.rpc.endpoint import SocketEndpoint
from tests.fixtures.providers import FakeProvider


@pytest.fixture
def runtime_dir() -> Iterator[Path]:
    directory = Path(tempfile.mkdtemp(prefix="aa-rpc-"))
    try:
        yield directory
    finally:
        shutil.rmtree(directory, ignore_errors=True)


def _socket(runtime_dir: Path) -> Path:
    return runtime_dir / "aa-sifter-v1.sock"


# -- configuration ---------------------------------------------------------


def test_parser_parses_serve_options() -> None:
    args = build_parser().parse_args(
        ["serve", "--socket", "/tmp/aa.sock", "--store-id", "store-a", "--timeout-ms", "1500"]
    )

    options = _serve_options(args)

    assert options.socket == "/tmp/aa.sock"
    assert options.store_id == "store-a"
    assert options.timeout_ms == 1500


def test_resolve_endpoint_honours_socket_override() -> None:
    endpoint = resolve_serve_endpoint(ServeOptions(socket="/tmp/aa-sifter-v1.sock"))

    assert endpoint.kind == "unix"
    assert endpoint.address == "/tmp/aa-sifter-v1.sock"


def test_resolve_endpoint_detects_named_pipe_override() -> None:
    endpoint = resolve_serve_endpoint(ServeOptions(socket="\\\\.\\pipe\\aa-sifter-v1"))

    assert endpoint.kind == "pipe"


def test_default_idempotency_path_sits_beside_database(config) -> None:
    path = default_idempotency_path(config)

    assert path.name == "idempotency.jsonl"
    assert path.parent == Path(config.resolved_database_path).parent


def test_build_server_uses_socket_override(config, runtime_dir: Path) -> None:
    aa_sifter = ComputeSifter(config, provider=FakeProvider())

    server = build_server(aa_sifter, ServeOptions(socket=str(_socket(runtime_dir))))

    assert server.endpoint.address == str(_socket(runtime_dir))


# -- serving ---------------------------------------------------------------


async def test_serve_refuses_when_already_served(config, runtime_dir: Path) -> None:
    aa_sifter = ComputeSifter(config, provider=FakeProvider())
    options = ServeOptions(socket=str(_socket(runtime_dir)))
    first = build_server(aa_sifter, options)
    await first.start()
    try:
        code = await serve(aa_sifter, options)
    finally:
        await first.stop()
        await aa_sifter.aclose()

    assert code == 1


async def test_serve_serves_and_stops_cleanly(config, runtime_dir: Path) -> None:
    aa_sifter = ComputeSifter(config, provider=FakeProvider())
    socket_path = _socket(runtime_dir)
    options = ServeOptions(socket=str(socket_path), store_id="store-a")
    stop = asyncio.Event()
    task = asyncio.create_task(serve(aa_sifter, options, stop=stop))
    try:
        for _ in range(200):
            if socket_path.exists():
                break
            await asyncio.sleep(0.01)
        endpoint = SocketEndpoint(address=str(socket_path), kind="unix")
        async with SifterClient(endpoint) as client:
            health = await client.health()
        assert health["available"] is True
    finally:
        stop.set()
        code = await asyncio.wait_for(task, timeout=5)
        await aa_sifter.aclose()

    assert code == 0
    assert not socket_path.exists()
