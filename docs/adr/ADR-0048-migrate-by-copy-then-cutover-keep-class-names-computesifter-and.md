# ADR-0048 — Migrate by copy-then-cutover; keep class names (`ComputeSifter`) and add a `Sifter` alias

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P4-001` |
| **Source** | [docs/phases/ph4-sifter/plan.md](../phases/ph4-sifter/plan.md) |
| **Scope** | phase · sifter |

## Context

Backfilled from the original decision record for **phase 4 (sifter)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

Migrate by copy-then-cutover; keep class names (`ComputeSifter`) and add a `Sifter` alias

## Rationale

Preserves a proven, tested implementation and avoids an unnecessary rewrite; the package/distribution are renamed to `aa_sifter`/`aa-sifter`

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../phases/ph4-sifter/plan.md).

## Related

- Legacy id `ADR-P4-001`
- [ADR index](./README.md)
