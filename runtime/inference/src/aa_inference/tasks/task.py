from __future__ import annotations

from enum import StrEnum

from pydantic import BaseModel, Field

from ..decisions.classifier import DecisionLevel
from ..models.provider import Tier


class TaskStatus(StrEnum):
    PENDING = "pending"
    READY = "ready"
    RUNNING = "running"
    BLOCKED_APPROVAL = "blocked_approval"
    BLOCKED_DEPENDENCY = "blocked_dependency"
    DONE = "done"
    FAILED = "failed"
    SKIPPED = "skipped"


class Task(BaseModel):
    id: str
    description: str
    dependencies: list[str] = Field(default_factory=list)
    status: TaskStatus = TaskStatus.PENDING
    assigned_model: str | None = None
    assigned_role: str = "builder"
    assigned_tier: Tier = Tier.LOCAL
    attempt_count: int = 0
    input_context: str = ""
    result: str = ""
    verification_state: str = "unverified"
    cost_tokens: int = 0
    duration_ms: float = 0.0
    decision_level: DecisionLevel = DecisionLevel.ROUTINE
    human_approval_status: str = "not_required"
    acceptance_criteria: list[str] = Field(default_factory=list)
    files_likely_relevant: list[str] = Field(default_factory=list)
    constraints: list[str] = Field(default_factory=list)

    @property
    def terminal(self) -> bool:
        return self.status in {TaskStatus.DONE, TaskStatus.FAILED, TaskStatus.SKIPPED}
