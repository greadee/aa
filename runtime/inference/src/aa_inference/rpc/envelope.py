"""JSON-RPC 2.0 envelope helpers for the aa inter-module RPC v1.

Framing is transport-agnostic: callers pass and receive one message object at a
time, so the kernel can host this over a named pipe or Unix socket without the
service knowing the transport.
"""

from __future__ import annotations

from typing import Any

RPC_VERSION = "1.0"

PARSE_ERROR = -32700
INVALID_REQUEST = -32600
METHOD_NOT_FOUND = -32601
INVALID_PARAMS = -32602
INTERNAL_ERROR = -32603

INCOMPATIBLE = "aa.incompatible"
STORE_MISMATCH = "aa.store_mismatch"
UNAUTHORIZED = "aa.unauthorized"
BUDGET_EXCEEDED = "aa.budget_exceeded"
APPROVAL_REQUIRED = "aa.approval_required"
UNAVAILABLE = "aa.unavailable"
CONFLICT = "aa.conflict"


def success(request_id: Any, result: Any) -> dict[str, Any]:
    return {"jsonrpc": "2.0", "id": request_id, "result": result}


def failure(
    request_id: Any,
    code: int | str,
    message: str,
    *,
    data: dict[str, Any] | None = None,
) -> dict[str, Any]:
    error: dict[str, Any] = {"code": code, "message": message}
    if data is not None:
        error["data"] = data
    return {"jsonrpc": "2.0", "id": request_id, "error": error}


class RpcError(Exception):
    """A failure that already carries an aa RPC v1 error code and data."""

    def __init__(
        self,
        code: int | str,
        message: str,
        *,
        data: dict[str, Any] | None = None,
    ) -> None:
        super().__init__(message)
        self.code = code
        self.message = message
        self.data = data

    def as_error(self, request_id: Any = None) -> dict[str, Any]:
        """Render this failure as a JSON-RPC error envelope."""
        return failure(request_id, self.code, self.message, data=self.data)


def is_compatible(aa: dict[str, Any] | None) -> bool:
    """A callee rejects an unsupported RPC major with ``aa.incompatible``."""
    if not aa:
        return True
    requested = str(aa.get("rpcVersion", RPC_VERSION))
    return requested.split(".")[0] == RPC_VERSION.split(".")[0]
