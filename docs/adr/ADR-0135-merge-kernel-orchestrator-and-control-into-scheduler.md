# ADR-0135 — Merge kernel orchestrator and control into scheduler

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) (D-13) |
| **Scope** | update phase · architecture-refactor-1 |

## Context

The kernel's supervised control cycle lived in `kernel/orchestrator` and the
assignment state machine and leases in `kernel/control`. Both are scheduling
concerns: ordering, dispatch readiness, concurrency policy, and the lifecycle of
an assignment under a lease.

## Decision

Merge both packages into a single **`kernel/scheduler`** package. The
orchestrator type becomes `Scheduler` (constructor `New`); the assignment
constructor `control.New` becomes `scheduler.NewAssignment`. The `api` service
consumes `scheduler.Scheduler`.

## Rationale

One owner for scheduling. The control loop and the assignment/lease state
machine are two faces of the same concern and change together; keeping them in
one package removes an artificial boundary.

## Alternatives and consequences

- **Keep `orchestrator` + `control`:** rejected — they are one scheduling
  concern split across two packages.
- **Keep the type named `Orchestrator`:** rejected — `scheduler.Orchestrator`
  obscures the ownership; `scheduler.Scheduler` is clearer.
- Consequence: `kernel/api` and its test were rewired; `kernel/orchestrator` and
  `kernel/control` are deleted.

## Related

- [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) D-13
- Module documentation: [docs/modules/kernel](../modules/kernel/README.md)
- [ADR index](./README.md)
