# kernel/scheduler

> Project history for the `kernel/scheduler` submodule of [aa-kernel](./README.md).

**Responsibility** — Runs aa-kernel's deterministic supervised control cycle
(ready → select → contract → context → lease → run → intake → gates → accept →
telemetry) and owns ordering, dispatch readiness, concurrency policy, the
assignment state machine, and expiring leases, recording through a `Sink`.

**Source** — [`kernel/scheduler/`](../../../kernel/scheduler/)

**Parent module** — [aa-kernel](./README.md) · [docs/modules](../README.md)
