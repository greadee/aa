from __future__ import annotations

from typing import Any

from pydantic import BaseModel, Field

from ..metrics.trace import Trace
from ..models.config import SifterConfig
from .approval import ApprovalProvider, ApprovalRequest, HumanDecision
from .classifier import DecisionClassification, DecisionClassifier, DecisionLevel, level_at_least
from .policy import StandingRuleEngine


class Proposal(BaseModel):
    description: str
    task_type: str = "general"
    architecture_impact: bool = False
    security_sensitive: bool = False
    reversible: bool = True
    scope_impact: str = "localized"
    metadata: dict[str, Any] = Field(default_factory=dict)


class DecisionGate:
    """Application-code enforcement of human authority over major decisions.

    This does not depend on any model remembering to ask. Major and critical
    decisions cannot reach the cloud reasoning stage until the human has
    authorized it.
    """

    def __init__(
        self,
        config: SifterConfig,
        *,
        classifier: DecisionClassifier | None = None,
        rules: StandingRuleEngine | None = None,
        approval_provider: ApprovalProvider | None = None,
        trace: Trace | None = None,
        history: Any | None = None,
    ):
        self.config = config
        self.classifier = classifier or DecisionClassifier()
        self.rules = rules or StandingRuleEngine()
        self.approval_provider = approval_provider
        self.trace = trace
        self.history = history

    def approval_needed(self, classification: DecisionClassification) -> bool:
        if classification.level == DecisionLevel.CRITICAL:
            return self.config.human_approval_for_critical_decisions
        if classification.level == DecisionLevel.MAJOR:
            return self.config.human_approval_for_major_decisions
        if classification.level == DecisionLevel.SIGNIFICANT:
            return self.config.human_approval_for_significant_decisions
        return False

    async def classify(
        self,
        proposal: str | Proposal,
        *,
        model_level: DecisionLevel | None = None,
        model_reason: str | None = None,
    ) -> DecisionClassification:
        text = proposal.description if isinstance(proposal, Proposal) else proposal
        conflicts = self.rules.detect_conflicts(text)
        classification = self.classifier.classify(
            text,
            model_level=model_level,
            model_reason=model_reason,
            rule_conflicts=conflicts,
        )
        if isinstance(proposal, Proposal):
            if proposal.architecture_impact:
                classification.level = _bump(classification.level, DecisionLevel.MAJOR)
            if proposal.security_sensitive:
                classification.level = _bump(classification.level, DecisionLevel.MAJOR)
        classification.requires_human_approval = self.approval_needed(classification)
        if self.trace is not None:
            self.trace.log(
                "decision",
                f"level={classification.level.value}",
                requires_approval=classification.requires_human_approval,
                signals=classification.signal_names,
            )
        return classification

    async def require_approval(self, request: ApprovalRequest) -> HumanDecision:
        if self.trace is not None:
            self.trace.log("approval", "waiting_for_user", category=request.category.value)
        if self.approval_provider is None:
            decision = HumanDecision(action=_default_action_for(request.category))
        else:
            decision = await self.approval_provider.request(request)
        if self.history is not None and self.trace is not None:
            self.history.record_approval(
                self.trace.trace_id,
                request.model_dump(mode="json"),
                {**decision.model_dump(mode="json"), "authorized": decision.authorized},
            )
        if self.trace is not None:
            self.trace.log(
                "user",
                decision.action.value,
                authorized=decision.authorized,
                selected=decision.selected_options,
                constraints=decision.constraints,
            )
        return decision


def _bump(current: DecisionLevel, floor: DecisionLevel) -> DecisionLevel:
    return floor if level_at_least(floor, current) else current


def _default_action_for(category: DecisionLevel):
    from .approval import ApprovalAction

    return ApprovalAction.STOP
