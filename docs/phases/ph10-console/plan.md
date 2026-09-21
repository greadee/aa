# Phase 10 — Interface: Plan

> Authored at the **start** of the phase. Branch and folder: `ph10-console` (the unified interface of `aa-console`).
> This phase is split into parts. Read with the part plans:
> [Phase 10a — Terminal](./parts/a-terminal/plan.md) ·
> [Phase 10b — Surface](./parts/b-surface/plan.md).

## Objective

Deliver `aa`'s unified interface. Phase 10 **is the interface**: the real control-plane server and the host composition root — one `aa host` daemon that composes the platform modules and serves the external, versioned HTTP/JSON control plane defined in `contracts/openapi/control-plane-v1.yaml` on loopback with bearer authentication. The console client and its surfaces are delivered as parts:

- **Phase 10a — Terminal:** the thin console client (typed on `contracts`) and its deterministic text/terminal surface that drives the whole workflow.
- **Phase 10b — Surface:** the pluggable `Surface` interface and the shared view-model layer that later surfaces implement.

The console reads and writes only through the control plane; it imports no module internals and calls no inter-module RPC. A future UI/TUI/GUI surface (web, desktop) is out of scope here and tracked by the long-lived UI placeholder PR.

## Why Parts

Phase 10 spans genuinely different workstreams with different dependencies and failure modes:

- the **interface core** — a host-level server and composition root that owns process lifecycle, transport, authentication, and module wiring; and
- the **console surfaces** — a thin client and a deterministic terminal that depend only on `contracts` and the control plane, plus the surface abstraction that future surfaces use.

Planning them as one monolith hid the server behind a fake and made the console's boundary look optional. Splitting the phase keeps the kernel's "reaches `sync`/`forge`/`sifter` over RPC, never by import" rule and the console's "`contracts` + control plane only" rule independently verifiable, and lets the interface core land first so the terminal integrates against a real control plane rather than only a fake.

## Parts

| Part | Scope | Plan | Status |
|---|---|---|---|
| 10a | Terminal: thin console client, typed commands, read models, approvals, configuration, deterministic text/terminal surface, app | [plan](./parts/a-terminal/plan.md) | planned |
| 10b | Surface: pluggable `Surface` interface, shared view-model layer, and the extension contract future surfaces implement | [plan](./parts/b-surface/plan.md) | planned |
| — | UI/TUI/GUI surfaces (local web, desktop) | PR #17 (placeholder) | future |

Part rules:

