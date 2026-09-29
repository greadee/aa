# ISS-IMP-7 — kernel/allocator + registry: role, model & compute allocation

**Type:** feature
**Status:** planned
**Branch:** `dev`
**Sprint PR:** [#21](https://github.com/greadee/aa/pull/21)
**Parent umbrella:** [ISS-IMP](post-refactor-implementation.md)
**Sprint plan:** [../updates/issue-impl-sep28/plan.md](../updates/issue-impl-sep28/plan.md) (Stage 4)
**Resolves:** A6 (model/compute allocation, `registry/models` population); A8 (routing/budgets extraction)

## Goal

Expand the refactored `kernel/allocator` around three independent allocation decisions — **role**
(expertise), **model/reasoning tier** (capability, including escalation), and **parallelism**
(compute) — producing an execution plan consumed by the scheduler. The allocator does not spawn
workers.

## Problem

Only role allocation carries behavior; `model_allocator` and `compute_allocator` are reserved; the
model registry is unpopulated; and `runtime/inference` still owns routing and budget policy (A8). The
three dimensions must remain independently testable even when combined into one execution plan.

## Requirements

- **Role allocation:** select the appropriate expertise from `registry/roles` and capabilities.
- **Model allocation:** choose the model/reasoning tier from the model registry; represent escalation;
  own model identity selection (never worker count/budgets).
- **Compute allocation:** worker count, parallelism, reasoning/execution budget, replication,
  placement; own resource quantity/topology (never model identity). Idle agents are not a reason to
  spawn workers; parallelism must be justified (latency/quality/coverage/reliability).
- Populate `registry/models` from the inference model catalog.
- Move routing/budget policy out of `runtime/inference` into `allocator`/`contract`; inference keeps
  provider/execution only.
- Produce an `ExecutionPlan` (contracts v2) for the scheduler; cost/compute awareness.

## Acceptance Criteria

- [ ] Role, model, and compute allocation implemented and independently testable.
- [ ] Model registry populated; model identity selection and escalation represented.
- [ ] Parallelism justified and bounded by cost/compute awareness; no auto-spawn.
- [ ] Routing/budget policy lives in the allocator/contract; inference is provider/execution only.
- [ ] An `ExecutionPlan` is produced and consumed by the scheduler (ISS-IMP-8).
- [ ] `go build`, `go vet`, `go test`, `gofmt`, `tools/archtest`, `runtime/inference` checks, and docs link check pass.
- [ ] tests added or updated
- [ ] documentation updated where required

## Dependencies

Follows ISS-IMP-3 (enriched task) and ISS-IMP-4. Unblocks ISS-IMP-8 (scheduling) and ISS-IMP-9 (learning).
