# ADR-0018 — Canonical records are JSON files under `.aa-project/records/<kind>/<id>.json`; events are an append-only `events.jsonl`

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P2-001` |
| **Source** | [docs/phases/ph2-memory/plan.md](../phases/ph2-memory/plan.md) |
| **Scope** | phase · memory |

## Context

Backfilled from the original decision record for **phase 2 (memory)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

Canonical records are JSON files under `.aa-project/records/<kind>/<id>.json`; events are an append-only `events.jsonl`

## Rationale

Portable, git-friendly, human-auditable, and trivially rebuildable

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../phases/ph2-memory/plan.md).

## Related

- Legacy id `ADR-P2-001`
- [ADR index](./README.md)
