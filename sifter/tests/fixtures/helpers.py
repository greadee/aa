"""Shared test helpers and task factories."""

from __future__ import annotations

from typing import Any

from aa_sifter.app import ComputeSifter
from aa_sifter.decisions.approval import (
    ApprovalAction,
    ApprovalProvider,
    ApprovalRequest,
    HumanDecision,
)
from aa_sifter.decisions.policy import StandingRuleEngine
from aa_sifter.history.sqlite import SqliteHistoryStore
from aa_sifter.models.config import SifterConfig
from aa_sifter.models.provider import ModelProvider

from .providers import FakeProvider

# ---------------------------------------------------------------------------
# Task fixtures
# ---------------------------------------------------------------------------
EASY_LOCAL_TASK = "Write unit tests for this small utility function."
MEDIUM_TASK = "Add input validation and error handling to the config loader."
HARD_DEBUGGING_TASK = "Find why these existing integration tests fail after the latest change."
MAJOR_ARCHITECTURE_TASK = "Replace the persistence architecture with a new design."
MAJOR_AUTH_TASK = "Replace authentication with OAuth while preserving old sessions."
DATABASE_SWAP_TASK = "Migrate the project to PostgreSQL."
CRITICAL_TASK = "Drop table users and delete from the production database."


class ScriptedApprovalProvider:
    """Returns predetermined human decisions in order."""

    def __init__(self, decisions: list[HumanDecision]):
        self.decisions = list(decisions)
        self.requests: list[ApprovalRequest] = []

    async def request(self, request: ApprovalRequest) -> HumanDecision:
        self.requests.append(request)
        if not self.decisions:
            return HumanDecision(action=ApprovalAction.STOP, note="no scripted decision left")
        return self.decisions.pop(0)


class RecordingApprovalProvider:
    """Always returns the configured decision and records requests."""

    def __init__(self, decision: HumanDecision | None = None):
        self.decision = decision or HumanDecision(action=ApprovalAction.STOP)
        self.requests: list[ApprovalRequest] = []

    async def request(self, request: ApprovalRequest) -> HumanDecision:
        self.requests.append(request)
        return self.decision


def make_sifter(
    config: SifterConfig,
    provider: ModelProvider | FakeProvider,
    *,
    approval_provider: ApprovalProvider | None = None,
    store: SqliteHistoryStore | None = None,
    verifier: Any | None = None,
    rules: StandingRuleEngine | None = None,
) -> ComputeSifter:
    return ComputeSifter(
        config,
        provider=provider,  # type: ignore[arg-type]
        approval_provider=approval_provider,
        history=store,
        verifier=verifier,
        rules=rules,
        trace_sink=None,
    )
