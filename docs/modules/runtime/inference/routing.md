# inference/src/aa_inference/routing

> Project history for the Python `routing` submodule of [aa-runtime](../README.md).

**Responsibility** — Deterministic routing policies, preflight assessment, cloud budgets, and escalation evaluation.

> **Ownership.** Model-locality/tier routing **policy** and the budget gate are
> owned by the control plane (`kernel/allocator/routing`, [ADR-0151](../../../adr/ADR-0151-routing-and-budget-policy-in-the-allocator.md)).
> This Python package remains the inference service's execution surface; when the
> execution path is wired it consumes the allocator's decision. Inference is
> provider/execution plus verification.

**Source** — [`inference/src/aa_inference/routing/`](../../../../runtime/inference/src/aa_inference/routing/)

**Parent module** — [aa-runtime](../README.md) · [docs/modules](../../README.md)
