from __future__ import annotations

from typing import Any, Protocol


class HistoryStore(Protocol):
    def record_run(self, run: dict[str, Any]) -> None: ...

    def record_approval(
        self, trace_id: str, request: dict[str, Any], decision: dict[str, Any]
    ) -> None: ...

    def record_event(
        self, trace_id: str, component: str, message: str, fields: dict[str, Any]
    ) -> None: ...

    def add_standing_rule(
        self,
        text: str,
        *,
        forbid_terms: list[str] | None = None,
        grants: list[str] | None = None,
        tags: list[str] | None = None,
    ) -> int: ...

    def list_standing_rules(self, *, active_only: bool = True) -> list[dict[str, Any]]: ...

    def deactivate_standing_rule(self, rule_id: int) -> bool: ...

    def recent_runs(self, limit: int = 20) -> list[dict[str, Any]]: ...

    def stats(self) -> dict[str, Any]: ...

    def close(self) -> None: ...
