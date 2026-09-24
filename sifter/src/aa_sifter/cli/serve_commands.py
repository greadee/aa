"""``aa-sifter serve``: host the RPC service over the per-user local socket.

The command assembles the production runtime: the sifter service bound to a
store identity, a durable idempotency log, the local socket listener, and the
server loop. It reports readiness on stderr, shuts down gracefully on SIGINT or
SIGTERM, and refuses to start when another process already owns the socket.
"""

from __future__ import annotations

import asyncio
import contextlib
import signal
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from ..rpc.deadline import DEFAULT_TIMEOUT_MS
from ..rpc.endpoint import SERVICE_NAME, EndpointKind, SocketEndpoint, default_endpoint
from ..rpc.idempotency import DEFAULT_WINDOW_SECONDS, IdempotencyStore
from ..rpc.listener import Ownership
from ..rpc.server import RpcServer
from ..rpc.service import SifterService

DEFAULT_GRACE_SECONDS = 5.0


@dataclass(frozen=True, slots=True)
class ServeOptions:
    """Resolved ``serve`` configuration."""

    socket: str | None = None
    store_id: str | None = None
    idempotency_file: str | None = None
    idempotency_window: float | None = None
    timeout_ms: int = DEFAULT_TIMEOUT_MS
    grace_seconds: float = DEFAULT_GRACE_SECONDS


def resolve_serve_endpoint(options: ServeOptions) -> SocketEndpoint:
    """The endpoint to serve, honouring an explicit ``--socket`` override."""
    if options.socket:
        kind: EndpointKind = "pipe" if options.socket.startswith("\\\\") else "unix"
        return SocketEndpoint(address=options.socket, kind=kind)
    return default_endpoint(SERVICE_NAME)


def default_idempotency_path(config: Any) -> Path:
    """The durable idempotency log beside the sifter's database."""
    return Path(config.resolved_database_path).with_name("idempotency.jsonl")


def build_server(aa_sifter: Any, options: ServeOptions) -> RpcServer:
    """Assemble the hosted service from the sifter and the serve options."""
    endpoint = resolve_serve_endpoint(options)
    window = options.idempotency_window or DEFAULT_WINDOW_SECONDS
    path = options.idempotency_file or str(default_idempotency_path(aa_sifter.config))
    store = IdempotencyStore(path, window_seconds=window)
    service = SifterService(aa_sifter, store_id=options.store_id, idempotency=store)
    return RpcServer(
        service,
        endpoint,
        default_timeout_ms=options.timeout_ms,
        grace_seconds=options.grace_seconds,
    )


async def serve(
    aa_sifter: Any,
    options: ServeOptions,
    *,
    stop: asyncio.Event | None = None,
) -> int:
    """Serve until stopped, returning a process exit code."""
    try:
        server = build_server(aa_sifter, options)
    except (OSError, ValueError) as exc:
        print(f"aa-sifter: cannot configure server: {exc}", file=sys.stderr)
        return 1
    try:
        ownership = await server.start()
    except OSError as exc:
        print(f"aa-sifter: cannot bind {server.endpoint.address}: {exc}", file=sys.stderr)
        return 1
    if ownership is Ownership.ATTACHED:
        print(
            f"aa-sifter: {server.endpoint.address} is already being served",
            file=sys.stderr,
        )
        return 1

    stopper = stop if stop is not None else asyncio.Event()
    _install_signal_handlers(stopper)
    print(
        f"aa-sifter: serving {server.endpoint.address} (store={options.store_id or 'unbound'})",
        file=sys.stderr,
    )
    try:
        await _wait_until_stopped(server, stopper)
    finally:
        await server.stop()
    return 0


async def _wait_until_stopped(server: RpcServer, stop: asyncio.Event) -> None:
    waiters = {
        asyncio.ensure_future(server.wait_closed()),
        asyncio.ensure_future(stop.wait()),
    }
    _, pending = await asyncio.wait(waiters, return_when=asyncio.FIRST_COMPLETED)
    for task in pending:
        task.cancel()
    await asyncio.gather(*pending, return_exceptions=True)


def _install_signal_handlers(stop: asyncio.Event) -> None:
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        with contextlib.suppress(NotImplementedError, AttributeError, ValueError):
            loop.add_signal_handler(sig, stop.set)
