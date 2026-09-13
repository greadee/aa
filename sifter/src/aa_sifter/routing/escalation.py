from __future__ import annotations

from enum import StrEnum

from pydantic import BaseModel, Field

from ..metrics.trace import Trace
from ..models.config import SifterConfig


class EscalationAction(StrEnum):
    NONE = "none"
    CLOUD_DEBUG = "cloud_debug"
    CLOUD_REVIEW = "cloud_review"
    CLOUD_PLAN = "cloud_plan"
    CLOUD_SYNTHESIS = "cloud_synthesis"
    USER_DECISION = "user_decision"
    ABORT = "abort"


class EscalationRequest(BaseModel):
    reason: str
    failed_attempts: int = 0
    needed_context: list[str] = Field(default_factory=list)
    recommended_action: EscalationAction = EscalationAction.CLOUD_REVIEW
    potential_major_decision: bool = False


class EscalationOutcome(BaseModel):
    escalate: bool
    action: EscalationAction
    requires_human_approval: bool
    reason: str


class EscalationPolicy:
    def __init__(self, config: SifterConfig, *, trace: Trace | None = None):
        self.config = config
        self.trace = trace

    def evaluate(
        self,
        request: EscalationRequest,
        *,
        force_user_if_major: bool = True,
    ) -> EscalationOutcome:
        requires_approval = bool(request.potential_major_decision and force_user_if_major)
        if requires_approval:
            outcome = EscalationOutcome(
                escalate=True,
                action=EscalationAction.USER_DECISION,
                requires_human_approval=True,
                reason=request.reason,
            )
        else:
            outcome = EscalationOutcome(
                escalate=True,
                action=request.recommended_action,
                requires_human_approval=False,
                reason=request.reason,
            )
        if self.trace is not None:
            self.trace.log(
                "escalation",
                f"action={outcome.action.value}",
                requires_approval=outcome.requires_human_approval,
                reason=request.reason,
            )
        return outcome

    def should_escalate(
        self,
        *,
        failed_attempts: int,
        verification_passed: bool | None,
        wrote_new_plan_without_progress: bool = False,
    ) -> bool:
        if failed_attempts > self.config.local_max_retries:
            return True
        if verification_passed is False:
            return True
        if wrote_new_plan_without_progress:
            return True
        return False

    def is_architecture_like(self, text: str) -> bool:
        lowered = text.lower()
        markers = [
            "architecture",
            "replace the",
            "persistence layer",
            "database",
            "rearchitect",
            "redesign",
            "migration",
            "security model",
            "framework",
        ]
        return any(marker in lowered for marker in markers)
