# ADR-0016 — New document declares `x-contract-version` 1.1; shared package constants are unchanged

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P1U-005` |
| **Source** | [contracts/module-upd-plan.md](../../contracts/module-upd-plan.md) |
| **Scope** | module update · contracts |

## Context

Backfilled from the original decision record for **module update 1 (contracts)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

New document declares `x-contract-version` 1.1; shared package constants are unchanged

## Rationale

POLICY encodes minor per document; bumping `v1.Version`/`CONTRACT_VERSION` would ripple into `forge`, `memory`, and `sifter` stamping and is not part of this branch

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../../contracts/module-upd-plan.md).

## Related

- Legacy id `ADR-P1U-005`
- [ADR index](./README.md)
