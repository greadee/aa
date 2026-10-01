# registry/models

> Project history for the `registry/models` submodule of [aa-registry](./README.md).

**Responsibility** — The durable model-definition store: provider/runtime,
identifier/version, capabilities, context limits, tool support, locality,
availability, cost, and latency. `ModelSpec` re-exports the contracts v2
`model_spec`; `Registry` stores and serves specs; `DefaultCatalog` mirrors the
inference service's advisory catalog so the model allocator has a populated
registry.

**Source** — [`registry/models/`](../../../registry/models/)

**Parent module** — [aa-registry](./README.md) · [docs/modules](../README.md)

## Registry

`NewRegistry()` is empty; `NewDefaultRegistry()` loads `DefaultCatalog()`
(10 models mirrored from `runtime/inference/.../catalog.toml`). `Add` validates
the spec and rejects duplicates; `Get`/`List`/`Count` read deterministically
(`List` sorted by id). Allocating a model is `kernel/allocator/model_allocator`.
