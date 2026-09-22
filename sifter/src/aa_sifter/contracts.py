"""Bridge between aa-contracts v1 objects and aa-sifter types.

This is the only module that knows the cross-module contract shapes. It never
holds project state: it validates incoming contract objects, converts a route
request into ``run`` inputs, and builds contract-shaped responses and memory
candidates. Validation is applied against the bundled JSON Schema at the
boundary and against the generated ``aa_contracts`` bindings when they are
importable (they are on the path in CI and in the monorepo).
"""

from __future__ import annotations

from typing import Any
from uuid import uuid4

from .contract_schema import SchemaError, validate_contract

try:  # pragma: no cover - availability depends on deployment path
    import aa_contracts as _aa_contracts
    from aa_contracts import CONTRACT_VERSION as _CONTRACT_VERSION

    CONTRACTS_AVAILABLE = True
except Exception:  # noqa: BLE001 - optional dependency at runtime
    _aa_contracts = None
    _CONTRACT_VERSION = "1.0"
    CONTRACTS_AVAILABLE = False

CONTRACT_VERSION = _CONTRACT_VERSION

ROUTE_VALUES = {"local", "hybrid", "cloud", "blocked"}
DECISION_LEVELS = {"routine", "significant", "major", "critical"}
TIER_VALUES = {"local", "expert", "none"}


class ContractError(ValueError):
    """Raised when a cross-module contract object is invalid."""


def contracts_available() -> bool:
    return CONTRACTS_AVAILABLE


def new_id(prefix: str) -> str:
    return f"{prefix}_{uuid4().hex[:16]}"


def validate(data: dict[str, Any]) -> dict[str, Any]:
    """Validate a contract object against the JSON Schema and the binding.

    The bundled JSON Schema is authoritative and always applied at the boundary;
    the generated ``aa_contracts`` binding, when importable, is an additional
    check. Raises :class:`ContractError` on failure.
    """
    try:
        validate_contract(data)
    except SchemaError as exc:
        raise ContractError(str(exc)) from exc
    if _aa_contracts is None:
        return data
    try:
        return _aa_contracts.validate(data)
    except ValueError as exc:
        raise ContractError(str(exc)) from exc


def route_request_kwargs(request: dict[str, Any]) -> dict[str, Any]:
    """Validate a ``route_request`` and return ``ComputeSifter.run`` inputs."""
    validate(request)
    budget = request.get("budget") or {}
    return {
        "prompt": request["prompt"],
        "context": request.get("context"),
        "policy": request.get("policy"),
        "max_cloud_cost": budget.get("maxCostUsd"),
        "metadata": dict(request.get("metadata") or {}),
    }


def make_route_response(
    request: dict[str, Any],
    *,
    route: str,
    decision_level: str,
    requires_human_approval: bool,
    reasons: list[str] | None = None,
    planner_tier: str | None = None,
    executor_tier: str | None = None,
    review_tier: str | None = None,
    blocked_reason: str | None = None,
    estimated_cost_usd: float | None = None,
) -> dict[str, Any]:
    if route not in ROUTE_VALUES:
        raise ContractError(f"route: invalid value {route!r}")
    if decision_level not in DECISION_LEVELS:
        raise ContractError(f"decisionLevel: invalid value {decision_level!r}")
    response: dict[str, Any] = {
        "contractVersion": CONTRACT_VERSION,
        "kind": "route_response",
        "id": new_id("resp"),
        "requestId": request["id"],
        "route": route,
        "decisionLevel": decision_level,
        "requiresHumanApproval": requires_human_approval,
        "reasons": list(reasons or []),
    }
    if planner_tier is not None:
        response["plannerTier"] = planner_tier
    if executor_tier is not None:
        response["executorTier"] = executor_tier
    if review_tier is not None:
        response["reviewTier"] = review_tier
    if blocked_reason is not None:
        response["blockedReason"] = blocked_reason
    if estimated_cost_usd is not None:
        response["estimatedCostUsd"] = max(0.0, float(estimated_cost_usd))
    validate(response)
    return response


def make_memory_record(
    *,
    summary: str,
    level: str = "session",
    lifecycle: str = "CANDIDATE",
    title: str | None = None,
    confidence: float = 0.5,
    evidence: list[dict[str, Any]] | None = None,
    applicability: dict[str, Any] | None = None,
) -> dict[str, Any]:
    """Build a lifecycle-gated ``memory_record`` candidate."""
    record: dict[str, Any] = {
        "contractVersion": CONTRACT_VERSION,
        "kind": "memory_record",
        "id": new_id("mem"),
        "level": level,
        "lifecycle": lifecycle,
        "title": title or summary[:120],
        "content": {"summary": summary},
        "confidence": max(0.0, min(1.0, confidence)),
    }
    if evidence:
        record["evidence"] = evidence
    if applicability:
        record["applicability"] = applicability
    validate(record)
    return record
