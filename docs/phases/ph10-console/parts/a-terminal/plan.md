# Phase 10a — Terminal: Plan

> Part of [Phase 10 — Interface](../../plan.md). Branch: `ph10-console`; this plan lives at `docs/phases/ph10-console/parts/a-terminal/plan.md`.
> Authored at the **start** of part 10a, before implementation.

## Objective

Deliver the thin console as a working terminal: a control-plane HTTP client typed on `contracts` with a deterministic fake, pure deterministic view models, typed idempotent audited commands, explicit approvals, workflow/tool configuration as data, and a deterministic text/terminal surface with a runnable entry point that drives the whole workflow (register → submit → browse → approve → configure). The console depends only on `contracts` and the control plane; it imports no module internals and calls no inter-module RPC.

The pluggable `Surface` abstraction and the shared view-model layer it consumes are generalized in [Phase 10b — Surface](../b-surface/plan.md); this part proves the terminal and the seam 10b builds on.

## Starting State

- Starting ref: `ph10-console @ <sha>` (after the phase plan, or in parallel against the OpenAPI and fakes).
- Available:
  - Phase 10's control-plane server (or the OpenAPI spec plus an `httptest` server) exposing the paths the console needs.
  - `contracts` v1 Go bindings and the OpenAPI control-plane specification (`contracts/openapi/control-plane-v1.yaml`).
  - The `console` scaffold (`doc.go`, `README.md`) and its directive `console/aa-tui.md`.
  - The architecture boundary test (`tools/archtest`) permitting `console -> contracts` only.
- Missing:
  - Any console implementation beyond `doc.go`: client, commands, view models, approvals, configuration, terminal surface, and application entry point.
- Known constraints:
  - `console` imports only `contracts` (archtest D5); no module internals and no inter-module RPC.
  - The default test suite makes no external network calls and is deterministic; ordering derives from ids and sequences, never the wall clock.
  - Every write is idempotent (keyed by the command contract id) and audited; approvals and gates are explicit and never auto-resolved.
  - Observation and display are metadata-only; secrets, raw prompts, and source are never rendered or logged.
  - Go standard library only in this part: no Wails/React/desktop surface, no heavyweight UI dependencies, and no generated TypeScript yet.

## Scope

### In Scope
- A control-plane client seam: an HTTP/JSON client typed on `contracts`, a deterministic in-memory fake, and an `httptest`-backed integration test against the OpenAPI paths.
- Typed commands (register project, submit work, approve/reject a gate, update workflow/tool configuration) with validation, idempotency keys, and an audit trail.
- Read models / browsing: projects, work packages, task graph, issues, strategies, assignments, nodes, tools, and the phase 9 learning/backtest reports as data.
- Approval and gate interactions that state consequences explicitly and require an explicit decision.
- Workflow configuration and tool/plugin/MCP management as data, validated before write.
- A deterministic text (TUI) surface with stable, golden-testable output.
- An application entry point that wires a chosen surface to a client and runs offline against the fake.

