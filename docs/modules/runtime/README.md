# aa-runtime — module documentation

> Project history for the `aa-runtime` module. Part of [docs/modules](../README.md).

**Source** — [`runtime/`](../../../runtime/) · **Updates** — [`updates/`](./updates/)

## Responsibility

Own execution mechanics for allocated work: instantiate and run workers
(Role × Model + bindings), expose the provider-neutral execution adapter, and
integrate execution isolation (reserved) and the inference service. It does not
decide allocation or scheduling.

## Owns

- Worker execution mechanics (the `worker` adapter seam).
- Worker lifecycle (reserved).
- The execution-isolation boundary (reserved; sandboxing is not implemented).
- The model provider/execution service boundary (the nested Python `inference` subproject).

## Must not

- Decide allocation or scheduling (that is `kernel/allocator` and `kernel/scheduler`).
- Own durable definitions (`registry`) or canonical history (`memory`).

## Submodules

| Submodule | Responsibility |
|---|---|
| [`worker`](./worker.md) | Provider-neutral execution adapter and deterministic fake |
| [`lifecycle`](./lifecycle.md) | Reserved: worker lifecycle |
| [`sandbox`](./sandbox.md) | Reserved: execution isolation (not implemented) |
| [`inference`](./inference.md) | Model provider/execution service (renamed from sifter; nested Python subproject) |
