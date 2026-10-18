# inference/src/aa_inference/routing

> Project history for the Python `routing` submodule of [aa-runtime](../README.md).

**Responsibility** — Execution-time helpers: preflight assessment, cloud budget
> accounting, and escalation evaluation. The duplicate routing policy engine
> (~~policy.py~~) was retired in phase 11.

> **Ownership.** Model-locality/tier routing **policy** and the budget gate are
> owned by the control plane (`kernel/allocator/routing`, [ADR-0151](../../../adr/ADR-0151-routing-and-budget-policy-in-the-allocator.md)).
> The RPC execution path consumes the allocator's `routingDecision` and fails
> closed when it is absent or inconsistent. Inference is provider/execution
> plus verification.

**Source** — [`inference/src/aa_inference/routing/`](../../../../runtime/inference/src/aa_inference/routing/)

**Parent module** — [aa-runtime](../README.md) · [docs/modules](../../README.md)
