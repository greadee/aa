"""Security tests over the RPC transport: identity, owner-only, and redaction.

These exercise the guarantees the v1 spec places on the boundary itself: the
socket is owner-only, a foreign store or peer is rejected, a cloud-disabled
profile produces zero expert traffic, and no secret reaches a provider on an
expert-tier call made through the transport.
"""

from __future__ import annotations

import os
import shutil
import socket
import stat
import tempfile
from collections.abc import Iterator
from pathlib import Path

import pytest

from aa_sifter.app import ComputeSifter
from aa_sifter.rpc.client import RpcClientError, SifterClient
from aa_sifter.rpc.endpoint import SocketEndpoint
from aa_sifter.rpc.listener import LocalListener
from aa_sifter.rpc.server import RpcServer
from aa_sifter.rpc.service import SifterService
from tests.fixtures.providers import FakeProvider

SECRET = "password = hunter2secret"
_PEER_CREDENTIALS_SUPPORTED = hasattr(os, "getuid") and (
    hasattr(socket, "SO_PEERCRED") or hasattr(socket.socket, "getpeereid")
)


@pytest.fixture
def runtime_dir() -> Iterator[Path]:
    directory = Path(tempfile.mkdtemp(prefix="aa-rpc-"))
    try:
        yield directory
    finally:
        shutil.rmtree(directory, ignore_errors=True)


def _endpoint(runtime_dir: Path) -> SocketEndpoint:
    return SocketEndpoint(address=str(runtime_dir / "aa-sifter-v1.sock"), kind="unix")


def _contents(provider: FakeProvider) -> str:
    return "\n".join(message.content for call in provider.messages for message in call)


async def _started(
    config,
    endpoint: SocketEndpoint,
    *,
    provider: FakeProvider | None = None,
    store_id: str | None = None,
    listener: LocalListener | None = None,
) -> RpcServer:
    service = SifterService(
        ComputeSifter(config, provider=provider or FakeProvider()),
        store_id=store_id,
    )
    server = RpcServer(service, endpoint, listener=listener)
    await server.start()
    return server


# -- owner-only socket -----------------------------------------------------


async def test_served_socket_is_owner_only(config, runtime_dir: Path) -> None:
    server = await _started(config, _endpoint(runtime_dir))
    try:
        mode = stat.S_IMODE(os.stat(server.endpoint.address).st_mode)
    finally:
        await server.stop()

    assert mode == 0o600


# -- store identity --------------------------------------------------------


async def test_store_mismatch_is_rejected_over_transport(config, runtime_dir: Path) -> None:
    server = await _started(config, _endpoint(runtime_dir), store_id="store-a")
    try:
        async with SifterClient(server.endpoint, store_id="store-b") as client:
            with pytest.raises(RpcClientError) as excinfo:
                await client.health()
    finally:
        await server.stop()

    assert excinfo.value.code == "aa.store_mismatch"


async def test_matching_store_is_accepted_over_transport(config, runtime_dir: Path) -> None:
    server = await _started(config, _endpoint(runtime_dir), store_id="store-a")
    try:
        async with SifterClient(server.endpoint, store_id="store-a") as client:
            assert (await client.health())["available"] is True
    finally:
        await server.stop()


# -- peer identity ---------------------------------------------------------


@pytest.mark.skipif(not _PEER_CREDENTIALS_SUPPORTED, reason="OS cannot report peer uid")
async def test_foreign_peer_never_reaches_the_service(config, runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    provider = FakeProvider()
    listener = LocalListener(endpoint, expected_uid=os.getuid() + 1)
    server = await _started(config, endpoint, provider=provider, listener=listener)
    try:
        async with SifterClient(endpoint) as client:
            with pytest.raises(ConnectionError):
                await client.health()
    finally:
        await server.stop()

    assert provider.local_calls == 0


# -- redaction over the transport ------------------------------------------


async def test_expert_generate_redacts_secret_over_transport(config, runtime_dir: Path) -> None:
    secure = config.model_copy(update={"cloud_allowed": True, "redact_secrets": True})
    provider = FakeProvider()
    server = await _started(secure, _endpoint(runtime_dir), provider=provider)
    try:
        async with SifterClient(server.endpoint) as client:
            result = await client.generate(
                {
                    "tier": "expert",
                    "messages": [{"role": "user", "content": SECRET}],
                }
            )
    finally:
        await server.stop()

    assert result["text"] == "cloud answer"
    assert provider.cloud_calls == 1
    sent = _contents(provider)
    assert "hunter2secret" not in sent
    assert "[REDACTED:" in sent


async def test_local_only_profile_yields_zero_expert_traffic(config, runtime_dir: Path) -> None:
    local_only = config.model_copy(update={"cloud_allowed": False, "redact_secrets": True})
    provider = FakeProvider()
    server = await _started(local_only, _endpoint(runtime_dir), provider=provider)
    try:
        async with SifterClient(server.endpoint) as client:
            with pytest.raises(RpcClientError) as excinfo:
                await client.generate(
                    {
                        "tier": "expert",
                        "messages": [{"role": "user", "content": "deploy the thing"}],
                    }
                )
    finally:
        await server.stop()

    assert excinfo.value.code == "aa.unavailable"
    assert provider.cloud_calls == 0


async def test_local_generate_does_not_leak_or_redact_locally(config, runtime_dir: Path) -> None:
    provider = FakeProvider()
    server = await _started(config, _endpoint(runtime_dir), provider=provider)
    try:
        async with SifterClient(server.endpoint) as client:
            await client.generate(
                {
                    "tier": "local",
                    "messages": [{"role": "user", "content": SECRET}],
                }
            )
    finally:
        await server.stop()

    assert "hunter2secret" in _contents(provider)
