# kernel/allocator/role_allocator

> Project history for the `kernel/allocator/role_allocator` submodule of [aa-kernel](./README.md).

**Responsibility** — Deterministic selection of the worker instances whose
expertise satisfies a work requirement, with rejection reasons. It answers "what
expertise is required" by matching role, capability, and (transitionally) trade;
accepted workers are ordered by cost weight then id.

**Source** — [`kernel/allocator/role_allocator/`](../../../kernel/allocator/role_allocator/)

**Parent module** — [aa-kernel](./README.md) · [docs/modules](../README.md)
