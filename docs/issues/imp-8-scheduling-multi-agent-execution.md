# ISS-IMP-8 — kernel/scheduler + runtime: scheduling & multi-agent execution

**Type:** feature
**Status:** complete
**Branch:** `dev`
**Sprint PR:** [#21](https://github.com/greadee/aa/pull/21)
**GitHub issue:** [#30](https://github.com/greadee/aa/issues/30)
**Parent umbrella:** [ISS-IMP](post-refactor-implementation.md)
**Sprint plan:** [../updates/issue-impl-sep28/plan.md](../updates/issue-impl-sep28/plan.md) (Stage 3)
**Resolves:** A5 (move `intake` to `runtime`)

## Goal

Turn allocation plans into controlled execution through the scheduler and runtime, using idle
capacity productively when the execution plan says additional workers are useful.

## Problem

`kernel/scheduler` (from orchestrator + control) runs a supervised cycle, but multi-agent execution,
concurrency limits, aggregation, cancellation, retries, partial-failure handling, and worker lifecycle
are not hardened. `kernel/intake` is a transitional package intended to move to `runtime` (A5).

## Requirements

- Task DAG/dependencies, worker dispatch, concurrency limits, synchronization, aggregation.
- Cancellation, retries, partial failures, worker lifecycle, idle-resource utilization.
- Serial vs parallel task handling driven by the execution plan (ISS-IMP-7).
- The scheduler must **not** redo allocation policy; the runtime must **not** decide allocation policy.
- Move `kernel/intake` → `runtime` and update archtest/docs.

## Acceptance Criteria

- [x] Scheduler dispatches and aggregates deterministically (`RunReport`).
- [x] Concurrency, cancellation, retries, and partial failures handled; attempt ids are unique per package.
- [x] Policy boundaries respected (allocation vs scheduling vs runtime).
- [x] `intake` moved to `runtime`; boundaries updated.
- [x] `go build`, `go vet`, `go test`, `gofmt`, `tools/archtest`, and docs link check pass.
- [x] tests added or updated
- [x] documentation updated where required

## Solution

`kernel/scheduler` gained a concurrency-safe `dispatchOne` shared by `Dispatch`
(one supervised step) and `Run` (drain the graph). `Run` dispatches the ready
set in sorted order, bounded by `Config.MaxConcurrency`, retries failed packages
up to `Config.MaxAttempts` (returning them to `READY`, with unique attempt ids
so intake stays idempotent), propagates cancellation, tolerates partial failure,
and returns a deterministic `RunReport`. Shared state is mutated only under a
mutex; the runtime call runs outside the lock so batch dispatches are genuinely
parallel. The scheduler consumes capabilities/contracts and never re-decides
allocation.

Decision record: [ADR-0149](../adr/ADR-0149-scheduler-concurrency-retries-and-aggregation.md).

**A5 execution.** `kernel/intake` **moved** to `runtime/intake` (execution
mechanics: validating and deduplicating untrusted result envelopes), per
[ADR-0143](../adr/ADR-0143-decide-intake-and-contract-ownership.md). The
scheduler now imports `runtime/intake`; the kernel no longer owns intake.
Architecture §5.3/§5.6, the module docs, `runtime/doc.go`, and the kernel
transitional-boundaries table were updated. This closes Issue 8.

## Dependencies

ISS-IMP-5 (sandbox) and ISS-IMP-7 (allocation). Unblocks ISS-IMP-9/10.
