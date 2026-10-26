# registry/policies

> Project history for the `registry/policies` submodule of [aa-registry](./README.md).

**Responsibility** — Durable policy-definition catalogue: what policies exist (approval, budget, routing, gate). Which policies
apply to a given workflow/project/execution is decided elsewhere; applicability
is not a registry concern. `Registry` stores and serves validated `PolicySpec` definitions (`Add` rejects duplicates; `Get`/`List`/`Count` read deterministically); applicability is decided elsewhere.

**Source** — [`registry/policies/`](../../../registry/policies/)

**Parent module** — [aa-registry](./README.md) · [docs/modules](../README.md)
