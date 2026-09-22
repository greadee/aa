"""aa-sifter RPC service (inter-module JSON-RPC v1)."""

from .endpoint import (
    SERVICE_NAME,
    SocketEndpoint,
    default_endpoint,
    runtime_directory,
)
from .envelope import RPC_VERSION
from .framing import (
    DEFAULT_MAX_FRAME_BYTES,
    FrameReader,
    FramingError,
    decode_frame,
    encode_frame,
)
from .listener import LocalListener, Ownership, is_endpoint_live
from .service import SifterService

__all__ = [
    "DEFAULT_MAX_FRAME_BYTES",
    "SERVICE_NAME",
    "FrameReader",
    "FramingError",
    "LocalListener",
    "Ownership",
    "RPC_VERSION",
    "SifterService",
    "SocketEndpoint",
    "decode_frame",
    "default_endpoint",
    "encode_frame",
    "is_endpoint_live",
    "runtime_directory",
]
