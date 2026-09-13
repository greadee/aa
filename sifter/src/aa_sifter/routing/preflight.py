from __future__ import annotations

from typing import Any

from pydantic import BaseModel, Field

from ..decisions.classifier import (
    DecisionClassification,
    DecisionClassifier,
    DecisionLevel,
)
from ..metrics.trace import Trace
from ..models.config import ModelConfig, SifterConfig
from ..models.provider import Message, ModelProvider, ProviderError

PREFLIGHT_SCHEMA: dict[str, Any] = {
    "type": "object",
    "properties": {
        "task_type": {"type": "string"},
        "complexity": {"type": "number"},
        "local_success_probability": {"type": "number"},
        "reasoning_difficulty": {"type": "number"},
        "context_requirement": {"type": "integer"},
        "estimated_subtasks": {"type": "integer"},
        "architecture_impact": {"type": "boolean"},
        "security_sensitive": {"type": "boolean"},
        "decision_level": {
            "type": "string",
            "enum": ["routine", "significant", "major", "critical"],
        },
        "requires_cloud": {"type": "boolean"},
        "reason": {"type": "string"},
    },
    "required": [
        "task_type",
        "complexity",
        "local_success_probability",
        "decision_level",
        "reason",
    ],
}


class PreflightAssessment(BaseModel):
    task_type: str = "general"
    complexity: float = 0.3
    local_success_probability: float = 0.8
    reasoning_difficulty: float = 0.3
    context_requirement: int = 4_000
    estimated_subtasks: int = 1
    architecture_impact: bool = False
    security_sensitive: bool = False
    decision_level: DecisionLevel = DecisionLevel.ROUTINE
    requires_human_approval: bool = False
    requires_cloud: bool = False
    recommended_route: str = "local"
    reason: str = ""
    model_assisted: bool = False
    signals: dict[str, Any] = Field(default_factory=dict)


_TASK_TYPES: dict[str, list[str]] = {
    "testing": ["test", "pytest", "unit test", "coverage"],
    "debugging": ["debug", "bug", "failing", "failure", "traceback", "error", "broken"],
    "implementation": ["implement", "add", "build", "create", "write", "feature"],
    "refactor": ["refactor", "restructure", "cleanup", "clean up"],
    "documentation": ["document", "readme", "docstring", "comment"],
    "review": ["review", "audit", "inspect", "analyze"],
    "architecture": ["architecture", "design", "plan", "migration"],
}

_COMPLEXITY_SIGNALS = [
    "concurrency",
    "distributed",
    "performance",
    "algorithm",
    "optimize",
    "security",
    "crypto",
    "migration",
    "refactor",
    "architecture",
    "asynchronous",
    "async",
    "race condition",
    "deadlock",
]

_SECURITY_SIGNALS = ["auth", "oauth", "password", "credential", "secret", "encryption", "security"]


