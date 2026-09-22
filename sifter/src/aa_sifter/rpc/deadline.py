"""Server-side deadlines for the aa inter-module RPC v1.

Every request carries ``aa.timeoutMs``. The server enforces it: a request that
does not finish in time is cancelled and answered with ``aa.unavailable`` and
``data.retryable: true``, and never leaves partial state behind. Cancellation
propagates into the handler, so an in-flight method stops instead of continuing
in the background.
"""

from __future__ import annotations

import asyncio
from collections.abc import Mapping
from typing import Any, Protocol

from .envelope import INVALID_PARAMS, UNAVAILABLE, failure

DEFAULT_TIMEOUT_MS = 30_000
TIMEOUT_FIELD = "timeoutMs"
_TIMEOUT_ALIASES = ("timeoutMs", "timeout_ms")


class DeadlineError(Exception):
    """The request stated a deadline that cannot be honoured."""

    def __init__(
        self, code: int | str, message: str, *, data: dict[str, Any] | None = None
    ) -> None:
        super().__init__(message)
        self.code = code
        self.message = message
        self.data = data

    def as_error(self, request_id: Any = None) -> dict[str, Any]:
        """Render this failure as a JSON-RPC error envelope."""
        return failure(request_id, self.code, self.message, data=self.data)


class RpcHandler(Protocol):
    """Anything with the sifter service's dispatch surface."""

    async def handle(self, message: dict[str, Any]) -> dict[str, Any]: ...


def _field(message: Mapping[str, Any]) -> Any:
    aa = message.get("aa")
    if isinstance(aa, Mapping):
        for key in _TIMEOUT_ALIASES:
            if key in aa:
                return aa[key]
    for key in _TIMEOUT_ALIASES:
        if key in message:
            return message[key]
    return None


def resolve_timeout_ms(
    message: Mapping[str, Any],
    *,
    default: int = DEFAULT_TIMEOUT_MS,
) -> int:
    """Read and validate ``aa.timeoutMs``, falling back to the server default."""
    value = _field(message)
    if value is None:
        return default
    if isinstance(value, bool) or not isinstance(value, (int, float)) or value <= 0:
        raise DeadlineError(INVALID_PARAMS, "aa.timeoutMs must be a positive number")
    return int(value)


async def dispatch_with_deadline(
    handler: RpcHandler,
    message: Mapping[str, Any],
    *,
    default_timeout_ms: int = DEFAULT_TIMEOUT_MS,
) -> dict[str, Any]:
    """Dispatch one request under its deadline, failing closed on timeout."""
    request_id = message.get("id")
    try:
        timeout_ms = resolve_timeout_ms(message, default=default_timeout_ms)
    except DeadlineError as exc:
        return exc.as_error(request_id)
    try:
        return await asyncio.wait_for(handler.handle(dict(message)), timeout=timeout_ms / 1000)
    except TimeoutError:
        return failure(
            request_id,
            UNAVAILABLE,
            "request exceeded its deadline",
            data={"retryable": True, "timeoutMs": timeout_ms},
        )
