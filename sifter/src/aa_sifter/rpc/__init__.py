"""aa-sifter RPC service (inter-module JSON-RPC v1)."""

from .deadline import (
    DEFAULT_TIMEOUT_MS,
    TIMEOUT_FIELD,
    DeadlineError,
    dispatch_with_deadline,
    resolve_timeout_ms,
)
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
from .identity import (
    STORE_ID_FIELD,
    IdentityError,
    PeerIdentity,
    StoreBinding,
    peer_credentials,
    requested_store_id,
    verify_peer,
)
from .listener import LocalListener, Ownership, is_endpoint_live
from .service import SifterService

__all__ = [
    "DEFAULT_MAX_FRAME_BYTES",
    "DEFAULT_TIMEOUT_MS",
    "SERVICE_NAME",
    "STORE_ID_FIELD",
    "TIMEOUT_FIELD",
    "DeadlineError",
    "FrameReader",
    "FramingError",
    "IdentityError",
    "LocalListener",
    "Ownership",
    "PeerIdentity",
    "RPC_VERSION",
    "SifterService",
    "SocketEndpoint",
    "StoreBinding",
    "decode_frame",
    "default_endpoint",
    "dispatch_with_deadline",
    "encode_frame",
    "is_endpoint_live",
    "peer_credentials",
    "requested_store_id",
    "resolve_timeout_ms",
    "runtime_directory",
    "verify_peer",
]
