# ADR-0149 — Scheduler concurrency, retries, and aggregation

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [issue-impl-sep28 Issue 8](../issues/imp-8-scheduling-multi-agent-execution.md) |
| **Scope** | update phase · issue-impl-sep28 (Stage 3) |

## Context

`kernel/scheduler` ran a supervised one-step cycle (`Dispatch`) but had no
concurrency limit, retries, cancellation handling, partial-failure policy, or a
deterministic aggregate over a graph. Multi-agent execution and idle-resource
use were not supported.

## Decision

- **Bounded, deterministic concurrency.** `Config.MaxConcurrency` bounds how many
  ready work packages run at once (1 = serial). `Run(ctx)` dispatches the ready
  set in sorted order in batches of at most `MaxConcurrency`.
- **Deterministic aggregate.** `Run` returns a `RunReport` of counts
  (`Dispatched`, `Accepted`, `Failed`, `Pending`, per-package `Attempts`) that
  does not depend on completion timing or goroutine scheduling.
- **Retries.** `Config.MaxAttempts` (default 1) retries a failed work package by
  returning it to `READY`; attempts are unique per package so intake idempotency
  is not violated by a different result under the same id.
- **Cancellation.** `Run` checks `ctx` between batches; a cancelled in-flight
  runtime call transitions the assignment to `canceled` and aborts the run.
- **Partial failure.** One failed work package does not abort independent work.
- **Concurrency safety.** Shared scheduler state (graph, assignments, results,
  telemetry, sequence) is mutated only under a mutex; the runtime call runs
  outside the lock so dispatches are actually parallel.
- **Policy boundaries.** The scheduler never re-decides allocation; it consumes
  capabilities/contracts and delegates planning to `kernel/allocator`.

## Rationale

- A deterministic aggregate keeps runs testable and replayable even when work
  executes in parallel.
- Bounding concurrency by explicit config (later supplied by the execution plan)
  keeps resource use predictable and the scheduler policy-free.
- Unique attempt ids preserve intake's content-hash idempotency across retries.

## Alternatives and consequences

- **Unbounded goroutines per ready package:** rejected — unbounded resource use
  and non-deterministic aggregation.
- **Retry with the same result id:** rejected — intake rejects a reused id with
  different content (`ErrConflict`).
- **Abort the run on first failure:** rejected — independent work should finish
  (partial failure).
- Consequence: `Run` and `Dispatch` share a concurrency-safe `dispatchOne`; the
  A5 `intake` → `runtime` move is the next commit and updates boundaries.

## Related

- [kernel/scheduler module doc](../modules/kernel/scheduler.md)
- [ADR-0033](./ADR-0033-assignment-transitions-are-an-explicit-state-machine-with-leases.md)
- [ADR index](./README.md)
