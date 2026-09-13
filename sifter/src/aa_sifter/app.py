from __future__ import annotations

import asyncio
import sys
import time
from collections.abc import Callable, Sequence
from contextvars import ContextVar
from typing import Any

from pydantic import BaseModel, Field

from .context.compression import estimate_tokens, fit_messages
from .context.handoff import HandoffBuilder, SecretRedactor
from .context.packet import EscalationPacket
from .decisions.approval import (
    ApprovalAction,
    ApprovalOption,
    ApprovalProvider,
    ApprovalRequest,
    AutoApprovalProvider,
    ConsoleApprovalProvider,
)
from .decisions.classifier import DecisionClassification, DecisionClassifier, DecisionLevel
from .decisions.gate import DecisionGate
from .decisions.policy import StandingRuleEngine
from .history.sqlite import SqliteHistoryStore
from .history.store import HistoryStore
from .memory import MemorySink, NopMemorySink, task_candidate
from .metrics.trace import Trace, configure_logging
from .metrics.usage import UsageMetrics, UsageTracker
from .model_defaults import DEFAULT_EXPERT_MAX_OUTPUT, DEFAULT_LOCAL_MAX_OUTPUT
from .models.config import ModelConfig, SifterConfig
from .models.factory import build_registry
from .models.provider import (
    GenerationResult,
    Message,
    ModelProvider,
    ModelRegistry,
    ProviderError,
    Tier,
)
from .routing.budget import BudgetExceeded, BudgetTracker
from .routing.escalation import (
    EscalationAction,
    EscalationOutcome,
    EscalationPolicy,
    EscalationRequest,
)
from .routing.policy import PolicyEngine, RouteDecision, RouteKind
from .routing.preflight import PreflightAssessment, PreflightAssessor
from .verification.verifier import NullVerifier, VerificationResult, Verifier

ProgressCallback = Callable[[str, dict[str, Any]], None]
_PROGRESS: ContextVar[ProgressCallback | None] = ContextVar("sifter_progress", default=None)


def _emit_progress(event: str, **data: Any) -> None:
    callback = _PROGRESS.get()
    if callback is not None:
        try:
            callback(event, data)
        except Exception:
            pass


_MAJOR_OPTIONS = [
    ApprovalOption(
        key="A",
        label="Keep the current approach",
        description="Proceed with the existing design and work around the limitation.",
    ),
    ApprovalOption(
        key="B",
        label="Allow DeepSeek to analyze these options",
        description="Cloud reasoning compares options and recommends, but does NOT implement.",
    ),
    ApprovalOption(
        key="C",
        label="Give different instructions",
        description="Provide explicit constraints or a different direction.",
    ),
    ApprovalOption(key="D", label="Stop this work"),
]


class SifterResult(BaseModel):
    trace_id: str
    status: str
    answer: str
    route: str = "local"
    decision_level: DecisionLevel = DecisionLevel.ROUTINE
    requires_human_approval: bool = False
    blocked_reason: str | None = None
    assessment: PreflightAssessment | None = None
    usage: UsageMetrics = Field(default_factory=UsageMetrics)
    approvals: list[dict[str, Any]] = Field(default_factory=list)
    metadata: dict[str, Any] = Field(default_factory=dict)

    @property
    def ok(self) -> bool:
        return self.status in {"completed", "done"}


