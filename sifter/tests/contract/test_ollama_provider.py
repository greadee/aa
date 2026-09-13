"""Contract tests for the Ollama provider HTTP behaviour.

All requests are intercepted by httpx.MockTransport, so these tests never
touch a real Ollama server or the network.
"""

from __future__ import annotations

import json

import httpx
import pytest

from aa_sifter.models.ollama import OllamaProvider
from aa_sifter.models.provider import Message, ProviderError, Tier


def _chat_response(text: str, *, model: str = "qwen3.5:9b") -> httpx.Response:
    return httpx.Response(
        200,
        json={
            "model": model,
            "message": {"role": "assistant", "content": text},
            "done": True,
            "prompt_eval_count": 11,
            "eval_count": 22,
            "total_duration": 5_000_000,
        },
    )


def _provider(handler, *, max_retries: int = 0) -> OllamaProvider:
    client = httpx.AsyncClient(
        base_url="http://ollama.test", transport=httpx.MockTransport(handler)
    )
    return OllamaProvider(host="http://ollama.test", client=client, max_retries=max_retries)


@pytest.mark.contract
@pytest.mark.asyncio
async def test_generate_formats_request_and_parses_usage():
    captured: dict = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["path"] = request.url.path
        captured["body"] = json.loads(request.content)
        return _chat_response("hello")

    provider = _provider(handler)
    result = await provider.generate(
        "qwen3.5:9b",
        [Message.system("sys"), Message.user("hi")],
        temperature=0.1,
        max_tokens=64,
    )

    assert captured["path"] == "/api/chat"
    body = captured["body"]
    assert body["model"] == "qwen3.5:9b"
    assert body["stream"] is False
    assert body["messages"][1] == {"role": "user", "content": "hi"}
    assert body["options"]["temperature"] == 0.1
    assert body["options"]["num_predict"] == 64
    assert result.text == "hello"
    assert result.input_tokens == 11
    assert result.output_tokens == 22
    assert result.tier == Tier.LOCAL
    await provider.aclose()


@pytest.mark.contract
@pytest.mark.asyncio
async def test_cloud_model_maps_to_expert_tier():
    provider = _provider(lambda request: _chat_response("cloud", model="deepseek-v4.1-flash:cloud"))
    result = await provider.generate("deepseek-v4.1-flash:cloud", [Message.user("hi")])
    assert result.tier == Tier.EXPERT
    assert result.model == "deepseek-v4.1-flash:cloud"
    await provider.aclose()


@pytest.mark.contract
@pytest.mark.asyncio
async def test_structured_generation_sends_schema_and_parses_json():
    captured: dict = {}
    schema = {"type": "object", "properties": {"answer": {"type": "string"}}}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["body"] = json.loads(request.content)
        return _chat_response('```json\n{"answer": "42"}\n```')

    provider = _provider(handler)
    result = await provider.generate_structured(
        "qwen3.5:9b", [Message.user("q")], schema, temperature=0.0
    )

    assert captured["body"]["format"] == schema
    assert result.structured == {"answer": "42"}
    await provider.aclose()


@pytest.mark.contract
@pytest.mark.asyncio
async def test_client_error_is_translated_and_not_retried():
    calls = {"n": 0}

    def handler(request: httpx.Request) -> httpx.Response:
        calls["n"] += 1
        return httpx.Response(402, json={"error": "payment required"})

    provider = _provider(handler, max_retries=3)
    with pytest.raises(ProviderError) as exc:
        await provider.generate("deepseek-v4.1-flash:cloud", [Message.user("hi")])
    assert exc.value.status_code == 402
    assert exc.value.retryable is False
    assert calls["n"] == 1, "non-retryable errors must not be retried"
    await provider.aclose()


@pytest.mark.contract
@pytest.mark.asyncio
async def test_server_error_is_retried_then_raised():
    calls = {"n": 0}

    def handler(request: httpx.Request) -> httpx.Response:
        calls["n"] += 1
        return httpx.Response(503, text="unavailable")

    provider = _provider(handler, max_retries=2)
    with pytest.raises(ProviderError) as exc:
        await provider.generate("qwen3.5:9b", [Message.user("hi")])
    assert exc.value.retryable is True
    assert calls["n"] == 3, "retries + initial attempt"
    await provider.aclose()


@pytest.mark.contract
@pytest.mark.asyncio
async def test_timeout_is_translated_to_retryable_error():
    def handler(request: httpx.Request) -> httpx.Response:
        raise httpx.TimeoutException("too slow", request=request)

    provider = _provider(handler)
    with pytest.raises(ProviderError) as exc:
        await provider.generate("qwen3.5:9b", [Message.user("hi")])
    assert exc.value.retryable is True
    await provider.aclose()


@pytest.mark.contract
@pytest.mark.asyncio
async def test_stream_yields_incremental_content():
    stream_body = (
        b'{"message": {"content": "hel"}}\n{"message": {"content": "lo"}}\n{"done": true}\n'
    )

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(200, content=stream_body)

    provider = _provider(handler)
    chunks = [chunk async for chunk in provider.stream("qwen3.5:9b", [Message.user("hi")])]
    assert "".join(chunks) == "hello"
    await provider.aclose()
