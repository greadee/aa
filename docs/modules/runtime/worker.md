# runtime/worker

> Project history for the `runtime/worker` submodule of [aa-runtime](./README.md).

**Responsibility** — The provider-neutral worker-execution adapter (`Adapter`,
`Request`, `Result`) with a deterministic scripted fake and a disabled adapter,
plus the `Worker` instance type (a runtime instantiation selected by the
allocator). Execution runs one allocated attempt; the kernel never calls a model
directly.

**Source** — [`runtime/worker/`](../../../runtime/worker/)

**Parent module** — [aa-runtime](./README.md) · [docs/modules](../README.md)
