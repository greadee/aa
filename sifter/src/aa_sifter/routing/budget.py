from __future__ import annotations

from pydantic import BaseModel

from ..metrics.trace import Trace
from ..models.config import ModelConfig, SifterConfig
from ..models.provider import GenerationResult


class BudgetExceeded(RuntimeError):
    def __init__(self, reason: str, *, kind: str, limit: float, used: float):
        super().__init__(reason)
        self.kind = kind
        self.limit = limit
        self.used = used


class Budget(BaseModel):
    max_calls: int = 5
    max_tokens: int = 200_000
    max_cost: float = 2.0
    max_parallel: int = 2

    calls: int = 0
    tokens: int = 0
    cost: float = 0.0

    def can_call(self) -> bool:
        return self.calls < self.max_calls

    def would_exceed(self, *, estimated_tokens: int = 0, estimated_cost: float = 0.0) -> str | None:
        if self.calls + 1 > self.max_calls:
            return "max_cloud_calls"
        if self.tokens + estimated_tokens > self.max_tokens:
            return "max_cloud_tokens"
        if self.cost + estimated_cost > self.max_cost:
            return "max_cloud_cost"
        return None


class BudgetTracker:
    def __init__(
        self,
        config: SifterConfig,
        *,
        max_cost: float | None = None,
        trace: Trace | None = None,
        cost_waiver: float | None = None,
    ):
        self.config = config
        self.trace = trace
        configured = config.max_cloud_cost_per_task if max_cost is None else max_cost
        effective = configured
        if cost_waiver is not None and cost_waiver > configured:
            effective = cost_waiver
            if self.trace is not None:
                self.trace.log(
                    "budget",
                    "standing rule raised the cloud cost cap",
                    configured=configured,
                    waived_ceiling=cost_waiver,
                )
        self.waiver_ceiling = cost_waiver
        self.budget = Budget(
            max_calls=config.max_cloud_calls_per_task,
            max_tokens=config.max_cloud_tokens_per_task,
            max_cost=effective,
            max_parallel=config.max_parallel_cloud_calls,
        )
        self.exhausted = False
        self.exhausted_reason: str | None = None

    def check(self, *, estimated_tokens: int = 0, estimated_cost: float = 0.0) -> None:
        reason = self.budget.would_exceed(
            estimated_tokens=estimated_tokens, estimated_cost=estimated_cost
        )
        if reason is not None:
            self.exhausted = True
            self.exhausted_reason = reason
            if self.trace is not None:
                self.trace.log(
                    "budget",
                    f"cloud request denied ({reason})",
                    level="warning",
                    used_cost=round(self.budget.cost, 4),
                    limit_cost=self.budget.max_cost,
                    calls=self.budget.calls,
                )
            raise BudgetExceeded(
                f"Cloud budget exceeded: {reason}",
                kind=reason,
                limit={
                    "max_cloud_calls": self.budget.max_calls,
                    "max_cloud_tokens": self.budget.max_tokens,
                    "max_cloud_cost": self.budget.max_cost,
                }[reason],
                used={
                    "max_cloud_calls": self.budget.calls,
                    "max_cloud_tokens": self.budget.tokens,
                    "max_cloud_cost": self.budget.cost,
                }[reason],
            )

    def record(self, result: GenerationResult, config: ModelConfig) -> None:
        self.budget.calls += 1
        self.budget.tokens += result.input_tokens + result.output_tokens
        self.budget.cost += config.estimate_cost(result.input_tokens, result.output_tokens)

    def cost_waiver(self, rules_cost_threshold: float | None) -> bool:
        if rules_cost_threshold is None:
            return False
        return self.budget.cost <= rules_cost_threshold

    @property
    def remaining_cost(self) -> float:
        return max(0.0, self.budget.max_cost - self.budget.cost)

    @property
    def remaining_calls(self) -> int:
        return max(0, self.budget.max_calls - self.budget.calls)

    def needs_increase_approval(
        self, projected_cost: float, *, waiver: float | None = None
    ) -> bool:
        if waiver is not None and projected_cost <= waiver:
            return False
        expected = max(self.budget.cost, 0.01)
        return projected_cost > expected * 10 and projected_cost > self.budget.max_cost
