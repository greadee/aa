# ADR-0014 — Redaction-first

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P1U-003` |
| **Source** | [contracts/module-upd-plan.md](../../contracts/module-upd-plan.md) |
| **Scope** | module update · contracts |

## Context

Backfilled from the original decision record for **module update 1 (contracts)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

Redaction-first: content is withheld or hashed, with `redacted` and `redactionVersion`

## Rationale

Metadata-only by default; hashes support dedup/correlation without carrying recoverable input

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../../contracts/module-upd-plan.md).

## Related

- Legacy id `ADR-P1U-003`
- [ADR index](./README.md)
