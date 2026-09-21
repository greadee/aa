# ISS-TRACE-2 — Memory: canonical trace store

**Type:** feature
**Status:** complete (phase 2 module update)
**Branch:** `ph2-memory`
**Phase PR:** [#13](https://github.com/greadee/aa/pull/13)
**Parent:** [ISS-TRACE-LOOP](trace-learning-substrate.md)
**Module plan:** [../../memory/module-upd-plan.md](../../memory/module-upd-plan.md)
**Module summary:** [../../memory/module-upd-summary.md](../../memory/module-upd-summary.md)

## Goal

Make `aa-memory` the canonical system of record for bounded per-step traces introduced by [ISS-TRACE-1](ph1-contracts-trace-contract.md): persist them idempotently, project and rebuild them, expose a deterministic read surface for learning, and bound the projected view without deleting canonical records.

## Problem

`contracts` defines a `trace`, but `memory` had no way to store, project, or read one. Without a canonical trace store, step-level evidence cannot survive, be rebuilt, or be compared across sessions, so cross-project learning has nothing durable to consume.

## Requirements

- Idempotent ingestion keyed by content hash; a higher revision supersedes and a stale revision is rejected.
- Canonical records of kind `trace`, projected and rebuild-equivalent; projections remain disposable.
- A deterministic, derived read surface (trace summaries) that does not depend on wall clock or input order.
- Retention that bounds the projected view only, mirroring `visualizer/retention`; canonical records are never deleted.
- No new dependencies; `memory` continues to import only `contracts`.

## Acceptance Criteria

- [x] `TraceRepository` ingests, gets, and lists traces, with by-attempt and by-project reads.
- [x] Re-ingesting identical content is a no-op; stale revisions are rejected.
- [x] `memory/query` derives deterministic `TraceSummary` values (steps, failures, duration, tokens, cost, outcome).
- [x] `memory/retention` trims a projected view (steps per trace, traces per attempt) without mutating input or canonical records.
- [x] `Memory.Rebuild()` is digest-stable with traces present, and traces persist across reopen.
- [x] `go build`, `go vet`, `go test`, `gofmt`, and boundary checks pass.

## Affected branches

The trace store is consumed by the learning loop on `ph9-joblearn` (which promotes derived knowledge through the existing `MemoryRecordRepository` lifecycle) and is populated by capture on `ph3-kernel`. Those solutions are not specified here.

## Notes

Traces are L2 evidence, not promoted knowledge, so they do not enter the memory lifecycle. Promotion of derived knowledge remains explicit and evidence-based through the existing repository.
