# Phase 8 — Visualizer: Plan

> Authored at the **start** of the phase. Branch and folder: `ph8-visualizer`.

## Objective

Deliver `aa-visualizer`'s time-travel capability: a deterministic session-state projection that folds the shared event taxonomy into a graph, a deterministic layout under a large-graph performance budget, a replay/time-travel cursor shared by live and replay, an identity mapping re-derived after replay, an explicit compatibility profile for observation metadata, retention over session trails, and memory session browsing over a query seam. The entire capability is testable offline.

## Starting State

- Starting ref: `main @ 6fd83a3` (after phase 7 merge and record).
- Available:
  - The `contracts` event taxonomy (`contracts/go/v1/event.go`): `Event` with `Sequence`, `Type`, `Aggregate`, `Actor`, `CausationID`, `CorrelationID`, and `Payload`.
  - The `memory` query API (`memory/query`) with `Events()` and derived histories, and the facade (`memory.Open`).
  - The architecture boundary harness allowing `visualizer -> contracts, obsv, memory` (`tools/archtest`).
  - A scaffolded `visualizer/` Go module (`go.mod`, `doc.go`, `README.md`, `aa-visualizer.md`).
  - The phase documentation conventions and templates (`docs/github-os/phase-documentation.md`).
- Missing:
  - Any visualizer implementation; `visualizer` holds `doc.go` only.
  - The session-state projection and the identity mapping.
  - The deterministic layout and its performance budget.
  - The replay/time-travel cursor and the live bridge.
  - The explicit observation compatibility profile.
  - Session browsing and retention.
- Known constraints:
  - `obsv` is scaffolded with no exported API; a real observation transport is not available. The visualizer uses an `EventSource` seam and a deterministic fake.
  - Live and replay must use the same events and the same renderer path.
  - Node identity is re-derived after replay, never persisted as truth.
  - No file contents or raw prompts are persisted; observation metadata is metadata-only.
  - `visualizer` never imports `kernel` and never defines its own observation protocol.
  - Ordering derives from event `Sequence`, never the wall clock.
  - The default test suite makes no network calls.
  - The phase branch name equals the phase folder name (`ph{N}-{scope}`) and the phase PR exists as a Draft from phase start.

## Scope

### In Scope
- Domain types for sessions, graph nodes, edges, timelines, and frames.
- An `EventSource` seam (`Replay`/`Subscribe`) over the shared `contracts.Event` taxonomy, with a deterministic in-memory fake.
- A session-state projection: fold an event stream into nodes and edges deterministically by `Sequence`.
- An identity mapping that re-derives stable node IDs from event aggregates and survives replay.
- An explicit compatibility profile that normalizes optional observation metadata (`secondary_paths`, `access_sequence`) and ignores unknown metadata.
- A deterministic layout with a large-graph performance budget.
- A replay/time-travel cursor: snapshot, step, and seek over a projected timeline.
- A live bridge that feeds a subscription through the same projection and render path.
- Session browsing over a `memory` query seam, plus deterministic session listing.
- Retention over session trails.

### Out of Scope
- The Wails + React/Three UI, camera controls, and pixel rendering; the renderer is a data projection behind a small render seam. A UI surface is a follow-up.
- The real `obsv` transport, journal, and emitter; `obsv` is scaffolded, so the `EventSource` seam and a deterministic fake stand in.
- Storing the graph or session state as canonical truth; state is always re-derived from events.
- Vector/graph indexes and cross-session geometry inference; a memory query seam is used as-is.
- Live network, file watching, and persistence of file contents or raw prompts.

### Required End State
- [ ] The projection is deterministic and independent of delivery order (events fold by `Sequence`).
- [ ] Node identity is stable across replay and re-derived, not persisted.
- [ ] The compatibility profile maps `secondary_paths`/`access_sequence` and ignores unknown metadata.
- [ ] Layout is deterministic and within the large-graph performance budget.
- [ ] Live and replay share one projection and render path.
- [ ] The replay cursor snapshots, steps, and seeks deterministically.
- [ ] Sessions are listed and loaded through the memory query seam.
- [ ] Retention trims old frames deterministically.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and boundary checks pass.

## Architecture Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P8-001 | The visualizer consumes the shared `contracts.Event` taxonomy through an `EventSource` seam | It must not define its own observation protocol; one event taxonomy keeps live and replay identical |
| ADR-P8-002 | Projection folds events by `Sequence`, never the wall clock | Deterministic replay across clocks and machines |
| ADR-P8-003 | Node identity is derived from aggregate kind + id and re-derived after replay | Identity is a view, not persisted truth |
| ADR-P8-004 | The compatibility profile is explicit and versioned; unknown metadata is ignored | Adopting observation metadata must be deliberate and tested |
| ADR-P8-005 | Layout is a pure function of the graph | Deterministic, testable, and independent of render order |
| ADR-P8-006 | Live and replay share one projection and one render path | A single code path prevents live/replay drift |
| ADR-P8-007 | The replay cursor is an index into a deterministic frame list | Time travel is a pure seek, not a stateful re-simulation |
| ADR-P8-008 | Session browsing goes through a `memory` query seam | The visualizer never owns canonical history |
| ADR-P8-009 | Retention is applied to the projected trail, not the source | The source is authoritative; the visualizer trims its own view |
| ADR-P8-010 | The renderer is a data projection with an explicit budget | Keeps the UI out of the tested core and enforces the performance exit criterion |

