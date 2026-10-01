# kernel/allocator/model_allocator

> Project history for the `kernel/allocator/model_allocator` submodule of [aa-kernel](./README.md).

**Responsibility** — Model allocation: select the model identity / inference
configuration for a work unit from `registry/models`, with an escalation target.
It owns model identity selection and never decides worker count, concurrency, or
compute budgets.

**Source** — [`kernel/allocator/model_allocator/`](../../../kernel/allocator/model_allocator/)

**Parent module** — [aa-kernel](./README.md) · [docs/modules](../README.md)

## Selection

`Allocate(reg, Requirement{Capabilities, Locality, MaxInputCostPer1K})` filters
available models by locality, capability subset, and cost cap, then ranks by
lowest cost, then latency, then id. When the primary is not reasoning-capable it
returns the strongest reasoning-capable eligible model as the escalation target.
No eligible model is `ErrNoModel`. Deterministic.
