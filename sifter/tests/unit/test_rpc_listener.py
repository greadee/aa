"""Local socket listener tests: owner-only creation and attach-or-own."""

from __future__ import annotations

import contextlib
import os
import shutil
import stat
import tempfile
from collections.abc import Iterator
from pathlib import Path

import pytest

from aa_sifter.rpc.endpoint import (
    SERVICE_NAME,
    SocketEndpoint,
    default_endpoint,
    runtime_directory,
)
from aa_sifter.rpc.listener import LocalListener, Ownership, is_endpoint_live


@pytest.fixture
def runtime_dir() -> Iterator[Path]:
    directory = Path(tempfile.mkdtemp(prefix="aa-rpc-"))
    try:
        yield directory
    finally:
        shutil.rmtree(directory, ignore_errors=True)


async def _handler(reader, writer) -> None:
    with contextlib.suppress(OSError):
        await reader.read()
    writer.close()
    with contextlib.suppress(OSError):
        await writer.wait_closed()


def _endpoint(runtime_dir: Path, name: str = "aa-sifter-v1.sock") -> SocketEndpoint:
    return SocketEndpoint(address=str(runtime_dir / name), kind="unix")


# -- endpoint resolution ---------------------------------------------------


def test_default_endpoint_uses_absolute_xdg_runtime_dir() -> None:
    endpoint = default_endpoint(env={"XDG_RUNTIME_DIR": "/run/user/1000"})

    assert endpoint.kind == "unix"
    assert endpoint.address == "/run/user/1000/aa-sifter-v1.sock"


def test_default_endpoint_falls_back_to_per_user_temp_directory() -> None:
    endpoint = default_endpoint(env={}, uid=1000)

    assert endpoint.kind == "unix"
    assert endpoint.address.endswith("aa-sifter-v1.sock")
    assert f"{os.sep}aa-1000{os.sep}" in endpoint.address


def test_default_endpoint_ignores_relative_xdg_runtime_dir() -> None:
    endpoint = default_endpoint(env={"XDG_RUNTIME_DIR": "relative/run"}, uid=7)

    assert f"{os.sep}aa-7{os.sep}" in endpoint.address


def test_default_endpoint_on_windows_is_a_named_pipe() -> None:
    endpoint = default_endpoint(platform="win32")

    assert endpoint.kind == "pipe"
    assert endpoint.address == "\\\\.\\pipe\\aa-sifter-v1"


def test_runtime_directory_is_per_user() -> None:
    assert SERVICE_NAME == "sifter"
    assert runtime_directory(env={}, uid=1).endswith("aa-1")


def test_endpoint_rejects_unknown_kind() -> None:
    with pytest.raises(ValueError):
        SocketEndpoint(address="/tmp/x", kind="bogus")  # type: ignore[arg-type]


def test_endpoint_rejects_empty_address() -> None:
    with pytest.raises(ValueError):
        SocketEndpoint(address="", kind="unix")


# -- listener ownership ----------------------------------------------------


async def test_start_creates_owner_only_socket(runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    listener = LocalListener(endpoint)
    try:
        ownership = await listener.start(_handler)

        assert ownership is Ownership.OWNED
        assert listener.owned is True
        assert listener.serving is True
        assert stat.S_ISSOCK(os.stat(endpoint.address).st_mode)
        assert stat.S_IMODE(os.stat(endpoint.address).st_mode) == 0o600
    finally:
        await listener.stop()


async def test_stop_removes_owned_socket(runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    listener = LocalListener(endpoint)
    await listener.start(_handler)

    await listener.stop()

    assert not os.path.exists(endpoint.address)
    assert listener.ownership is None


async def test_second_listener_attaches_instead_of_binding(runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    first = LocalListener(endpoint)
    second = LocalListener(endpoint)
    try:
        assert await first.start(_handler) is Ownership.OWNED
        assert await second.start(_handler) is Ownership.ATTACHED
        assert second.owned is False
        assert second.serving is False
    finally:
        await second.stop()
        await first.stop()


async def test_attached_listener_stop_leaves_owner_socket(runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    first = LocalListener(endpoint)
    second = LocalListener(endpoint)
    try:
        await first.start(_handler)
        await second.start(_handler)

        await second.stop()

        assert os.path.exists(endpoint.address)
        assert await is_endpoint_live(endpoint) is True
    finally:
        await second.stop()
        await first.stop()


async def test_start_recovers_stale_socket_file(runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    Path(endpoint.address).write_bytes(b"stale owner")
    listener = LocalListener(endpoint)
    try:
        assert await listener.start(_handler) is Ownership.OWNED
        assert stat.S_ISSOCK(os.stat(endpoint.address).st_mode)
    finally:
        await listener.stop()


async def test_start_creates_private_runtime_directory(runtime_dir: Path) -> None:
    nested = runtime_dir / "run" / "aa-sifter-v1.sock"
    listener = LocalListener(SocketEndpoint(address=str(nested), kind="unix"))
    try:
        assert await listener.start(_handler) is Ownership.OWNED
        assert stat.S_IMODE(os.stat(nested.parent).st_mode) == 0o700
    finally:
        await listener.stop()


async def test_socket_mode_is_configurable(runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    listener = LocalListener(endpoint, socket_mode=0o640)
    try:
        await listener.start(_handler)
        assert stat.S_IMODE(os.stat(endpoint.address).st_mode) == 0o640
    finally:
        await listener.stop()


async def test_stop_is_idempotent(runtime_dir: Path) -> None:
    endpoint = _endpoint(runtime_dir)
    listener = LocalListener(endpoint)
    await listener.start(_handler)

    await listener.stop()
    await listener.stop()

    assert listener.ownership is None


async def test_pipe_binding_requires_platform_backend() -> None:
    listener = LocalListener(SocketEndpoint(address="\\\\.\\pipe\\aa-sifter-v1", kind="pipe"))

    with pytest.raises(NotImplementedError):
        await listener.start(_handler)


async def test_probe_missing_endpoint_is_not_live(runtime_dir: Path) -> None:
    assert await is_endpoint_live(_endpoint(runtime_dir)) is False


async def test_probe_pipe_endpoint_is_not_live() -> None:
    endpoint = SocketEndpoint(address="\\\\.\\pipe\\aa-sifter-v1", kind="pipe")

    assert await is_endpoint_live(endpoint) is False
