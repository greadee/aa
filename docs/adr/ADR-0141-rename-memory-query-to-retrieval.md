# ADR-0141 — Rename `memory/query` to `memory/retrieval`

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) (D-8) |
| **Scope** | update phase · architecture-refactor-1 |

## Context

The context/retrieval boundary is explicit in the target architecture:
`aa-memory/retrieval` finds candidate information; `aa-kernel/context` decides
what is needed and assembles the final context. The package that provides
deterministic reads over the memory projection was named `memory/query`.

## Decision

Rename `memory/query` to **`memory/retrieval`** (package `retrieval`). The read
API, histories, and trace summaries are unchanged; only the name changes.
Consumers (`memory` facade, `visualizer/browse`) import `memory/retrieval`.

## Rationale

The name states the role in the boundary: retrieval returns candidate
information; it does not assemble a context window. It aligns the package with
the context/retrieval split the architecture names.

## Alternatives and consequences

- **Keep `query`:** rejected — it obscures the retrieval role.
- **Split retrieval into a new package beside query:** rejected — the package
  already is retrieval; a parallel package would duplicate it.
- Consequence: the memory facade and the visualizer browse seam import
  `memory/retrieval`; docs and architecture references updated.

## Related

- [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) D-8
- Modules: [docs/modules/memory](../modules/memory/README.md)
- [ADR index](./README.md)