- A part never creates a phase or renames the branch; all parts share `ph10-console` and the phase PR (#11).
- Each part authors its own `plan.md` and `plan.uml` at its start, and a `summary.md` and `summary.uml` at its end.
- A part's slices map to commits on `ph10-console` (or on a child branch merged into it); a part is complete when its own required end state and exit criteria pass.
- The phase is complete when the interface core and parts 10a and 10b are complete and the phase PR merges to `main`.

## Starting State

- Starting ref: `main @ 79b495d` (after the phase 9 merge and record); phase branch `ph10-console`.
- Available:
  - `contracts` v1 Go bindings and JSON Schemas, the external control-plane specification (`contracts/openapi/control-plane-v1.yaml`): loopback HTTP/JSON, bearer auth, health/projects/work/memory/nodes/tools paths.
  - `kernel/api`: an in-process control-plane service (disabled by default) over the orchestrator, plus status/dispatch/approve.
  - `kernel/orchestrator`, `memory` store/query/repositories, `obsv` runtime, and `toolbox` registry/workflow from phases 2–9.
  - The inter-module RPC v1 spec (`contracts/rpc/rpc-v1.md`) and the transport-agnostic adapter pattern in `forge`, `sync`, and `toolbox`.
  - The `sifter` JSON-RPC service (in-process today) and its production RPC wiring plan on `ph4-sifter`.
  - The `console` scaffold (`doc.go`, `README.md`), its directive `console/aa-tui.md`, and the architecture boundary test (`tools/archtest`) permitting `console -> contracts` only.
- Missing:
  - Any HTTP/JSON control-plane transport, bearer authentication, composition root, host lifecycle, RPC clients, or SSE event stream.
  - The wiring from OpenAPI paths to module services, and the OpenAPI conformance tests.
- Known constraints:
  - `kernel` imports only `contracts`, `obsv`, `memory`, and `toolbox`; it reaches `sync`, `forge`, and `sifter` over RPC, never by import (archtest).
  - `console` imports only `contracts` (archtest D5); it must not import module internals and must not call inter-module RPC.
  - Execution is disabled by default and enabled explicitly.
  - The listener is loopback-only; a bearer token is required (loopback alone is not authentication).
  - The default test suite makes no external network calls; the host is exercised with `httptest` and in-process fakes.
  - Metadata-only: user questions, secrets, prompts, and source are never logged.

## Scope

Phase scope is the union of the part scopes; the parts own the console detail.

### In Scope
- A control-plane HTTP/JSON server on a loopback listener implementing every path in the OpenAPI spec: `/health`, `/status`, `/projects`, `/projects/{projectId}`, `/projects/{projectId}/work-packages`, `/projects/{projectId}/graph`, `/projects/{projectId}/events` (SSE), `/projects/{projectId}/issues`, `/projects/{projectId}/strategies`, `/assignments`, `/approvals/{gateId}`, `/nodes`, `/tools`.
- Bearer authentication, correlation ids, request logging that never records bodies or secrets, and mapping module errors to HTTP status codes and the v1 error codes.
- The host composition root: wiring `kernel`, `memory`, `obsv`, and `toolbox` directly and `sync`, `forge`, and `sifter` through RPC clients; startup order, readiness, and graceful shutdown.
- RPC client seams for `sync`, `forge`, and `sifter` (typed on `contracts`, implemented to the v1 spec) with deterministic fakes for tests and offline use.
- Execution enablement (disabled by default; explicit and recorded) and host configuration.
- OpenAPI conformance tests asserting every path and payload shape.

### Out of Scope
- The console client, commands, views, approvals, configuration, and surfaces (parts 10a/10b).
- The local web and desktop surfaces (future UI, tracked by PR #17).
- Authentication beyond a loopback bearer token; pairing, device keys, and mTLS (phase 11).
- Reimplementing orchestration, memory, or learning; the interface composes and exposes.
- Packaging, installers, and process supervision beyond a foreground serve command.

## Architecture Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P10-001 | The interface is thin: the console depends only on `contracts` and the external control-plane HTTP API | Avoids coupling UX to volatile internals; enforced by `archtest` (D5, D22) |
| ADR-P10-002 | Phase 10 is split into parts — 10a terminal, 10b surface — sharing one branch and one phase PR | Separate workstreams with different dependencies and review needs; independent plans and exit criteria |
| ADR-P10-003 | The external control-plane HTTP/JSON API is the only console interface; the console never calls inter-module RPC | One versioned surface; keeps the RPC surface internal and the console decoupled |
| ADR-P10-004 | The control-plane server is in scope for the phase; the earlier "server deferred, a fake stands in" position is superseded | The console cannot be verified end to end against a fake alone; the phase owns the real server |
| ADR-P10-005 | Deterministic, offline testability: a fake client for surfaces and `httptest` for the server | CI makes no external network calls; renders and responses are reproducible |
| ADR-P10-006 | Every console write is idempotent and audited; approvals and gates require an explicit decision | Repeated commands must not duplicate work; human consent is never auto-resolved (D18) |
| ADR-P10-007 | The host daemon is built from the `kernel` module (`kernel/cmd/aad`), using the kernel's allowed imports and RPC clients for `sync`/`forge`/`sifter` | Respects the existing boundary (`kernel` must not import those modules) and needs no new broad-import module or archtest exception; the host is the kernel daemon (architecture §4.3) |
| ADR-P10-008 | The transport is loopback HTTP/JSON per the OpenAPI spec; the inter-module RPC surface stays internal | One external versioned surface for humans and the visualizer; the RPC surface is not exposed |
| ADR-P10-009 | Bearer authentication is required on every path except `/health`; the token is generated per host and stored owner-only | Loopback alone is not authentication; least privilege on a shared machine |
| ADR-P10-010 | Execution is disabled by default; enabling is explicit, recorded, and reversible | Security boundary inherited from the kernel (`api.ErrDisabled`) |
| ADR-P10-011 | `/events` is server-sent events ordered by sequence; clients resume with `after` | Replayable, deterministic event delivery without a wall-clock dependency |
| ADR-P10-012 | RPC clients for `sync`/`forge`/`sifter` are typed on `contracts` and implemented to the v1 spec; deterministic fakes stand in for tests and offline use | The kernel's provider-agnostic seam; testable without network or a running sidecar |
| ADR-P10-013 | The composition root owns startup order, readiness, and graceful shutdown, and is the only place that knows module wiring | One place to reason about wiring; modules keep no knowledge of each other |
| ADR-P10-014 | The OpenAPI document is the contract; handler tests assert path and shape conformance | Prevents silent drift between the spec and the server |

Console- and surface-specific decisions are recorded in the part plans as `ADR-P10A-*` (terminal) and `ADR-P10B-*` (surface).

Diagrams: [plan.uml](./plan.uml).

## Part Dependencies

```text
Phase 10 — interface core (control-plane server + host composition)
    -> 10a — Terminal (thin console; integrates against the real control plane)
        -> 10b — Surface (pluggable Surface abstraction + shared view-model layer)
            -> future UI/TUI/GUI surfaces (PR #17)
```

Part 10a may begin against the deterministic fake and `httptest` before the interface core lands; its end-to-end exit criterion requires the real server. Part 10b generalizes the surface seam proven by 10a and is the extension point future surfaces implement.

## Slices

Each slice is independently validatable and maps to exactly one commit on `ph10-console`. The slices below deliver the interface core; see the part plans for 10a and 10b.

### Slice 1 — Phase plan and UML
**Goal** — Plan the phase (interface core and parts 10a/10b). **Expected Output** — this `plan.md` + `plan.uml` + the part plans. **Commit Message** — `add ph10 console interface plan`. **Validation** — docs link check. **Dependencies** — none.

### Slice 2 — Server core
**Goal** — Router, middleware, bearer auth, correlation ids, and error mapping to HTTP/status and v1 error codes. **Expected Output** — `kernel/controlplane/*.go` + tests. **Commit Message** — `add console control plane server core`. **Validation** — auth, method-not-allowed, error-mapping tests.

### Slice 3 — Health, status, and readiness
**Goal** — `/health` (unauthenticated liveness and versions) and `/status` (aggregate host status). **Expected Output** — handlers + tests. **Commit Message** — `add console control plane health and status`. **Validation** — health is open; status requires auth; shape conformance.

### Slice 4 — Projects
**Goal** — `/projects` list/register and `/projects/{projectId}` read, backed by `memory`. **Expected Output** — handlers + tests. **Commit Message** — `add console control plane projects`. **Validation** — create/get/list, 404, 409, idempotent register.

### Slice 5 — Work reads
**Goal** — `/projects/{projectId}/work-packages`, `/projects/{projectId}/graph`, and `/assignments`. **Expected Output** — handlers + tests. **Commit Message** — `add console control plane work reads`. **Validation** — ordering by id/sequence; empty and partial data.

### Slice 6 — Events stream
**Goal** — `/projects/{projectId}/events` as server-sent events with an `after` cursor. **Expected Output** — handler + tests. **Commit Message** — `add console control plane events stream`. **Validation** — ordered delivery, resume, client disconnect.

### Slice 7 — Memory reads
**Goal** — `/projects/{projectId}/issues` and `/projects/{projectId}/strategies`. **Expected Output** — handlers + tests. **Commit Message** — `add console control plane memory reads`. **Validation** — shape and ordering conformance.

### Slice 8 — Approvals
**Goal** — `/approvals/{gateId}` POST with an explicit `decision`/`actor`, wired to the kernel human gate; never auto-resolve. **Expected Output** — handler + tests. **Commit Message** — `add console control plane approvals`. **Validation** — approve/reject, 403, idempotent repeat.

### Slice 9 — Nodes and tools
**Goal** — `/nodes` and `/tools` read models. **Expected Output** — handlers + tests. **Commit Message** — `add console control plane nodes and tools`. **Validation** — shape conformance; fakes.

### Slice 10 — RPC clients
**Goal** — Typed `sync`/`forge`/`sifter` clients over the v1 envelope with deterministic fakes and error mapping. **Expected Output** — `kernel/rpcclient/*.go` + tests. **Commit Message** — `add console host rpc clients`. **Validation** — envelope, timeout, retryable, and fake tests.

### Slice 11 — Composition root and lifecycle
**Goal** — The `aa host` daemon: wiring, startup order, readiness, graceful shutdown, and a foreground `aad serve` command. **Expected Output** — `kernel/cmd/aad/main.go`, composition package + tests. **Commit Message** — `add aa host composition and lifecycle`. **Validation** — start/stop, auth token generation, boundary check.

### Slice 12 — Execution enablement and configuration
**Goal** — Disabled-by-default execution with an explicit enable operation, and host configuration. **Expected Output** — config + tests. **Commit Message** — `add aa host execution config`. **Validation** — disabled refuses dispatch; enable is recorded and reversible.

### Slice 13 — OpenAPI conformance tests
**Goal** — Assert every OpenAPI path and payload against the running handler. **Expected Output** — conformance tests. **Commit Message** — `add aa host openapi conformance tests`. **Validation** — path coverage; schema shape.

### Slice 14 — Phase summary and UML
**Goal** — Record the as-built interface core. **Expected Output** — `summary.md` + `summary.uml`. **Commit Message** — `add ph10 console interface summary`. **Validation** — link check.

## Required End State

The phase is complete when the interface core and parts 10a and 10b are complete. At phase level:

- [ ] The server serves every OpenAPI path on a loopback listener with bearer auth (interface core).
- [ ] Bearer auth is enforced on every path except `/health`; the token is never logged.
- [ ] Execution is disabled unless explicitly enabled.
- [ ] `/events` streams ordered events and resumes from `after`.
- [ ] The composition root wires `kernel`, `memory`, `obsv`, `toolbox`, and the `sync`/`forge`/`sifter` clients; archtest stays green.
- [ ] A deterministic terminal (10a) drives the whole workflow through the control plane.
- [ ] The pluggable `Surface` abstraction and shared view-model layer exist for future surfaces (10b).
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and boundary checks pass.

## Exit Criteria

- [ ] The interface core, part 10a, and part 10b meet their own exit criteria.
- [ ] Every OpenAPI path is implemented and conformance-tested.
- [ ] One scripted run drives register → submit → browse → approve → configure through a surface against the real control plane.
- [ ] `kernel` still imports only `contracts`/`obsv`/`memory`/`toolbox` (archtest); `console` imports only `contracts`.
- [ ] The phase PR is reviewed and merged to `main`; the branch is retained.

## Test Plan

Phase-level; each part adds detail.

| Layer | What is tested |
|---|---|
| contract | OpenAPI path/shape conformance; `contracts` v1 payloads; error-code mapping |
| integration | `httptest` server + client round-trips; surface end-to-end flow against the real control plane |
| boundary | `console` imports only `contracts`; `kernel` does not import `sync`/`forge`/`sifter` (archtest) |
| determinism | event and list ordering by sequence/id, never wall clock; golden renders |
| security | bearer token supplied but never logged; approvals explicit; metadata-only rendering; no secrets |
| accessibility | stable, keyboard-navigable text surface; semantic surface when a future GUI lands |
