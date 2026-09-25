# ADR-0081 — Revision sync is one-way and receiver-authoritative; deletion is a guarded tombstone

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P6-006` |
| **Source** | [docs/phases/ph6-sync/plan.md](../phases/ph6-sync/plan.md) |
| **Scope** | phase · sync |

## Context

Backfilled from the original decision record for **phase 6 (sync)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

Revision sync is one-way and receiver-authoritative; deletion is a guarded tombstone

## Rationale

Prevents replica divergence and accidental data loss

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../phases/ph6-sync/plan.md).

## Related

- Legacy id `ADR-P6-006`
- [ADR index](./README.md)
