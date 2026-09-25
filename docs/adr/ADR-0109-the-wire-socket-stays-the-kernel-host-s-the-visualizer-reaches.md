# ADR-0109 — The wire socket stays the kernel host's; the visualizer reaches it without changing callers

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P8U-004` |
| **Source** | [visualizer/module-upd-plan.md](../../visualizer/module-upd-plan.md) |
| **Scope** | module update · visualizer |

## Context

Backfilled from the original decision record for **module update 8 (visualizer)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

The wire socket stays the kernel host's; the visualizer reaches it without changing callers

## Rationale

Transport ownership is `obsv`/kernel; the visualizer adds no transport of its own

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../../visualizer/module-upd-plan.md).

## Related

- Legacy id `ADR-P8U-004`
- [ADR index](./README.md)
