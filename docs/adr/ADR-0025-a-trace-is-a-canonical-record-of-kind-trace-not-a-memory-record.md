# ADR-0025 — A trace is a canonical record of kind `trace`, not a `memory_record`

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P2U-001` |
| **Source** | [docs/modules/memory/updates/trace-store/plan.md](../modules/memory/updates/trace-store/plan.md) |
| **Scope** | module update · memory |

## Context

Backfilled from the original decision record for **module update 2 (memory)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

A trace is a canonical record of kind `trace`, not a `memory_record`

## Rationale

Traces are L2 evidence, not promoted knowledge; running them through the memory lifecycle would conflate evidence with validated knowledge

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../modules/memory/updates/trace-store/plan.md).

## Related

- Legacy id `ADR-P2U-001`
- [ADR index](./README.md)
