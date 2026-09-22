"""aa-sifter RPC service (inter-module JSON-RPC v1)."""

from .envelope import RPC_VERSION
from .framing import (
    DEFAULT_MAX_FRAME_BYTES,
    FrameReader,
    FramingError,
    decode_frame,
    encode_frame,
)
from .service import SifterService

__all__ = [
    "DEFAULT_MAX_FRAME_BYTES",
    "FrameReader",
    "FramingError",
    "RPC_VERSION",
    "SifterService",
    "decode_frame",
    "encode_frame",
]
