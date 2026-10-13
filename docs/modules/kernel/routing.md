# kernel/allocator/routing

> Project history for the `kernel/allocator/routing` submodule of [aa-kernel](./README.md).

**Responsibility** — Own the deterministic model-locality/tier routing policy and
the budget gate. Control-plane policy: the allocator decides where a unit runs
and within what budget; inference only provides and executes models.

**Source** — [`kernel/allocator/routing/`](../../../kernel/allocator/routing/)

**Parent module** — [aa-kernel](./README.md) · [docs/modules](../README.md)

## Decision

`Decide(Request{Policy, Significance, CloudAllowed, estimated tokens/costs, Budget})`
resolves a policy (`local_only`, `cloud_only`, `local_first`, `expert_first`,
`adaptive`, `budget_constrained`) and significance (`routine`, `significant`,
`major`, `critical`) into a locality/tier decision. It fails closed: the expert
tier is blocked when cloud is disabled or the estimated cost exceeds
`Budget.maxCostUsd`. Deterministic; no model is called. Moving policy here leaves
`runtime/inference` provider/execution only. See
[ADR-0151](../../adr/ADR-0151-routing-and-budget-policy-in-the-allocator.md).
