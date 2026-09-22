"""NDJSON framing tests for the aa inter-module RPC v1 transport."""

from __future__ import annotations

import json

import pytest

from aa_sifter.rpc.envelope import INVALID_REQUEST, PARSE_ERROR
from aa_sifter.rpc.framing import (
    DEFAULT_MAX_FRAME_BYTES,
    FrameReader,
    FramingError,
    decode_frame,
    encode_frame,
)


def test_encode_decode_roundtrip() -> None:
    message = {"jsonrpc": "2.0", "id": "rpc_1", "method": "sifter.health", "params": {}}

    frame = encode_frame(message)

    assert frame.endswith(b"\n")
    assert frame.count(b"\n") == 1
    assert decode_frame(frame[:-1]) == message


def test_encode_frame_is_compact_and_utf8() -> None:
    frame = encode_frame({"note": "café"})

    assert b"caf\xc3\xa9" in frame
    assert b": " not in frame
    assert frame.decode("utf-8").strip() == json.dumps(
        {"note": "café"}, separators=(",", ":"), ensure_ascii=False
    )


def test_encode_frame_rejects_non_mapping() -> None:
    with pytest.raises(FramingError) as excinfo:
        encode_frame(["not", "an", "object"])  # type: ignore[arg-type]

    assert excinfo.value.code == INVALID_REQUEST


def test_encode_frame_reports_unserializable_value() -> None:
    with pytest.raises(FramingError) as excinfo:
        encode_frame({"bad": object()})

    assert excinfo.value.code == INVALID_REQUEST
    assert "serializable" in excinfo.value.message


def test_decode_frame_rejects_malformed_json() -> None:
    with pytest.raises(FramingError) as excinfo:
        decode_frame(b'{"jsonrpc": "2.0"')

    assert excinfo.value.code == PARSE_ERROR


def test_decode_frame_rejects_non_object_json() -> None:
    for raw in (b"[]", b'"hello"', b"42", b"null"):
        with pytest.raises(FramingError) as excinfo:
            decode_frame(raw)
        assert excinfo.value.code == INVALID_REQUEST


def test_decode_frame_rejects_invalid_utf8() -> None:
    with pytest.raises(FramingError) as excinfo:
        decode_frame(b"\xff\xfe")

    assert excinfo.value.code == PARSE_ERROR


def test_decode_frame_accepts_crlf_terminator_body() -> None:
    assert decode_frame(b'{"id":"rpc_1"}\r') == {"id": "rpc_1"}


def test_reader_splits_multiple_frames_in_one_chunk() -> None:
    reader = FrameReader()
    chunk = encode_frame({"id": 1}) + encode_frame({"id": 2}) + encode_frame({"id": 3})

    messages = reader.feed(chunk)

    assert [message["id"] for message in messages] == [1, 2, 3]
    assert reader.pending_bytes == 0


def test_reader_reassembles_frame_across_chunks() -> None:
    reader = FrameReader()
    frame = encode_frame({"id": "rpc_split", "method": "sifter.route"})

    assert reader.feed(frame[:10]) == []
    assert reader.pending_bytes == 10
    messages = reader.feed(frame[10:])

    assert messages == [{"id": "rpc_split", "method": "sifter.route"}]
    assert reader.pending_bytes == 0


def test_reader_ignores_blank_lines() -> None:
    reader = FrameReader()

    messages = reader.feed(b"\n \n" + encode_frame({"id": 1}) + b"\r\n")

    assert messages == [{"id": 1}]


def test_reader_handles_crlf_frames() -> None:
    reader = FrameReader()

    messages = reader.feed(b'{"id": 1}\r\n{"id": 2}\r\n')

    assert [message["id"] for message in messages] == [1, 2]


def test_reader_feed_empty_chunk_is_noop() -> None:
    reader = FrameReader()

    assert reader.feed(b"") == []
    assert reader.pending_bytes == 0


def test_reader_rejects_oversized_incomplete_frame() -> None:
    reader = FrameReader(max_frame_bytes=8)

    reader.feed(b"12345")
    with pytest.raises(FramingError) as excinfo:
        reader.feed(b"6789")

    assert excinfo.value.code == INVALID_REQUEST
    assert "exceeds 8 bytes" in excinfo.value.message


def test_reader_rejects_oversized_complete_frame() -> None:
    reader = FrameReader(max_frame_bytes=8)

    with pytest.raises(FramingError):
        reader.feed(b'{"id": 123456}\n')


def test_reader_rejects_bad_json_frame() -> None:
    reader = FrameReader()

    with pytest.raises(FramingError) as excinfo:
        reader.feed(b"{not json}\n")

    assert excinfo.value.code == PARSE_ERROR


def test_reader_requires_positive_bound() -> None:
    with pytest.raises(ValueError):
        FrameReader(max_frame_bytes=0)


def test_default_bound_is_one_mib() -> None:
    assert DEFAULT_MAX_FRAME_BYTES == 1024 * 1024


def test_framing_error_renders_error_envelope() -> None:
    error = FramingError(PARSE_ERROR, "bad frame")

    envelope = error.as_error("rpc_9")

    assert envelope == {
        "jsonrpc": "2.0",
        "id": "rpc_9",
        "error": {"code": PARSE_ERROR, "message": "bad frame"},
    }
