"""Per-user local socket addresses for the aa inter-module RPC v1.

The v1 spec fixes the addressing scheme: a per-user named pipe on Windows
(``\\\\.\\pipe\\aa-<service>-v1``) and a per-user Unix domain socket elsewhere
(``$XDG_RUNTIME_DIR/aa-<service>-v1.sock``). This module resolves that address
without opening anything, so callers (listener, client, ``serve`` command) agree
on one endpoint.
"""

from __future__ import annotations

import os
import sys
import tempfile
from collections.abc import Mapping
from dataclasses import dataclass
from typing import Literal

SERVICE_NAME = "sifter"
PIPE_ROOT = "\\\\.\\pipe"

EndpointKind = Literal["unix", "pipe"]


@dataclass(frozen=True, slots=True)
class SocketEndpoint:
    """The address a local RPC service owns."""

    address: str
    kind: EndpointKind

    def __post_init__(self) -> None:
        if self.kind not in ("unix", "pipe"):
            raise ValueError(f"unknown endpoint kind {self.kind!r}")
        if not self.address:
            raise ValueError("endpoint address must not be empty")


def runtime_directory(
    *,
    env: Mapping[str, str] | None = None,
    uid: int | None = None,
) -> str:
    """The per-user directory that hosts Unix domain socket endpoints.

    ``$XDG_RUNTIME_DIR`` is used when it is set to an absolute path; otherwise a
    per-user directory under the system temp directory is used so a missing or
    relative XDG value never falls back to a shared path.
    """
    source = os.environ if env is None else env
    xdg = source.get("XDG_RUNTIME_DIR")
    if xdg and os.path.isabs(xdg):
        return xdg
    resolved_uid = getattr(os, "getuid", lambda: 0)() if uid is None else uid
    return os.path.join(tempfile.gettempdir(), f"aa-{resolved_uid}")


def default_endpoint(
    service: str = SERVICE_NAME,
    *,
    env: Mapping[str, str] | None = None,
    platform: str | None = None,
    uid: int | None = None,
) -> SocketEndpoint:
    """Resolve the default endpoint for a service on this platform."""
    resolved_platform = sys.platform if platform is None else platform
    if resolved_platform == "win32":
        return SocketEndpoint(address=f"{PIPE_ROOT}\\aa-{service}-v1", kind="pipe")
    directory = runtime_directory(env=env, uid=uid)
    return SocketEndpoint(address=os.path.join(directory, f"aa-{service}-v1.sock"), kind="unix")
