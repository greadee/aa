# Phase 2 — Memory: Summary

> Authored at the **end** of the phase. Derived from [plan.md](./plan.md). As-built diagram: [summary.uml](./summary.uml).

## Delivered

| Slice | Status | Commit | Notes |
|---|---|---|---|
| Phase plan and UML | Complete | `25585d6` | plan.md + plan.uml |
| Record model and portable layout | Complete | `a09d77f` | `memory/store/record.go`, `layout.go` |
| Canonical store | Complete | `bb83049` | `memory/store/store.go` |
| Rebuildable projection | Complete | `704e39f` | `memory/projection` |
| Query API and histories | Complete | `784b2bc` | `memory/query` |
| Issue and strategy repositories | Complete | `5939414` | `memory/repo` + promotion |
| Ingestion, provenance, retention | Complete | `8f7cfc5` | `store/ingest.go`, `store/retention.go` |
| Memory facade | Complete | `cf0fab4` | `memory/memory.go` |
| Phase summary and UML | Complete | (this commit) | summary.md + summary.uml |

- Tracking issue: none (phase executed via the phase PR)
- Phase PR: (see GitHub)
- Branch: `ph2-memory` (retained after merge)

## Validation

| Gate | Result | Evidence |
|---|---|---|
| Go build / vet | Pass | all `memory/...` packages |
| Go tests | Pass | store, projection, query, repo, and facade suites |
| Rebuild equivalence | Pass | incremental and rebuilt projections share a digest |
| Idempotent ingestion | Pass | duplicate records/events counted as unchanged |
| Provenance | Pass | absent/present/bad-timestamp cases covered |
| Retention | Pass | below-floor events excluded; floor is monotonic |
| Boundaries | Pass | `archtest` reports memory depends only on contracts |
| Formatting | Pass | `gofmt -l .` clean |
| Docs links | Pending | runs on CI after push |
| Phase audit | Not yet run | before merge to `main` |

## Decisions Affirmed

| # | Decision | Outcome | Evidence |
|---|---|---|---|
| ADR-P2-001 | Canonical JSON records + append-only event log | Affirmed | store round-trips and reopens persist records |
| ADR-P2-002 | Identity `(kind,id)`, monotonic revision, canonical hash | Affirmed | idempotent puts, stale-revision rejection, stable hashes |
| ADR-P2-003 | Projection is an interface with an in-memory reference | Affirmed | rebuild-equivalence and digest stability tests |
| ADR-P2-004 | Ordering is explicit; never wall clock | Affirmed | histories sort by sequence; deterministic digests |
| ADR-P2-005 | Thin typed repositories over `contracts` types | Affirmed | issues/strategies encode to `contracts` v1 objects |
| ADR-P2-006 | Promotion is an explicit state machine | Affirmed | illegal CANDIDATE→ACTIVE rejected; supersession tested |
| ADR-P2-007 | Retention is an explicit event floor | Affirmed | monotonic floor; below-floor events excluded |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | A SQLite projection | In-memory reference projection behind the `Projection` interface | Prove rebuild-equivalence without an external dependency; a SQLite adapter can be added without changing callers |
| 2 | A git history reader | Histories derived from canonical events only | A git adapter is a separate integration; event-derived histories are deterministic and sufficient for the phase exit |
| 3 | Provenance enforced on all records | `RequireProvenance` is opt-in via `IngestOptions`; `ValidateProvenance` runs when present | Contract schemas mark provenance optional; enforcement belongs where a producer is trusted |
| 4 | A dedicated memory-record repository | The promotion state machine is shared and used by the strategy repository | Avoids duplicating lifecycle logic; a memory-record repository can wrap the same transitions later |
| 5 | Event retention by sequence | Implemented as a monotonic floor persisted in `retention.json` | Simple, auditable, and never rewrites history |

## Deferred / Follow-Up

- SQLite projection adapter for large stores.
- A git history reader producing work events from repository history.
- A typed memory-record repository over the shared promotion transitions.
- Provenance enforcement policy per record kind.
- Phase audit before merge to `main`.

## Metrics

- Commits: 9 (8 slices + this summary)
- Child PRs: 0 (single phase PR)
- Packages: `store`, `projection`, `query`, `repo`, plus the `memory` facade
- ADRs: ADR-P2-001 … ADR-P2-007
- Pitfalls recorded: 0
- Tests added: 35 (store 20, projection 5, query 3, repo 5, facade 2)

## Retrospective

### Repeat
- Interface-first design: the `Projection` interface let the reference implementation land before storage choices.
- Determinism as a test invariant (digests, sequence ordering) caught ordering mistakes early.
- Fail-closed tests for traversal paths, stale revisions, and illegal lifecycle transitions.

### Avoid
- Keep provenance policy explicit per kind rather than assuming it.
- Do not let the in-memory projection quietly become canonical; it stays disposable by construction.

### As-Built Diagram

[summary.uml](./summary.uml)
