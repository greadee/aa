"""Typed client for the aa inter-module RPC v1 transport.

The kernel reaches the sifter only through this governed boundary: the client
frames requests, attaches the caller identity and deadline, and turns a v1 error
response into an exception. :class:`RuntimeGateway` is the seam the kernel's
production runtime adapter is written against; the scripted fake stays
test-only.
"""

from __future__ import annotations

import asyncio
import contextlib
from collections import deque
from collections.abc import Mapping
from typing import Any, Protocol, runtime_checkable
from uuid import uuid4

from .deadline import DEFAULT_TIMEOUT_MS
from .endpoint import SocketEndpoint
from .envelope import RPC_VERSION, RpcError
from .framing import FrameReader, encode_frame

_READ_SIZE = 65536


class RpcClientError(RpcError):
    """A v1 error returned by the remote service."""


@runtime_checkable
class RuntimeGateway(Protocol):
    """The operations a kernel runtime adapter needs from the sifter."""

    async def health(self) -> dict[str, Any]: ...

    async def route(self, request: Mapping[str, Any]) -> dict[str, Any]: ...

    async def generate(self, params: Mapping[str, Any]) -> dict[str, Any]: ...


class SifterClient:
    """A typed, serialized client for one sifter socket."""

    def __init__(
        self,
        endpoint: SocketEndpoint,
        *,
        caller: str = "kernel",
        caller_instance_id: str | None = None,
        timeout_ms: int = DEFAULT_TIMEOUT_MS,
        store_id: str | None = None,
        rpc_version: str = RPC_VERSION,
    ) -> None:
        self.endpoint = endpoint
        self.caller = caller
        self.caller_instance_id = caller_instance_id or f"inst_{uuid4().hex}"
        self.timeout_ms = timeout_ms
        self.store_id = store_id
        self.rpc_version = rpc_version
        self._reader: asyncio.StreamReader | None = None
        self._writer: asyncio.StreamWriter | None = None
        self._framing = FrameReader()
        self._queue: deque[dict[str, Any]] = deque()
        self._lock = asyncio.Lock()

    async def __aenter__(self) -> SifterClient:
        await self.connect()
        return self

    async def __aexit__(self, *exc_info: object) -> None:
        await self.close()

    async def connect(self) -> SifterClient:
        self._reader, self._writer = await asyncio.open_unix_connection(self.endpoint.address)
        return self

    async def close(self) -> None:
        writer, self._writer = self._writer, None
        self._reader = None
        if writer is not None:
            writer.close()
            with contextlib.suppress(OSError):
                await writer.wait_closed()

    # -- calls -------------------------------------------------------------
    async def request(
        self,
        method: str,
        params: Mapping[str, Any] | None = None,
        *,
        timeout_ms: int | None = None,
    ) -> Any:
        """Send one request and return its result, or raise :class:`RpcClientError`."""
        async with self._lock:
            return await self._request(method, params or {}, timeout_ms)

    async def health(self) -> dict[str, Any]:
        return await self.request("sifter.health")

    async def route(self, request: Mapping[str, Any]) -> dict[str, Any]:
        return await self.request("sifter.route", request)

    async def generate(self, params: Mapping[str, Any]) -> dict[str, Any]:
        return await self.request("sifter.generate", params)

    async def recommend(self, params: Mapping[str, Any]) -> dict[str, Any]:
        return await self.request("sifter.recommend", params)

    # -- internals ---------------------------------------------------------
    async def _request(
        self,
        method: str,
        params: Mapping[str, Any],
        timeout_ms: int | None,
    ) -> Any:
        if self._reader is None or self._writer is None:
            raise RuntimeError("sifter client is not connected")
        deadline = self.timeout_ms if timeout_ms is None else timeout_ms
        request_id = f"rpc_{uuid4().hex}"
        message = {
            "jsonrpc": "2.0",
            "id": request_id,
            "method": method,
            "params": params,
            "aa": self._aa(deadline),
        }
        self._writer.write(encode_frame(message))
        await self._writer.drain()
        try:
            response = await asyncio.wait_for(self._recv(), timeout=deadline / 1000)
        except TimeoutError as exc:
            raise RpcClientError(
                "aa.unavailable",
                "request exceeded its deadline",
                data={"retryable": True, "timeoutMs": deadline},
            ) from exc
        error = response.get("error")
        if error is not None:
            raise RpcClientError(
                error.get("code"),
                str(error.get("message", "")),
                data=error.get("data"),
            )
        return response.get("result")

    async def _recv(self) -> dict[str, Any]:
        while not self._queue:
            if self._reader is None:
                raise RuntimeError("sifter client is not connected")
            chunk = await self._reader.read(_READ_SIZE)
            if not chunk:
                raise ConnectionError("sifter connection closed")
            self._queue.extend(self._framing.feed(chunk))
        return self._queue.popleft()

    def _aa(self, timeout_ms: int) -> dict[str, Any]:
        aa: dict[str, Any] = {
            "rpcVersion": self.rpc_version,
            "caller": self.caller,
            "callerInstanceId": self.caller_instance_id,
            "timeoutMs": timeout_ms,
        }
        if self.store_id is not None:
            aa["storeId"] = self.store_id
        return aa
