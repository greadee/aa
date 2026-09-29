# ADR-0026 — Ingestion is idempotent by content hash and supersedes by increasing revision

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P2U-002` |
| **Source** | [docs/modules/memory/updates/trace-store/plan.md](../modules/memory/updates/trace-store/plan.md) |
| **Scope** | module update · memory |

## Context

Backfilled from the original decision record for **module update 2 (memory)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

Ingestion is idempotent by content hash and supersedes by increasing revision

## Rationale

Matches the store's existing record semantics; re-ingesting identical evidence is a no-op

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../modules/memory/updates/trace-store/plan.md).

## Related

- Legacy id `ADR-P2U-002`
- [ADR index](./README.md)
