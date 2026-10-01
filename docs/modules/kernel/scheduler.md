# kernel/scheduler

> Project history for the `kernel/scheduler` submodule of [aa-kernel](./README.md).

**Responsibility** — Runs aa-kernel's deterministic supervised control cycle
(ready → select → contract → context → lease → run → intake → gates → accept →
telemetry) and owns ordering, dispatch readiness, concurrency policy, the
assignment state machine, and expiring leases, recording through a `Sink`.

**Source** — [`kernel/scheduler/`](../../../kernel/scheduler/)

**Parent module** — [aa-kernel](./README.md) · [docs/modules](../README.md)

## Dispatch, concurrency, and cancellation

- `Dispatch(ctx)` runs the next ready work package (supervised, one step).
- `Run(ctx)` drains the graph: it dispatches the ready set in sorted order,
  bounded by `Config.MaxConcurrency` (1 = serial), and retries failed work
  packages up to `Config.MaxAttempts`. It returns a deterministic `RunReport`
  (`Dispatched`, `Accepted`, `Failed`, `Pending`, per-package `Attempts`).
- Concurrency is bounded and deterministic: the ready set is split into batches
  of at most `MaxConcurrency`; results are aggregated by count, not timing.
- Cancelling `ctx` stops dispatching and propagates to in-flight runtime calls
  (an assignment transitions to `canceled`).
- A failing work package does not abort independent work (partial failure);
  attempts are unique per package so intake never sees a reused result id.
- The scheduler owns scheduling only: it consumes capabilities and an execution
  contract and never re-decides allocation, which stays in `kernel/allocator`.

Decision record: [ADR-0149](../../adr/ADR-0149-scheduler-concurrency-retries-and-aggregation.md).
