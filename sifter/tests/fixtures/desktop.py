from __future__ import annotations

import threading
from dataclasses import dataclass

import httpx

from aa_sifter.desktop.server import ComputeSifterServer, create_server, server_url
from aa_sifter.desktop.service import ComputeSifterService


@dataclass
class RunningServer:
    service: ComputeSifterService
    server: ComputeSifterServer
    thread: threading.Thread
    token: str

    @property
    def url(self) -> str:
        return server_url(self.server)

    def client(self, timeout: float = 15.0) -> httpx.Client:
        return httpx.Client(
            base_url=self.url,
            headers={"X-Sifter-Token": self.token},
            timeout=timeout,
        )

    def close(self) -> None:
        try:
            self.server.shutdown()
        except Exception:  # noqa: BLE001
            pass
        self.server.server_close()
        self.thread.join(timeout=5)
        self.service.close()


def start_server(service: ComputeSifterService, token: str = "test-token") -> RunningServer:
    server = create_server(service, host="127.0.0.1", port=0, token=token)
    thread = threading.Thread(target=server.serve_forever, daemon=True, name="test-server")
    thread.start()
    return RunningServer(service, server, thread, token)
