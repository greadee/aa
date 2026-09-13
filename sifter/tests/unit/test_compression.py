from __future__ import annotations

from aa_sifter.context.compression import compress_messages, estimate_tokens, truncate
from aa_sifter.models.provider import Message


def test_estimate_tokens_is_positive():
    assert estimate_tokens("") == 1
    assert estimate_tokens("a" * 400) == 100


def test_compress_keeps_short_conversations_intact():
    messages = [Message.system("sys"), Message.user("hello")]
    assert compress_messages(messages, max_tokens=1000) == messages


def test_compress_summarizes_old_turns():
    messages = [Message.system("sys")]
    messages.extend(Message.user(f"message number {i}") for i in range(20))
    compressed = compress_messages(messages, max_tokens=10, keep_recent=4)
    assert len(compressed) < len(messages)
    assert compressed[0].content == "sys"
    assert compressed[-1].content == "message number 19"


def test_truncate_long_text():
    text = "a" * 10_000
    result = truncate(text, max_tokens=100)
    assert len(result) < len(text)
    assert "truncated" in result
