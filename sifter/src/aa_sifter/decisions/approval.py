from __future__ import annotations

from collections.abc import Callable, Sequence
from enum import StrEnum
from typing import Protocol

from pydantic import BaseModel, Field

from .classifier import DecisionLevel


class ApprovalAction(StrEnum):
    APPROVE = "approve"
    REJECT = "reject"
    MODIFY = "modify"
    CHOOSE_OPTION = "choose_option"
    MORE_LOCAL_ANALYSIS = "ask_for_more_local_analysis"
    ALLOW_EXPERT_ANALYSIS = "allow_expert_analysis"
    CONTINUE_CURRENT = "continue_with_current_architecture"
    STOP = "stop_task"


_AUTHORIZING = {
    ApprovalAction.APPROVE,
    ApprovalAction.CHOOSE_OPTION,
    ApprovalAction.ALLOW_EXPERT_ANALYSIS,
    ApprovalAction.CONTINUE_CURRENT,
}
_NON_AUTHORIZING = {
    ApprovalAction.REJECT,
    ApprovalAction.STOP,
    ApprovalAction.MODIFY,
    ApprovalAction.MORE_LOCAL_ANALYSIS,
}


class ApprovalOption(BaseModel):
    key: str
    label: str
    description: str = ""


class ApprovalRequest(BaseModel):
    category: DecisionLevel
    issue: str
    why_it_matters: str = ""
    current_approach: str = ""
    options: list[ApprovalOption] = Field(default_factory=list)
    recommended_action: str = ""
    cloud_reasoning_invoked: bool = False
    stage: str = "analysis"
    context: dict = Field(default_factory=dict)


class HumanDecision(BaseModel):
    action: ApprovalAction
    selected_options: list[str] = Field(default_factory=list)
    constraints: list[str] = Field(default_factory=list)
    note: str = ""

    @property
    def authorized(self) -> bool:
        return self.action in _AUTHORIZING

    @property
    def allows_expert_analysis(self) -> bool:
        return self.action == ApprovalAction.ALLOW_EXPERT_ANALYSIS


class ApprovalProvider(Protocol):
    async def request(self, request: ApprovalRequest) -> HumanDecision: ...


def render_request(request: ApprovalRequest) -> str:
    lines: list[str] = []
    lines.append(f"{request.category.value.upper()} DECISION REQUIRED")
    lines.append("")
    lines.append("Issue:")
    lines.append(request.issue)
    if request.why_it_matters:
        lines.append("")
        lines.append("Why this matters:")
        lines.append(request.why_it_matters)
    if request.current_approach:
        lines.append("")
        lines.append("Current approach:")
        lines.append(request.current_approach)
    if request.options:
        lines.append("")
        lines.append("Possible directions:")
        for option in request.options:
            lines.append(f"{option.key}. {option.label}")
            if option.description:
                lines.append(f"   {option.description}")
    if request.recommended_action:
        lines.append("")
        lines.append("Recommended next step:")
        lines.append(request.recommended_action)
    lines.append("")
    if request.cloud_reasoning_invoked:
        lines.append("Cloud reasoning HAS been invoked for analysis.")
    else:
        lines.append("Cloud reasoning has NOT been invoked yet.")
    lines.append("")
    lines.append("Choose:")
    existing = {option.key.upper() for option in request.options}
    for option in request.options:
        lines.append(f"[{option.key}] {option.label}")
    if "C" not in existing:
        lines.append("[C] Give different instructions")
    if "D" not in existing:
        lines.append("[D] Stop this work")
    return "\n".join(lines)