class PreflightAssessor:
    """Deterministic task assessment combined with optional local model input.

    The policy engine has final routing authority. Model self-assessment can
    raise concern but cannot lower deterministic risk.
    """

    def __init__(
        self,
        config: SifterConfig,
        *,
        classifier: DecisionClassifier | None = None,
        trace: Trace | None = None,
    ):
        self.config = config
        self.classifier = classifier or DecisionClassifier()
        self.trace = trace

    def assess(
        self,
        prompt: str,
        context: str | None = None,
        *,
        failed_attempts: int = 0,
        retry_count: int = 0,
        metadata: dict[str, Any] | None = None,
        classification: DecisionClassification | None = None,
    ) -> PreflightAssessment:
        text = f"{prompt}\n{context or ''}".lower()
        prompt_tokens = max(1, len(prompt) // 4)
        context_tokens = max(0, len(context or "") // 4)
        context_requirement = prompt_tokens + context_tokens

        task_type = self._task_type(text)
        matching = sum(1 for signal in _COMPLEXITY_SIGNALS if signal in text)
        length_factor = min(1.0, len(prompt) / 4_000.0)
        complexity = _clamp(0.15 + 0.1 * matching + 0.5 * length_factor)
        reasoning_difficulty = _clamp(0.2 + 0.12 * matching + 0.4 * length_factor)

        classification = classification or self.classifier.classify(prompt)
        architecture_impact = classification.signals.architecture_change
        security_sensitive = classification.signals.security_boundary or any(
            signal in text for signal in _SECURITY_SIGNALS
        )

        local_probability = 0.92 - 0.5 * complexity - 0.25 * reasoning_difficulty
        local_probability -= min(0.4, 0.2 * failed_attempts)
        local_probability -= min(0.2, 0.05 * retry_count)
        if architecture_impact:
            local_probability -= 0.15
        local_probability = _clamp(local_probability)

        context_pressure = context_requirement > self.config.local_context_limit
        requires_cloud = (
            local_probability < self.config.local_confidence_threshold
            or context_pressure
            or classification.level in {DecisionLevel.MAJOR, DecisionLevel.CRITICAL}
        )

        estimated_subtasks = 1 + int(complexity * 4)
        if task_type in {"architecture", "implementation"} and complexity > 0.5:
            estimated_subtasks = max(estimated_subtasks, 3)

        if requires_cloud or architecture_impact:
            route = (
                "cloud" if context_pressure else ("hybrid" if local_probability > 0.4 else "cloud")
            )
        else:
            route = "local"

        reasons: list[str] = []
        if classification.level != DecisionLevel.ROUTINE:
            reasons.append(f"decision level {classification.level.value}")
        if context_pressure:
            reasons.append(
                f"context ~{context_requirement} tokens exceeds local limit {self.config.local_context_limit}"
            )
        if failed_attempts:
            reasons.append(f"{failed_attempts} prior failed attempt(s)")
        if not reasons:
            reasons.append("localized work within local model capability")
        reason = "; ".join(reasons)

        assessment = PreflightAssessment(
            task_type=task_type,
            complexity=round(complexity, 3),
            local_success_probability=round(local_probability, 3),
            reasoning_difficulty=round(reasoning_difficulty, 3),
            context_requirement=context_requirement,
            estimated_subtasks=estimated_subtasks,
            architecture_impact=architecture_impact,
            security_sensitive=security_sensitive,
            decision_level=classification.level,
            requires_human_approval=classification.requires_human_approval,
            requires_cloud=requires_cloud,
            recommended_route=route,
            reason=reason,
        )
        if metadata:
            assessment.signals["metadata"] = metadata
        return assessment

    async def assess_with_model(
        self,
        provider: ModelProvider,
        model: ModelConfig,
        prompt: str,
        context: str | None = None,
        *,
        base: PreflightAssessment | None = None,
        classification: DecisionClassification | None = None,
    ) -> PreflightAssessment:
        base = base or self.assess(prompt, context, classification=classification)
        messages = [
            Message.system(
                "You assess software tasks. Respond ONLY with JSON matching the schema. "
                "Be conservative: if unsure whether a decision is significant or major, choose major."
            ),
            Message.user(
                "Assess this task.\n\nTask:\n"
                + prompt
                + (f"\n\nContext (truncated):\n{context[:4000]}" if context else "")
            ),
        ]
        try:
            result = await provider.generate_structured(
                model.name,
                messages,
                PREFLIGHT_SCHEMA,
                temperature=0.0,
                timeout=model.timeout_seconds,
            )
        except ProviderError as exc:
            if self.trace is not None:
                self.trace.log("preflight", f"model assessment failed: {exc}", level="warning")
            return base
        parsed = result.structured
        if not parsed:
            if self.trace is not None:
                self.trace.log(
                    "preflight", "model assessment returned no structured output", level="warning"
                )
            return base
        return self.merge_assessment(base, parsed)

    def merge_assessment(
        self, base: PreflightAssessment, parsed: dict[str, Any]
    ) -> PreflightAssessment:
        merged = base.model_copy(deep=True)
        merged.model_assisted = True
        model_level_raw = parsed.get("decision_level")
        if isinstance(model_level_raw, str) and model_level_raw in DecisionLevel._value2member_map_:
            model_level = DecisionLevel(model_level_raw)
            if _severity(model_level) > _severity(base.decision_level):
                merged.decision_level = model_level
        merged.complexity = _merge_float(base.complexity, parsed.get("complexity"), higher=True)
        merged.reasoning_difficulty = _merge_float(
            base.reasoning_difficulty, parsed.get("reasoning_difficulty"), higher=True
        )
        merged.local_success_probability = _merge_float(
            base.local_success_probability, parsed.get("local_success_probability"), higher=False
        )
        try:
            merged.context_requirement = max(
                base.context_requirement, int(parsed.get("context_requirement") or 0)
            )
        except (TypeError, ValueError):
            pass
        merged.architecture_impact = base.architecture_impact or bool(
            parsed.get("architecture_impact")
        )
        merged.security_sensitive = base.security_sensitive or bool(
            parsed.get("security_sensitive")
        )
        merged.requires_cloud = base.requires_cloud or bool(parsed.get("requires_cloud"))
        model_reason = parsed.get("reason")
        if model_reason:
            merged.reason = f"{base.reason}; model: {model_reason}"
        return merged

    @staticmethod
    def _task_type(text: str) -> str:
        for task_type, keywords in _TASK_TYPES.items():
            if any(keyword in text for keyword in keywords):
                return task_type
        return "general"


def _clamp(value: float, low: float = 0.0, high: float = 1.0) -> float:
    return max(low, min(high, value))


def _severity(level: DecisionLevel) -> int:
    return {"routine": 0, "significant": 1, "major": 2, "critical": 3}[level.value]


def _merge_float(base: float, candidate: Any, *, higher: bool) -> float:
    try:
        value = float(candidate)
    except (TypeError, ValueError):
        return base
    value = _clamp(value)
    return max(base, value) if higher else min(base, value)
