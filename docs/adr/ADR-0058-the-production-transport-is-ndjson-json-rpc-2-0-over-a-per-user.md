# ADR-0058 — The production transport is NDJSON JSON-RPC 2.0 over a per-user local socket, exactly as `contracts/rpc/rpc-v1.md`

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P4U-001` |
| **Source** | [docs/modules/sifter/updates/production-rpc-wiring/plan.md](../modules/sifter/updates/production-rpc-wiring/plan.md) |
| **Scope** | module update · sifter |

## Context

Backfilled from the original decision record for **module update 4 (sifter)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

The production transport is NDJSON JSON-RPC 2.0 over a per-user local socket, exactly as `contracts/rpc/rpc-v1.md`

## Rationale

One cross-module transport spec; the service stays transport-agnostic

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../modules/sifter/updates/production-rpc-wiring/plan.md).

## Related

- Legacy id `ADR-P4U-001`
- [ADR index](./README.md)
