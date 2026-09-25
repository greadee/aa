# ADR-0054 — Contracts are validated with `aa_contracts` at the RPC boundary; sifter types never leak cross-module

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P4-007` |
| **Source** | [docs/phases/ph4-sifter/plan.md](../phases/ph4-sifter/plan.md) |
| **Scope** | phase · sifter |

## Context

Backfilled from the original decision record for **phase 4 (sifter)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

Contracts are validated with `aa_contracts` at the RPC boundary; sifter types never leak cross-module

## Rationale

One schema source; the Python binding is the only cross-module dependency

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../phases/ph4-sifter/plan.md).

## Related

- Legacy id `ADR-P4-007`
- [ADR index](./README.md)
