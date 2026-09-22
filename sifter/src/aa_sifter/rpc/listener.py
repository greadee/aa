"""Local socket listener with owner-only creation and attach-or-own semantics.

The v1 transport gives every per-user service exactly one owner. A process that
wants to serve first checks whether a live owner already listens on the
endpoint: if so it *attaches* (the caller decides whether to become a client or
refuse to start) instead of binding a second socket. Otherwise it takes
ownership, recovering a stale socket file left by a crashed owner, creates the
socket owner-only, and removes it again on shutdown.

Only the owner ever unlinks the socket; an attached process never touches a file
it does not own.
"""

from __future__ import annotations

import asyncio
import contextlib
import os
from collections.abc import Awaitable, Callable
from enum import StrEnum

from .endpoint import SocketEndpoint

ConnectionHandler = Callable[[asyncio.StreamReader, asyncio.StreamWriter], Awaitable[None]]


class Ownership(StrEnum):
    """Whether a listener took the endpoint or found a live owner already."""

    OWNED = "owned"
    ATTACHED = "attached"


async def is_endpoint_live(endpoint: SocketEndpoint) -> bool:
    """Probe an endpoint for a live owner without consuming a real request."""
    if endpoint.kind != "unix":
        return False
    path = endpoint.address
    if not os.path.exists(path):
        return False
    try:
        _, writer = await asyncio.open_unix_connection(path)
    except OSError:
        return False
    writer.close()
    with contextlib.suppress(OSError):
        await writer.wait_closed()
    return True


def _remove_socket_file(path: str) -> None:
    with contextlib.suppress(FileNotFoundError):
        os.unlink(path)


class LocalListener:
    """Bind (or attach to) one per-user Unix domain socket endpoint."""

    def __init__(
        self,
        endpoint: SocketEndpoint,
        *,
        directory_mode: int = 0o700,
        socket_mode: int = 0o600,
        backlog: int = 16,
    ) -> None:
        self.endpoint = endpoint
        self.directory_mode = directory_mode
        self.socket_mode = socket_mode
        self.backlog = backlog
        self._server: asyncio.AbstractServer | None = None
        self._ownership: Ownership | None = None

    @property
    def ownership(self) -> Ownership | None:
        """``None`` before :meth:`start`, then the outcome of the last start."""
        return self._ownership

    @property
    def owned(self) -> bool:
        return self._ownership is Ownership.OWNED

    @property
    def serving(self) -> bool:
        """Whether this process actually bound the endpoint."""
        return self._server is not None

    async def start(self, handler: ConnectionHandler) -> Ownership:
        """Take the endpoint, or report that a live owner already holds it."""
        if self.endpoint.kind == "unix":
            return await self._start_unix(handler)
        return await self._start_pipe(handler)

    async def stop(self) -> None:
        """Close an owned server and remove the socket file it created."""
        server, self._server = self._server, None
        if server is not None:
            server.close()
            with contextlib.suppress(OSError):
                await server.wait_closed()
        if self._ownership is Ownership.OWNED and self.endpoint.kind == "unix":
            _remove_socket_file(self.endpoint.address)
        self._ownership = None

    # -- transports --------------------------------------------------------
    async def _start_unix(self, handler: ConnectionHandler) -> Ownership:
        path = self.endpoint.address
        if await is_endpoint_live(self.endpoint):
            self._ownership = Ownership.ATTACHED
            return self._ownership

        self._ensure_directory(path)
        _remove_socket_file(path)
        try:
            self._server = await asyncio.start_unix_server(handler, path=path, backlog=self.backlog)
        except OSError:
            # Another owner won the race between the probe and the bind.
            if await is_endpoint_live(self.endpoint):
                self._ownership = Ownership.ATTACHED
                return self._ownership
            raise
        os.chmod(path, self.socket_mode)
        self._ownership = Ownership.OWNED
        return self._ownership

    async def _start_pipe(self, handler: ConnectionHandler) -> Ownership:
        raise NotImplementedError(
            "Windows named-pipe hosting is specified by aa RPC v1 but needs a "
            "platform backend; use a Unix domain socket endpoint on this platform"
        )

    def _ensure_directory(self, path: str) -> None:
        directory = os.path.dirname(path)
        if not directory or os.path.isdir(directory):
            return
        os.makedirs(directory, mode=self.directory_mode, exist_ok=True)
        with contextlib.suppress(OSError):
            os.chmod(directory, self.directory_mode)
