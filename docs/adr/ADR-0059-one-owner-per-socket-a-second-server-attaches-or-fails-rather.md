# ADR-0059 — One owner per socket; a second server attaches or fails rather than binding twice

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P4U-002` |
| **Source** | [docs/modules/sifter/updates/production-rpc-wiring/plan.md](../modules/sifter/updates/production-rpc-wiring/plan.md) |
| **Scope** | module update · sifter |

## Context

Backfilled from the original decision record for **module update 4 (sifter)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

One owner per socket; a second server attaches or fails rather than binding twice

## Rationale

Mirrors the `obsv` attach-or-own precedent and prevents split-brain

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../modules/sifter/updates/production-rpc-wiring/plan.md).

## Related

- Legacy id `ADR-P4U-002`
- [ADR index](./README.md)
