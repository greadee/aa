# kernel/allocator/compute_allocator

> Project history for the `kernel/allocator/compute_allocator` submodule of [aa-kernel](./README.md).

**Responsibility** — Compute allocation: worker count, parallelism, budgets, and
placement — resource quantity and topology — without changing model identity.

**Source** — [`kernel/allocator/compute_allocator/`](../../../kernel/allocator/compute_allocator/)

**Parent module** — [aa-kernel](./README.md) · [docs/modules](../README.md)

## Planning

`Plan(Input{Units, IndependentUnits, Justification, MaxConcurrency, Budget, Placement})`
is opt-in and deterministic: without a valid `Justification` (`latency`,
`quality`, `coverage`, `reliability`) it stays serial. With one, workers are
bounded by the independent-unit count and `MaxConcurrency`. Idle agents are never
a reason to spawn workers; unknown justifications and negative counts are
`ErrInvalid`.
