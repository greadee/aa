# Phase 2 — Memory: Plan

> Authored at the **start** of the phase. Branch and folder: `ph2-memory`.

## Objective

Make `aa-memory` the canonical, portable, rebuildable system of record: work, git, job, and project histories; the issue and strategy repositories; the memory hierarchy with evidence-based promotion; provenance and retention; and a deterministic query API. Directory records are the truth; the projection is disposable and must rebuild to an identical digest.

## Starting State

- Starting ref: `main @ f49e211` (after phase 1 merge).
- Available:
  - `aa-contracts` v1 schemas, bindings, and conformance suite (phase 1).
  - `memory` module scaffold (`github.com/greadee/aa/memory`) with `doc.go` only.
  - Architecture boundary harness (`tools/archtest`) enforcing `memory -> contracts, obsv`.
  - Terminology, state registry, and memory lifecycle in `docs/reference/terminology.md`.
- Missing:
  - Any record, store, projection, repository, or query implementation.
  - Portable `.aa-project/` layout.
  - Rebuild-equivalence and idempotent-ingestion guarantees.
- Known constraints:
  - `memory` must not depend on `kernel`, `sync`, `forge`, or `sifter` (RPC-only from kernel).
  - Directory records are canonical; the projection must be rebuildable and never authoritative.
  - No external module dependencies beyond `contracts` for phase 2; a SQLite adapter is deferred.
  - No raw prompts, source, credentials, or terminal transcripts.

## Scope

### In Scope
- A canonical record model and the portable `.aa-project/` layout with path safety.
- A canonical file store: materialized records plus an append-only event log, atomic writes, hashing.
- A rebuildable projection (reference in-memory implementation) with deterministic digests.
- A deterministic query API and derived work, project, and job histories.
- The issue repository and the strategy repository.
- The memory hierarchy and evidence-based promotion lifecycle.
- Idempotent ingestion, provenance checks, and retention.
- A `memory` facade wiring store, projection, repositories, and query.

### Out of Scope
- Work execution, scheduling, or model calls (kernel/sifter).
- File transfer (sync) and forge operations (forge).
- A SQLite projection adapter (deferred to `ph3-kernel`/follow-up; the projection interface is designed for it).
- Git history reading as a live integration (a git-derived work history reader is deferred; event-derived histories are in scope).
- Multi-writer or two-way history.

### Required End State
- [ ] Records can be written, read, listed, and hashed deterministically.
- [ ] The projection rebuilds to an identical digest after being discarded.
- [ ] Ingestion is idempotent for both records and events.
- [ ] Issues and strategies are created, transitioned, and listed through repositories.
- [ ] Memory lifecycle transitions are enforced.
- [ ] Query and histories are deterministic and contract-typed.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and boundary checks pass.

## Architecture Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P2-001 | Canonical records are JSON files under `.aa-project/records/<kind>/<id>.json`; events are an append-only `events.jsonl` | Portable, git-friendly, human-auditable, and trivially rebuildable |
| ADR-P2-002 | Record identity is `(kind, id)` with a monotonic `revision`; the hash is over canonicalized bytes | Stable identity and deterministic digests |
| ADR-P2-003 | The projection is an interface; the reference implementation is in-memory | Proves rebuild-equivalence without a storage dependency; a SQLite adapter can be added later |
| ADR-P2-004 | Ordering never depends on wall clock; events carry an explicit `sequence` | Determinism and replayability |
| ADR-P2-005 | Repositories are thin typed layers over the store and projection, using `contracts` types | One source of truth for shapes; memory does not redefine contracts |
| ADR-P2-006 | Promotion transitions are an explicit state machine | Fail-closed, evidence-based knowledge promotion |
| ADR-P2-007 | Retention is an explicit floor over the event log | Bounded, auditable retention without rewriting history |

Diagrams: [plan.uml](./plan.uml).

## Slices

Each slice maps to exactly one commit.

### Slice 1 — Phase plan and UML
**Goal** — Record the plan and intended memory infrastructure. **Commit** — `add ph2 memory phase plan`. **Validation** — docs link check.

### Slice 2 — Record model and portable layout
**Goal** — Define `Record`, deterministic hashing, and the `.aa-project/` layout with traversal-safe path construction. **Inputs** — contracts identifiers. **Output** — `memory/store/record.go`, `memory/store/layout.go` + tests. **Commit** — `add memory record model and portable layout`. **Validation** — unit tests for hashing stability and path safety.

### Slice 3 — Canonical store
**Goal** — Read/write records and events with atomic writes; initialize the project manifest. **Output** — `memory/store/store.go` + tests. **Commit** — `add memory canonical store`. **Validation** — round-trip, atomic replace, listing, manifest tests.

### Slice 4 — Rebuildable projection
**Goal** — Define the projection interface and rebuild from the canonical store to an identical digest. **Output** — `memory/projection/*.go` + tests. **Commit** — `add memory rebuildable projection`. **Validation** — rebuild-equivalence test; digest stability.

### Slice 5 — Query API and histories
**Goal** — Deterministic queries and derived work/project/job histories from events. **Output** — `memory/query/*.go` + tests. **Commit** — `add memory query api and histories`. **Validation** — history derivation tests.

### Slice 6 — Issue and strategy repositories
**Goal** — Typed repositories over the store with lifecycle enforcement. **Output** — `memory/repo/*.go` + tests. **Commit** — `add memory repositories and promotion`. **Validation** — create/get/list/transition; illegal transitions rejected.

### Slice 7 — Idempotent ingestion, provenance, retention
**Goal** — Idempotent record/event ingestion, provenance presence checks, and an event retention floor. **Output** — additions to `store` + tests. **Commit** — `add memory ingestion provenance and retention`. **Validation** — duplicate ingestion is a no-op; below-floor events are excluded.

### Slice 8 — Memory facade
**Goal** — One entry point wiring store, projection, repositories, and query. **Output** — `memory/memory.go` + tests. **Commit** — `add memory facade`. **Validation** — end-to-end facade test.

### Slice 9 — Phase summary and UML
**Goal** — Record delivered, affirmed, and deviated. **Output** — `summary.md`, `summary.uml`. **Commit** — `add ph2 memory phase summary`. **Validation** — link check.

## Exit Criteria

- [ ] Records and events persist portably and deterministically.
- [ ] Projection rebuild-equivalence is proven by test.
- [ ] Ingestion is idempotent; retention is enforced.
- [ ] Issue/strategy repositories and promotion are tested.
- [ ] Query and histories are deterministic.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, boundary checks pass.
- [ ] Phase PR merged to `main`; branch retained.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | hashing, path safety, atomic writes, lifecycle transitions, history derivation |
| contract | records decode as `contracts` types; provenance required |
| integration | store -> projection -> query round trip; facade end-to-end |
| boundary | `memory` imports only `contracts`/`obsv` (archtest) |
| determinism | rebuild produces an identical projection digest |
| fail-closed | traversal paths, illegal transitions, and below-floor reads are rejected/excluded |
