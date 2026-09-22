"""Newline-delimited JSON-RPC 2.0 framing for the aa inter-module RPC v1.

Framing is transport-agnostic. The transport (named pipe / Unix domain socket)
owns the bytes; this module owns the boundary the v1 spec fixes: one JSON object
per line, each object bounded in size. A :class:`FrameReader` turns an arbitrary
stream of byte chunks into whole message objects, and :func:`encode_frame` turns
one message object back into a single newline-terminated frame.

A framing failure is fatal to the connection: the reader cannot know where the
next frame starts, so :class:`FramingError` reports the JSON-RPC code the server
should return (a parse error for malformed JSON, an invalid-request error for a
well-formed value that is not an object) and never silently resynchronizes.
"""

from __future__ import annotations

import json
from collections.abc import Mapping
from typing import Any

from .envelope import INVALID_REQUEST, PARSE_ERROR, failure

# Largest single frame body accepted, excluding its terminator.
DEFAULT_MAX_FRAME_BYTES = 1024 * 1024


class FramingError(Exception):
    """A frame could not be parsed or was not a JSON-RPC request object."""

    def __init__(self, code: int, message: str) -> None:
        super().__init__(message)
        self.code = code
        self.message = message

    def as_error(self, request_id: Any = None) -> dict[str, Any]:
        """Render this failure as a JSON-RPC error envelope."""
        return failure(request_id, self.code, self.message)


def encode_frame(message: Mapping[str, Any]) -> bytes:
    """Serialize one message object to a single newline-terminated frame."""
    if not isinstance(message, Mapping):
        raise FramingError(INVALID_REQUEST, "frame must be a JSON object")
    try:
        payload = json.dumps(dict(message), separators=(",", ":"), ensure_ascii=False)
    except (TypeError, ValueError) as exc:
        raise FramingError(INVALID_REQUEST, f"frame is not serializable: {exc}") from exc
    return payload.encode("utf-8") + b"\n"


def decode_frame(frame: bytes | str) -> dict[str, Any]:
    """Parse one frame body (without its terminator) into a message object."""
    if isinstance(frame, bytes):
        try:
            text = frame.decode("utf-8")
        except UnicodeDecodeError as exc:
            raise FramingError(PARSE_ERROR, "frame is not valid UTF-8") from exc
    else:
        text = frame
    try:
        value = json.loads(text)
    except json.JSONDecodeError as exc:
        raise FramingError(PARSE_ERROR, f"frame is not valid JSON: {exc.msg}") from exc
    if not isinstance(value, dict):
        raise FramingError(INVALID_REQUEST, "frame must be a JSON object")
    return value


class FrameReader:
    """Incrementally split a byte stream into newline-delimited JSON objects."""

    def __init__(self, *, max_frame_bytes: int = DEFAULT_MAX_FRAME_BYTES) -> None:
        if max_frame_bytes <= 0:
            raise ValueError("max_frame_bytes must be positive")
        self.max_frame_bytes = max_frame_bytes
        self._buffer = bytearray()

    def feed(self, chunk: bytes) -> list[dict[str, Any]]:
        """Append ``chunk`` and return every frame it completes.

        Frames may be split across chunks and several may arrive in one chunk.
        Blank lines are ignored. A frame that exceeds ``max_frame_bytes`` raises
        :class:`FramingError`, because the connection can no longer be parsed.
        """
        if not chunk:
            return []
        self._buffer.extend(chunk)
        messages: list[dict[str, Any]] = []
        while True:
            newline = self._buffer.find(b"\n")
            if newline == -1:
                break
            line = bytes(self._buffer[:newline])
            del self._buffer[: newline + 1]
            if not line.strip():
                continue
            messages.append(self._parse(line))
        if len(self._buffer) > self.max_frame_bytes:
            raise FramingError(
                INVALID_REQUEST,
                f"frame exceeds {self.max_frame_bytes} bytes",
            )
        return messages

    @property
    def pending_bytes(self) -> int:
        """Bytes buffered for an as-yet incomplete frame."""
        return len(self._buffer)

    def _parse(self, line: bytes) -> dict[str, Any]:
        if len(line) > self.max_frame_bytes:
            raise FramingError(
                INVALID_REQUEST,
                f"frame exceeds {self.max_frame_bytes} bytes",
            )
        return decode_frame(line)
