from __future__ import annotations

import re
from collections.abc import Sequence

from pydantic import BaseModel, Field

from ..metrics.trace import Trace
from ..models.provider import Message

_SECRET_PATTERNS: list[tuple[str, re.Pattern[str]]] = [
    (
        "private_key",
        re.compile(
            r"-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----", re.DOTALL
        ),
    ),
    ("aws_access_key", re.compile(r"\bAKIA[0-9A-Z]{16}\b")),
    ("github_token", re.compile(r"\b(gh[pousr]_[A-Za-z0-9]{20,})\b")),
    ("slack_token", re.compile(r"\bxox[baprs]-[A-Za-z0-9-]{10,}\b")),
    ("openai_key", re.compile(r"\bsk-[A-Za-z0-9]{16,}\b")),
    ("bearer_token", re.compile(r"(?i)\bbearer\s+[A-Za-z0-9._\-]{12,}")),
    (
        "env_assignment",
        re.compile(
            r"(?im)^\s*[A-Z0-9_]*(?:KEY|TOKEN|SECRET|PASSWORD|PASSWD|PWD|CREDENTIAL)[A-Z0-9_]*\s*=\s*.+$"
        ),
    ),
    ("password_field", re.compile(r"(?i)\bpassword\s*[:=]\s*[^\s\"']{4,}")),
    (
        "connection_string",
        re.compile(r"(?i)\b(?:postgres(?:ql)?|mysql|mongodb|redis)(?:\+\w+)?://[^\s]+"),
    ),
]


class RedactionResult(BaseModel):
    text: str
    findings: list[str] = Field(default_factory=list)

    @property
    def redacted(self) -> bool:
        return bool(self.findings)


class SecretRedactor:
    """Removes likely secrets before any text is handed to the cloud model."""

    def __init__(self, enabled: bool = True, *, trace: Trace | None = None):
        self.enabled = enabled
        self.trace = trace

    def redact(self, text: str) -> RedactionResult:
        if not self.enabled or not text:
            return RedactionResult(text=text)
        findings: list[str] = []
        redacted = text
        for name, pattern in _SECRET_PATTERNS:
            if pattern.search(redacted):
                findings.append(name)
                redacted = pattern.sub(f"[REDACTED:{name}]", redacted)
        result = RedactionResult(text=redacted, findings=findings)
        if findings and self.trace is not None:
            self.trace.log(
                "security", "redacted potential secrets before cloud handoff", patterns=findings
            )
        return result

    def redact_messages(
        self, messages: list[dict[str, str]]
    ) -> tuple[list[dict[str, str]], list[str]]:
        findings: list[str] = []
        output: list[dict[str, str]] = []
        for message in messages:
            result = self.redact(message.get("content", ""))
            findings.extend(result.findings)
            output.append({**message, "content": result.text})
        return output, sorted(set(findings))

    def redact_model_messages(self, messages: Sequence[Message]) -> tuple[list[Message], list[str]]:
        """Redact typed :class:`Message` objects for an outbound cloud call.

        This is the shared implementation behind the single outbound redaction
        chokepoint in ``ComputeSifter._call_tier`` and ``generate``.
        """
        findings: list[str] = []
        output: list[Message] = []
        for message in messages:
            result = self.redact(message.content)
            findings.extend(result.findings)
            output.append(message.model_copy(update={"content": result.text}))
        return output, sorted(set(findings))


class LocalTaskSpec(BaseModel):
    task: str
    acceptance_criteria: list[str] = Field(default_factory=list)
    files_likely_relevant: list[str] = Field(default_factory=list)
    constraints: list[str] = Field(default_factory=list)
    dependencies: list[str] = Field(default_factory=list)
    task_id: str | None = None
    role: str = "builder"


class HandoffBuilder:
    def __init__(self, *, redactor: SecretRedactor | None = None, trace: Trace | None = None):
        self.redactor = redactor or SecretRedactor()
        self.trace = trace

    def to_local_task(
        self,
        *,
        task: str,
        acceptance_criteria: list[str] | None = None,
        files: list[str] | None = None,
        constraints: list[str] | None = None,
        dependencies: list[str] | None = None,
        task_id: str | None = None,
    ) -> LocalTaskSpec:
        return LocalTaskSpec(
            task=task,
            acceptance_criteria=list(acceptance_criteria or []),
            files_likely_relevant=list(files or []),
            constraints=list(constraints or []),
            dependencies=list(dependencies or []),
            task_id=task_id,
        )

    def from_plan(self, plan: dict) -> list[LocalTaskSpec]:
        specs: list[LocalTaskSpec] = []
        for index, item in enumerate(plan.get("tasks", [])):
            specs.append(
                LocalTaskSpec(
                    task=item.get("task", ""),
                    acceptance_criteria=list(item.get("acceptance_criteria", [])),
                    files_likely_relevant=list(item.get("files_likely_relevant", [])),
                    constraints=list(item.get("constraints", [])),
                    dependencies=list(item.get("dependencies", [])),
                    task_id=item.get("id") or f"task-{index + 1:03d}",
                )
            )
        return specs