### Out of Scope
- The shared `Surface` abstraction and view-model layer formalized for other surfaces (part 10b).
- The local web and desktop surfaces (future UI, tracked by PR #17).
- The control-plane server and composition root (Phase 10 core).
- Generated TypeScript bindings and any browser tooling/build pipeline.
- Authentication/authorization beyond carrying a loopback bearer token (phase 11).
- Reimplementing orchestration, memory, or learning logic; the console only presents and commands.

## Architecture Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P10A-001 | The console is thin: it depends only on `contracts` and the control-plane HTTP API | Avoids coupling UX to volatile internals; enforced by archtest (D5, D22) |
| ADR-P10A-002 | The control-plane client is a seam with a deterministic fake and an `httptest` server | Offline, deterministic tests and an offline-capable console without a running host |
| ADR-P10A-003 | Views are pure, deterministic projections of client responses | Golden-testable rendering; ordering from ids/sequences, never wall clock |
| ADR-P10A-004 | Every write is idempotent (keyed by the command contract id) and audited | Repeated commands must not duplicate work; the control plane records intent |
| ADR-P10A-005 | Approvals and gates state consequences and require an explicit decision | Human consent is runtime-enforced and never auto-resolved (D18) |
| ADR-P10A-006 | Workflow and tool/plugin configuration are data, edited through the control plane | Configurability without console code changes (D16, D17) |
| ADR-P10A-007 | The text surface is stable and golden-tested; no desktop/Wails and no generated TS in this part | Deterministic verification now; part 10b preserves the extension points |

Diagrams: [plan.uml](./plan.uml).

## Slices

Each slice is independently validatable and maps to exactly one commit on `ph10-console`.

### Slice 1 — Part plan and UML
**Goal** — Plan part 10a and its intended infrastructure. **Expected Output** — this `plan.md` + `plan.uml`. **Commit Message** — `add ph10 console 10a terminal plan`. **Validation** — docs link check. **Dependencies** — phase plan.

### Slice 2 — Control-plane client
**Goal** — An HTTP/JSON client typed on `contracts`, a transport seam, a deterministic fake, and an `httptest` integration test against the OpenAPI paths. **Expected Output** — `console/client/*.go` + tests. **Commit Message** — `add console control plane client`. **Validation** — request/response conformance, auth header, error mapping, offline fake.

### Slice 3 — Command model
**Goal** — Typed commands (register project, submit work, approve/reject, update config) with validation, idempotency keys, and an audit record routed through the client. **Expected Output** — `console/command/*.go` + tests. **Commit Message** — `add console command model`. **Validation** — validation, idempotency, and audit tests.

### Slice 4 — Read models and browsing
**Goal** — Deterministic view models for projects, work packages, task graph, issues, strategies, assignments, nodes, tools, and learning reports. **Expected Output** — `console/view/*.go` + tests. **Commit Message** — `add console read models`. **Validation** — ordering, empty, and partial-response tests.

### Slice 5 — Approvals and gates
**Goal** — Explicit approval/gate interaction: consequences are rendered, decisions are explicit, and nothing is auto-resolved. **Expected Output** — `console/approval/*.go` + tests. **Commit Message** — `add console approvals and gates`. **Validation** — consequence, explicit-decision, and no-auto-resolve tests.

### Slice 6 — Workflow and tool configuration
**Goal** — Browse and edit workflows and tool/plugin/MCP manifests as data through the control plane, with validation before write. **Expected Output** — `console/config/*.go` + tests. **Commit Message** — `add console workflow and tool configuration`. **Validation** — validation, round-trip, and idempotency tests.

### Slice 7 — Text (TUI) surface
**Goal** — A deterministic text surface rendering the view models and driving commands, with stable golden output. **Expected Output** — `console/tui/*.go` + tests. **Commit Message** — `add console tui surface`. **Validation** — golden render, input handling, and no-network tests.

### Slice 8 — Application and end-to-end flow
**Goal** — Wire a chosen surface to a client and a runnable entry point, and prove the surface can drive the whole workflow offline. **Expected Output** — `console/app/*.go`, `console/cmd/aa/main.go` + tests. **Commit Message** — `add console app and end to end flow`. **Validation** — scripted end-to-end flow; surface-over-fake test.

### Slice 9 — Part summary and UML
**Goal** — Record the as-built part. **Expected Output** — `summary.md` + `summary.uml` + updated `console/README.md`. **Commit Message** — `add ph10 console 10a terminal summary`. **Validation** — link check.

## Required End State

- [ ] A control-plane client typed on `contracts` with a deterministic fake and an `httptest` integration test.
- [ ] Read models for projects, work, graph, issues, strategies, assignments, nodes, tools, and learning reports.
- [ ] Typed commands with validation, idempotent writes, and an audit trail.
- [ ] Approvals and gates state consequences and require explicit decisions; nothing is auto-resolved.
- [ ] Workflow and tool/plugin/MCP configuration are edited as data through the control plane.
- [ ] A deterministic text (TUI) surface drives the whole workflow; output is golden-tested.
- [ ] The console imports only `contracts`; no module internals and no inter-module RPC.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and boundary checks pass.

## Exit Criteria

- [ ] The text surface drives register → submit → browse → approve → configure against the control plane.
- [ ] Writes are idempotent and audited; approvals are explicit.
- [ ] Golden renders are deterministic; the default suite makes no network calls.
- [ ] The seam is ready for part 10b to generalize into a pluggable `Surface` abstraction.
- [ ] Part summary and UML authored.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | view models, command validation, idempotency, approvals, config rendering, surface input handling |
| contract | control-plane request/response payloads conform to `contracts` v1 and the OpenAPI paths; error-code mapping |
| integration | client against an in-process `httptest` control-plane server; the text surface over one fake client |
| boundary | `console` imports only `contracts` (archtest); no module internals, no inter-module RPC |
| e2e | a scripted run drives the whole workflow from the text surface (register → submit → browse → approve → configure) |
| security | bearer token is supplied but never logged; approvals explicit; metadata-only rendering; no secrets |
| determinism | golden renders and view orderings independent of input order and wall clock |