class ComputeSifter:
    """One unified local/cloud compute interface with a human approval gate."""

    def __init__(
        self,
        config: SifterConfig | None = None,
        *,
        provider: ModelProvider | None = None,
        registry: ModelRegistry | None = None,
        approval_provider: ApprovalProvider | None = None,
        history: HistoryStore | None = None,
        verifier: Verifier | None = None,
        rules: StandingRuleEngine | None = None,
        trace_sink: Any = None,
        memory_sink: MemorySink | None = None,
    ):
        self.config = config or SifterConfig.from_env()
        configure_logging(self.config.debug)
        if provider is not None:
            self.provider: ModelProvider | None = provider
            self.registry = registry or build_registry(self.config, injected=provider)
        else:
            self.provider = None
            self.registry = registry or build_registry(self.config)
        self.history = history
        if self.history is None:
            try:
                self.history = SqliteHistoryStore(self.config.resolved_database_path)
            except Exception:
                self.history = None
        self.rules = rules or StandingRuleEngine(self.history)
        self.approval_provider = approval_provider or self._default_approval_provider()
        self.classifier = DecisionClassifier()
        self.handoff = HandoffBuilder(redactor=SecretRedactor(self.config.redact_secrets))
        self.redactor = self.handoff.redactor
        self.assessor = PreflightAssessor(
            self.config, classifier=self.classifier, redactor=self.redactor
        )
        self.policy_engine = PolicyEngine(self.config)
        self.escalation_policy = EscalationPolicy(self.config)
        self.verifier = verifier or NullVerifier()
        self.trace_sink = trace_sink
        self.memory_sink: MemorySink = memory_sink or NopMemorySink()

    def _default_approval_provider(self) -> ApprovalProvider:
        interactive = sys.stdin is not None and sys.stdin.isatty()
        if self.config.non_interactive or not interactive:
            return AutoApprovalProvider()
        return ConsoleApprovalProvider()

    def local_config(self) -> ModelConfig:
        return self.config.local_model_config()

    def cloud_config(self) -> ModelConfig:
        return self.config.cloud_model_config()

    def _provider_for(self, model: ModelConfig) -> ModelProvider:
        return self.registry.provider_for(model.name, default=model.provider)

    async def run(
        self,
        prompt: str,
        context: str | None = None,
        policy: str | None = None,
        max_cloud_cost: float | None = None,
        metadata: dict[str, Any] | None = None,
        progress: ProgressCallback | None = None,
    ) -> SifterResult:
        token = _PROGRESS.set(progress)
        try:
            return await self._run_impl(
                prompt,
                context=context,
                policy=policy,
                max_cloud_cost=max_cloud_cost,
                metadata=metadata,
            )
        finally:
            _PROGRESS.reset(token)

    async def _run_impl(
        self,
        prompt: str,
        context: str | None = None,
        policy: str | None = None,
        max_cloud_cost: float | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> SifterResult:
        start = time.perf_counter()
        metadata = dict(metadata or {})
        metadata.setdefault("prompt", prompt)
        metadata.setdefault("policy", policy or self.config.routing_policy)
        trace = Trace(
            debug=self.config.debug,
            sink=self.trace_sink,
            history=self.history,
        )
        usage = UsageMetrics()
        tracker = UsageTracker()
        tracker.configure(local=self.local_config(), cloud=self.cloud_config())
        waiver = None if max_cloud_cost is not None else self.rules.cost_waiver_threshold()
        budget = BudgetTracker(
            self.config, max_cost=max_cloud_cost, trace=trace, cost_waiver=waiver
        )
        autonomy_grant = self.rules.has_grant("cloud_auto_escalation") or self.rules.has_grant(
            "cloud_debug_failed_tests"
        )
        approvals: list[dict[str, Any]] = []
        approved_decisions: list[str] = []
        human_constraints: list[str] = []
        allow_cloud_analysis = False
        force_local = False

        trace.log("aa_sifter", "start", prompt=prompt[:200])
        gate = DecisionGate(
            self.config,
            classifier=self.classifier,
            rules=self.rules,
            approval_provider=self.approval_provider,
            trace=trace,
            history=self.history,
        )

        try:
            classification = await gate.classify(prompt)
            assessment = self.assessor.assess(prompt, context, classification=classification)
            _emit_progress(
                "analyzing",
                decision_level=classification.level.value,
                requires_human_approval=classification.requires_human_approval,
                task_type=assessment.task_type,
            )

            if classification.requires_human_approval:
                outcome = await self._handle_major_decision(
                    prompt=prompt,
                    context=context,
                    classification=classification,
                    gate=gate,
                    trace=trace,
                    usage=usage,
                    tracker=tracker,
                    budget=budget,
                    approvals=approvals,
                )
                if outcome is None:
                    return self._finish(
                        trace,
                        status="stopped",
                        answer="No action taken. The decision requires human authorization.",
                        classification=classification,
                        assessment=assessment,
                        usage=usage,
                        approvals=approvals,
                        metadata=metadata,
                        blocked_reason="awaiting human decision",
                        start=start,
                    )
                human_constraints, approved_decisions, allow_cloud_analysis, force_local = outcome
                assessment = self.assessor.assess(
                    prompt,
                    context,
                    failed_attempts=0,
                    classification=classification,
                )

            effective_policy = "local_only" if force_local else policy
            if not self.config.cloud_allowed:
                effective_policy = "local_only"
            route_decision = self.policy_engine.decide(
                assessment,
                policy=effective_policy,
                cloud_budget_available=budget.remaining_calls > 0 and not budget.exhausted,
            )
            trace.log("router", f"route={route_decision.kind.value}", reason=route_decision.reasons)
            _emit_progress(
                "routing",
                route=route_decision.kind.value,
                executor=route_decision.executor_tier.value,
                uses_cloud=route_decision.uses_cloud,
            )
            cloud_permitted = (effective_policy or self.config.routing_policy) != "local_only"

            answer, verification, failed_attempts = await self._execute_route(
                prompt=prompt,
                context=context,
                route=route_decision,
                assessment=assessment,
                classification=classification,
                human_constraints=human_constraints,
                approved_decisions=approved_decisions,
                trace=trace,
                usage=usage,
                tracker=tracker,
                budget=budget,
            )

            status = "completed" if verification.passed else "failed"
            if not verification.passed:
                answer, resolved = await self._escalate_after_failure(
                    prompt=prompt,
                    answer=answer,
                    verification=verification,
                    failed_attempts=failed_attempts,
                    classification=classification,
                    human_constraints=human_constraints,
                    approved_decisions=approved_decisions,
                    gate=gate,
                    trace=trace,
                    usage=usage,
                    tracker=tracker,
                    budget=budget,
                    approvals=approvals,
                    allowed=cloud_permitted
                    and (allow_cloud_analysis or not classification.is_major or autonomy_grant),
                    autonomy_grant=autonomy_grant,
                )
                status = "completed" if resolved else "failed"

            return self._finish(
                trace,
                status=status,
                answer=answer,
                classification=classification,
                assessment=assessment,
                usage=usage,
                approvals=approvals,
                metadata={
                    **metadata,
                    "approved_decisions": approved_decisions,
                    "human_constraints": human_constraints,
                },
                blocked_reason=None,
                start=start,
                route=route_decision.kind.value,
            )
        except asyncio.CancelledError:
            trace.log("aa_sifter", "cancelled", level="warning")
            raise
        except BudgetExceeded as exc:
            return self._finish(
                trace,
                status="blocked",
                answer=f"Cloud budget limit reached: {exc}",
                classification=locals().get("classification"),
                assessment=locals().get("assessment"),
                usage=usage,
                approvals=approvals,
                metadata=metadata,
                blocked_reason=str(exc),
                start=start,
            )
        except ProviderError as exc:
            return self._finish(
                trace,
                status="error",
                answer=f"Model provider error: {exc}",
                classification=locals().get("classification"),
                assessment=locals().get("assessment"),
                usage=usage,
                approvals=approvals,
                metadata=metadata,
                blocked_reason=str(exc),
                start=start,
            )

    async def _handle_major_decision(
        self,
        *,
        prompt: str,
        context: str | None,
        classification: DecisionClassification,
        gate: DecisionGate,
        trace: Trace,
        usage: UsageMetrics,
        tracker: UsageTracker,
        budget: BudgetTracker,
        approvals: list[dict[str, Any]],
    ) -> tuple[list[str], list[str], bool, bool] | None:
        request = self._build_approval_request(prompt, context, classification)
        decision = await gate.require_approval(request)
        approvals.append(
            {
                "stage": "analysis",
                "action": decision.action.value,
                "selected_options": decision.selected_options,
                "constraints": decision.constraints,
            }
        )
        usage.human_approval_requests += 1
        usage.major_decisions += 1

        if decision.action in {ApprovalAction.STOP, ApprovalAction.REJECT}:
            return None
        if decision.action == ApprovalAction.MODIFY:
            return None
        if decision.action == ApprovalAction.MORE_LOCAL_ANALYSIS:
            trace.log("aa_sifter", "continuing with local-only analysis per user")
            return decision.constraints, [], False, True
        if decision.action == ApprovalAction.CONTINUE_CURRENT:
            return decision.constraints, ["keep current architecture"], False, True

        # ALLOW_EXPERT_ANALYSIS / APPROVE / CHOOSE_OPTION
        constraints = list(decision.constraints)
        if decision.action in {ApprovalAction.APPROVE, ApprovalAction.CHOOSE_OPTION}:
            if decision.selected_options:
                constraints.extend(f"Selected option {o}" for o in decision.selected_options)
            return constraints, [], True, False

        trace.log("deepseek", "comparing approved alternatives")
        analysis, rec_level = await self._expert_analysis(
            prompt=prompt,
            context=context,
            constraints=constraints,
            trace=trace,
            usage=usage,
            tracker=tracker,
            budget=budget,
        )
        usage.human_approved_cloud_analyses += 1

        implement_request = ApprovalRequest(
            category=rec_level,
            issue="DeepSeek has analyzed the options and produced a recommendation.",
            why_it_matters=(
                "Implementing this recommendation changes project direction. Permission "
                "to analyze is not permission to implement."
            ),
            current_approach="No implementation changes have been made yet.",
            options=[
                ApprovalOption(key="A", label="Approve implementing the recommendation"),
                ApprovalOption(key="B", label="Give different instructions"),
                ApprovalOption(key="C", label="Stop this work"),
            ],
            recommended_action=analysis[:2000],
            cloud_reasoning_invoked=True,
            stage="implementation",
            context={"recommendation": analysis[:4000]},
        )
        second = await gate.require_approval(implement_request)
        approvals.append(
            {
                "stage": "implementation",
                "action": second.action.value,
                "selected_options": second.selected_options,
                "constraints": second.constraints,
            }
        )
        usage.human_approval_requests += 1
        if second.action not in {ApprovalAction.APPROVE, ApprovalAction.CHOOSE_OPTION}:
            return None
        constraints.extend(second.constraints)
        approved = [f"DeepSeek-recommended direction approved: {analysis[:500]}"]
        return constraints, approved, True, False

    def _build_approval_request(
        self, prompt: str, context: str | None, classification: DecisionClassification
    ) -> ApprovalRequest:
        signal_names = classification.signal_names
        if classification.level == DecisionLevel.CRITICAL:
            why = (
                "This is a critical, potentially destructive or irreversible action. "
                "Compute Sifter will default to the safest non-destructive behavior until you decide."
            )
        else:
            why = (
                "This decision materially affects architecture, security, data, scope, "
                "external dependencies or cost. DeepSeek must not make it implicitly."
            )
        return ApprovalRequest(
            category=classification.level,
            issue=prompt,
            why_it_matters=why,
            current_approach=(context[:1500] if context else "Not specified."),
            options=_MAJOR_OPTIONS,
            recommended_action=(
                "Ask DeepSeek to compare the options and recommend; no implementation "
                "will occur until you separately approve it."
            ),
            cloud_reasoning_invoked=False,
            stage="analysis",
            context={"signals": signal_names, "reasons": classification.reasons},
        )

    async def _expert_analysis(
        self,
        *,
        prompt: str,
        context: str | None,
        constraints: list[str],
        trace: Trace,
        usage: UsageMetrics,
        tracker: UsageTracker,
        budget: BudgetTracker,
    ) -> tuple[str, DecisionLevel]:
        packet = EscalationPacket(
            original_task=prompt,
            current_plan="Human has permitted expert analysis of major options.",
            reason_for_escalation="Major decision requires expert comparison of approved options.",
            human_constraints=list(constraints),
            open_questions=["Which option best fits the stated constraints?"],
        )
        redaction = self.handoff.redactor.redact(packet.to_prompt())
        if redaction.redacted:
            trace.log(
                "security", "redacted secrets before cloud handoff", patterns=redaction.findings
            )
        messages = [
            Message.system(
                "You are a senior engineer. Compare the options, state tradeoffs, and "
                "recommend ONE direction. Do NOT implement code. Respect the human "
                "constraints as authoritative and non-negotiable. End with a clear recommendation."
            ),
            Message.user(
                redaction.text + (f"\n\nAdditional context:\n{context[:4000]}" if context else "")
            ),
        ]
        answer, _ = await self._call_tier(
            Tier.EXPERT,
            messages,
            trace=trace,
            usage=usage,
            tracker=tracker,
            budget=budget,
            purpose="expert_analysis",
        )
        return answer, DecisionLevel.MAJOR

    async def _execute_route(
        self,
        *,
        prompt: str,
        context: str | None,
        route: RouteDecision,
        assessment: PreflightAssessment,
        classification: DecisionClassification,
        human_constraints: list[str],
        approved_decisions: list[str],
        trace: Trace,
        usage: UsageMetrics,
        tracker: UsageTracker,
        budget: BudgetTracker,
    ) -> tuple[str, VerificationResult, int]:
        base_messages = self._execution_messages(
            prompt, context, human_constraints, approved_decisions
        )
        failed_attempts = 0

        if route.kind == RouteKind.CLOUD:
            answer, _ = await self._call_tier(
                Tier.EXPERT,
                base_messages,
                trace=trace,
                usage=usage,
                tracker=tracker,
                budget=budget,
                purpose="cloud_execute",
            )
            verification = await self.verifier.verify(answer=answer)
            _emit_progress("verifying", passed=verification.passed, stage="cloud")
            return answer, verification, failed_attempts

        if route.kind == RouteKind.HYBRID:
            plan_messages = [
                Message.system(
                    "Produce a concise implementation plan with acceptance criteria and "
                    "likely files. Do NOT write the final answer yet. Respect human constraints."
                ),
                Message.user(prompt),
            ]
            plan, _ = await self._call_tier(
                Tier.EXPERT,
                plan_messages,
                trace=trace,
                usage=usage,
                tracker=tracker,
                budget=budget,
                purpose="cloud_plan",
            )
            exec_messages = self._execution_messages(
                prompt, context, human_constraints, approved_decisions + [f"Plan: {plan}"]
            )
            answer, _ = await self._call_tier(
                Tier.LOCAL,
                exec_messages,
                trace=trace,
                usage=usage,
                tracker=tracker,
                budget=budget,
                purpose="local_execute",
            )
            verification = await self.verifier.verify(answer=answer)
            _emit_progress("verifying", passed=verification.passed, stage="hybrid")
            if not verification.passed:
                failed_attempts += 1
            else:
                review_messages = [
                    Message.system(
                        "Review the implementation against the plan. Point out concrete issues only."
                    ),
                    Message.user(f"Plan:\n{plan}\n\nImplementation:\n{answer}"),
                ]
                try:
                    review, _ = await self._call_tier(
                        Tier.EXPERT,
                        review_messages,
                        trace=trace,
                        usage=usage,
                        tracker=tracker,
                        budget=budget,
                        purpose="cloud_review",
                    )
                    answer = f"{answer}\n\n---\nExpert review:\n{review}"
                except BudgetExceeded:
                    trace.log("budget", "skipping expert review", level="warning")
            return answer, verification, failed_attempts

        # LOCAL with bounded retries and repair.
        answer = ""
        verification = VerificationResult(passed=False)
        max_attempts = self.config.local_max_retries + 1
        for attempt in range(1, max_attempts + 1):
            trace.log("qwen", f"attempt={attempt}")
            messages = base_messages
            if attempt > 1:
                messages = base_messages + [
                    Message.assistant(answer),
                    Message.user(
                        "The previous attempt did not pass verification. Fix the specific "
                        "problem and return a corrected complete answer."
                    ),
                ]
            try:
                answer, _ = await self._call_tier(
                    Tier.LOCAL,
                    messages,
                    trace=trace,
                    usage=usage,
                    tracker=tracker,
                    budget=budget,
                    purpose="local_execute",
                )
            except ProviderError as exc:
                trace.log("qwen", f"local inference failed: {exc}", level="warning")
                failed_attempts += 1
                answer = f"Local model error: {exc}"
                continue
            verification = await self.verifier.verify(answer=answer)
            _emit_progress("verifying", passed=verification.passed, stage="local")
            if verification.passed:
                return answer, verification, failed_attempts
            failed_attempts += 1
            usage.retries += 1
        return answer, verification, failed_attempts

    def _execution_messages(
        self,
        prompt: str,
        context: str | None,
        human_constraints: list[str],
        approved_decisions: list[str],
    ) -> list[Message]:
        system = (
            "You are a capable local software engineer. Follow existing project patterns. "
            "Return a complete, coherent answer."
        )
        if human_constraints or approved_decisions:
            authoritative = []
            if human_constraints:
                authoritative.append(
                    "HUMAN CONSTRAINTS (authoritative, must not be reinterpreted as optional):\n"
                    + "\n".join(f"- {c}" for c in human_constraints)
                )
            if approved_decisions:
                authoritative.append(
                    "APPROVED DECISIONS (authoritative):\n"
                    + "\n".join(f"- {d}" for d in approved_decisions)
                )
            system += "\n\n" + "\n\n".join(authoritative)
        messages = [Message.system(system)]
        if context:
            messages.append(Message.user(f"Context:\n{context}"))
        messages.append(Message.user(prompt))
        return messages

    async def _escalate_after_failure(
        self,
        *,
        prompt: str,
        answer: str,
        verification: VerificationResult,
        failed_attempts: int,
        classification: DecisionClassification,
        human_constraints: list[str],
        approved_decisions: list[str],
        gate: DecisionGate,
        trace: Trace,
        usage: UsageMetrics,
        tracker: UsageTracker,
        budget: BudgetTracker,
        approvals: list[dict[str, Any]],
        allowed: bool,
        autonomy_grant: bool = False,
    ) -> tuple[str, bool]:
        architecture_like = self.escalation_policy.is_architecture_like(prompt)
        request = EscalationRequest(
            reason=f"local verification failed: {verification.summary}",
            failed_attempts=failed_attempts,
            recommended_action=(
                EscalationAction.CLOUD_PLAN if architecture_like else EscalationAction.CLOUD_DEBUG
            ),
            potential_major_decision=architecture_like and classification.is_major,
        )
        outcome: EscalationOutcome = self.escalation_policy.evaluate(request)
        usage.escalations += 1
        _emit_progress("escalating", reason=request.reason, action=outcome.action.value)

        if outcome.requires_human_approval and not autonomy_grant:
            trace.log("decision", "escalation implies architecture change; asking user")
            decision = await gate.require_approval(
                ApprovalRequest(
                    category=DecisionLevel.MAJOR,
                    issue=(
                        "The local model failed repeatedly and the failure appears to imply "
                        "an architectural change."
                    ),
                    why_it_matters="This exceeds routine debugging and may change project direction.",
                    options=_MAJOR_OPTIONS,
                    recommended_action=(
                        "Ask DeepSeek to diagnose, but do not implement architectural changes "
                        "without a separate approval."
                    ),
                    cloud_reasoning_invoked=False,
                    stage="escalation",
                )
            )
            approvals.append(
                {
                    "stage": "escalation",
                    "action": decision.action.value,
                    "selected_options": decision.selected_options,
                }
            )
            usage.human_approval_requests += 1
            if not decision.authorized:
                return (
                    answer + "\n\n[Blocked: architectural escalation requires human approval.]",
                    False,
                )
            allowed = decision.action == ApprovalAction.ALLOW_EXPERT_ANALYSIS
        elif outcome.requires_human_approval and autonomy_grant:
            trace.log(
                "decision",
                "standing rule grants autonomous cloud escalation",
                level="warning",
            )
        if not allowed:
            return answer + "\n\n[Local-only policy: cloud escalation not permitted.]", False

        packet = EscalationPacket(
            original_task=prompt,
            current_plan="Local execution attempted and failed verification.",
            completed_work=answer,
            failed_attempts=[f"attempt {i + 1} failed" for i in range(failed_attempts)],
            test_results=verification.summary,
            reason_for_escalation=request.reason,
            human_constraints=human_constraints,
            approved_decisions=approved_decisions,
            decision_level=classification.level,
        )
        redaction = self.handoff.redactor.redact(packet.to_prompt())
        messages = [
            Message.system(
                "You are a senior engineer diagnosing a failed attempt. Identify the concrete "
                "root cause and provide corrected complete work. Respect authoritative constraints."
            ),
            Message.user(redaction.text),
        ]
        try:
            diagnosis, _ = await self._call_tier(
                Tier.EXPERT,
                messages,
                trace=trace,
                usage=usage,
                tracker=tracker,
                budget=budget,
                purpose="cloud_escalation",
            )
            return f"{answer}\n\n---\nExpert escalation:\n{diagnosis}", True
        except BudgetExceeded as exc:
            return answer + f"\n\n[Escalation skipped: {exc}]", False

    @staticmethod
    def _is_cloud_bound(tier: Tier, model: ModelConfig) -> bool:
        """True when a call can leave the machine (expert tier or ``:cloud`` model)."""
        return tier == Tier.EXPERT or str(model.name).endswith(":cloud")

    def _redact_cloud_messages(
        self, messages: list[Message], trace: Trace, purpose: str
    ) -> list[Message]:
        redacted, findings = self.redactor.redact_model_messages(messages)
        if findings:
            trace.log(
                "security",
                "redacted secrets before cloud handoff",
                purpose=purpose,
                patterns=findings,
            )
        return redacted

    def _fit_to_context(
        self, messages: list[Message], model: ModelConfig, max_output: int
    ) -> list[Message]:
        """Bound the outbound messages to the model context window (ADR-P4-004)."""
        limit = int(model.context_limit or 0)
        if limit <= 0:
            return list(messages)
        budget = max(1_024, limit - max(0, max_output))
        return fit_messages(list(messages), max_tokens=budget)

    async def _call_tier(
        self,
        tier: Tier,
        messages: Sequence[Message],
        *,
        trace: Trace,
        usage: UsageMetrics,
        tracker: UsageTracker,
        budget: BudgetTracker,
        purpose: str,
        temperature: float | None = None,
    ) -> tuple[str, GenerationResult]:
        model = self.local_config() if tier == Tier.LOCAL else self.cloud_config()
        provider = self._provider_for(model)
        tag = "local" if tier == Tier.LOCAL else "expert"
        outbound = self._is_cloud_bound(tier, model)
        if tier == Tier.LOCAL:
            max_output = self.config.local_max_output_tokens or DEFAULT_LOCAL_MAX_OUTPUT
        else:
            max_output = self.config.expert_max_output_tokens or DEFAULT_EXPERT_MAX_OUTPUT
        prepared = self._fit_to_context(list(messages), model, max_output)
        if outbound:
            if not self.config.cloud_allowed:
                raise ProviderError(
                    "cloud is disabled by the active profile (cloud_allowed = false)"
                )
            prepared = self._redact_cloud_messages(prepared, trace, purpose)
        if tier == Tier.EXPERT:
            input_tokens = sum(estimate_tokens(message.content) for message in prepared)
            budget.check(
                estimated_tokens=input_tokens,
                estimated_cost=model.estimate_cost(input_tokens, 0),
            )
        if temperature is None:
            temperature = (
                self.config.local_temperature
                if tier == Tier.LOCAL
                else self.config.expert_temperature
            )
        if temperature is None:
            temperature = 0.2
        _emit_progress(
            "local_call" if tier == Tier.LOCAL else "expert_call",
            purpose=purpose,
            model=model.name,
            provider=model.provider,
        )
        start = time.perf_counter()
        result = await provider.generate(
            model.name,
            prepared,
            temperature=temperature,
            max_tokens=max_output,
            timeout=model.timeout_seconds,
        )
        usage.record_generation(result, model)
        tracker.record(result)
        if tier == Tier.EXPERT:
            budget.record(result, model)
        trace.log(
            tag,
            f"{purpose}",
            input_tokens=result.input_tokens,
            output_tokens=result.output_tokens,
            duration_ms=round((time.perf_counter() - start) * 1000.0, 1),
        )
        return result.text, result

    def _emit_memory_candidate(self, result: SifterResult) -> None:
        try:
            record = task_candidate(
                trace_id=result.trace_id,
                status=result.status,
                route=str(result.route),
                decision_level=result.decision_level.value,
                requires_human_approval=result.requires_human_approval,
                prompt=str(result.metadata.get("prompt", "")),
                cloud_calls=result.usage.cloud_calls,
                cloud_cost=result.usage.cloud_cost,
            )
            self.memory_sink.emit(record)
        except Exception:  # noqa: BLE001 - a memory sink must never break a run
            return None

    def _finish(
        self,
        trace: Trace,
        *,
        status: str,
        answer: str,
        classification: DecisionClassification | None,
        assessment: PreflightAssessment | None,
        usage: UsageMetrics,
        approvals: list[dict[str, Any]],
        metadata: dict[str, Any],
        blocked_reason: str | None,
        start: float,
        route: str | None = None,
    ) -> SifterResult:
        usage.wall_clock_ms = (time.perf_counter() - start) * 1000.0
        trace.log(
            "aa_sifter",
            "complete",
            status=status,
            cloud_calls=usage.cloud_calls,
            cost=round(usage.cloud_cost, 6),
            duration_ms=round(usage.wall_clock_ms, 1),
        )
        _emit_progress(
            "complete",
            status=status,
            cloud_calls=usage.cloud_calls,
            cloud_cost=round(usage.cloud_cost, 6),
            duration_ms=round(usage.wall_clock_ms, 1),
        )
        result = SifterResult(
            trace_id=trace.trace_id,
            status=status,
            answer=answer,
            route=route or (assessment.recommended_route if assessment else "local"),
            decision_level=classification.level if classification else DecisionLevel.ROUTINE,
            requires_human_approval=bool(classification and classification.requires_human_approval),
            blocked_reason=blocked_reason,
            assessment=assessment,
            usage=usage,
            approvals=approvals,
            metadata={
                **metadata,
                "cloud_cost": round(usage.cloud_cost, 6),
                "trace_events": trace.events if self.config.debug else [],
            },
        )
        self._emit_memory_candidate(result)
        if self.history is not None:
            try:
                self.history.record_run(
                    {
                        "trace_id": trace.trace_id,
                        "prompt": metadata.get("prompt", ""),
                        "task_type": assessment.task_type if assessment else None,
                        "complexity": assessment.complexity if assessment else None,
                        "decision_level": classification.level.value if classification else None,
                        "approval_requested": bool(
                            classification and classification.requires_human_approval
                        ),
                        "user_decision": approvals[-1]["action"] if approvals else None,
                        "route": result.route,
                        "policy": metadata.get("policy"),
                        "models": {
                            "local": self.config.local_model,
                            "cloud": self.config.cloud_model,
                        },
                        "input_tokens": usage.local_input_tokens + usage.cloud_input_tokens,
                        "output_tokens": usage.local_output_tokens + usage.cloud_output_tokens,
                        "cloud_calls": usage.cloud_calls,
                        "cloud_cost": usage.cloud_cost,
                        "retries": usage.retries,
                        "escalations": usage.escalations,
                        "verification_success": result.ok,
                        "final_success": result.ok,
                        "duration_ms": usage.wall_clock_ms,
                        "metadata": metadata,
                    }
                )
            except Exception:
                pass
        return result

    async def generate(
        self,
        messages: Sequence[Message],
        *,
        tier: str = "local",
        metadata: dict[str, Any] | None = None,
    ) -> GenerationResult:
        """Low-level inference entry point for aa-sync agents.

        Major-decision gating is the caller's responsibility when using this
        method directly; prefer :meth:`run` for full governance. Cloud egress is
        still redacted and the cloud-disabled profile is still enforced.
        """
        selected = Tier.EXPERT if str(tier).lower() in {"expert", "cloud"} else Tier.LOCAL
        model = self.cloud_config() if selected == Tier.EXPERT else self.local_config()
        provider = self._provider_for(model)
        prepared = list(messages)
        if self._is_cloud_bound(selected, model):
            if not self.config.cloud_allowed:
                raise ProviderError(
                    "cloud is disabled by the active profile (cloud_allowed = false)"
                )
            prepared, _findings = self.redactor.redact_model_messages(prepared)
        return await provider.generate(model.name, prepared, timeout=model.timeout_seconds)

    def as_inference_layer(self) -> ComputeSifter:
        return self

    async def aclose(self) -> None:
        try:
            await self.registry.aclose()
        finally:
            if isinstance(self.history, SqliteHistoryStore):
                self.history.close()
