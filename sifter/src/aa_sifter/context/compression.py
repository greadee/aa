from __future__ import annotations

from ..models.provider import Message, Role


def estimate_tokens(text: str) -> int:
    return max(1, len(text) // 4)


def compress_messages(
    messages: list[Message],
    *,
    max_tokens: int,
    keep_recent: int = 6,
) -> list[Message]:
    """Deterministic context compaction: keep system messages and recent turns."""
    if not messages:
        return []
    total = sum(estimate_tokens(m.content) for m in messages)
    if total <= max_tokens:
        return list(messages)
    system = [m for m in messages if m.role == Role.SYSTEM]
    rest = [m for m in messages if m.role != Role.SYSTEM]
    recent = rest[-keep_recent:]
    older = rest[:-keep_recent] if len(rest) > keep_recent else []
    if not older:
        return system + recent
    summary_parts = []
    for message in older:
        snippet = message.content.strip().replace("\n", " ")
        summary_parts.append(f"{message.role.value}: {snippet[:160]}")
    summary = Message.system("Compressed earlier context:\n" + "\n".join(summary_parts))
    return system + [summary] + recent


def truncate(text: str, max_tokens: int) -> str:
    if estimate_tokens(text) <= max_tokens:
        return text
    max_chars = max_tokens * 4
    head = int(max_chars * 0.7)
    tail = max_chars - head
    return text[:head] + "\n...[truncated]...\n" + text[-tail:]
