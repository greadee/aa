from __future__ import annotations

import json
import sqlite3
import threading
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

_MIGRATIONS: list[str] = [
    """
    CREATE TABLE IF NOT EXISTS runs (
        trace_id TEXT PRIMARY KEY,
        created_at TEXT NOT NULL,
        prompt TEXT NOT NULL,
        task_type TEXT,
        complexity REAL,
        decision_level TEXT,
        approval_requested INTEGER DEFAULT 0,
        user_decision TEXT,
        route TEXT,
        policy TEXT,
        models TEXT,
        input_tokens INTEGER DEFAULT 0,
        output_tokens INTEGER DEFAULT 0,
        cloud_calls INTEGER DEFAULT 0,
        cloud_cost REAL DEFAULT 0,
        retries INTEGER DEFAULT 0,
        escalations INTEGER DEFAULT 0,
        verification_success INTEGER,
        final_success INTEGER,
        duration_ms REAL DEFAULT 0,
        metadata TEXT
    );

    CREATE TABLE IF NOT EXISTS approvals (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        trace_id TEXT NOT NULL,
        created_at TEXT NOT NULL,
        category TEXT NOT NULL,
        issue TEXT NOT NULL,
        options TEXT,
        recommended_action TEXT,
        action TEXT NOT NULL,
        selected_options TEXT,
        constraints TEXT,
        authorized INTEGER NOT NULL
    );

    CREATE TABLE IF NOT EXISTS standing_rules (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        created_at TEXT NOT NULL,
        text TEXT NOT NULL,
        forbid_terms TEXT,
        grants TEXT,
        tags TEXT,
        active INTEGER NOT NULL DEFAULT 1
    );

    CREATE TABLE IF NOT EXISTS events (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        trace_id TEXT NOT NULL,
        created_at TEXT NOT NULL,
        component TEXT NOT NULL,
        message TEXT NOT NULL,
        fields TEXT
    );

    CREATE INDEX IF NOT EXISTS idx_events_trace ON events(trace_id);
    CREATE INDEX IF NOT EXISTS idx_approvals_trace ON approvals(trace_id);
    CREATE INDEX IF NOT EXISTS idx_runs_created ON runs(created_at);
    """,
]


def _now() -> str:
    return datetime.now(UTC).isoformat()


