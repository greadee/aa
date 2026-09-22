"""NDJSON JSON-RPC server loop for the aa inter-module RPC v1.

The server owns one per-user local socket (through the listener), reads
newline-delimited frames, dispatches each message under its deadline, and writes
one framed response per request. It reports readiness while it owns the socket
and shuts down gracefully: it stops accepting, drains in-flight requests for a
grace period, then cancels whatever remains.
"""

from __future__ import annotations

import asyncio
import contextlib

from .deadline import DEFAULT_TIMEOUT_MS, RpcHandler, dispatch_with_deadline
from .endpoint import SocketEndpoint
from .framing import FrameReader, FramingError, encode_frame
from .listener import LocalListener, Ownership

_READ_SIZE = 65536
_GRACE_SECONDS = 5.0


async def _close_writer(writer: asyncio.StreamWriter) -> None:
    writer.close()
    with contextlib.suppress(OSError):
        await writer.wait_closed()


class RpcServer:
    """Serve a :class:`~aa_sifter.rpc.service.SifterService` over one socket."""

    def __init__(
        self,
        service: RpcHandler,
        endpoint: SocketEndpoint,
        *,
        default_timeout_ms: int = DEFAULT_TIMEOUT_MS,
        listener: LocalListener | None = None,
        read_size: int = _READ_SIZE,
        grace_seconds: float = _GRACE_SECONDS,
    ) -> None:
        self.service = service
        self.endpoint = endpoint
        self.default_timeout_ms = default_timeout_ms
        self.read_size = read_size
        self.grace_seconds = grace_seconds
        self._listener = listener or LocalListener(endpoint)
        self._connections: set[asyncio.Task[None]] = set()
        self._stop = asyncio.Event()
        self._ready = False
        self._closing = False
        self.ownership: Ownership | None = None

    @property
    def ready(self) -> bool:
        """Whether this server owns the socket and is accepting requests."""
        return self._ready

    async def start(self) -> Ownership:
        """Bind (or attach to) the endpoint and begin serving."""
        self.ownership = await self._listener.start(self._handle_connection)
        self._ready = self.ownership is Ownership.OWNED
        return self.ownership

    async def wait_closed(self) -> None:
        """Block until :meth:`stop` has completed."""
        await self._stop.wait()

    async def stop(self) -> None:
        """Stop accepting, drain in-flight requests, and release the socket."""
        if self._stop.is_set():
            return
        self._ready = False
        self._closing = True
        # Let already-accepted connections start their handlers so they observe
        # the closing flag, then cancel the in-flight ones before we close the
        # listening socket.
        await asyncio.sleep(0)
        await asyncio.sleep(0)
        await self._drain()
        self._listener.close()
        await self._listener.wait_closed()
        self._stop.set()

    async def _drain(self) -> None:
        tasks = [task for task in self._connections if not task.done()]
        if not tasks:
            return
        _, pending = await asyncio.wait(tasks, timeout=self.grace_seconds)
        for task in pending:
            task.cancel()
        if pending:
            await asyncio.gather(*pending, return_exceptions=True)

    # -- connection loop ---------------------------------------------------
    async def _handle_connection(
        self,
        reader: asyncio.StreamReader,
        writer: asyncio.StreamWriter,
    ) -> None:
        task = asyncio.current_task()
        if task is not None:
            self._connections.add(task)
        framing = FrameReader()
        try:
            if self._closing:
                return
            while True:
                chunk = await reader.read(self.read_size)
                if not chunk:
                    break
                try:
                    messages = framing.feed(chunk)
                except FramingError as exc:
                    await self._respond(writer, exc.as_error(None))
                    break
                for message in messages:
                    response = await dispatch_with_deadline(
                        self.service,
                        message,
                        default_timeout_ms=self.default_timeout_ms,
                    )
                    await self._respond(writer, response)
        except (ConnectionError, OSError):
            pass
        finally:
            if task is not None:
                self._connections.discard(task)
            await _close_writer(writer)

    async def _respond(self, writer: asyncio.StreamWriter, message: dict) -> None:
        writer.write(encode_frame(message))
        await writer.drain()
