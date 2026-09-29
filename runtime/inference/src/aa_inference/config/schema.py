from __future__ import annotations

from typing import Any

from pydantic import BaseModel, Field, model_validator

CONFIG_VERSION = 1


class HardwareProfile(BaseModel):
    """Best-effort description of a machine. All fields are optional."""

    cpu_name: str | None = None
    cpu_cores: int | None = None
    cpu_threads: int | None = None

    gpu_name: str | None = None
    gpu_vendor: str | None = None
    gpu_vram_gb: float | None = None

    system_ram_gb: float | None = None

    operating_system: str | None = None
    architecture: str | None = None

    source: str = "manual"

    model_config = {"extra": "ignore"}

    @property
    def has_gpu(self) -> bool:
        return bool(self.gpu_name) and (self.gpu_vram_gb or 0) > 0


class UserGoal(BaseModel):
    workload: list[str] = Field(default_factory=list)

    quality_priority: float = 0.5
    speed_priority: float = 0.5
    cost_priority: float = 0.5
    privacy_priority: float = 0.5

    cloud_allowed: bool = True
    preferred_context: int | None = None

    model_config = {"extra": "ignore"}

    def required_capabilities(self) -> set[str]:
        capabilities: set[str] = set()
        workload = {item.lower() for item in self.workload}
        if workload & {
            "coding",
            "software development",
            "code generation",
            "software engineering",
            "software engineering agents",
            "local autonomous agents",
            "multi-agent orchestration",
        }:
            capabilities.add("coding")
        if workload & {
            "agents",
            "agent workflows",
            "software engineering agents",
            "local autonomous agents",
            "multi-agent orchestration",
        }:
            capabilities.update({"tool_use", "agentic"})
        if workload & {"reasoning", "research"}:
            capabilities.add("reasoning")
        if "large repository analysis" in workload or "summarization" in workload:
            capabilities.add("long_context")
        if "data analysis" in workload:
            capabilities.add("structured_output")
        if not capabilities:
            capabilities.add("general")
        return capabilities


class BudgetConfig(BaseModel):
    max_cloud_calls_per_task: int = 5
    max_cloud_tokens_per_task: int = 200_000
    max_cloud_cost_per_task: float = 2.0
    max_parallel_cloud_calls: int = 2
    max_parallel_local_calls: int = 1

    model_config = {"extra": "ignore"}


class ApprovalConfig(BaseModel):
    major: bool = True
    critical: bool = True
    significant: bool = False
    non_interactive: bool = False

    model_config = {"extra": "ignore", "validate_assignment": True}

    @model_validator(mode="after")
    def _protect_gates(self) -> ApprovalConfig:
        # Major and critical approval can never be silently disabled.
        if not self.major:
            object.__setattr__(self, "major", True)
        if not self.critical:
            object.__setattr__(self, "critical", True)
        return self


class UserPreferenceConfig(BaseModel):
    local_first: bool = True
    cost_sensitive: bool = True
    privacy_sensitive: bool = True
    cloud_allowed: bool = True
    preferred_context: int | None = None

    model_config = {"extra": "ignore"}


class ModelTierConfig(BaseModel):
    provider: str
    model: str

    endpoint: str | None = None
    api_key_env: str | None = None

    context_limit: int | None = None
    max_output_tokens: int | None = None
    temperature: float | None = None

    input_cost_per_mtok: float | None = None
    output_cost_per_mtok: float | None = None

    enabled: bool = True

    options: dict[str, Any] = Field(default_factory=dict)

    model_config = {"extra": "ignore"}


class ComputeProfile(BaseModel):
    name: str

    local: ModelTierConfig | None = None
    expert: ModelTierConfig | None = None
    local_fallbacks: list[ModelTierConfig] = Field(default_factory=list)

    routing_policy: str = "adaptive"

    budget: BudgetConfig = Field(default_factory=BudgetConfig)
    approval: ApprovalConfig = Field(default_factory=ApprovalConfig)
    preferences: UserPreferenceConfig = Field(default_factory=UserPreferenceConfig)
    hardware: HardwareProfile | None = None
    goal: UserGoal | None = None

    model_config = {"extra": "ignore"}


class ApplicationConfig(BaseModel):
    version: int = CONFIG_VERSION
    active_profile: str = "default"
    active_hardware: str | None = None
    profiles: dict[str, ComputeProfile] = Field(default_factory=dict)
    hardware: dict[str, HardwareProfile] = Field(default_factory=dict)

    model_config = {"extra": "ignore"}

    def active(self) -> ComputeProfile:
        if self.active_profile not in self.profiles:
            raise KeyError(f"active profile '{self.active_profile}' is not defined")
        return self.profiles[self.active_profile]


class Recommendation(BaseModel):
    profile: ComputeProfile
    reasons: list[str] = Field(default_factory=list)
    local_alternatives: list[ModelTierConfig] = Field(default_factory=list)
    expert_alternatives: list[ModelTierConfig] = Field(default_factory=list)
    confidence: str = "medium"
    notes: list[str] = Field(default_factory=list)
