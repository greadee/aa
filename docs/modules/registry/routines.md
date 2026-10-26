# registry/routines

> Project history for the `registry/routines` submodule of [aa-registry](./README.md).

**Responsibility** — Durable routine-definition store: an
organization-owned, reusable process for recurring work. A project deploys a
routine as a workflow; the workflow engine lives in `toolbox`. `RoutineSpec`
re-exports the contracts v2 `routine`; `Registry` stores and serves validated
definitions (`Add` rejects duplicates; `Get`/`List`/`Count` read deterministically).

**Source** — [`registry/routines/`](../../../registry/routines/)

**Parent module** — [aa-registry](./README.md) · [docs/modules](../README.md)
