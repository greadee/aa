# kernel/observability

> Project history for the `kernel/observability` submodule of [aa-kernel](./README.md).

**Responsibility** — Structured, correlated events across the execution seams and
the end-to-end identity chain that attributes them, with deterministic metrics
and cost/token/compute accounting.

**Source** — [`kernel/observability/`](../../../kernel/observability/)

**Parent module** — [aa-kernel](./README.md) · [docs/modules](../README.md)

## Events and identity

- **Seam events** — `task.received`, `context.needs`, `retrieval`,
  `context.assembly`, `allocation.role`, `allocation.capability`,
  `allocation.parallelism`, `plan.execution`, `scheduler.dispatch`,
  `runtime.start`/`runtime.stop`, `sandbox`, `tool.call`, `agent.result`,
  `evaluation.learning`, `failure.retry`.
- **Identity chain** — `Identity{Organization, Project, Subtask, Crew, Worker,
  Role, Team, Model}`; `Chain()` is the canonical `org/project/subtask/crew/worker/role/team/model`.
- **Correlation** — every `Event` carries a `CorrelationID` (work package/attempt).
- **Accounting** — `Summarize` gives totals (events, failures, retries, tokens,
  cost, compute-ms); `Accounting` aggregates per identity chain, sorted.
- `Recorder` is the seam to `obsv`/memory; `MemRecorder` is the deterministic
  reference. Derivation is model-free.

Decision record: [ADR-0153](../../adr/ADR-0153-observability-identity-chain-and-accounting.md).
