from __future__ import annotations

from enum import StrEnum

from pydantic import BaseModel, Field


class Capability(StrEnum):
    GENERAL = "general"
    CODING = "coding"
    REASONING = "reasoning"
    VISION = "vision"
    TOOL_USE = "tool_use"
    STRUCTURED_OUTPUT = "structured_output"
    LONG_CONTEXT = "long_context"
    AGENTIC = "agentic"
    RESEARCH = "research"


class CostMetadata(BaseModel):
    input_per_mtok: float = 0.0
    output_per_mtok: float = 0.0


class ModelMetadata(BaseModel):
    provider: str
    model_id: str
    display_name: str

    parameter_count_b: float | None = None
    minimum_vram_gb: float | None = None
    recommended_vram_gb: float | None = None
    minimum_ram_gb: float | None = None
    context_window: int | None = None

    capabilities: list[str] = Field(default_factory=list)
    strengths: list[str] = Field(default_factory=list)

    local: bool = False
    cloud: bool = False
    tool_use: bool = False
    structured_output: bool = False

    cost: CostMetadata | None = None

    def has_capabilities(self, required: set[str]) -> bool:
        if not required:
            return True
        return required.issubset(set(self.capabilities))

    def supports(self, capability: str) -> bool:
        return capability in self.capabilities

    @property
    def key(self) -> str:
        return f"{self.provider}/{self.model_id}"
