# ADR-0044 — The journal is append-only, sequence-ordered, and replay-equivalent; retention trims a projected view only

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P3U-004` |
| **Source** | [obsv/module-upd-plan.md](../../obsv/module-upd-plan.md) |
| **Scope** | module update · obsv |

## Context

Backfilled from the original decision record for **module update 3 (obsv)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

The journal is append-only, sequence-ordered, and replay-equivalent; retention trims a projected view only

## Rationale

Deterministic replay and a canonical record mirror the `memory`/`visualizer` retention precedent

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../../obsv/module-upd-plan.md).

## Related

- Legacy id `ADR-P3U-004`
- [ADR index](./README.md)
