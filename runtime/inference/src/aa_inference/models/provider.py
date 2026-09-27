from __future__ import annotations

import time
from collections.abc import AsyncIterator, Sequence
from enum import StrEnum
from typing import Any, Protocol, runtime_checkable

from pydantic import BaseModel, Field


class Tier(StrEnum):
    LOCAL = "local"
    EXPERT = "expert"


class Role(StrEnum):
    SYSTEM = "system"
    USER = "user"
    ASSISTANT = "assistant"
    TOOL = "tool"


class Message(BaseModel):
    role: Role
    content: str

    @classmethod
    def system(cls, content: str) -> Message:
        return cls(role=Role.SYSTEM, content=content)

    @classmethod
    def user(cls, content: str) -> Message:
        return cls(role=Role.USER, content=content)

    @classmethod
    def assistant(cls, content: str) -> Message:
        return cls(role=Role.ASSISTANT, content=content)


class GenerationResult(BaseModel):
    text: str
    model: str
    tier: Tier
    input_tokens: int = 0
    output_tokens: int = 0
    duration_ms: float = 0.0
    structured: dict[str, Any] | None = None
    provider_raw: dict[str, Any] = Field(default_factory=dict)


class ProviderError(RuntimeError):
    def __init__(self, message: str, *, status_code: int | None = None, retryable: bool = False):
        super().__init__(message)
        self.status_code = status_code
        self.retryable = retryable


@runtime_checkable
class ModelProvider(Protocol):
    name: str

    async def generate(
        self,
        model: str,
        messages: Sequence[Message],
        *,
        temperature: float = 0.2,
        max_tokens: int | None = None,
        timeout: float | None = None,
    ) -> GenerationResult: ...

    async def generate_structured(
        self,
        model: str,
        messages: Sequence[Message],
        schema: dict[str, Any],
        *,
        temperature: float = 0.0,
        max_tokens: int | None = None,
        timeout: float | None = None,
    ) -> GenerationResult: ...

    def stream(
        self,
        model: str,
        messages: Sequence[Message],
        *,
        temperature: float = 0.2,
        timeout: float | None = None,
    ) -> AsyncIterator[str]: ...

    async def aclose(self) -> None: ...


class ModelRegistry:
    """Maps model configurations to provider implementations by provider name."""

    def __init__(self, providers: dict[str, ModelProvider]):
        self._providers = dict(providers)

    def register(self, provider: ModelProvider) -> None:
        self._providers[provider.name] = provider

    def provider_for(self, model_name: str, default: str | None = None) -> ModelProvider:
        if ":" in model_name and model_name.split(":", 1)[1] == "cloud":
            if "ollama" in self._providers:
                return self._providers["ollama"]
        if default and default in self._providers:
            return self._providers[default]
        if len(self._providers) == 1:
            return next(iter(self._providers.values()))
        raise ProviderError(f"No provider registered for model '{model_name}'")

    def get(self, name: str) -> ModelProvider:
        if name not in self._providers:
            raise ProviderError(f"Provider '{name}' is not registered")
        return self._providers[name]

    async def aclose(self) -> None:
        for provider in self._providers.values():
            await provider.aclose()


def now_ms() -> float:
    return time.perf_counter() * 1000.0