class SqliteHistoryStore:
    """SQLite-backed execution history, approval audit trail and standing rules."""

    def __init__(self, path: str | Path):
        self.path = Path(path).expanduser()
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self._lock = threading.RLock()
        self._conn = sqlite3.connect(str(self.path), check_same_thread=False)
        self._conn.row_factory = sqlite3.Row
        self._migrate()

    def _migrate(self) -> None:
        with self._lock:
            current = self._conn.execute("PRAGMA user_version").fetchone()[0]
            for version in range(current, len(_MIGRATIONS)):
                self._conn.executescript(_MIGRATIONS[version])
                self._conn.execute(f"PRAGMA user_version = {version + 1}")
            self._conn.execute("PRAGMA journal_mode=WAL")
            self._conn.commit()

    def record_run(self, run: dict[str, Any]) -> None:
        columns = [
            "trace_id",
            "created_at",
            "prompt",
            "task_type",
            "complexity",
            "decision_level",
            "approval_requested",
            "user_decision",
            "route",
            "policy",
            "models",
            "input_tokens",
            "output_tokens",
            "cloud_calls",
            "cloud_cost",
            "retries",
            "escalations",
            "verification_success",
            "final_success",
            "duration_ms",
            "metadata",
        ]
        payload = dict(run)
        payload.setdefault("created_at", _now())
        for key in ("models", "metadata"):
            if key in payload and not isinstance(payload[key], str):
                payload[key] = json.dumps(payload[key], default=str)
        for key in ("approval_requested", "verification_success", "final_success"):
            if key in payload and payload[key] is not None:
                payload[key] = 1 if payload[key] else 0
        values = [payload.get(column) for column in columns]
        placeholders = ", ".join("?" for _ in columns)
        with self._lock:
            self._conn.execute(
                f"INSERT OR REPLACE INTO runs ({', '.join(columns)}) VALUES ({placeholders})",
                values,
            )
            self._conn.commit()

    def record_approval(
        self, trace_id: str, request: dict[str, Any], decision: dict[str, Any]
    ) -> None:
        with self._lock:
            self._conn.execute(
                """
                INSERT INTO approvals
                    (trace_id, created_at, category, issue, options,
                     recommended_action, action, selected_options, constraints, authorized)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    trace_id,
                    _now(),
                    str(request.get("category", "")),
                    str(request.get("issue", "")),
                    json.dumps(request.get("options", []), default=str),
                    str(request.get("recommended_action", "")),
                    str(decision.get("action", "")),
                    json.dumps(decision.get("selected_options", []), default=str),
                    json.dumps(decision.get("constraints", []), default=str),
                    1 if decision.get("authorized") else 0,
                ),
            )
            self._conn.commit()

    def record_event(
        self, trace_id: str, component: str, message: str, fields: dict[str, Any]
    ) -> None:
        try:
            with self._lock:
                self._conn.execute(
                    "INSERT INTO events (trace_id, created_at, component, message, fields) VALUES (?, ?, ?, ?, ?)",
                    (trace_id, _now(), component, message, json.dumps(fields, default=str)),
                )
                self._conn.commit()
        except sqlite3.Error:
            pass

    def add_standing_rule(
        self,
        text: str,
        *,
        forbid_terms: list[str] | None = None,
        grants: list[str] | None = None,
        tags: list[str] | None = None,
    ) -> int:
        with self._lock:
            cursor = self._conn.execute(
                """
                INSERT INTO standing_rules (created_at, text, forbid_terms, grants, tags, active)
                VALUES (?, ?, ?, ?, ?, 1)
                """,
                (
                    _now(),
                    text,
                    json.dumps(forbid_terms or []),
                    json.dumps(grants or []),
                    json.dumps(tags or []),
                ),
            )
            self._conn.commit()
            return int(cursor.lastrowid or 0)

    def list_standing_rules(self, *, active_only: bool = True) -> list[dict[str, Any]]:
        query = "SELECT * FROM standing_rules"
        if active_only:
            query += " WHERE active = 1"
        query += " ORDER BY id"
        with self._lock:
            rows = self._conn.execute(query).fetchall()
        return [_rule_row(row) for row in rows]

    def deactivate_standing_rule(self, rule_id: int) -> bool:
        with self._lock:
            cursor = self._conn.execute(
                "UPDATE standing_rules SET active = 0 WHERE id = ?", (rule_id,)
            )
            self._conn.commit()
            return cursor.rowcount > 0

    def recent_runs(self, limit: int = 20) -> list[dict[str, Any]]:
        with self._lock:
            rows = self._conn.execute(
                "SELECT * FROM runs ORDER BY created_at DESC LIMIT ?", (limit,)
            ).fetchall()
        return [dict(row) for row in rows]

    def stats(self) -> dict[str, Any]:
        with self._lock:
            row = self._conn.execute(
                """
                SELECT
                    COUNT(*) AS runs,
                    COALESCE(SUM(cloud_calls), 0) AS cloud_calls,
                    COALESCE(SUM(cloud_cost), 0) AS cloud_cost,
                    COALESCE(SUM(final_success), 0) AS successes,
                    COALESCE(SUM(approval_requested), 0) AS approval_requests,
                    COALESCE(SUM(escalations), 0) AS escalations
                FROM runs
                """
            ).fetchone()
        data = dict(row)
        runs = data.get("runs") or 0
        data["success_rate"] = (data["successes"] / runs) if runs else 0.0
        return data

    def close(self) -> None:
        with self._lock:
            self._conn.close()


def _rule_row(row: sqlite3.Row) -> dict[str, Any]:
    data = dict(row)
    for key in ("forbid_terms", "grants", "tags"):
        try:
            data[key] = json.loads(data.get(key) or "[]")
        except (json.JSONDecodeError, TypeError):
            data[key] = []
    data["active"] = bool(data.get("active"))
    return data
