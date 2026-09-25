# ADR-0064 — Idempotency keys are recorded for the retention window of the affected aggregate

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P4U-007` |
| **Source** | [docs/modules/sifter/updates/production-rpc-wiring/plan.md](../modules/sifter/updates/production-rpc-wiring/plan.md) |
| **Scope** | module update · sifter |

## Context

Backfilled from the original decision record for **module update 4 (sifter)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

Idempotency keys are recorded for the retention window of the affected aggregate

## Rationale

The spec requires replay-safe mutating calls

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../modules/sifter/updates/production-rpc-wiring/plan.md).

## Related

- Legacy id `ADR-P4U-007`
- [ADR index](./README.md)
