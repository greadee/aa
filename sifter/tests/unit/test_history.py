from __future__ import annotations

from aa_sifter.history.sqlite import SqliteHistoryStore


def test_run_round_trip(tmp_path):
    store = SqliteHistoryStore(tmp_path / "h.db")
    store.record_run(
        {
            "trace_id": "t1",
            "prompt": "hello",
            "decision_level": "routine",
            "route": "local",
            "final_success": True,
            "cloud_calls": 0,
            "cloud_cost": 0.0,
        }
    )
    runs = store.recent_runs()
    assert len(runs) == 1
    assert runs[0]["prompt"] == "hello"
    stats = store.stats()
    assert stats["runs"] == 1
    assert stats["success_rate"] == 1.0
    store.close()


def test_approval_audit(tmp_path):
    store = SqliteHistoryStore(tmp_path / "h.db")
    store.record_approval(
        "t1",
        {"category": "major", "issue": "x", "options": [], "recommended_action": "y"},
        {"action": "allow_expert_analysis", "authorized": True, "constraints": ["c"]},
    )
    rows = store._conn.execute("SELECT * FROM approvals").fetchall()
    assert len(rows) == 1
    assert rows[0]["authorized"] == 1
    store.close()


def test_standing_rules_persist(tmp_path):
    store = SqliteHistoryStore(tmp_path / "h.db")
    rule_id = store.add_standing_rule(
        "SQLite is approved", forbid_terms=["sqlite"], grants=["x"], tags=["persistence"]
    )
    rules = store.list_standing_rules()
    assert rules[0]["id"] == rule_id
    assert rules[0]["forbid_terms"] == ["sqlite"]
    assert store.deactivate_standing_rule(rule_id) is True
    assert store.list_standing_rules() == []
    store.close()


def test_migrations_idempotent(tmp_path):
    path = tmp_path / "h.db"
    store = SqliteHistoryStore(path)
    store.close()
    store2 = SqliteHistoryStore(path)
    assert store2._conn.execute("PRAGMA user_version").fetchone()[0] >= 1
    store2.close()


def test_pending_decision_persists_across_restart(tmp_path):
    path = tmp_path / "h.db"
    first = SqliteHistoryStore(path)
    first.record_approval(
        "trace-1",
        {"category": "major", "issue": "persistence", "options": [], "recommended_action": ""},
        {"action": "allow_expert_analysis", "authorized": True, "constraints": ["keep sqlite"]},
    )
    first.close()

    second = SqliteHistoryStore(path)
    row = second._conn.execute(
        "SELECT * FROM approvals WHERE trace_id = ?", ("trace-1",)
    ).fetchone()
    assert row is not None
    assert row["action"] == "allow_expert_analysis"
    assert row["authorized"] == 1
    second.close()


def test_rejected_decision_persists_across_restart(tmp_path):
    path = tmp_path / "h.db"
    first = SqliteHistoryStore(path)
    first.record_approval(
        "trace-2",
        {"category": "critical", "issue": "drop table", "options": [], "recommended_action": ""},
        {"action": "stop_task", "authorized": False, "constraints": []},
    )
    first.close()

    second = SqliteHistoryStore(path)
    row = second._conn.execute(
        "SELECT * FROM approvals WHERE trace_id = ?", ("trace-2",)
    ).fetchone()
    assert row["action"] == "stop_task"
    assert row["authorized"] == 0
    second.close()


def test_standing_rule_persists_across_restart(tmp_path):
    path = tmp_path / "h.db"
    first = SqliteHistoryStore(path)
    rule_id = first.add_standing_rule("Never change authentication architecture.")
    first.close()

    second = SqliteHistoryStore(path)
    rules = second.list_standing_rules()
    assert any(rule["id"] == rule_id for rule in rules)
    second.close()
