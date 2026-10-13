# kernel/context

> Project history for the `kernel/context` submodule of [aa-kernel](./README.md).

**Responsibility** — Package context compiles deterministic, bounded, provenance-preserving context bundles from retrieval candidates.

**Source** — [`kernel/context/`](../../../kernel/context/)

**Parent module** — [aa-kernel](./README.md) · [docs/modules](../README.md)

## Context compilation

`kernel/context` owns the task's working context:

- **Needs discovery** — `DeriveNeeds(objective, workPackageID)` derives `Kinds`,
  `Terms`, and a `Limit` purely, without a model.
- **Selection** — `CompileFromRetrieval(ctx, retriever, …)` consumes the
  ISS-IMP-2 `memory/retrieval.Retriever` interface (ISS-IMP-3 boundary).
- **Budgeting and compression** — the pure `Compiler` estimates tokens
  (`ceil(runes/4)`) and truncates on a rune boundary when the budget is tight.
- **Provenance** — every `Section` carries the candidate's
  `retrieval.Provenance`, and the bundle `Digest` includes it.
- **Assembly** — `Bundle` is the final context package.
- **Overflow** — `CompileFromRetrieval` fails closed with `ErrBudgetExceeded`
  (returning the partial, `Truncated` bundle) when selection does not fit.

Context never chooses roles, models, or compute — that is allocation
(`kernel/allocator`).

Decision record: [ADR-0146](../../adr/ADR-0146-context-compilation-and-a8-extraction.md).
