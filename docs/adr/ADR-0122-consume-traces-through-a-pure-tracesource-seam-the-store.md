# ADR-0122 — Consume traces through a pure `TraceSource` seam; the store adapter lives beside the engine like `promote/memory.go`

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P9U-001` |
| **Source** | [kernel/joblearn/module-upd-plan.md](../../kernel/joblearn/module-upd-plan.md) |
| **Scope** | module update · kernel/joblearn |

## Context

Backfilled from the original decision record for **module update 9 (kernel/joblearn)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

Consume traces through a pure `TraceSource` seam; the store adapter lives beside the engine like `promote/memory.go`

## Rationale

Keeps the engine pure and unit-testable while `kernel` remains the only importer of `memory`

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../../kernel/joblearn/module-upd-plan.md).

## Related

- Legacy id `ADR-P9U-001`
- [ADR index](./README.md)
