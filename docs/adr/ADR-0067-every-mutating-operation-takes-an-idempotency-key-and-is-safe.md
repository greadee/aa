# ADR-0067 — Every mutating operation takes an idempotency key and is safe to repeat

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P5-002` |
| **Source** | [docs/phases/ph5-forge/plan.md](../phases/ph5-forge/plan.md) |
| **Scope** | phase · forge |

## Context

Backfilled from the original decision record for **phase 5 (forge)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

Every mutating operation takes an idempotency key and is safe to repeat

## Rationale

Retries and reconnects must not duplicate issues, PRs, or releases

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../phases/ph5-forge/plan.md).

## Related

- Legacy id `ADR-P5-002`
- [ADR index](./README.md)
