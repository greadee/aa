from __future__ import annotations

import json
import logging
import sys
import uuid
from datetime import UTC, datetime
from typing import Any, TextIO

_LOGGER_NAME = "aa_sifter"


def configure_logging(debug: bool = False, *, json_logs: bool = False) -> logging.Logger:
    logger = logging.getLogger(_LOGGER_NAME)
    logger.handlers.clear()
    handler = logging.StreamHandler(sys.stderr)
    if json_logs:
        handler.setFormatter(_JsonFormatter())
    else:
        handler.setFormatter(logging.Formatter("%(message)s"))
    logger.addHandler(handler)
    logger.setLevel(logging.DEBUG if debug else logging.INFO)
    logger.propagate = False
    return logger


class _JsonFormatter(logging.Formatter):
    def format(self, record: logging.LogRecord) -> str:
        payload: dict[str, Any] = {
            "ts": datetime.now(UTC).isoformat(),
            "level": record.levelname,
            "component": getattr(record, "component", "-"),
            "message": record.getMessage(),
        }
        fields = getattr(record, "fields", None)
        if fields:
            payload.update(fields)
        return json.dumps(payload, default=str)


class Trace:
    """Structured, auditable event log for one task execution."""

    def __init__(
        self,
        trace_id: str | None = None,
        *,
        debug: bool = False,
        sink: TextIO | None = None,
        logger: logging.Logger | None = None,
        history: Any | None = None,
        show_trace: bool | None = None,
    ):
        self.trace_id = trace_id or uuid.uuid4().hex[:16]
        self.debug = debug
        self.sink = sink if sink is not None else sys.stderr
        self.show_trace = debug if show_trace is None else show_trace
        self.logger = logger or logging.getLogger(_LOGGER_NAME)
        self.history = history
        self.events: list[dict[str, Any]] = []

    def child(self, prefix: str) -> Trace:
        child = Trace(
            self.trace_id,
            debug=self.debug,
            sink=self.sink,
            logger=self.logger,
            history=self.history,
            show_trace=self.show_trace,
        )
        child._prefix = prefix  # type: ignore[attr-defined]
        return child

    def log(
        self, component: str, message: str, *, level: str = "info", **fields: Any
    ) -> dict[str, Any]:
        event = {
            "ts": datetime.now(UTC).isoformat(),
            "trace_id": self.trace_id,
            "component": component,
            "message": message,
            "fields": fields,
        }
        self.events.append(event)
        log_level = getattr(logging, level.upper(), logging.INFO)
        self.logger.log(
            log_level,
            f"[{component}] {message}",
            extra={"component": component, "fields": fields},
        )
        if self.show_trace and self.sink is not None:
            rendered = " ".join(f"{k}={_render(v)}" for k, v in fields.items())
            suffix = f" {rendered}" if rendered else ""
            print(f"[{component}] {message}{suffix}", file=self.sink)
        if self.history is not None:
            self.history.record_event(self.trace_id, component, message, fields)
        return event


def _render(value: Any) -> str:
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, float):
        return f"{value:.3g}"
    if isinstance(value, (list, tuple, dict)):
        return json.dumps(value, default=str)
    return str(value)
