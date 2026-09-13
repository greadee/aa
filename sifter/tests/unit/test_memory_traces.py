"""Memory trace seam: each run emits a lifecycle-gated candidate."""

from __future__ import annotations

from aa_sifter.app import ComputeSifter
from aa_sifter.contracts import contracts_available, validate
from aa_sifter.memory import NopMemorySink
from tests.fixtures.providers import FakeProvider


class Collector:
    def __init__(self) -> None:
        self.records: list[dict] = []

    def emit(self, record: dict) -> None:
        self.records.append(record)


async def test_run_emits_a_memory_candidate(config):
    cfg = config.model_copy(update={"cloud_allowed": True})
    collector = Collector()
    sifter = ComputeSifter(cfg, provider=FakeProvider(), memory_sink=collector)

    result = await sifter.run("Write unit tests for a small utility function.")

    assert result.status == "completed"
    assert len(collector.records) == 1
    record = collector.records[0]
    assert record["kind"] == "memory_record"
    assert record["lifecycle"] == "CANDIDATE"
    assert record["content"]["summary"]
    assert record["provenance"]["traceId"] == result.trace_id
    if contracts_available():
        validate(record)


async def test_default_sink_is_a_noop(config):
    sifter = ComputeSifter(config.model_copy(update={"cloud_allowed": True}))
    assert isinstance(sifter.memory_sink, NopMemorySink)

    result = await sifter.run("Add logging to the parser.")

    assert result.status in {"completed", "failed"}


def test_sink_is_a_protocol_runtime_checkable(config):
    from aa_sifter.memory import MemorySink

    assert isinstance(NopMemorySink(), MemorySink)


async def test_sink_failure_does_not_break_a_run(config):
    class Exploding:
        def emit(self, record: dict) -> None:
            raise RuntimeError("sink down")

    sifter = ComputeSifter(
        config.model_copy(update={"cloud_allowed": True}),
        provider=FakeProvider(),
        memory_sink=Exploding(),
    )

    result = await sifter.run("Write unit tests for a small utility function.")
    assert result.status == "completed"
