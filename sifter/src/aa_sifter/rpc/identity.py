"""Peer and store identity for the aa inter-module RPC v1.

The v1 spec derives identity from the transport, never from the request body:
the server verifies the peer's OS user where the OS allows it, and binds itself
to one store identity so a caller attached to a different store is rejected
with ``aa.store_mismatch``. When the OS cannot report the peer's user (for
example a Windows named pipe without a SID backend), the owner-only socket and
its private directory are the fallback gate.
"""

from __future__ import annotations

import os
import socket
import struct
from collections.abc import Mapping
from dataclasses import dataclass
from typing import Any

from .envelope import STORE_MISMATCH, UNAUTHORIZED, failure

#: Field the additive v1 minor carries the caller's store identity in.
STORE_ID_FIELD = "storeId"


class IdentityError(Exception):
    """A transport or store identity check failed."""

    def __init__(self, code: str, message: str, *, data: dict[str, Any] | None = None) -> None:
        super().__init__(message)
        self.code = code
        self.message = message
        self.data = data

    def as_error(self, request_id: Any = None) -> dict[str, Any]:
        """Render this failure as a JSON-RPC error envelope."""
        return failure(request_id, self.code, self.message, data=self.data)


@dataclass(frozen=True, slots=True)
class PeerIdentity:
    """The OS user behind one accepted connection, when the OS reports it."""

    uid: int | None
    gid: int | None
    pid: int | None = None

    @property
    def verified(self) -> bool:
        """Whether the OS actually reported the peer's user."""
        return self.uid is not None


def _current_uid() -> int | None:
    getuid = getattr(os, "getuid", None)
    return getuid() if getuid is not None else None


def peer_credentials(connection: socket.socket) -> PeerIdentity:
    """Read the peer's credentials from an accepted Unix socket.

    Linux answers ``SO_PEERCRED``; the BSDs and macOS answer ``getpeereid``.
    When neither is available the identity is unverified, and callers fall back
    to the owner-only socket and its private directory.
    """
    option = getattr(socket, "SO_PEERCRED", None)
    if option is not None:
        try:
            raw = connection.getsockopt(socket.SOL_SOCKET, option, struct.calcsize("3i"))
        except OSError:
            return PeerIdentity(uid=None, gid=None, pid=None)
        pid, uid, gid = struct.unpack("3i", raw)
        return PeerIdentity(uid=uid, gid=gid, pid=pid)

    getpeereid = getattr(connection, "getpeereid", None)
    if getpeereid is not None:
        try:
            uid, gid = getpeereid()
        except OSError:
            return PeerIdentity(uid=None, gid=None, pid=None)
        return PeerIdentity(uid=uid, gid=gid, pid=None)

    return PeerIdentity(uid=None, gid=None, pid=None)


def verify_peer(
    connection: socket.socket,
    *,
    expected_uid: int | None = None,
) -> PeerIdentity:
    """Verify the peer's user matches the service owner.

    Raises :class:`IdentityError` (``aa.unauthorized``) on a mismatch. When the
    OS cannot report the peer's user the identity is returned unverified and the
    owner-only socket and directory are the gate instead.
    """
    identity = peer_credentials(connection)
    expected = _current_uid() if expected_uid is None else expected_uid
    if identity.uid is not None and expected is not None and identity.uid != expected:
        raise IdentityError(
            UNAUTHORIZED,
            "peer user does not own the service",
            data={"peerUid": identity.uid, "expectedUid": expected},
        )
    return identity


@dataclass(frozen=True, slots=True)
class StoreBinding:
    """The one store a service answers for over its lifetime.

    ``None`` means the service is unbound and accepts any caller. An unbound
    caller (no ``aa.storeId``) is always accepted, so adding the field is a
    minor, backward-compatible change.
    """

    store_id: str | None = None

    def check(self, requested: str | None) -> None:
        """Reject a caller bound to a different store."""
        if self.store_id is None or requested is None:
            return
        if requested != self.store_id:
            raise IdentityError(
                STORE_MISMATCH,
                "request is bound to a different store",
                data={"storeId": self.store_id, "requestedStoreId": requested},
            )


def requested_store_id(message: Mapping[str, Any]) -> str | None:
    """Read the caller's store identity from an envelope, if it carries one."""
    aa = message.get("aa")
    if isinstance(aa, Mapping):
        value = aa.get(STORE_ID_FIELD)
        if isinstance(value, str) and value:
            return value
    value = message.get(STORE_ID_FIELD)
    if isinstance(value, str) and value:
        return value
    return None
