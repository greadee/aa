"""v1 error mapping tests for the aa inter-module RPC boundary."""

from __future__ import annotations

import pytest

from aa_sifter.contracts import ContractError
from aa_sifter.models.provider import ProviderError
from aa_sifter.routing.budget import BudgetExceeded
from aa_sifter.rpc.envelope import (
    APPROVAL_REQUIRED,
    BUDGET_EXCEEDED,
    CONFLICT,
    INTERNAL_ERROR,
    INVALID_PARAMS,
    UNAUTHORIZED,
    UNAVAILABLE,
)
from aa_sifter.rpc.errors import ConflictError, UnavailableError, map_exception
from aa_sifter.rpc.identity import IdentityError
from aa_sifter.rpc.service import SifterService
from aa_sifter.tasks.executor import ApprovalRequiredError


def _error(exc: BaseException, request_id: str = "rpc_e") -> dict:
    return map_exception(exc, request_id)


# -- domain mapping --------------------------------------------------------


def test_contract_error_maps_to_invalid_params() -> None:
    assert _error(ContractError("bad request"))["error"] == {
        "code": INVALID_PARAMS,
        "message": "bad request",
    }


def test_budget_exceeded_maps_to_governed_budget_error() -> None:
    exc = BudgetExceeded("budget spent", kind="max_cloud_cost", limit=1.0, used=2.0)

    error = _error(exc)["error"]

    assert error["code"] == BUDGET_EXCEEDED
    assert error["data"] == {
        "kind": "max_cloud_cost",
        "limit": 1.0,
        "used": 2.0,
        "retryable": False,
    }


def test_approval_required_maps_to_governed_approval_error() -> None:
    exc = ApprovalRequiredError("task-7", "major decision")

    error = _error(exc)["error"]

    assert error["code"] == APPROVAL_REQUIRED
    assert error["data"] == {
        "taskId": "task-7",
        "reason": "major decision",
        "retryable": False,
    }


@pytest.mark.parametrize(
    ("status", "code"),
    [
        (401, UNAUTHORIZED),
        (403, UNAUTHORIZED),
        (402, BUDGET_EXCEEDED),
        (409, CONFLICT),
    ],
)
def test_provider_status_maps_to_governed_code(status: int, code: str) -> None:
    error = _error(ProviderError("provider said no", status_code=status))["error"]

    assert error["code"] == code
    assert error["data"]["statusCode"] == status
    assert error["data"]["retryable"] is False


@pytest.mark.parametrize("status", [429, 500, 502, 503, 504])
def test_retryable_provider_status_is_retryable(status: int) -> None:
    error = _error(ProviderError("try later", status_code=status))["error"]

    assert error["code"] == UNAVAILABLE
    assert error["data"] == {"retryable": True, "statusCode": status}


def test_provider_error_without_status_is_unavailable() -> None:
    error = _error(ProviderError("cloud disabled"))["error"]

    assert error["code"] == UNAVAILABLE
    assert error["data"] == {"retryable": False}


def test_provider_error_retryable_flag_is_honoured() -> None:
    error = _error(ProviderError("timed out", retryable=True))["error"]

    assert error["code"] == UNAVAILABLE
    assert error["data"]["retryable"] is True


def test_conflict_error_carries_conflict_code() -> None:
    error = _error(ConflictError("duplicate key", data={"idempotencyKey": "k"}))["error"]

    assert error["code"] == CONFLICT
    assert error["data"] == {"retryable": False, "idempotencyKey": "k"}


def test_unavailable_error_defaults_to_retryable() -> None:
    assert _error(UnavailableError("down"))["error"]["data"] == {"retryable": True}


def test_unavailable_error_can_be_permanent() -> None:
    error = _error(UnavailableError("gone", retryable=False))["error"]

    assert error["data"] == {"retryable": False}


def test_rpc_error_renders_itself() -> None:
    error = _error(IdentityError(UNAUTHORIZED, "peer mismatch"))["error"]

    assert error["code"] == UNAUTHORIZED


def test_unknown_exception_maps_to_internal_error() -> None:
    assert _error(RuntimeError("unexpected"))["error"] == {
        "code": INTERNAL_ERROR,
        "message": "unexpected",
    }


# -- service integration ---------------------------------------------------


class _RaisingSifter:
    def __init__(self, exc: BaseException) -> None:
        self.exc = exc

    async def generate(self, messages, *, tier: str = "local"):
        raise self.exc


def _generate_message() -> dict:
    return {
        "jsonrpc": "2.0",
        "id": "rpc_e",
        "method": "sifter.generate",
        "params": {"tier": "local", "messages": [{"role": "user", "content": "hi"}]},
    }


async def test_service_maps_budget_exceeded() -> None:
    exc = BudgetExceeded("budget spent", kind="max_cloud_cost", limit=1.0, used=2.0)
    service = SifterService(_RaisingSifter(exc))

    response = await service.handle(_generate_message())

    assert response["error"]["code"] == BUDGET_EXCEEDED


async def test_service_maps_provider_unavailable() -> None:
    service = SifterService(_RaisingSifter(ProviderError("down", status_code=503)))

    response = await service.handle(_generate_message())

    assert response["error"]["code"] == UNAVAILABLE
    assert response["error"]["data"]["retryable"] is True


async def test_service_maps_approval_required() -> None:
    service = SifterService(_RaisingSifter(ApprovalRequiredError("task-1", "review")))

    response = await service.handle(_generate_message())

    assert response["error"]["code"] == APPROVAL_REQUIRED
    assert response["error"]["data"]["taskId"] == "task-1"
