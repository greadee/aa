# ADR-0051 — Token limits are enforced, not configured

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P4-004` |
| **Source** | [docs/phases/ph4-sifter/plan.md](../phases/ph4-sifter/plan.md) |
| **Scope** | phase · sifter |

## Context

Backfilled from the original decision record for **phase 4 (sifter)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

Token limits are enforced, not configured: size messages to `context_limit`, pass `max_output_tokens`

## Rationale

Prevents silent over-limit cloud sends and runaway output

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../phases/ph4-sifter/plan.md).

## Related

- Legacy id `ADR-P4-004`
- [ADR index](./README.md)
