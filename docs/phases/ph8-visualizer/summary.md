# Phase 8 — Visualizer: Summary

> Authored at the **end** of the phase. Derived from [plan.md](./plan.md). As-built diagram: [summary.uml](./summary.uml).

## Delivered

| Slice | Status | Commit | Notes |
|---|---|---|---|
| Phase plan and UML | Complete | `f44de6e` | plan.md + plan.uml |
| Core types and errors | Complete | `d726147` | `visualizer/{types,errors,clock}.go` |
| Event source, fake, and compatibility profile | Complete | `b85a933` | `visualizer/source`, `visualizer/compat` |
| Session projection and identity | Complete | `afd8393` | `visualizer/session` (fold by `Sequence`, derived identity) |
| Deterministic layout | Complete | `95ace68` | `visualizer/layout` (pure layout + budget + benchmark) |
| Replay cursor | Complete | `d562fb2` | `visualizer/replay` (frames + snapshot/step/seek) |
| Live bridge | Complete | `0c9886b` | `visualizer/bridge` (subscription through the replay path) |
| Session browsing and retention | Complete | `79d6bd8` | `visualizer/browse`, `visualizer/retention` |
| Phase summary and UML | Complete | this commit | summary.md + summary.uml + README |

- Starting ref: `main @ 6fd83a3` (after phase 7 merge and record).
- Tracking issue: none (phase executed via the phase PR).
- Milestone: none (matches phases 0–7 on this repository).
- Phase PR: [#9](https://github.com/greadee/aa1/pull/9) — merged
- Merge commit: `e26b9fb`
- Branch: `ph8-visualizer` (retained after merge on `aa1`)

## Validation

| Gate | Result | Evidence |
|---|---|---|
| Go build / vet | Pass | all `visualizer/...` packages |
| Go tests | Pass | 60 tests + 1 benchmark across 9 packages |
| Formatting | Pass | `gofmt -l .` clean |
| Boundaries | Pass | `archtest` ok; `visualizer` imports `contracts` and `memory` only (its own subpackages aside) |
| Projection determinism | Pass | `TestProjectIsDeterministicAcrossOrder`; folds by `Sequence`, not delivery order |
| Identity | Pass | `TestProjectIdentityStable`; ids re-derived from aggregate kind + id |
| Compatibility profile | Pass | `TestProjectAttachesCompatibilityMetadata`, `TestProjectRejectsInvalidCompatibilityMetadata`; unknown keys ignored |
| Layout determinism | Pass | `TestLayoutIsDeterministic`, `TestLayoutIsOrderIndependent` |
| Layout budget | Pass | `TestLayoutBudget`; `TestLayoutLargeGraphWithinBudget` (50k nodes < 5s); `BenchmarkLayoutLargeGraph` (100k nodes) |
| Replay cursor | Pass | `TestCursorSnapshotAndStep`, `TestCursorSeek`, `TestCursorSeekSequence` |
| Live equals replay | Pass | `TestConsumeLiveEqualsReplay`, `TestConsumeLiveEqualsReplayOutOfOrder` |
| Memory browsing | Pass | `TestBrowserOverMemoryQuery` (real `store`/`projection`/`query`); listing, sequence-ordered load, timeline == replay |
| Retention | Pass | `TestTrimMaxFramesKeepsNewest`, `TestTrimMinSequenceDropsOld`, `TestTrimCombined`, `TestTrimDoesNotMutateInput` |
| Security / privacy | Pass | metadata-only observations; no file contents or raw prompts; `kernel` never imported |
| Docs links | Pass | runs on CI after push |
| Phase audit | Complete | see Audit below |

## Decisions Affirmed

| # | Decision | Outcome | Evidence |
|---|---|---|---|
| ADR-P8-001 | Consume the shared `contracts.Event` taxonomy through an `EventSource` seam | Affirmed | `source.EventSource` + `source.Fake`; live bridge consumes it; no new protocol |
| ADR-P8-002 | Projection folds by `Sequence`, never the wall clock | Affirmed | `orderedCopy` + `TestProjectIsDeterministicAcrossOrder` |
| ADR-P8-003 | Node identity is derived and re-derived after replay | Affirmed | `NodeIDFor`/`SessionNodeID`; `TestProjectIdentityStable`; `test.Build` re-derives every frame |
| ADR-P8-004 | The compatibility profile is explicit, versioned; unknown metadata ignored | Affirmed | `compat.Version`, `compat.Normalize`; invalid supported values rejected, unknown keys ignored |
| ADR-P8-005 | Layout is a pure function of the graph | Affirmed | `TestLayoutIsDeterministic`, `TestLayoutIsOrderIndependent` |
| ADR-P8-006 | Live and replay share one projection and one render path | Affirmed | `bridge` re-derives frames with `replay.Build`; `TestConsumeLiveEqualsReplay` |
| ADR-P8-007 | The replay cursor is an index into a deterministic frame list | Affirmed | `replay.Cursor`; `TestCursorSnapshotAndStep`, `TestCursorSeek` |
| ADR-P8-008 | Session browsing goes through a `memory` query seam | Affirmed | `browse.EventSource` satisfied by `*query.Query`; `TestBrowserOverMemoryQuery` |
| ADR-P8-009 | Retention is applied to the projected trail, not the source | Affirmed | `retention.Trim` over `replay.Frame`s; source untouched; no-mutation test |
| ADR-P8-010 | The renderer is a data projection with an explicit budget | Affirmed | `layout.Budget`/`DefaultBudget`; `replay.Frame` carries graph + placement; benchmark |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | Adopt `obsv` `Subscribe`/`Replay` as the live transport | An `EventSource` seam plus a deterministic in-memory fake | `obsv` is scaffolded with no exported API; live adoption is a follow-up |
| 2 | Reach `obsv`, `memory`, and `contracts` interfaces | `obsv` unused so far; `memory` consumed through the `browse` seam | No observation API exists yet; past-session reading is the only memory need this phase |
| 3 | Deterministic session listing (session key unspecified) | Sessions derived per event: payload `sessionId`, then `ProjectID`, then `DefaultSessionID` | The canonical taxonomy carries no explicit session id; the derivation keeps every event attributable |
| 4 | A `memory` query seam of visualizer-owned records | The `browse.EventSource` seam mirrors `memory`'s `Events() ([]store.EventRecord, error)` | The phase plan adopts the memory query API "as-is"; the boundary is permitted by `archtest` |
| 5 | Wails + React/Three UI, camera, and cross-session geometry | A tested data projection (graph + placement + frames); no UI | Explicitly out of scope; a UI surface is a follow-up |

## Deferred / Follow-Up

- A real `obsv` `Replay`/`Subscribe` transport behind `source.EventSource`; the fake then becomes test-only.
- The Wails + React/Three surface, camera controls, and pixel rendering over `replay.Frame`.
- Adopting an explicit observation session id (obsv) as the primary `SessionKey`, retiring the `ProjectID` fallback.
- Cross-session geometry inference and vector/graph indexes beyond the memory query seam.
- UI end-to-end tests (Playwright) and golden replay fixtures.

## Metrics

- Commits: 8 implementation + this summary
- Packages: 9 (`visualizer`, `source`, `compat`, `session`, `layout`, `replay`, `bridge`, `browse`, `retention`)
- Tests: 60 tests + 1 benchmark
- ADRs: ADR-P8-001 … ADR-P8-010
- Pitfalls recorded: 0
- PRs: 1 phase PR (#9)
- Issues: none

## Audit

| Severity | Count | Notes |
|---|---|---|
| P0 | 0 | — |
| P1 | 0 | — |
| P2 | 0 | — |
| P3 | 2 | Live `obsv` transport deferred (deviation 1); UI surface deferred (deviation 5) |

Audit result: no blocking findings. Boundary rule holds: `visualizer` imports only `contracts` and `memory` (plus its own subpackages); `kernel` is never imported. State is always re-derived from events and never persisted as truth; observation metadata is metadata-only.

## Retrospective

### Repeat
- Keep one `EventSource` seam and one `replay.Build` path; "live equals replay" came down to a single `DeepEqual` test.
- Make layout and projection pure and order-independent; determinism tests were almost trivial because no wall-clock or random state exists.
- Put retention on the derived trail, never the source; the visualizer bounds its own view and leaves `aa-memory` authoritative.
- Prove the memory seam against a real `store`/`projection`/`query`, not only a fake, so the "used as-is" claim is tested.

### Avoid
- Do not let session identity be implicit; deriving a key from `ProjectID` is a stopgap until an observation session id exists.
- Do not re-project per event when a single fold suffices; `replay.Build` amortizes work across the frame list.
- Do not widen a seam to a module's private types unless necessary; the `browse` seam is typed on `store.EventRecord` only because the memory query is adopted as-is.

### As-Built Diagram

[summary.uml](./summary.uml)
