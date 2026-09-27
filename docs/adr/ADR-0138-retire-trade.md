# ADR-0138 — Retire the `trade` concept

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) (D-3) |
| **Scope** | update phase · architecture-refactor-1 |

## Context

The agent ontology carried both **role** (durable responsibility) and **trade**
(durable capability) as separate axes, plus a separate capability vocabulary.
`trade` overlapped role and capability, was unvalidated, and was used only as an
extra selection/learning dimension. The target ontology is **Role = expertise
spec, Model = inference, Worker = Role × Model instance**, with capabilities a
separate authority vocabulary.

## Decision

Retire `trade` everywhere in the current generation:

- **contracts v2** — remove `Actor.trade`, `WorkPackage.trade`,
  `WorkflowStep.trade`, and `Applicability.trades` (schemas, Go/TS/Python
  bindings, fixtures).
- **registry/runtime** — remove `Worker.trade`; the role allocator selects on
  role and capability only.
- **joblearn** — remove the `Trade` attribution field and the `ByTrade`
  dimension; per-trade routing becomes per-role routing.

`v1` keeps the former field as the previous generation; consumers move to `v2`.

## Rationale

One vocabulary for expertise: role (spec) plus capability (authority). Removing
the overlapping third axis removes ambiguity, unvalidated strings, and a
selection/learning dimension that duplicated role.

## Alternatives and consequences

- **Keep trade as a deprecated alias:** rejected — it re-opens the overlap.
- **Keep trade only in joblearn:** rejected — attribution must match the
  ontology.
- Consequence: per-trade learned routing is replaced by per-role routing;
  capability-based allocation returns with the ten-issue Sifter allocation work.

## Related

- [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) D-3
- [ADR-0137 — Contracts v2 generation](./ADR-0137-contracts-v2-generation.md)
- [ADR index](./README.md)