Diagrams: [plan.uml](./plan.uml).

## Slices

Each slice is independently validatable and maps to exactly one commit.

### Slice 1 — Phase plan and UML
**Goal** — Plan the phase and its intended infrastructure. **Inputs** — architecture, contracts, the visualizer directive. **Expected Output** — `docs/phases/ph8-visualizer/plan.md` + `plan.uml`. **Model Class** — Strong. **Commit Message** — `add ph8 visualizer phase plan`. **Validation** — docs link check. **Dependencies** — none.

### Slice 2 — Core types and errors
**Goal** — Session, node, edge, graph, frame, and cursor types plus sentinel errors and a deterministic clock. **Inputs** — the visualizer directive. **Expected Output** — `visualizer/types.go`, `visualizer/errors.go`, `visualizer/clock.go` + tests. **Model Class** — General. **Commit Message** — `add visualizer core types and errors`. **Validation** — `go test ./...`. **Dependencies** — slice 1. **Documentation** — `visualizer/README.md`.

### Slice 3 — Event source, fake, and compatibility profile
**Goal** — The `EventSource` seam, a deterministic in-memory fake, and the explicit observation-metadata compatibility profile. **Inputs** — slice 2, `contracts/go/v1/event.go`. **Expected Output** — `visualizer/source/*.go`, `visualizer/compat/*.go` + tests. **Model Class** — Strong. **Commit Message** — `add visualizer event source and compatibility profile`. **Validation** — `go test ./...`; profile and unknown-metadata tests. **Dependencies** — slice 2. **Documentation** — `visualizer/README.md`.

### Slice 4 — Session projection and identity
**Goal** — Fold events into a deterministic graph and re-derive stable node identity. **Inputs** — slice 3. **Expected Output** — `visualizer/session/*.go` + tests. **Model Class** — Strong. **Commit Message** — `add visualizer session projection and identity`. **Validation** — `go test ./...`; determinism and identity tests. **Dependencies** — slice 3. **Documentation** — `visualizer/README.md`.

### Slice 5 — Deterministic layout
**Goal** — A pure, deterministic layout with a large-graph performance budget. **Inputs** — slice 4. **Expected Output** — `visualizer/layout/*.go` + tests and benchmark. **Model Class** — Strong. **Commit Message** — `add visualizer deterministic layout`. **Validation** — `go test ./...`; budget test. **Dependencies** — slice 4. **Documentation** — `visualizer/README.md`.

### Slice 6 — Replay cursor
**Goal** — Build a deterministic frame list and a cursor that snapshots, steps, and seeks. **Inputs** — slices 4–5. **Expected Output** — `visualizer/replay/*.go` + tests. **Model Class** — General. **Commit Message** — `add visualizer replay cursor`. **Validation** — `go test ./...`. **Dependencies** — slice 5. **Documentation** — `visualizer/README.md`.

### Slice 7 — Live bridge
**Goal** — Feed a subscription through the same projection and render path as replay. **Inputs** — slices 3, 6. **Expected Output** — `visualizer/bridge/*.go` + tests. **Model Class** — General. **Commit Message** — `add visualizer live bridge`. **Validation** — `go test ./...`; live-equals-replay test. **Dependencies** — slice 6. **Documentation** — `visualizer/README.md`.

### Slice 8 — Session browsing and retention
**Goal** — List and load sessions over the memory query seam, and trim trails by retention. **Inputs** — slices 4, 6. **Expected Output** — `visualizer/browse/*.go`, `visualizer/retention/*.go` + tests. **Model Class** — General. **Commit Message** — `add visualizer session browsing and retention`. **Validation** — `go test ./...`. **Dependencies** — slice 7. **Documentation** — `visualizer/README.md`.

### Slice 9 — Phase summary and UML
**Goal** — Record the as-built phase. **Inputs** — all prior slices. **Expected Output** — `summary.md` + `summary.uml` + updated `visualizer/README.md`. **Model Class** — Strong. **Commit Message** — `add ph8 visualizer phase summary`. **Validation** — link check. **Dependencies** — slice 8.

## Exit Criteria

- [ ] Projection is deterministic and sequence-ordered.
- [ ] Identity is stable across replay and re-derived.
- [ ] The compatibility profile is explicit and tested; unknown metadata is ignored.
- [ ] Layout is deterministic and within the large-graph budget.
- [ ] Live and replay share one path.
- [ ] The replay cursor snapshots, steps, and seeks deterministically.
- [ ] Sessions list and load through the memory query seam.
- [ ] Retention trims trails deterministically.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and boundary checks pass.
- [ ] Phase PR merged to `main`; branch retained.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | sequence folding, identity derivation, profile mapping, cursor seek, retention trim |
| contract | `contracts.Event` conformance for fixtures; unknown metadata ignored |
| integration | source -> projection -> layout -> replay; live path equals replay path |
| security | no file contents or raw prompts; metadata-only; kernel never imported |
| determinism | stable graph, stable layout, stable frames, order-independent delivery |
| performance | large-graph layout within budget (benchmark) |
| boundary | `visualizer` imports only contracts, obsv, memory (archtest) |
