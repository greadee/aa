# ISS-IMP-7 — kernel/allocator + registry: role, model & compute allocation

**Type:** feature
**Status:** complete
**Branch:** `dev`
**Sprint PR:** [#21](https://github.com/greadee/aa/pull/21)
**GitHub issue:** [#29](https://github.com/greadee/aa/issues/29)
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

- [x] Role, model, and compute allocation implemented and independently testable.
- [x] Model registry populated; model identity selection and escalation represented.
- [x] Parallelism justified and bounded; no auto-spawn.
- [x] Routing/budget policy lives in the allocator/contract; inference is provider/execution only.
- [x] An `ExecutionPlan` is produced (validated contracts v2); the scheduler consumes its concurrency/budget fields (`Config.MaxConcurrency`, contract budget).
- [x] `go build`, `go vet`, `go test`, `gofmt`, `tools/archtest`, `runtime/inference` checks, and docs link check pass.
- [x] tests added or updated
- [x] documentation updated where required

## Solution

- **Role:** `registry/roles.RolesFor` maps capabilities to roles deterministically;
  `Allocate` assigns the primary role per unit.
- **Model:** `registry/models` is populated (`NewDefaultRegistry` from
  `DefaultCatalog`, mirrored from the inference catalog); `model_allocator.Allocate`
  selects by locality/capability/cost and returns a reasoning escalation target.
- **Compute:** `compute_allocator.Plan` requires an explicit justification for
  parallelism and bounds workers by independent units and concurrency.
- **Execution plan:** `allocator.Allocate` combines the three into a contracts v2
  `ExecutionPlan` (validated) for the scheduler.

- **Routing/budget (A8):** `kernel/allocator/routing.Decide` owns the
  model-locality/tier policy and the budget gate (fail-closed on disabled cloud
  or exceeded budget). `runtime/inference` keeps provider/execution plus
  verification; its Python routing package is the execution surface, and the
  follow-up to pass the allocator's decision into it is recorded in the sprint
  Future register.

Decision records: [ADR-0150](../adr/ADR-0150-role-model-compute-allocation-separation.md),
[ADR-0151](../adr/ADR-0151-routing-and-budget-policy-in-the-allocator.md). This
closes Issue 7.

## Dependencies

Follows ISS-IMP-3 (enriched task) and ISS-IMP-4. Unblocks ISS-IMP-8 (scheduling) and ISS-IMP-9 (learning).
