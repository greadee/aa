# ADR-0136 — Kernel transitional package placement and intended ownership

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) (D-15) |
| **Scope** | update phase · architecture-refactor-1 |

## Context

The architecture refactor moved planning, allocation, and scheduling into
`kernel/allocator` and `kernel/scheduler`, and moved execution mechanics and
worker instances to `aa-runtime`. Several kernel packages
(`context`, `contract`, `gate`, `telemetry`, `joblearn`, `api`, `intake`) remain
where they were. Leaving their placement undocumented risks mistaking a
transitional location for final ownership.

## Decision

Keep those packages in the kernel **for now**, and record each one's intended
home and constraints in
[docs/modules/kernel/transitional-boundaries.md](../modules/kernel/transitional-boundaries.md).
`intake` is intended to move to `runtime`; `contract` may move under the
allocator or become a shared kernel package; the rest are intended to stay, with
internal restructuring reserved. No new dependency may entrench a transitional
placement, and the narrowest seam is preferred so a package can move without
touching callers.

## Rationale

Documenting intent without forcing a move keeps the refactor behavior-preserving
while preventing accidental coupling that would make a later move expensive. It
also makes the remaining work visible rather than implicit.

## Alternatives and consequences

- **Move every package now:** rejected — some intended homes need contracts or
  behavior that does not exist yet, so moves would be speculative.
- **Leave placement undocumented:** rejected — it invites entrenchment.
- Consequence: the module docs and architecture carry a pointer to the intended
  ownership; moves become their own slices with ADR updates.

## Related

- [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) D-15
- [Kernel transitional boundaries](../modules/kernel/transitional-boundaries.md)
- [ADR index](./README.md)
