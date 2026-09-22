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
from .envelope import RPC_VERSION, RpcError
from .errors import ConflictError, UnavailableError, map_exception
from .framing import (
    DEFAULT_MAX_FRAME_BYTES,
    FrameReader,
    FramingError,
    decode_frame,
    encode_frame,
)
from .idempotency import (
    DEFAULT_WINDOW_SECONDS,
    IdempotencyStore,
    request_fingerprint,
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
from .server import RpcServer
from .service import SifterService

__all__ = [
    "DEFAULT_MAX_FRAME_BYTES",
    "DEFAULT_TIMEOUT_MS",
    "DEFAULT_WINDOW_SECONDS",
    "SERVICE_NAME",
    "STORE_ID_FIELD",
    "TIMEOUT_FIELD",
    "ConflictError",
    "DeadlineError",
    "FrameReader",
    "FramingError",
    "IdentityError",
    "IdempotencyStore",
    "LocalListener",
    "Ownership",
    "PeerIdentity",
    "RPC_VERSION",
    "RpcError",
    "RpcServer",
    "SifterService",
    "SocketEndpoint",
    "StoreBinding",
    "UnavailableError",
    "decode_frame",
    "default_endpoint",
    "dispatch_with_deadline",
    "encode_frame",
    "is_endpoint_live",
    "map_exception",
    "peer_credentials",
    "request_fingerprint",
    "requested_store_id",
    "resolve_timeout_ms",
    "runtime_directory",
    "verify_peer",
]
