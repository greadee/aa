from __future__ import annotations

from pydantic import BaseModel, Field

from ..decisions.classifier import DecisionLevel


class EscalationPacket(BaseModel):
    original_task: str
    current_plan: str = ""
    completed_work: str = ""
    relevant_files: list[str] = Field(default_factory=list)
    failed_attempts: list[str] = Field(default_factory=list)
    test_results: str = ""
    open_questions: list[str] = Field(default_factory=list)
    reason_for_escalation: str = ""
    human_constraints: list[str] = Field(default_factory=list)
    approved_decisions: list[str] = Field(default_factory=list)
    decision_level: DecisionLevel = DecisionLevel.ROUTINE
    constraints_are_authoritative: bool = True

    def to_prompt(self) -> str:
        sections: list[str] = [f"Original task:\n{self.original_task}"]
        if self.current_plan:
            sections.append(f"Current plan:\n{self.current_plan}")
        if self.completed_work:
            sections.append(f"Completed work:\n{self.completed_work}")
        if self.relevant_files:
            sections.append("Relevant files:\n" + "\n".join(f"- {f}" for f in self.relevant_files))
        if self.failed_attempts:
            sections.append(
                "Failed attempts:\n" + "\n".join(f"- {attempt}" for attempt in self.failed_attempts)
            )
        if self.test_results:
            sections.append(f"Test results:\n{self.test_results}")
        if self.open_questions:
            sections.append("Open questions:\n" + "\n".join(f"- {q}" for q in self.open_questions))
        if self.reason_for_escalation:
            sections.append(f"Reason for escalation:\n{self.reason_for_escalation}")
        if self.human_constraints:
            sections.append(
                "HUMAN CONSTRAINTS (authoritative, must not be reinterpreted as optional):\n"
                + "\n".join(f"- {c}" for c in self.human_constraints)
            )
        if self.approved_decisions:
            sections.append(
                "APPROVED DECISIONS (authoritative):\n"
                + "\n".join(f"- {d}" for d in self.approved_decisions)
            )
        return "\n\n".join(sections)
