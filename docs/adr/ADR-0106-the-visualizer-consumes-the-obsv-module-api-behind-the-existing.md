# ADR-0106 — The visualizer consumes the `obsv` module API behind the existing `source.EventSource` seam; the fake becomes test-only

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P8U-001` |
| **Source** | [docs/modules/visualizer/updates/obsv-adoption/plan.md](../modules/visualizer/updates/obsv-adoption/plan.md) |
| **Scope** | module update · visualizer |

## Context

Backfilled from the original decision record for **module update 8 (visualizer)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

The visualizer consumes the `obsv` module API behind the existing `source.EventSource` seam; the fake becomes test-only

## Rationale

One seam already exists; swapping the implementation must not change projection, layout, or replay

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../modules/visualizer/updates/obsv-adoption/plan.md).

## Related

- Legacy id `ADR-P8U-001`
- [ADR index](./README.md)
