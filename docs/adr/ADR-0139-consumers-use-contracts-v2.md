# ADR-0139 — Consumers use contracts v2; registry and allocator expose the spec/plan contracts

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) (D-7, D-17) |
| **Scope** | update phase · architecture-refactor-1 |

## Context

`contracts` gained a v2 generation (ADR-0137) and `trade` was retired
(ADR-0138). Consumers still imported `contracts/go/v1`, and the new spec/plan
objects had no consumers.

## Decision

Switch every Go consumer from `contracts/go/v1` to **`contracts/go/v2`**. Wire
the new contracts where they belong:

- `registry/roles`, `registry/models`, `registry/teams`, `registry/routines`
  re-export the v2 specifications (`RoleSpec`, `ModelSpec`, `TeamSpec`,
  `RoutineSpec`).
- `kernel/allocator` re-exports `WorkPlan` and `ExecutionPlan` (the Planner's and
  allocators' outputs).

No new behavior is added; this is aliasing and import migration.

## Rationale

Consumers on the current generation keep the single-source-of-truth rule, and the
registry/allocator name the contracts they own. `v1` remains available for
history and any external consumer that has not migrated.

## Alternatives and consequences

- **Keep consumers on v1:** rejected — it would leave the new generation unused
  and `trade` present in the active path.
- **Add adapters v1↔v2:** not needed — no consumer used the removed `trade`
  field, so the migration is a clean switch.
- Consequence: TypeScript and Python bindings are bumped to v2 (the Python
  package exports `v2`). `v1` directories remain.

## Related

- [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) D-7, D-17
- [ADR-0137 — Contracts v2 generation](./ADR-0137-contracts-v2-generation.md)
- [ADR index](./README.md)
