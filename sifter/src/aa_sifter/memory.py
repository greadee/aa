"""Memory trace seam.

The sifter never writes canonical state (aa-sifter must not hold project state).
Instead, each completed run can emit a lifecycle-gated ``memory_record``
candidate through a :class:`MemorySink`; promotion is a separate, evidence-gated
step owned by ``aa-memory`` (ADR-P4-008).
"""

from __future__ import annotations

from typing import Any, Protocol, runtime_checkable


@runtime_checkable
class MemorySink(Protocol):
    def emit(self, record: dict[str, Any]) -> None: ...


class NopMemorySink:
    """Default sink: records nothing and never blocks a run."""

    def emit(self, record: dict[str, Any]) -> None:
        return None


def task_candidate(
    *,
    trace_id: str,
    status: str,
    route: str,
    decision_level: str,
    requires_human_approval: bool,
    prompt: str,
    cloud_calls: int,
    cloud_cost: float,
) -> dict[str, Any]:
    """Build a ``memory_record`` candidate describing one sifter run."""
    from .contracts import make_memory_record

    summary = f"aa-sifter task {status} via {route} at decision level {decision_level}"
    record = make_memory_record(
        summary=summary,
        level="task",
        lifecycle="CANDIDATE",
        title=f"{route} run: {prompt[:80]}",
        confidence=0.6 if status == "completed" else 0.3,
        applicability={
            "route": route,
            "decisionLevel": decision_level,
            "requiresHumanApproval": requires_human_approval,
            "cloudCalls": cloud_calls,
            "cloudCostUsd": round(cloud_cost, 6),
        },
    )
    record["provenance"] = {"source": "aa-sifter", "traceId": trace_id}
    return record
