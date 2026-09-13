"""Deterministic model providers for tests.

No test in the default suite makes a network call. These fakes let a test
simulate local success/failure/timeout/malformed output and cloud
success/failure/timeout/malformed output.
"""

from __future__ import annotations

import asyncio
from collections.abc import AsyncIterator, Sequence
from typing import Any

from aa_sifter.models.provider import (
    GenerationResult,
    Message,
    ProviderError,
)

_FAILING_PREFIX = "could not produce valid output"

_KNOWN_CLOUD_MODELS = {"deepseek-chat", "deepseek-reasoner", "gpt-4o-mini"}


def _is_cloud(model: str) -> bool:
    return model.endswith(":cloud") or model in _KNOWN_CLOUD_MODELS


class FakeProvider:
    """Configurable deterministic provider that records every call.

    Counters count *attempts* (including failed attempts), which lets tests
    assert retry and escalation behaviour precisely.
    """

    name = "ollama"

    def __init__(
        self,
        *,
        local_text: str = "local answer",
        cloud_text: str = "cloud answer",
        structured: dict[str, Any] | None = None,
        fail_local: bool = False,
        fail_cloud: bool = False,
        fail_local_times: int = 0,
        timeout_local: bool = False,
        timeout_cloud: bool = False,
        malformed: bool = False,
        delay: float = 0.0,
        input_tokens: int = 100,
        output_tokens: int = 50,
    ):
        self.local_text = local_text
        self.cloud_text = cloud_text
        self.structured = structured
        self.fail_local = fail_local
        self.fail_cloud = fail_cloud
        self.fail_local_times = fail_local_times
        self.timeout_local = timeout_local
        self.timeout_cloud = timeout_cloud
        self.malformed = malformed
        self.delay = delay
        self.input_tokens = input_tokens
        self.output_tokens = output_tokens
        self.local_calls = 0
        self.cloud_calls = 0
        self.calls: list[str] = []
        self.messages: list[list[Message]] = []
        self.models: list[str] = []

    # -- helpers -----------------------------------------------------------
    def _text(self, cloud: bool) -> str:
        return self.cloud_text if cloud else self.local_text

    def _raise_if_failing(self, cloud: bool) -> None:
        if cloud:
            if self.timeout_cloud:
                raise ProviderError("cloud request timed out", retryable=True)
            if self.fail_cloud:
                raise ProviderError("cloud unavailable (402)", status_code=402)
            return
        if self.timeout_local:
            raise ProviderError("local request timed out", retryable=True)
        if self.fail_local and (
            self.fail_local_times == 0 or self.local_calls <= self.fail_local_times
        ):
            raise ProviderError("local inference failed", retryable=True)

    def _record(self, model: str, messages: Sequence[Message]) -> bool:
        cloud = _is_cloud(model)
        self.models.append(model)
        self.calls.append(model)
        self.messages.append(list(messages))
        if cloud:
            self.cloud_calls += 1
        else:
            self.local_calls += 1
        return cloud

    def _result(self, model: str, text: str, structured: dict[str, Any] | None) -> GenerationResult:
        return GenerationResult(
            text=text,
            model=model,
            tier="expert" if _is_cloud(model) else "local",
            input_tokens=self.input_tokens,
            output_tokens=self.output_tokens,
            duration_ms=1.0,
            structured=structured,
        )

    # -- ModelProvider protocol -------------------------------------------
    async def generate(
        self,
        model: str,
        messages: Sequence[Message],
        *,
        temperature: float = 0.2,
        max_tokens: int | None = None,
        timeout: float | None = None,
    ) -> GenerationResult:
        cloud = self._record(model, messages)
        if self.delay:
            await asyncio.sleep(self.delay)
        self._raise_if_failing(cloud)
        return self._result(model, self._text(cloud), None)

    async def generate_structured(
        self,
        model: str,
        messages: Sequence[Message],
        schema: dict[str, Any],
        *,
        temperature: float = 0.0,
        max_tokens: int | None = None,
        timeout: float | None = None,
    ) -> GenerationResult:
        cloud = self._record(model, messages)
        if self.delay:
            await asyncio.sleep(self.delay)
        self._raise_if_failing(cloud)
        if self.malformed:
            return self._result(model, _FAILING_PREFIX, None)
        return self._result(model, self._text(cloud), self.structured)

    def stream(
        self,
        model: str,
        messages: Sequence[Message],
        *,
        temperature: float = 0.2,
        timeout: float | None = None,
    ) -> AsyncIterator[str]:
        async def _gen() -> AsyncIterator[str]:
            yield self._text(_is_cloud(model))

        return _gen()

    async def aclose(self) -> None:
        return None


