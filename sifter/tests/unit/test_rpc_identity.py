"""Peer and store identity tests for the aa inter-module RPC v1 transport."""

from __future__ import annotations

import asyncio
import contextlib
import os
import shutil
import socket
import tempfile
from collections.abc import Iterator
from pathlib import Path

import pytest

from aa_sifter.app import ComputeSifter
from aa_sifter.rpc.endpoint import SocketEndpoint
from aa_sifter.rpc.envelope import STORE_MISMATCH, UNAUTHORIZED
from aa_sifter.rpc.identity import (
    STORE_ID_FIELD,
    IdentityError,
    StoreBinding,
    peer_credentials,
    requested_store_id,
    verify_peer,
)
from aa_sifter.rpc.listener import LocalListener
from aa_sifter.rpc.service import SifterService
from tests.fixtures.providers import FakeProvider

_POSIX_UID = hasattr(os, "getuid")
_PEER_CREDENTIALS_SUPPORTED = _POSIX_UID and (
    hasattr(socket, "SO_PEERCRED") or hasattr(socket.socket, "getpeereid")
)


@pytest.fixture
def runtime_dir() -> Iterator[Path]:
    directory = Path(tempfile.mkdtemp(prefix="aa-rpc-"))
    try:
        yield directory
    finally:
        shutil.rmtree(directory, ignore_errors=True)


def _endpoint(runtime_dir: Path):
    return SocketEndpoint(address=str(runtime_dir / "aa-sifter-v1.sock"), kind="unix")


async def _close(writer: asyncio.StreamWriter) -> None:
    writer.close()
    with contextlib.suppress(OSError):
        await writer.wait_closed()


# -- store identity --------------------------------------------------------


def test_store_binding_accepts_matching_store() -> None:
    StoreBinding("store-a").check("store-a")


def test_store_binding_rejects_mismatch() -> None:
    with pytest.raises(IdentityError) as excinfo:
        StoreBinding("store-a").check("store-b")

    assert excinfo.value.code == STORE_MISMATCH
    assert excinfo.value.data == {"storeId": "store-a", "requestedStoreId": "store-b"}


def test_store_binding_unbound_service_accepts_any_store() -> None:
    StoreBinding().check("store-b")


def test_store_binding_ignores_unbound_caller() -> None:
    StoreBinding("store-a").check(None)


def test_requested_store_id_reads_aa_block() -> None:
    assert requested_store_id({"aa": {STORE_ID_FIELD: "store-a"}}) == "store-a"


def test_requested_store_id_reads_top_level() -> None:
    assert requested_store_id({STORE_ID_FIELD: "store-a"}) == "store-a"


def test_requested_store_id_absent_or_invalid() -> None:
    assert requested_store_id({}) is None
    assert requested_store_id({"aa": {STORE_ID_FIELD: ""}}) is None
    assert requested_store_id({"aa": {STORE_ID_FIELD: 5}}) is None


def test_identity_error_renders_envelope() -> None:
    error = IdentityError(UNAUTHORIZED, "denied", data={"peerUid": 9})

    assert error.as_error("rpc_x") == {
        "jsonrpc": "2.0",
        "id": "rpc_x",
        "error": {"code": UNAUTHORIZED, "message": "denied", "data": {"peerUid": 9}},
    }


def test_identity_error_omits_empty_data() -> None:
    assert "data" not in IdentityError(UNAUTHORIZED, "denied").as_error(None)["error"]


# -- peer identity ---------------------------------------------------------


@pytest.mark.skipif(not _PEER_CREDENTIALS_SUPPORTED, reason="OS cannot report peer uid")
def test_peer_credentials_report_current_user() -> None:
    left, right = socket.socketpair()
    try:
        identity = peer_credentials(left)

        assert identity.verified is True
        assert identity.uid == os.getuid()
    finally:
        left.close()
        right.close()


