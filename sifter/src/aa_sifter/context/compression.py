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


def fit_messages(
    messages: list[Message],
    *,
    max_tokens: int,
    keep_recent: int = 6,
) -> list[Message]:
    """Deterministically fit messages into a token budget.

    First compact multi-turn history; if a single (often oversized) message still
    exceeds the budget, halve the largest messages until the budget is met. The
    result is content-deterministic for identical inputs.
    """
    compacted = compress_messages(messages, max_tokens=max_tokens, keep_recent=keep_recent)
    if sum(estimate_tokens(message.content) for message in compacted) <= max_tokens:
        return compacted
    result = list(compacted)
    while result and sum(estimate_tokens(m.content) for m in result) > max_tokens:
        index = max(range(len(result)), key=lambda i: estimate_tokens(result[i].content))
        current = estimate_tokens(result[index].content)
        if current <= 1:
            break
        reduced = truncate(result[index].content, max(1, current // 2))
        if estimate_tokens(reduced) >= current:
            break
        result[index] = result[index].model_copy(update={"content": reduced})
    return result
