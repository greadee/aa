from __future__ import annotations

import asyncio
import json
import random
from collections.abc import AsyncIterator, Sequence
from typing import Any

import httpx

from .provider import (
    GenerationResult,
    Message,
    ModelProvider,
    ProviderError,
    Tier,
)


class OllamaProvider(ModelProvider):
    """Async Ollama provider implementing the ModelProvider protocol."""

    name = "ollama"

    def __init__(
        self,
        host: str = "http://localhost:11434",
        *,
        max_retries: int = 2,
        backoff_base: float = 0.5,
        default_timeout: float = 300.0,
        client: httpx.AsyncClient | None = None,
    ):
        self.host = host.rstrip("/")
        self.max_retries = max_retries
        self.backoff_base = backoff_base
        self.default_timeout = default_timeout
        self._client = client
        self._owns_client = client is None

    def _get_client(self) -> httpx.AsyncClient:
        if self._client is None:
            self._client = httpx.AsyncClient(base_url=self.host, timeout=self.default_timeout)
        return self._client

    async def aclose(self) -> None:
        if self._owns_client and self._client is not None:
            await self._client.aclose()
            self._client = None

    async def _post(
        self, path: str, payload: dict[str, Any], timeout: float | None
    ) -> dict[str, Any]:
        client = self._get_client()
        last_error: Exception | None = None
        for attempt in range(self.max_retries + 1):
            try:
                response = await client.post(
                    path, json=payload, timeout=timeout or self.default_timeout
                )
            except httpx.TimeoutException as exc:
                last_error = ProviderError(f"Ollama request timed out: {exc}", retryable=True)
            except httpx.TransportError as exc:
                last_error = ProviderError(f"Ollama transport error: {exc}", retryable=True)
            else:
                if response.status_code >= 500:
                    last_error = ProviderError(
                        f"Ollama server error {response.status_code}",
                        status_code=response.status_code,
                        retryable=True,
                    )
                elif response.status_code >= 400:
                    detail = _safe_detail(response)
                    raise ProviderError(
                        f"Ollama error {response.status_code}: {detail}",
                        status_code=response.status_code,
                        retryable=False,
                    )
                else:
                    return response.json()
            if attempt < self.max_retries:
                await asyncio.sleep(self.backoff_base * (2**attempt) + random.random() * 0.1)
        assert last_error is not None
        raise last_error

    async def generate(
        self,
        model: str,
        messages: Sequence[Message],
        *,
        temperature: float = 0.2,
        max_tokens: int | None = None,
        timeout: float | None = None,
    ) -> GenerationResult:
        payload: dict[str, Any] = {
            "model": model,
            "messages": [_message_payload(m) for m in messages],
            "stream": False,
            "options": _options(temperature, max_tokens),
        }
        data = await self._post("/api/chat", payload, timeout)
        return _parse_response(model, data)

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
        payload: dict[str, Any] = {
            "model": model,
            "messages": [_message_payload(m) for m in messages],
            "stream": False,
            "format": schema,
            "options": _options(temperature, max_tokens),
        }
        data = await self._post("/api/chat", payload, timeout)
        result = _parse_response(model, data)
        parsed = _try_parse_json(result.text)
        if parsed is not None:
            result.structured = parsed
        return result

    async def stream(
        self,
        model: str,
        messages: Sequence[Message],
        *,
        temperature: float = 0.2,
        timeout: float | None = None,
    ) -> AsyncIterator[str]:
        payload: dict[str, Any] = {
            "model": model,
            "messages": [_message_payload(m) for m in messages],
            "stream": True,
            "options": _options(temperature, None),
        }
        client = self._get_client()
        async with client.stream(
            "POST", "/api/chat", json=payload, timeout=timeout or self.default_timeout
        ) as response:
            if response.status_code >= 400:
                body = await response.aread()
                raise ProviderError(
                    f"Ollama stream error {response.status_code}: {body[:500]!r}",
                    status_code=response.status_code,
                )
            async for line in response.aiter_lines():
                if not line.strip():
                    continue
                try:
                    chunk = json.loads(line)
                except json.JSONDecodeError:
                    continue
                content = chunk.get("message", {}).get("content")
                if content:
                    yield content
                if chunk.get("done"):
                    break


def _message_payload(message: Message) -> dict[str, str]:
    return {"role": message.role.value, "content": message.content}


def _options(temperature: float, max_tokens: int | None) -> dict[str, Any]:
    options: dict[str, Any] = {"temperature": temperature}
    if max_tokens is not None:
        options["num_predict"] = max_tokens
    return options


def _parse_response(model: str, data: dict[str, Any]) -> GenerationResult:
    message = data.get("message", {}) or {}
    text = message.get("content", "") or data.get("response", "")
    tier = Tier.EXPERT if model.endswith(":cloud") else Tier.LOCAL
    return GenerationResult(
        text=text,
        model=data.get("model", model),
        tier=tier,
        input_tokens=int(data.get("prompt_eval_count") or 0),
        output_tokens=int(data.get("eval_count") or 0),
        duration_ms=float(data.get("total_duration") or 0) / 1_000_000.0,
        provider_raw={
            "done_reason": data.get("done_reason"),
            "load_duration": data.get("load_duration"),
        },
    )


def _safe_detail(response: httpx.Response) -> str:
    try:
        return response.text[:500]
    except Exception:  # pragma: no cover - defensive
        return "<unreadable body>"


def _try_parse_json(text: str) -> dict[str, Any] | None:
    candidate = text.strip()
    if not candidate:
        return None
    if candidate.startswith("```"):
        candidate = candidate.strip("`")
        if candidate.startswith("json"):
            candidate = candidate[4:]
        candidate = candidate.strip()
    start = candidate.find("{")
    end = candidate.rfind("}")
    if start == -1 or end == -1 or end < start:
        return None
    try:
        parsed = json.loads(candidate[start : end + 1])
    except json.JSONDecodeError:
        return None
    return parsed if isinstance(parsed, dict) else None
