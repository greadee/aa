from __future__ import annotations

from pydantic import BaseModel

from ..models.config import ModelConfig
from ..models.provider import GenerationResult, Tier


class UsageMetrics(BaseModel):
    local_input_tokens: int = 0
    local_output_tokens: int = 0
    cloud_input_tokens: int = 0
    cloud_output_tokens: int = 0
    local_calls: int = 0
    cloud_calls: int = 0
    cloud_cost: float = 0.0
    wall_clock_ms: float = 0.0
    local_inference_ms: float = 0.0
    cloud_inference_ms: float = 0.0
    tool_execution_ms: float = 0.0
    subtasks: int = 0
    escalations: int = 0
    retries: int = 0
    human_approval_requests: int = 0
    human_approved_cloud_analyses: int = 0
    major_decisions: int = 0

    def record_generation(
        self, result: GenerationResult, config: ModelConfig | None = None
    ) -> None:
        if result.tier == Tier.LOCAL:
            self.local_calls += 1
            self.local_input_tokens += result.input_tokens
            self.local_output_tokens += result.output_tokens
            self.local_inference_ms += result.duration_ms
        else:
            self.cloud_calls += 1
            self.cloud_input_tokens += result.input_tokens
            self.cloud_output_tokens += result.output_tokens
            self.cloud_inference_ms += result.duration_ms
            if config is not None:
                self.cloud_cost += config.estimate_cost(result.input_tokens, result.output_tokens)

    @property
    def total_tokens(self) -> int:
        return (
            self.local_input_tokens
            + self.local_output_tokens
            + self.cloud_input_tokens
            + self.cloud_output_tokens
        )

    def as_dict(self) -> dict[str, float | int]:
        return self.model_dump()


class UsageTracker:
    def __init__(self) -> None:
        self.metrics = UsageMetrics()
        self._local_config: ModelConfig | None = None
        self._cloud_config: ModelConfig | None = None

    def configure(
        self, *, local: ModelConfig | None = None, cloud: ModelConfig | None = None
    ) -> None:
        self._local_config = local
        self._cloud_config = cloud

    def record(self, result: GenerationResult) -> None:
        config = self._local_config if result.tier == Tier.LOCAL else self._cloud_config
        self.metrics.record_generation(result, config)

    def summary(self) -> dict[str, float | int]:
        data = self.metrics.model_dump()
        data["total_tokens"] = self.metrics.total_tokens
        return data


__all__ = ["UsageMetrics", "UsageTracker"]