class FakeLocalProvider(FakeProvider):
    """Responds locally; cloud is treated as unavailable."""

    def __init__(self, **kwargs: Any):
        kwargs.setdefault("fail_cloud", True)
        super().__init__(**kwargs)


class FakeCloudProvider(FakeProvider):
    """Responds from the cloud tier."""

    def __init__(self, **kwargs: Any):
        kwargs.setdefault("local_text", "unused")
        super().__init__(**kwargs)


class FailingProvider(FakeProvider):
    """Fails on every tier."""

    def __init__(self, **kwargs: Any):
        kwargs.setdefault("fail_local", True)
        kwargs.setdefault("fail_cloud", True)
        super().__init__(**kwargs)


class TimeoutProvider(FakeProvider):
    """Times out on every tier."""

    def __init__(self, **kwargs: Any):
        kwargs.setdefault("timeout_local", True)
        kwargs.setdefault("timeout_cloud", True)
        super().__init__(**kwargs)


class MalformedProvider(FakeProvider):
    """Returns unstructured / unparseable output."""

    def __init__(self, **kwargs: Any):
        kwargs.setdefault("malformed", True)
        super().__init__(**kwargs)


class SlowProvider(FakeProvider):
    """Adds a delay to simulate slow inference."""

    def __init__(self, *, delay: float = 0.05, **kwargs: Any):
        super().__init__(delay=delay, **kwargs)


class ScriptedProvider(FakeProvider):
    """Returns a scripted sequence of outcomes per tier.

    Each outcome is either a string (successful text) or an exception instance.
    """

    def __init__(
        self,
        *,
        local_outcomes: Sequence[str | Exception] | None = None,
        cloud_outcomes: Sequence[str | Exception] | None = None,
        **kwargs: Any,
    ):
        super().__init__(**kwargs)
        self.local_outcomes = list(local_outcomes or [])
        self.cloud_outcomes = list(cloud_outcomes or [])

    def _next_outcome(self, cloud: bool) -> str | Exception | None:
        queue = self.cloud_outcomes if cloud else self.local_outcomes
        if not queue:
            return None
        return queue.pop(0)

    async def generate(
        self,
        model: str,
        messages: Sequence[Message],
        *,
        temperature: float = 0.2,
        max_tokens: int | None = None,
        timeout: float | None = None,
    ) -> GenerationResult:
        outcome = self._next_outcome(_is_cloud(model))
        cloud = self._record(model, messages)
        if isinstance(outcome, Exception):
            raise outcome
        return self._result(model, outcome if outcome is not None else self._text(cloud), None)

    async def generate_structured(
        self,
        model: str,
        messages: Sequence[Message],
        schema: dict[str, Any],
        *,
        temperature: float = 0.0,
        max_tokens: int | None = None,
        timeout: float | None = None,
    ) -> GenerationResult:
        outcome = self._next_outcome(_is_cloud(model))
        cloud = self._record(model, messages)
        if isinstance(outcome, Exception):
            raise outcome
        structured = None if self.malformed else self.structured
        text = outcome if outcome is not None else self._text(cloud)
        return self._result(model, text, structured)
