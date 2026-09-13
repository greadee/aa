from __future__ import annotations

import sys

import pytest

from aa_sifter.verification.verifier import (
    CallableVerifier,
    CommandVerifier,
    CompositeVerifier,
    FileChangeVerifier,
    NullVerifier,
)


@pytest.mark.asyncio
async def test_command_verifier_pass_and_fail():
    ok = CommandVerifier([f'"{sys.executable}" -c "raise SystemExit(0)"'])
    result = await ok.verify(answer="x")
    assert result.passed
    bad = CommandVerifier([f'"{sys.executable}" -c "raise SystemExit(1)"'])
    result = await bad.verify(answer="x")
    assert not result.passed


@pytest.mark.asyncio
async def test_callable_verifier():
    verifier = CallableVerifier("contains", lambda text: ("ok" in text, ""))
    assert (await verifier.verify(answer="ok done")).passed
    assert not (await verifier.verify(answer="nope")).passed


@pytest.mark.asyncio
async def test_file_change_verifier(tmp_path):
    (tmp_path / "created.py").write_text("x")
    verifier = FileChangeVerifier(["created.py", "missing.py"])
    result = await verifier.verify(answer="x", workdir=tmp_path)
    assert not result.passed
    assert any(c.passed for c in result.checks)


@pytest.mark.asyncio
async def test_composite_and_null():
    null = NullVerifier()
    assert (await null.verify(answer="x")).passed
    composite = CompositeVerifier(
        [CallableVerifier("a", lambda t: True), CallableVerifier("b", lambda t: False)]
    )
    assert not (await composite.verify(answer="x")).passed
