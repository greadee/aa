"""Map sifter domain failures onto the aa inter-module RPC v1 error set.

The v1 error set is fixed: the JSON-RPC numeric codes plus the namespaced
``aa.*`` codes. Every failure that reaches the RPC boundary is rendered here, so
the kernel sees one governed vocabulary instead of provider-specific text. The
governed failures carry ``data.retryable`` so a caller knows whether a retry can
succeed.
"""

from __future__ import annotations

from typing import Any

from ..contracts import ContractError
from ..models.provider import ProviderError
from ..routing.budget import BudgetExceeded
from ..tasks.executor import ApprovalRequiredError
from .envelope import (
    APPROVAL_REQUIRED,
    BUDGET_EXCEEDED,
    CONFLICT,
    INTERNAL_ERROR,
    INVALID_PARAMS,
    UNAUTHORIZED,
    UNAVAILABLE,
    RpcError,
    failure,
)

_UNAUTHORIZED_STATUS = frozenset({401, 403})
_BUDGET_STATUS = frozenset({402})
_CONFLICT_STATUS = frozenset({409})
_RETRYABLE_STATUS = frozenset({429, 500, 502, 503, 504})


class ConflictError(RpcError):
    """A mutating request conflicts with state already recorded."""

    def __init__(self, message: str, *, data: dict[str, Any] | None = None) -> None:
        super().__init__(CONFLICT, message, data={"retryable": False, **(data or {})})


class UnavailableError(RpcError):
    """A required service or runtime is unavailable."""

    def __init__(
        self,
        message: str,
        *,
        retryable: bool = True,
        data: dict[str, Any] | None = None,
    ) -> None:
        super().__init__(UNAVAILABLE, message, data={"retryable": retryable, **(data or {})})


def _provider_error(exc: ProviderError, request_id: Any) -> dict[str, Any]:
    status = exc.status_code
    code: str = UNAVAILABLE
    retryable = exc.retryable
    if status in _UNAUTHORIZED_STATUS:
        code, retryable = UNAUTHORIZED, False
    elif status in _BUDGET_STATUS:
        code, retryable = BUDGET_EXCEEDED, False
    elif status in _CONFLICT_STATUS:
        code, retryable = CONFLICT, False
    elif status in _RETRYABLE_STATUS:
        retryable = True
    data: dict[str, Any] = {"retryable": retryable}
    if status is not None:
        data["statusCode"] = status
    return failure(request_id, code, str(exc), data=data)


def map_exception(exc: BaseException, request_id: Any = None) -> dict[str, Any]:
    """Render a mapped failure as the correct v1 error envelope.

    Anything already carrying a v1 code (identity, deadline, framing, conflict,
    explicit unavailability) renders itself. Known domain exceptions are mapped;
    everything else becomes a generic internal error and never leaks a stack.
    """
    if isinstance(exc, RpcError):
        return exc.as_error(request_id)
    if isinstance(exc, ContractError):
        return failure(request_id, INVALID_PARAMS, str(exc))
    if isinstance(exc, BudgetExceeded):
        return failure(
            request_id,
            BUDGET_EXCEEDED,
            str(exc),
            data={
                "kind": exc.kind,
                "limit": exc.limit,
                "used": exc.used,
                "retryable": False,
            },
        )
    if isinstance(exc, ApprovalRequiredError):
        return failure(
            request_id,
            APPROVAL_REQUIRED,
            str(exc),
            data={"taskId": exc.task_id, "reason": exc.reason, "retryable": False},
        )
    if isinstance(exc, ProviderError):
        return _provider_error(exc, request_id)
    return failure(request_id, INTERNAL_ERROR, str(exc))
