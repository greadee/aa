from __future__ import annotations

from aa_sifter.metrics.trace import Trace
from aa_sifter.metrics.usage import UsageMetrics, UsageTracker
from aa_sifter.models.provider import GenerationResult, Tier


def _result(tier: Tier, input_tokens: int = 100, output_tokens: int = 50) -> GenerationResult:
    return GenerationResult(
        text="x",
        model="m",
        tier=tier,
        input_tokens=input_tokens,
        output_tokens=output_tokens,
        duration_ms=10.0,
    )


def test_usage_metrics_record_generation():
    metrics = UsageMetrics()
    metrics.record_generation(_result(Tier.LOCAL))
    metrics.record_generation(_result(Tier.EXPERT))
    assert metrics.local_calls == 1
    assert metrics.cloud_calls == 1
    assert metrics.total_tokens == 300


def test_usage_tracker_computes_cloud_cost(config):
    tracker = UsageTracker()
    tracker.configure(local=config.local_model_config(), cloud=config.cloud_model_config())
    tracker.record(_result(Tier.EXPERT, input_tokens=1_000_000, output_tokens=1_000_000))
    assert tracker.metrics.cloud_cost > 0
    summary = tracker.summary()
    assert summary["total_tokens"] == 2_000_000


def test_trace_logs_and_serializes_history(store):
    trace = Trace(history=store, show_trace=False)
    trace.log("router", "route=local", confidence=0.9)
    assert trace.events[0]["component"] == "router"
    events = store._conn.execute("SELECT COUNT(*) FROM events").fetchone()[0]
    assert events == 1


def test_trace_child_is_loggable():
    trace = Trace(show_trace=False)
    child = trace.child("sub")
    child.log("x", "y")
    assert child.trace_id == trace.trace_id


def test_trace_writes_to_sink_when_enabled():
    import io

    sink = io.StringIO()
    trace = Trace(show_trace=True, sink=sink)
    trace.log("router", "route=local", confidence=0.9, enabled=True, names=["a"])
    output = sink.getvalue()
    assert "[router] route=local" in output
    assert "confidence=" in output


def test_configure_logging_and_json_formatter():
    import json
    import logging

    from aa_sifter.metrics.trace import _JsonFormatter, configure_logging

    logger = configure_logging(debug=True, json_logs=True)
    assert logger.name == "aa_sifter"
    record = logging.LogRecord("aa_sifter", logging.INFO, __file__, 1, "hello", None, None)
    record.component = "x"
    record.fields = {"a": 1}
    payload = json.loads(_JsonFormatter().format(record))
    assert payload["message"] == "hello"
    assert payload["a"] == 1
