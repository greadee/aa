from __future__ import annotations

from typing import Any

from ..context.handoff import HandoffBuilder, LocalTaskSpec
from ..metrics.trace import Trace
from ..models.config import ModelConfig
from ..models.provider import Message, ModelProvider, ProviderError

PLAN_SCHEMA: dict[str, Any] = {
    "type": "object",
    "properties": {
        "tasks": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "id": {"type": "string"},
                    "task": {"type": "string"},
                    "acceptance_criteria": {"type": "array", "items": {"type": "string"}},
                    "files_likely_relevant": {"type": "array", "items": {"type": "string"}},
                    "constraints": {"type": "array", "items": {"type": "string"}},
                    "dependencies": {"type": "array", "items": {"type": "string"}},
                },
                "required": ["task"],
            },
        }
    },
    "required": ["tasks"],
}


class Decomposer:
    def __init__(self, *, handoff: HandoffBuilder | None = None, trace: Trace | None = None):
        self.handoff = handoff or HandoffBuilder()
        self.trace = trace

    async def decompose(
        self,
        prompt: str,
        provider: ModelProvider,
        model: ModelConfig,
        *,
        context: str = "",
        constraints: list[str] | None = None,
    ) -> list[LocalTaskSpec]:
        task_content = (
            f"Task:\n{prompt}"
            + (f"\n\nContext:\n{context[:6000]}" if context else "")
            + (f"\n\nConstraints: {'; '.join(constraints)}" if constraints else "")
        )
        redaction = self.handoff.redactor.redact(task_content)
        if redaction.redacted and self.trace is not None:
            self.trace.log(
                "security",
                "redacted secrets before cloud handoff",
                purpose="decompose",
                patterns=redaction.findings,
            )
        messages = [
            Message.system(
                "Break the task into at most 6 independent or sequentially dependent "
                "implementation subtasks. Respond ONLY with JSON matching the schema. "
                "Always include explicit human constraints in every subtask that may affect them."
            ),
            Message.user(redaction.text),
        ]
        try:
            result = await provider.generate_structured(
                model.name, messages, PLAN_SCHEMA, temperature=0.0, timeout=model.timeout_seconds
            )
        except ProviderError as exc:
            if self.trace is not None:
                self.trace.log(
                    "plan", f"decomposition failed, using single task: {exc}", level="warning"
                )
            return [self._single(prompt, constraints)]
        if not result.structured or not result.structured.get("tasks"):
            return [self._single(prompt, constraints)]
        specs = self.handoff.from_plan(result.structured)
        for spec in specs:
            if constraints:
                for constraint in constraints:
                    if constraint not in spec.constraints:
                        spec.constraints.append(constraint)
        if self.trace is not None:
            self.trace.log("plan", f"decomposed into {len(specs)} subtask(s)")
        return specs

    def _single(self, prompt: str, constraints: list[str] | None) -> LocalTaskSpec:
        return LocalTaskSpec(task=prompt, constraints=list(constraints or []), task_id="task-001")