def parse_decision_input(raw: str, options: Sequence[ApprovalOption]) -> HumanDecision:
    text = raw.strip()
    lowered = text.lower()
    valid_keys = {option.key.upper() for option in options}
    if lowered in {"approve", "approved", "yes", "y", "ok", "a"} and "A" not in valid_keys:
        return HumanDecision(action=ApprovalAction.APPROVE)
    if lowered in {"reject", "no", "n"}:
        return HumanDecision(action=ApprovalAction.REJECT)
    if lowered in {"stop", "d", "cancel", "abort"}:
        return HumanDecision(action=ApprovalAction.STOP)
    if lowered in {"modify", "c", "change", "different"}:
        return HumanDecision(action=ApprovalAction.MODIFY, note=text)
    if lowered in {"more", "more local", "analyze locally", "local analysis"}:
        return HumanDecision(action=ApprovalAction.MORE_LOCAL_ANALYSIS)
    if lowered in {"allow deepseek", "allow expert", "allow expert analysis", "deepseek"}:
        return HumanDecision(action=ApprovalAction.ALLOW_EXPERT_ANALYSIS)
    if lowered in {"continue", "keep", "keep current", "current"}:
        return HumanDecision(action=ApprovalAction.CONTINUE_CURRENT)
    if text.upper() in valid_keys:
        key = text.upper()
        option = next(o for o in options if o.key.upper() == key)
        label = option.label.lower()
        if "stop" in label:
            return HumanDecision(action=ApprovalAction.STOP)
        if "keep" in label or "current" in label:
            return HumanDecision(action=ApprovalAction.CONTINUE_CURRENT)
        if "analysis" in label or "analyze" in label or "deepseek" in label or "expert" in label:
            return HumanDecision(
                action=ApprovalAction.ALLOW_EXPERT_ANALYSIS, selected_options=[key]
            )
        if "different" in label or "instruction" in label:
            return HumanDecision(action=ApprovalAction.MODIFY, selected_options=[key])
        return HumanDecision(action=ApprovalAction.CHOOSE_OPTION, selected_options=[key])
    if lowered.startswith("constraint:") or lowered.startswith("constraints:"):
        constraints = [part.strip() for part in text.split(":", 1)[1].split(";") if part.strip()]
        return HumanDecision(action=ApprovalAction.APPROVE, constraints=constraints, note=text)
    return HumanDecision(action=ApprovalAction.MODIFY, note=text)


class ConsoleApprovalProvider:
    """Interactive human approval via stdin/stdout."""

    def __init__(
        self,
        *,
        input_fn: Callable[[str], str] = input,
        output_fn: Callable[[str], None] = print,
    ):
        self._input = input_fn
        self._output = output_fn

    async def request(self, request: ApprovalRequest) -> HumanDecision:
        self._output(render_request(request))
        prompts = ", ".join(o.key for o in request.options) or "approve/reject"
        raw = self._input(f"Decision [{prompts}, C=instructions, D=stop]: ")
        return parse_decision_input(raw, request.options)


class AutoApprovalProvider:
    """Deterministic approval provider for non-interactive use and tests.

    Defaults to the safest non-destructive behavior (STOP) for major and
    critical decisions. Critical decisions can never be auto-approved.
    """

    def __init__(
        self,
        *,
        default: HumanDecision | None = None,
        major: HumanDecision | None = None,
        critical: HumanDecision | None = None,
        significant: HumanDecision | None = None,
    ):
        self._significant = significant
        self._major = major or default
        self._critical = critical
        self.requests: list[ApprovalRequest] = []

    async def request(self, request: ApprovalRequest) -> HumanDecision:
        self.requests.append(request)
        if request.category == DecisionLevel.CRITICAL:
            if self._critical is not None:
                return self._critical
            return HumanDecision(
                action=ApprovalAction.STOP, note="critical decision not auto-approved"
            )
        if request.category == DecisionLevel.MAJOR:
            if self._major is not None:
                return self._major
            return HumanDecision(
                action=ApprovalAction.STOP, note="major decision not auto-approved"
            )
        if self._significant is not None:
            return self._significant
        return HumanDecision(action=ApprovalAction.APPROVE, note="non-interactive routine approval")
