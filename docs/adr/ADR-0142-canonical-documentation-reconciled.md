# ADR-0142 — Canonical documentation reconciled to the refactored topology

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) |
| **Scope** | update phase · architecture-refactor-1 |

## Context

The refactor moved modules and packages across slices (registry, runtime, ui,
contracts v2, trade retirement, sifter→inference, query→retrieval). The canonical
documentation — architecture, terminology, module directives, indexes, and
adjacency lists — must reflect the finished topology so the ten-issue phase can
plan against it.

## Decision

Reconcile the canonical documentation to the refactored state:

- **Modules:** `contracts, registry, obsv, kernel, runtime, memory, sync, forge,
  toolbox, visualizer, ui` (plus the `github-os` process area).
- **Kernel:** `allocator` (planner, role/model/compute allocators) and
  `scheduler` own planning/allocation/scheduling; transitional packages are
  documented in [kernel transitional boundaries](../modules/kernel/transitional-boundaries.md).
- **Runtime:** `worker`, `lifecycle`, reserved `sandbox`, and the nested Python
  `inference` subproject.
- **Contracts v2** is the current generation; `trade` is retired; consumers use
  `v2`.
- Update the architecture README, terminology, module directives/READMEs,
  `docs/index/*`, `MANIFEST.json`, and the `aa-*` directives' cross-references.

Historical phase, issue, and backfilled ADR documents keep their original names.

## Rationale

Canonical docs are the input to the next phase; leaving stale names would
re-introduce drift. Historical records are preserved so the decision trail stays
intact.

## Alternatives and consequences

- **Rewrite history:** rejected — historical records must stay as delivered.
- Consequence: docs use the refactored names; the ten-issue handoff's pre-refactor
  names are translated by its Issue 1.

## Related

- [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md)
- [Terminology](../reference/terminology.md)
- [ADR index](./README.md)
