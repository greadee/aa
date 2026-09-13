from __future__ import annotations

import asyncio
import json
import os
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


class OpenAICompatibleProvider(ModelProvider):
    """Provider for OpenAI-compatible chat completion endpoints.

    Works with DeepSeek, vLLM, LM Studio, OpenRouter and similar services.
    """

    name = "openai-compatible"

    def __init__(
        self,
        base_url: str,
        *,
        api_key: str | None = None,
        api_key_env: str | None = None,
        max_retries: int = 2,
        backoff_base: float = 0.5,
        default_timeout: float = 300.0,
        client: httpx.AsyncClient | None = None,
    ):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key or (os.environ.get(api_key_env) if api_key_env else None)
        self.api_key_env = api_key_env
        self.max_retries = max_retries
        self.backoff_base = backoff_base
        self.default_timeout = default_timeout
        self._client = client
        self._owns_client = client is None

    def _get_client(self) -> httpx.AsyncClient:
        if self._client is None:
            self._client = httpx.AsyncClient(base_url=self.base_url, timeout=self.default_timeout)
        return self._client

    def _headers(self) -> dict[str, str]:
        headers = {"Content-Type": "application/json"}
        if self.api_key:
            headers["Authorization"] = f"Bearer {self.api_key}"
        return headers

    async def aclose(self) -> None:
        if self._owns_client and self._client is not None:
            await self._client.aclose()
            self._client = None

    async def _post(self, payload: dict[str, Any], timeout: float | None) -> dict[str, Any]:
        client = self._get_client()
        last_error: Exception | None = None
        for attempt in range(self.max_retries + 1):
            try:
                response = await client.post(
                    "/chat/completions",
                    json=payload,
                    headers=self._headers(),
                    timeout=timeout or self.default_timeout,
                )
            except httpx.TimeoutException as exc:
                last_error = ProviderError(f"request timed out: {exc}", retryable=True)
            except httpx.TransportError as exc:
                last_error = ProviderError(f"transport error: {exc}", retryable=True)
            else:
                if response.status_code >= 500:
                    last_error = ProviderError(
                        f"server error {response.status_code}",
                        status_code=response.status_code,
                        retryable=True,
                    )
                elif response.status_code >= 400:
                    raise ProviderError(
                        f"API error {response.status_code}: {_safe_detail(response)}",
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
            "messages": [_message_payload(message) for message in messages],
            "temperature": temperature,
            "stream": False,
        }
        if max_tokens is not None:
            payload["max_tokens"] = max_tokens
        data = await self._post(payload, timeout)
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
            "messages": [
                Message.system(
                    "Respond with a single valid JSON object matching this schema: "
                    + json.dumps(schema)
                ).content,
                *[_message_payload(message) for message in messages],
            ],
            "temperature": temperature,
            "stream": False,
            "response_format": {"type": "json_object"},
        }
        if max_tokens is not None:
            payload["max_tokens"] = max_tokens
        data = await self._post(payload, timeout)
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
        payload = {
            "model": model,
            "messages": [_message_payload(message) for message in messages],
            "temperature": temperature,
            "stream": True,
        }
        client = self._get_client()
        async with client.stream(
            "POST",
            "/chat/completions",
            json=payload,
            headers=self._headers(),
            timeout=timeout or self.default_timeout,
        ) as response:
            if response.status_code >= 400:
                body = await response.aread()
                raise ProviderError(
                    f"stream error {response.status_code}: {body[:500]!r}",
                    status_code=response.status_code,
                )
            async for line in response.aiter_lines():
                if not line.startswith("data:"):
                    continue
                chunk = line[len("data:") :].strip()
                if chunk == "[DONE]":
                    break
                try:
                    parsed = json.loads(chunk)
                except json.JSONDecodeError:
                    continue
                delta = parsed.get("choices", [{}])[0].get("delta", {})
                content = delta.get("content")
                if content:
                    yield content


class DeepSeekProvider(OpenAICompatibleProvider):
    name = "deepseek"

    def __init__(
        self,
        *,
        base_url: str = "https://api.deepseek.com/v1",
        api_key: str | None = None,
        api_key_env: str | None = "DEEPSEEK_API_KEY",
        **kwargs: Any,
    ):
        super().__init__(base_url, api_key=api_key, api_key_env=api_key_env, **kwargs)


def _message_payload(message: Message) -> dict[str, str]:
    return {"role": message.role.value, "content": message.content}


def _parse_response(model: str, data: dict[str, Any]) -> GenerationResult:
    choices = data.get("choices") or [{}]
    message = choices[0].get("message", {}) if choices else {}
    text = message.get("content", "") or ""
    usage = data.get("usage") or {}
    return GenerationResult(
        text=text,
        model=data.get("model", model),
        tier=Tier.EXPERT,
        input_tokens=int(usage.get("prompt_tokens") or 0),
        output_tokens=int(usage.get("completion_tokens") or 0),
        duration_ms=0.0,
        provider_raw={"finish_reason": choices[0].get("finish_reason") if choices else None},
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
