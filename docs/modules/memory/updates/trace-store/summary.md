# memory — Module Update Summary: trace store

> Authored at the **end** of the module update, on branch `ph2-memory`. Derived from [plan.md](plan.md).
> Issue: [ISS-TRACE-2](../../../../issues/ph2-memory-trace-store.md) · Parent umbrella: [ISS-TRACE-LOOP](../../../../issues/trace-learning-substrate.md) · Phase PR: [#13](https://github.com/greadee/aa/pull/13)

## Delivered

| Slice | Status | Commit message | Notes |
|---|---|---|---|
| Issue documentation and module plan | Complete | `add ph2 memory module update plan and issue documentation` | `docs/issues/`, this plan |
| Trace repository | Complete | `add memory trace repository` | `memory/repo/trace.go` + tests |
| Trace query summaries | Complete | `add memory trace query summaries` | `memory/query/trace.go` + tests |
| Projected-view retention | Complete | `add memory trace retention` | `memory/retention/` + tests |
| Facade wiring | Complete | `wire memory trace store` | `Memory.Traces()`, README, integration test |
| Module update summary | Complete | `add ph2 memory module update summary` | this file |
| Phase PR link | Complete | `link phase 2 module update pull request` | [#13](https://github.com/greadee/aa/pull/13) |

## What was added

- `memory/repo` `TraceRepository` — idempotent `Ingest` of `v1.Trace` as canonical records of kind `trace`; `Get`, `List`, `ListByAttempt`, `ListByProject`.
- `memory/query` — `TraceSummary` and `TraceSummaries`/`TraceSummariesByAttempt`, deriving steps, failures, duration, tokens, cost, and an outcome (`succeeded`/`failed`/`blocked`/`skipped`).
- `memory/retention` — `Policy` and `Trim` bounding steps per trace and traces per attempt on a projected view only.
- `memory.Memory.Traces()` on the facade, plus rebuild-equivalence and persistence coverage.

## What this changes about the module and the app

- `aa-memory` gains a canonical trace store. Traces are L2 evidence: stored, projected, and rebuilt like any record, but they do not enter the memory lifecycle.
- Learning can now read a deterministic, cross-session trace surface instead of thin per-attempt telemetry. The canonical store remains authoritative; retention never deletes records.
- The app is otherwise unchanged: no new dependency, and `memory` still imports only `contracts`.

## Is this part of a larger change?

Yes. This is the second sub-issue of the umbrella change documented at [ISS-TRACE-LOOP](../../../../issues/trace-learning-substrate.md). It depends on [ISS-TRACE-1](../../../../issues/ph1-contracts-trace-contract.md) and unblocks the learning and capture sub-issues on `ph3-kernel` and `ph9-joblearn`; those solutions are not specified here.

## Validation

| Gate | Result | Evidence |
|---|---|---|
| Go build / vet / test | Pass | `memory` suite incl. trace repo, query, retention, rebuild-equivalence |
| Go formatting | Pass | `gofmt -l .` clean |
| Boundaries | Pass | `tools/archtest` reports ok (`memory` -> `contracts` only) |
| Rebuild equivalence | Pass | `TestTraceStoreRebuildEquivalence` digest-stable with traces |
| Determinism | Pass | summaries and retention stable across input order |

## Decisions affirmed

| # | Decision | Outcome |
|---|---|---|
| ADR-P2U-001 | Traces are canonical `trace` records, not `memory_record`s | Affirmed — evidence stays distinct from promoted knowledge |
| ADR-P2U-002 | Idempotent ingest with revision supersede | Affirmed — re-ingest is a no-op; stale revisions rejected |
| ADR-P2U-003 | Deterministic derived read surface | Affirmed |
| ADR-P2U-004 | Retention trims the projected view only | Affirmed — canonical records untouched |
| ADR-P2U-005 | Promotion stays with `MemoryRecordRepository` | Affirmed — no second lifecycle |
| ADR-P2U-006 | No new dependencies | Affirmed — boundary check clean |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | Retention as a store-level prune | Retention as projected-view trimming (`memory/retention`) | Canonical records must stay rebuild-equivalent and append-oriented; this mirrors the `visualizer/retention` precedent |

## Follow-Up

- `ph3-kernel`: capture real traces via live `obsv` and the kernel host (ISS-OBSV-1).
- `ph9-joblearn`: distill and evaluate subagent artifacts from the trace surface (ISS-LEARN-1).
