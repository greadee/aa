# runtime/lifecycle

> Project history for the `runtime/lifecycle` submodule of [aa-runtime](./README.md).

**Responsibility** — Worker lifecycle: instantiation, start/run/stop,
termination, cancellation, cleanup, and lifecycle events. The manager implements
the provider-neutral `worker.Adapter` seam (`Run`) and the scheduler's
`WorkerLifecycle` hook (`Begin`/`End`). Events are failure-open and are wired to
`obsv` in the composition root; the manager holds no durable definitions or
canonical history.

**Source** — [`runtime/lifecycle/`](../../../runtime/lifecycle/)

**Parent module** — [aa-runtime](./README.md) · [docs/modules](../README.md)
