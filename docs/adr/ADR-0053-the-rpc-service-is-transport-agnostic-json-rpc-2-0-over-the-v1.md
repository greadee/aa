# ADR-0053 — The RPC service is transport-agnostic JSON-RPC 2.0 over the v1 envelope

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Legacy id** | `ADR-P4-006` |
| **Source** | [docs/phases/ph4-sifter/plan.md](../phases/ph4-sifter/plan.md) |
| **Scope** | phase · sifter |

## Context

Backfilled from the original decision record for **phase 4 (sifter)**.
Only the decision and its stated rationale were recorded at the time; alternatives
and consequences were not.

## Decision

The RPC service is transport-agnostic JSON-RPC 2.0 over the v1 envelope

## Rationale

Lets the kernel host it over the local socket without coupling the service to transport

## Alternatives and consequences

Not recorded in the original decision record. See the [source](../phases/ph4-sifter/plan.md).

## Related

- Legacy id `ADR-P4-006`
- [ADR index](./README.md)
