"""Manual live cloud-provider smoke test.

Never runs in normal CI. It is deselected by the default marker filter
(`not live`) and by `SIFTER_LIVE_SMOKE` being unset. The
`.github/workflows/live-provider-smoke.yml` workflow sets the variable and
makes exactly one minimal request.
"""

from __future__ import annotations

import os

import pytest

from aa_sifter.models.ollama import OllamaProvider
from aa_sifter.models.provider import Message

pytestmark = pytest.mark.live

_MARKER = "AA_SIFTER_DEEPSEEK_OK"


@pytest.mark.asyncio
async def test_live_cloud_returns_deterministic_marker():
    if os.environ.get("SIFTER_LIVE_SMOKE") != "1":
        pytest.skip("set SIFTER_LIVE_SMOKE=1 to run the paid live smoke test")

    host = os.environ.get("OLLAMA_HOST", "http://localhost:11434")
    model = os.environ.get("SIFTER_CLOUD_MODEL", "deepseek-v4.1-flash:cloud")

    provider = OllamaProvider(host=host, max_retries=0, default_timeout=60)
    try:
        result = await provider.generate(
            model,
            [Message.user(f"Return exactly this token and nothing else: {_MARKER}")],
            temperature=0.0,
            max_tokens=16,
        )
    finally:
        await provider.aclose()

    assert _MARKER in result.text
