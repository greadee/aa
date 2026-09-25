# ADR-0096 — The visualizer consumes the shared `contracts.Event` taxonomy through an `EventSource` seam

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P8-001` |
| **Source** | [docs/phases/ph8-visualizer/plan.md](../phases/ph8-visualizer/plan.md) |
| **Scope** | phase · visualizer |

## Context

Backfilled from the original decision record for **phase 8 (visualizer)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

The visualizer consumes the shared `contracts.Event` taxonomy through an `EventSource` seam

## Rationale

It must not define its own observation protocol; one event taxonomy keeps live and replay identical

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../phases/ph8-visualizer/plan.md).

## Related

- Legacy id `ADR-P8-001`
- [ADR index](./README.md)
