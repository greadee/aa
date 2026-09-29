# memory — Module Update Plan: trace store

> Authored at the **start** of the module update, on branch `ph2-memory` (stacked on the `ph1-contracts` trace contract).
> Issue: [ISS-TRACE-2](../../../../issues/ph2-memory-trace-store.md) · Parent umbrella: [ISS-TRACE-LOOP](../../../../issues/trace-learning-substrate.md)

## Objective

Make `aa-memory` the canonical system of record for bounded per-step **traces**: ingest them idempotently, project them, expose a deterministic read surface for the learning loop, and bound the projected view without ever deleting canonical records. This completes the storage half of the trace substrate that belonged to `ph2-memory`.

## Starting State

- Starting ref: `ph1-contracts @ 9dd6c01` (fast-forwarded `ph2-memory`), which adds `v1.Trace`/`v1.TraceStep` to `contracts`.
- Available: canonical record store (`store`), rebuildable projection (`projection`), deterministic query and derived histories (`query`), typed repositories with the memory lifecycle (`repo`), the `memory.Open` facade, and the visualizer's projected-view `retention` pattern to mirror.
- Missing: any way to store, project, or read a `v1.Trace`.
- Known constraints:
  - Directory records under `.aa-project` are canonical; projections are derived and must rebuild to an identical digest.
  - Ingestion is idempotent; history is append-oriented (supersede, never silently overwrite).
  - `memory` executes no work and calls no models; it must not store raw prompts, source, or credentials — traces are already redacted at the contract boundary.
  - Retention is applied to projected views, never to canonical records (the `visualizer/retention` precedent).

## Scope

### In Scope
- A `TraceRepository` (`memory/repo`) with idempotent ingestion, revision handling, and get/list/by-attempt/by-project reads.
- A deterministic trace read surface (`memory/query`) summarizing a trace for learning.
- Projected-view retention (`memory/retention`) that bounds steps per trace and traces per attempt.
- Facade wiring (`memory.Memory.Traces()`), documentation, and tests including rebuild-equivalence.

### Out of Scope
- Trace capture and the observation protocol (`ph3-kernel`).
- Distillation and evaluation of subagent artifacts (`ph9-joblearn`).
- Any new contract object; the schema is delivered by `ph1-contracts`.
- A promotion lifecycle for traces: traces are evidence, and promotion of derived knowledge continues through the existing `MemoryRecordRepository` lifecycle, which `ph9-joblearn` consumes.

## Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P2U-001 | A trace is a canonical record of kind `trace`, not a `memory_record` | Traces are L2 evidence, not promoted knowledge; running them through the memory lifecycle would conflate evidence with validated knowledge |
| ADR-P2U-002 | Ingestion is idempotent by content hash and supersedes by increasing revision | Matches the store's existing record semantics; re-ingesting identical evidence is a no-op |
| ADR-P2U-003 | The read surface is derived and deterministic | Learning must be reproducible; summaries are computed from canonical records, sorted by id, never by wall clock |
| ADR-P2U-004 | Retention trims the projected view only | Mirrors `visualizer/retention`: canonical records stay authoritative and rebuild-equivalent |
| ADR-P2U-005 | Promotion stays with `MemoryRecordRepository`; candidates reference traces by id in `evidence` | Avoids a second lifecycle; keeps evidence and knowledge distinct |
| ADR-P2U-006 | No new dependencies | `memory` still depends only on `contracts` |

## Slices

Each slice maps to exactly one commit.

| Slice | Goal | Commit message |
|---|---|---|
| 1 | Issue documentation and this module plan | `add ph2 memory module update plan and issue documentation` |
| 2 | `TraceRepository` and tests | `add memory trace repository` |
| 3 | Deterministic trace summaries and tests | `add memory trace query summaries` |
| 4 | Projected-view retention and tests | `add memory trace retention` |
| 5 | Facade wiring and module README | `wire memory trace store` |
| 6 | Module update summary | `add ph2 memory module update summary` |
| 7 | Link the phase PR | `link phase 2 module update pull request` |

## Required End State

- [ ] `v1.Trace` can be ingested, stored, projected, and read back.
- [ ] Re-ingesting identical content is a no-op; a higher revision supersedes.
- [ ] Trace summaries are deterministic across runs and index order.
- [ ] Retention bounds the projected view without mutating canonical records or inputs.
- [ ] `Memory.Rebuild()` is digest-stable with traces present.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and boundary checks pass.

## Exit Criteria

- [ ] Trace store delivered with repository, query, and retention.
- [ ] Rebuild-equivalence and idempotency tests pass.
- [ ] Module README and module summary updated.
- [ ] Phase PR opened on `ph2-memory` (stacked on `ph1-contracts`).

## Test Plan

| Layer | What is tested |
|---|---|
| unit | repository defaults/validation/idempotency/revision; summaries; retention trimming never mutates input |
| contract | ingested traces keep their `contractVersion` and validate on read |
| integration | ingest -> project -> query -> rebuild yields an identical digest |
| determinism | summaries and retention are stable regardless of input order |
| boundary | `memory` imports only `contracts` (archtest) |
| docs | link check |
