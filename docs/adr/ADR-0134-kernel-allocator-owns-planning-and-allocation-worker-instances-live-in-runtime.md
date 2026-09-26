# ADR-0134 — Kernel allocator owns planning and allocation; worker instances live in runtime

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) (D-10, D-11, D-17) |
| **Scope** | update phase · architecture-refactor-1 |

## Context

`kernel/registry` mixed three concerns: durable role definitions, worker
instances, and deterministic worker selection. `kernel/plan` held work planning.
The target architecture separates planning and allocation inside the kernel, and
keeps runtime instances out of the durable definition store.

## Decision

- `kernel/plan` → **`kernel/allocator/planner`** (the Planner).
- `kernel/registry` is **retired**:
  - role definitions → `registry/roles` (already relocated);
  - deterministic worker selection → **`kernel/allocator/role_allocator`**;
  - worker instances (`Worker`, `WorkerID`, `Trade`) → **`runtime/worker`**.
- `kernel/allocator/model_allocator` and `kernel/allocator/compute_allocator` are
  **reserved** boundaries (no behavior).

The allocator answers *what work* (planner), *whose expertise* (role allocator),
and — later — *which model* (model allocator) and *how much compute* (compute
allocator). It produces plans; it does not execute them.

## Rationale

One owner per concern: definitions in `registry`, instances in `runtime`,
allocation in `kernel/allocator`. It removes the mixed `kernel/registry` and
gives role, model, and compute allocation distinct, independently testable
boundaries.

## Alternatives and consequences

- **Keep `kernel/registry`:** rejected — it conflates definitions, instances, and
  selection.
- **Split selection into three allocators now:** rejected — model and compute
  allocation are new design, not a structural move; only role allocation carries
  current behavior.
- Consequence: `kernel/allocator/role_allocator` imports `runtime/worker` and
  `registry/roles`; `runtime/worker` imports `registry/roles`. Layering and
  `tools/archtest` were updated.

## Related

- [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) D-10, D-11, D-17
- Modules: [docs/modules/kernel](../modules/kernel/README.md), [docs/modules/runtime](../modules/runtime/README.md)
- [ADR index](./README.md)