@pytest.mark.skipif(not _PEER_CREDENTIALS_SUPPORTED, reason="OS cannot report peer uid")
def test_verify_peer_accepts_owner() -> None:
    left, right = socket.socketpair()
    try:
        identity = verify_peer(left, expected_uid=os.getuid())

        assert identity.uid == os.getuid()
    finally:
        left.close()
        right.close()


@pytest.mark.skipif(not _PEER_CREDENTIALS_SUPPORTED, reason="OS cannot report peer uid")
def test_verify_peer_rejects_other_user() -> None:
    left, right = socket.socketpair()
    try:
        with pytest.raises(IdentityError) as excinfo:
            verify_peer(left, expected_uid=os.getuid() + 1)

        assert excinfo.value.code == UNAUTHORIZED
        assert excinfo.value.data["peerUid"] == os.getuid()
    finally:
        left.close()
        right.close()


class _NoCredentialsSocket:
    def getsockopt(self, *args: object) -> bytes:
        raise OSError("peer credentials unsupported")

    def __getattr__(self, name: str) -> object:
        raise AttributeError(name)


def test_verify_peer_falls_back_to_directory_ownership() -> None:
    identity = verify_peer(_NoCredentialsSocket(), expected_uid=123456)  # type: ignore[arg-type]

    assert identity.verified is False


# -- listener enforcement --------------------------------------------------


async def test_listener_serves_owner_peer(runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    called = asyncio.Event()

    async def handler(reader: asyncio.StreamReader, writer: asyncio.StreamWriter) -> None:
        called.set()
        await _close(writer)

    listener = LocalListener(endpoint)
    try:
        await listener.start(handler)
        _, writer = await asyncio.open_unix_connection(endpoint.address)
        writer.write(b"\n")
        await writer.drain()
        await _close(writer)

        await asyncio.wait_for(called.wait(), timeout=2)
        assert called.is_set()
    finally:
        await listener.stop()


@pytest.mark.skipif(not _PEER_CREDENTIALS_SUPPORTED, reason="OS cannot report peer uid")
async def test_listener_rejects_foreign_peer(runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    called = asyncio.Event()

    async def handler(reader: asyncio.StreamReader, writer: asyncio.StreamWriter) -> None:
        called.set()
        await _close(writer)

    listener = LocalListener(endpoint, expected_uid=os.getuid() + 1)
    try:
        await listener.start(handler)
        reader, writer = await asyncio.open_unix_connection(endpoint.address)

        assert await asyncio.wait_for(reader.read(), timeout=2) == b""
        await asyncio.sleep(0)
        assert not called.is_set()
        await _close(writer)
    finally:
        await listener.stop()


# -- service store wiring --------------------------------------------------


def _rpc_service(config, store_id: str | None = None) -> SifterService:
    return SifterService(ComputeSifter(config, provider=FakeProvider()), store_id=store_id)


async def test_service_rejects_store_mismatch(config) -> None:
    service = _rpc_service(config, store_id="store-a")

    response = await service.handle(
        {
            "jsonrpc": "2.0",
            "id": "rpc_store",
            "method": "sifter.health",
            "params": {},
            "aa": {"rpcVersion": "1.0", STORE_ID_FIELD: "store-b"},
        }
    )

    assert response["error"]["code"] == STORE_MISMATCH
    assert response["error"]["data"]["requestedStoreId"] == "store-b"


async def test_service_accepts_matching_store(config) -> None:
    service = _rpc_service(config, store_id="store-a")

    response = await service.handle(
        {
            "jsonrpc": "2.0",
            "id": "rpc_store",
            "method": "sifter.health",
            "params": {},
            "aa": {"rpcVersion": "1.0", STORE_ID_FIELD: "store-a"},
        }
    )

    assert response["result"]["available"] is True


async def test_service_accepts_unbound_store(config) -> None:
    service = _rpc_service(config)

    response = await service.handle(
        {"jsonrpc": "2.0", "id": "rpc_store", "method": "sifter.health", "params": {}}
    )

    assert response["result"]["available"] is True
