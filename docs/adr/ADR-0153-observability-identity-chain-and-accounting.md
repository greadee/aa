# ADR-0153 — Observability identity chain and accounting

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [issue-impl-sep28 Issue 10](../issues/imp-10-observability-hardening-e2e.md) (A9) |
| **Scope** | update phase · issue-impl-sep28 (Stage 5) |

## Context

The refactor introduced new execution seams (context, retrieval, allocation,
plan, scheduler, runtime, sandbox, tool, learning) with no unified observability,
and attribution did not span `organization → project → subtask → crew → worker →
{role → team, model}`. There was no end-to-end scenario coverage for the finished
pipeline.

## Decision

- **Structured seam events.** `kernel/observability` defines named events for
  each seam (`task.received`, `context.needs`, `retrieval`, `context.assembly`,
  `allocation.role`, `allocation.capability`, `allocation.parallelism`,
  `plan.execution`, `scheduler.dispatch`, `runtime.start`/`stop`, `sandbox`,
  `tool.call`, `agent.result`, `evaluation.learning`, `failure.retry`).
- **Identity chain.** `Identity{Organization, Project, Subtask, Crew, Worker,
  Role, Team, Model}` with a canonical `Chain()`; every `Event` carries a
  `CorrelationID` (work package/attempt).
- **Accounting.** `Summarize` reports totals (events, failures, retries, tokens,
  cost, compute-ms); `Accounting` aggregates per identity chain.
- **Deterministic and model-free.** Aggregation depends only on the recorded
  events; a `Recorder` seam keeps the kernel unaware of the sink (obsv/memory).
- **End-to-end scenarios.** Regression/integration tests cover simple,
  escalated, decomposable/parallel, and tool-using tasks across the built
  pipeline.

## Rationale

- One identity chain makes every event attributable end to end and reconciles
  work package ↔ worker ↔ role/team/model.
- Correlation IDs link events across seams without a global ordering dependency.
- Cost/token/compute accounting per chain makes resource use visible and
  auditable.

## Alternatives and consequences

- **Per-module ad-hoc logging:** rejected — no unified attribution or accounting.
- **Fold identity into `obsv`:** rejected — `obsv` is product-neutral; the chain
  is aa-specific, so it lives in the kernel with an `obsv`-backed recorder.
- Consequence: the `Recorder` seam is wired to `obsv`/memory at composition.

## Related

- [observability module doc](../modules/kernel/observability.md)
- [ADR-0012](./ADR-0012-add-a-new-trace-object-rather-than-extend-telemetry.md)
- [ADR index](./README.md)
