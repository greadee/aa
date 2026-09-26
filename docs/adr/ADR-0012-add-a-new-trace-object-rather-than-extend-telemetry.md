# ADR-0012 — Add a new `trace` object rather than extend `telemetry`

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P1U-001` |
| **Source** | [docs/modules/contracts/updates/trace-contract/plan.md](../modules/contracts/updates/trace-contract/plan.md) |
| **Scope** | module update · contracts |

## Context

Backfilled from the original decision record for **module update 1 (contracts)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

Add a new `trace` object rather than extend `telemetry`

## Rationale

`telemetry` is a per-attempt aggregate with a different authority and retention; D8 keeps operational telemetry, work history, and canonical memory distinct

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../modules/contracts/updates/trace-contract/plan.md).

## Related

- Legacy id `ADR-P1U-001`
- [ADR index](./README.md)
