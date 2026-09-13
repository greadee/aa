"""Contract tests for the OpenAI-compatible provider using httpx.MockTransport.

These tests never contact a real cloud endpoint.
"""

from __future__ import annotations

import json

import httpx
import pytest

from aa_sifter.models.openai_compatible import DeepSeekProvider, OpenAICompatibleProvider
from aa_sifter.models.provider import Message, ProviderError, Tier


def _chat_response(text: str = "hello", model: str = "deepseek-chat") -> httpx.Response:
    return httpx.Response(
        200,
        json={
            "model": model,
            "choices": [
                {"message": {"role": "assistant", "content": text}, "finish_reason": "stop"}
            ],
            "usage": {"prompt_tokens": 7, "completion_tokens": 3},
        },
    )


def _provider(handler, *, api_key: str | None = "test-key", max_retries: int = 0):
    client = httpx.AsyncClient(
        base_url="http://cloud.test/v1", transport=httpx.MockTransport(handler)
    )
    return OpenAICompatibleProvider(
        "http://cloud.test/v1", api_key=api_key, client=client, max_retries=max_retries
    )


@pytest.mark.contract
@pytest.mark.asyncio
async def test_request_shape_auth_and_usage():
    captured: dict = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["path"] = request.url.path
        captured["auth"] = request.headers.get("authorization")
        captured["body"] = json.loads(request.content)
        return _chat_response()

    provider = _provider(handler)
    result = await provider.generate(
        "deepseek-chat", [Message.system("sys"), Message.user("hi")], temperature=0.1, max_tokens=32
    )
    assert captured["path"] == "/v1/chat/completions"
    assert captured["auth"] == "Bearer test-key"
    body = captured["body"]
    assert body["model"] == "deepseek-chat"
    assert body["messages"][1] == {"role": "user", "content": "hi"}
    assert body["max_tokens"] == 32
    assert result.text == "hello"
    assert result.input_tokens == 7
    assert result.output_tokens == 3
    assert result.tier == Tier.EXPERT
    await provider.aclose()


@pytest.mark.contract
@pytest.mark.asyncio
async def test_deepseek_provider_defaults():
    provider = DeepSeekProvider(api_key="x")
    assert provider.name == "deepseek"
    assert provider.base_url == "https://api.deepseek.com/v1"
    await provider.aclose()


@pytest.mark.contract
@pytest.mark.asyncio
async def test_structured_requests_json_object_mode():
    captured: dict = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["body"] = json.loads(request.content)
        return _chat_response('{"answer": "42"}')

    provider = _provider(handler)
    result = await provider.generate_structured(
        "deepseek-chat", [Message.user("q")], {"type": "object"}, temperature=0.0
    )
    assert captured["body"]["response_format"] == {"type": "json_object"}
    assert result.structured == {"answer": "42"}
    await provider.aclose()


@pytest.mark.contract
@pytest.mark.asyncio
async def test_client_error_not_retried():
    calls = {"n": 0}

    def handler(request: httpx.Request) -> httpx.Response:
        calls["n"] += 1
        return httpx.Response(401, json={"error": "unauthorized"})

    provider = _provider(handler, max_retries=3)
    with pytest.raises(ProviderError) as exc:
        await provider.generate("deepseek-chat", [Message.user("hi")])
    assert exc.value.status_code == 401
    assert exc.value.retryable is False
    assert calls["n"] == 1
    await provider.aclose()


@pytest.mark.contract
@pytest.mark.asyncio
async def test_server_error_retried_then_raised():
    calls = {"n": 0}

    def handler(request: httpx.Request) -> httpx.Response:
        calls["n"] += 1
        return httpx.Response(503, text="unavailable")

    provider = _provider(handler, max_retries=2)
    with pytest.raises(ProviderError) as exc:
        await provider.generate("deepseek-chat", [Message.user("hi")])
    assert exc.value.retryable is True
    assert calls["n"] == 3
    await provider.aclose()


@pytest.mark.contract
@pytest.mark.asyncio
async def test_timeout_translated():
    def handler(request: httpx.Request) -> httpx.Response:
        raise httpx.TimeoutException("slow", request=request)

    provider = _provider(handler)
    with pytest.raises(ProviderError) as exc:
        await provider.generate("deepseek-chat", [Message.user("hi")])
    assert exc.value.retryable is True
    await provider.aclose()


@pytest.mark.contract
@pytest.mark.asyncio
async def test_stream_parses_sse():
    body = (
        b'data: {"choices": [{"delta": {"content": "hel"}}]}\n'
        b'data: {"choices": [{"delta": {"content": "lo"}}]}\n'
        b"data: [DONE]\n"
    )

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, content=body)

    provider = _provider(handler)
    chunks = [chunk async for chunk in provider.stream("deepseek-chat", [Message.user("hi")])]
    assert "".join(chunks) == "hello"
    await provider.aclose()
