# ADR-0008 — Bindings are dependency-light

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P1-003` |
| **Source** | [docs/phases/ph1-contracts/plan.md](../phases/ph1-contracts/plan.md) |
| **Scope** | phase · contracts |

## Context

Backfilled from the original decision record for **phase 1 (contracts)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

Bindings are dependency-light: Go stdlib only, Python stdlib-first (Pydantic optional for `sifter`)

## Rationale

Keeps `contracts` trivially embeddable and avoids dependency cycles

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../phases/ph1-contracts/plan.md).

## Related

- Legacy id `ADR-P1-003`
- [ADR index](./README.md)
