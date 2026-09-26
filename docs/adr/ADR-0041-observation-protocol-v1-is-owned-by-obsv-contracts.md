# ADR-0041 — Observation protocol v1 is owned by `obsv`; `contracts` orchestration events are untouched

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P3U-001` |
| **Source** | [docs/modules/obsv/updates/observation-substrate/plan.md](../modules/obsv/updates/observation-substrate/plan.md) |
| **Scope** | module update · obsv |

## Context

Backfilled from the original decision record for **module update 3 (obsv)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

Observation protocol v1 is owned by `obsv`; `contracts` orchestration events are untouched

## Rationale

`contracts/go/v1/event.go` already defers observation events to `aa-obsv`; one owner per vocabulary

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../modules/obsv/updates/observation-substrate/plan.md).

## Related

- Legacy id `ADR-P3U-001`
- [ADR index](./README.md)
