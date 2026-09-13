from __future__ import annotations

import json
import queue
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import parse_qs, urlparse

from .service import ComputeSifterService

WEB_DIR = Path(__file__).parent / "web"
_STATIC_TYPES = {
    ".html": "text/html; charset=utf-8",
    ".js": "application/javascript; charset=utf-8",
    ".css": "text/css; charset=utf-8",
    ".svg": "image/svg+xml",
    ".ico": "image/x-icon",
}


class ComputeSifterServer(ThreadingHTTPServer):
    daemon_threads = True
    allow_reuse_address = False

    def __init__(self, address: tuple[str, int], service: ComputeSifterService, token: str):
        super().__init__(address, _Handler)
        self.service = service
        self.token = token
        self.started_at = time.time()
        self.shutdown_requested = threading.Event()


class _Handler(BaseHTTPRequestHandler):
    server_version = "ComputeSifter"

    # -- plumbing ----------------------------------------------------------
    def log_message(self, fmt: str, *args: object) -> None:  # silence default logging
        return

    @property
    def sifter_server(self) -> ComputeSifterServer:
        return self.server  # type: ignore[return-value]

    def _authorized(self, query: dict[str, list[str]]) -> bool:
        token = self.sifter_server.token
        header = self.headers.get("X-Sifter-Token")
        if header and header == token:
            return True
        auth = self.headers.get("Authorization", "")
        if auth.startswith("Bearer ") and auth[7:] == token:
            return True
        provided = (query.get("token") or [""])[0]
        return provided == token

    def _send_json(self, status: int, payload: object) -> None:
        body = json.dumps(payload, default=str).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _read_json(self) -> dict:
        length = int(self.headers.get("Content-Length") or 0)
        if length <= 0:
            return {}
        raw = self.rfile.read(length)
        try:
            return json.loads(raw.decode("utf-8"))
        except json.JSONDecodeError:
            return {}

    def _serve_static(self, relative: str) -> None:
        web_root = WEB_DIR.resolve()
        path = (WEB_DIR / relative).resolve()
        if not str(path).startswith(str(web_root)):
            self._send_json(404, {"error": "not found"})
            return
        if not path.exists() or not path.is_file():
            self._send_json(404, {"error": "not found"})
            return
        content = path.read_bytes()
        if path.name == "index.html":
            content = content.replace(b"__SIFTER_TOKEN__", self.sifter_server.token.encode())
        self.send_response(200)
        self.send_header("Content-Type", _STATIC_TYPES.get(path.suffix, "application/octet-stream"))
        self.send_header("Content-Length", str(len(content)))
        self.end_headers()
        self.wfile.write(content)

    # -- GET ---------------------------------------------------------------
    def do_GET(self) -> None:  # noqa: N802 - http.server API
        parsed = urlparse(self.path)
        path = parsed.path
        query = parse_qs(parsed.query)

        if path in {"/", "/index.html"}:
            self._serve_static("index.html")
            return
        if path in {"/app.js", "/styles.css"}:
            self._serve_static(path.lstrip("/"))
            return

        if not path.startswith("/api/"):
            self._send_json(404, {"error": "not found"})
            return
        if not self._authorized(query):
            self._send_json(401, {"error": "unauthorized"})
            return

        service = self.sifter_server.service
        if path == "/api/health":
            self._send_json(200, service.health())
        elif path == "/api/status":
            self._send_json(200, service.status())
        elif path == "/api/profiles":
            self._send_json(200, {"profiles": service.profiles()})
        elif path == "/api/models":
            self._send_json(200, service.models())
        elif path == "/api/system":
            self._send_json(200, service.system())
        elif path == "/api/doctor":
            live = (query.get("live") or ["false"])[0].lower() == "true"
            self._send_json(200, service.doctor(live=live))
        elif path == "/api/metrics":
            self._send_json(200, service.metrics())
        elif path == "/api/tasks":
            self._send_json(200, {"tasks": service.list_tasks()})
        elif path.startswith("/api/tasks/"):
            parts = path.strip("/").split("/")
            task_id = parts[2] if len(parts) > 2 else ""
            if len(parts) == 4 and parts[3] == "events":
                self._stream_events(task_id, query)
                return
            record = service.get(task_id)
            self._send_json(200, record.summary() if record else {"error": "not found"})
        else:
            self._send_json(404, {"error": "not found"})

    # -- SSE ---------------------------------------------------------------
    def _stream_events(self, task_id: str, query: dict[str, list[str]]) -> None:
        service = self.sifter_server.service
        record = service.get(task_id)
        if record is None:
            self._send_json(404, {"error": "not found"})
            return
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.send_header("Connection", "keep-alive")
        self.end_headers()

        subscriber = record.subscribe()
        try:
            for event in list(record.events):
                self._write_event(event)
            if record.terminal:
                return
            while True:
                try:
                    event = subscriber.get(timeout=15)
                except queue.Empty:
                    self.wfile.write(b": ping\n\n")
                    self.wfile.flush()
                    if record.terminal:
                        break
                    continue
                self._write_event(event)
                if event["event"] == "done":
                    break
        except (BrokenPipeError, ConnectionResetError, OSError):
            pass
        finally:
            record.unsubscribe(subscriber)

    def _write_event(self, event: dict) -> None:
        payload = json.dumps(event, default=str)
        self.wfile.write(f"data: {payload}\n\n".encode())
        self.wfile.flush()

    # -- POST --------------------------------------------------------------
    def do_POST(self) -> None:  # noqa: N802 - http.server API
        parsed = urlparse(self.path)
        path = parsed.path
        query = parse_qs(parsed.query)
        if not path.startswith("/api/"):
            self._send_json(404, {"error": "not found"})
            return
        if not self._authorized(query):
            self._send_json(401, {"error": "unauthorized"})
            return

        service = self.sifter_server.service
        body = self._read_json()

        if path == "/api/tasks":
            prompt = str(body.get("prompt", "")).strip()
            if not prompt:
                self._send_json(400, {"error": "prompt is required"})
                return
            record = service.submit(
                prompt,
                context=body.get("context"),
                policy=body.get("policy"),
                execution_mode=body.get("execution_mode", "adaptive"),
            )
            self._send_json(202, record.summary())
            return

        if path.startswith("/api/tasks/"):
            parts = path.strip("/").split("/")
            task_id = parts[2] if len(parts) > 2 else ""
            action = parts[3] if len(parts) > 3 else ""
            if action == "approval":
                ok = service.resolve_approval(
                    task_id,
                    action=str(body.get("action", "")),
                    selected_options=body.get("selected_options"),
                    constraints=body.get("constraints"),
                    note=str(body.get("note", "")),
                )
                self._send_json(200 if ok else 400, {"ok": ok})
                return
            if action == "cancel":
                self._send_json(200, {"cancelled": service.cancel(task_id)})
                return
            self._send_json(404, {"error": "not found"})
            return

        if path == "/api/system/detect":
            self._send_json(200, service.detect_system(save_as=body.get("save_as")))
            return

        if path.startswith("/api/profiles/") and path.endswith("/activate"):
            name = path.strip("/").split("/")[2]
            try:
                self._send_json(200, service.activate_profile(name))
            except Exception as exc:  # noqa: BLE001 - report to client
                self._send_json(400, {"error": str(exc)})
            return

        if path == "/api/profiles":
            name = str(body.get("name", "")).strip()
            if not name:
                self._send_json(400, {"error": "name is required"})
                return
            try:
                result = service.create_profile(
                    name, source=body.get("source"), force=bool(body.get("force"))
                )
                self._send_json(201, result)
            except Exception as exc:  # noqa: BLE001
                self._send_json(400, {"error": str(exc)})
            return

        if path == "/api/models/set":
            try:
                result = service.set_model(
                    str(body.get("tier", "")),
                    str(body.get("provider", "")),
                    str(body.get("model", "")),
                    context_limit=body.get("context_limit"),
                    endpoint=body.get("endpoint"),
                    api_key_env=body.get("api_key_env"),
                )
                self._send_json(200, result)
            except Exception as exc:  # noqa: BLE001
                self._send_json(400, {"error": str(exc)})
            return

        if path == "/api/shutdown":
            self._send_json(200, {"ok": True})
            self.sifter_server.shutdown_requested.set()
            threading.Thread(target=self.sifter_server.shutdown, daemon=True).start()
            return

        self._send_json(404, {"error": "not found"})


def create_server(
    service: ComputeSifterService,
    *,
    host: str = "127.0.0.1",
    port: int = 0,
    token: str,
) -> ComputeSifterServer:
    server = ComputeSifterServer((host, port), service, token)
    return server


def server_url(server: ComputeSifterServer) -> str:
    address = server.server_address
    host = str(address[0])
    port = int(address[1])
    return f"http://{host}:{port}"
