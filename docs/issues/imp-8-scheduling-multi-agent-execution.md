# ISS-IMP-8 — kernel/scheduler + runtime: scheduling & multi-agent execution

**Type:** feature
**Status:** planned
**Branch:** `dev`
**Sprint PR:** [#21](https://github.com/greadee/aa/pull/21)
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

- [ ] Scheduler consumes the execution plan, dispatches, and aggregates deterministically.
- [ ] Concurrency, cancellation, retries, and partial failures handled; worker lifecycle managed.
- [ ] Policy boundaries respected (allocation vs scheduling vs runtime).
- [ ] `intake` moved to `runtime`; boundaries updated.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, `tools/archtest`, and docs link check pass.
- [ ] tests added or updated
- [ ] documentation updated where required

## Dependencies

ISS-IMP-5 (sandbox) and ISS-IMP-7 (allocation). Unblocks ISS-IMP-9/10.
