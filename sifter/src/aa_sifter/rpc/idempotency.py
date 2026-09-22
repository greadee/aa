"""Durable idempotency for mutating RPC methods.

A repeated mutating call with the same ``idempotencyKey`` and the same request
returns the original result instead of running again. Reusing a key for a
*different* request is a conflict (``aa.conflict``). Records live for a bounded
retention window and, when a path is configured, survive a restart as an
append-only JSONL log that is compacted when expired entries are pruned.
"""

from __future__ import annotations

import contextlib
import hashlib
import json
import os
import tempfile
import time
from collections.abc import Callable
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from .errors import ConflictError

DEFAULT_WINDOW_SECONDS = 24 * 60 * 60


def request_fingerprint(payload: Any) -> str:
    """A stable, content-free fingerprint of a request payload."""
    canonical = json.dumps(payload, sort_keys=True, separators=(",", ":"), default=str)
    return hashlib.sha256(canonical.encode("utf-8")).hexdigest()


@dataclass(frozen=True, slots=True)
class _Record:
    fingerprint: str
    result: dict[str, Any]
    created_at: float


def _encode(key: str, record: _Record) -> str:
    return json.dumps(
        {
            "key": key,
            "fingerprint": record.fingerprint,
            "result": record.result,
            "createdAt": record.created_at,
        },
        separators=(",", ":"),
        ensure_ascii=False,
    )


class IdempotencyStore:
    """Records applied idempotency keys for a bounded retention window."""

    def __init__(
        self,
        path: str | Path | None = None,
        *,
        window_seconds: float = DEFAULT_WINDOW_SECONDS,
        clock: Callable[[], float] = time.time,
    ) -> None:
        if window_seconds <= 0:
            raise ValueError("window_seconds must be positive")
        self.path = Path(path) if path is not None else None
        self.window_seconds = window_seconds
        self._clock = clock
        self._records: dict[str, _Record] = {}
        if self.path is not None:
            self._load()

    @property
    def size(self) -> int:
        """The number of live (unexpired) records."""
        self._drop_expired()
        return len(self._records)

    def get(self, key: str) -> dict[str, Any] | None:
        """Return the live result for a key, or ``None``."""
        record = self._live(key)
        return None if record is None else record.result

    def resolve(self, key: str, fingerprint: str) -> dict[str, Any] | None:
        """The original result for an identical request; ``aa.conflict`` otherwise."""
        record = self._live(key)
        if record is None:
            return None
        if record.fingerprint != fingerprint:
            raise ConflictError(
                "idempotency key reused with a different request",
                data={"idempotencyKey": key},
            )
        return record.result

    def remember(self, key: str, fingerprint: str, result: dict[str, Any]) -> None:
        """Record the result of an applied key durably."""
        self._drop_expired()
        record = _Record(fingerprint=fingerprint, result=result, created_at=self._clock())
        self._records[key] = record
        self._append(key, record)

    def forget_expired(self) -> int:
        """Drop expired records from memory and return how many were removed."""
        before = len(self._records)
        self._drop_expired()
        return before - len(self._records)

    def prune(self) -> int:
        """Drop expired records and compact the durable log."""
        removed = self.forget_expired()
        if self.path is not None:
            self._rewrite()
        return removed

    def close(self) -> None:
        """Every append is flushed and fsynced, so there is nothing to close."""
        return None

    # -- internals ---------------------------------------------------------
    def _live(self, key: str) -> _Record | None:
        record = self._records.get(key)
        if record is None:
            return None
        if self._expired(record):
            self._records.pop(key, None)
            return None
        return record

    def _expired(self, record: _Record) -> bool:
        return self._clock() - record.created_at > self.window_seconds

    def _drop_expired(self) -> None:
        expired = [key for key, record in self._records.items() if self._expired(record)]
        for key in expired:
            self._records.pop(key, None)

    def _load(self) -> None:
        assert self.path is not None
        try:
            text = self.path.read_text(encoding="utf-8")
        except FileNotFoundError:
            return
        for line in text.splitlines():
            if not line.strip():
                continue
            try:
                raw = json.loads(line)
                record = _Record(
                    fingerprint=str(raw["fingerprint"]),
                    result=dict(raw["result"]),
                    created_at=float(raw["createdAt"]),
                )
                key = str(raw["key"])
            except (KeyError, TypeError, ValueError):
                continue
            if key and not self._expired(record):
                self._records[key] = record

    def _append(self, key: str, record: _Record) -> None:
        if self.path is None:
            return
        self.path.parent.mkdir(parents=True, exist_ok=True)
        with self.path.open("a", encoding="utf-8") as handle:
            handle.write(_encode(key, record) + "\n")
            handle.flush()
            os.fsync(handle.fileno())

    def _rewrite(self) -> None:
        assert self.path is not None
        self.path.parent.mkdir(parents=True, exist_ok=True)
        descriptor, temporary = tempfile.mkstemp(
            dir=str(self.path.parent), prefix=f"{self.path.name}.", suffix=".tmp"
        )
        try:
            with os.fdopen(descriptor, "w", encoding="utf-8") as handle:
                for key, record in self._records.items():
                    handle.write(_encode(key, record) + "\n")
                handle.flush()
                os.fsync(handle.fileno())
            os.replace(temporary, self.path)
        except BaseException:
            with contextlib.suppress(OSError):
                os.unlink(temporary)
            raise
