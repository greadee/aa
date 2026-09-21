# Phase 10 — Console: Plan

> Authored at the **start** of the phase. Branch and folder: `ph10-console`.

## Objective

Deliver `aa`'s unified interface: one thin entry point through which a human can drive and inspect the whole workflow. The console reads and writes only through the external, versioned HTTP/JSON control plane defined in `contracts/openapi/control-plane-v1.yaml`; it never imports module internals and never calls inter-module RPC directly. Deliver a pluggable **surface** interface with a deterministic text (TUI) surface and a local web surface over the same view models, typed commands with idempotent, audited writes, read models for projects, work, memory, nodes, and tools, explicit approval/gate interactions, and workflow/tool configuration as data. The entire console is testable offline against a deterministic fake control-plane client, with no model calls and no desktop surface.

## Starting State

- Starting ref: `main @ 79b495d` (after phase 9 merge and record).
- Available:
  - `contracts` v1 Go bindings and JSON Schemas for every cross-module object the console displays or mutates (`ProjectRecord`, `WorkPackage`, `TaskGraph`, `Issue`, `Strategy`, `MemoryRecord`, `ToolManifest`, `Workflow`, `Event`, `Telemetry`).
  - The external control-plane specification (`contracts/openapi/control-plane-v1.yaml`): loopback HTTP/JSON, bearer auth, health/projects/work/memory/nodes/tools paths.
  - The inter-module RPC v1 spec (`contracts/rpc/rpc-v1.md`) and the transport-agnostic adapter pattern used by `forge`, `sync`, and `toolbox` (envelope, error codes, idempotency) as a design precedent — not for direct use by the console.
  - `kernel/api`: an in-process control-plane service (disabled by default), whose doc comment records that the transport "lands with the console".
  - Phase 9 deterministic data reports (gate decisions, backtest reports) and the visualizer browse seam as inputs to surface.
  - The `console` scaffold (`doc.go`, `README.md`), its directive `console/aa-tui.md`, and the architecture boundary test (`tools/archtest`) that permits `console -> contracts` only.
- Missing:
  - Any console implementation beyond `doc.go`.
  - A control-plane client and its transport seam.
  - A `Surface` interface and any surface (TUI or web).
  - Read models/browsing, typed commands, approvals, and configuration.
  - A runnable entry point that wires a surface to the control plane.
- Known constraints:
  - `console` imports only `contracts` (archtest D5). It must not import `kernel`, `memory`, `visualizer`, or any other module internals.
  - The control plane is reached over loopback HTTP/JSON with a bearer token; loopback alone is not authentication. The server side is a composition concern and does not exist yet, so a deterministic fake client and an in-process `httptest` server stand in.
  - The default test suite makes no external network calls and is deterministic; ordering derives from ids and sequences, never the wall clock.
  - Go standard library only in this phase: no Wails/React/desktop surface, no heavyweight UI dependencies, and no generated TypeScript yet.
  - Every write is idempotent (keyed by the command contract id) and audited; approvals and gates are explicit and never auto-resolved.
  - Observation and display are metadata-only; secrets, raw prompts, and source are never rendered or logged.
  - The phase branch name equals the phase folder name (`ph10-console`) and the phase PR exists as a Draft from phase start.

## Scope

### In Scope
- A pluggable `Surface` interface implemented by a deterministic text (TUI) surface and a local web surface, sharing one view-model layer.
- A control-plane client seam: an HTTP/JSON client typed on `contracts`, a deterministic in-memory fake, and a `httptest`-backed integration test against the OpenAPI paths.
- Typed commands (register project, submit work, approve/reject a gate, update workflow/tool configuration) with validation, idempotency keys, and an audit trail.
- Read models / browsing: projects, work packages, task graph, issues, strategies, assignments, nodes, and tools, plus the phase 9 learning/backtest reports as data.
- Approval and gate interactions that state consequences explicitly and require an explicit decision.
- Workflow configuration and tool/plugin/MCP management as data (compiled through the control plane), never code.
- A deterministic text surface with stable, golden-testable output and an accessibility-first local web surface (semantic HTML + plain-text alternative).
- An application entry point that wires a chosen surface (TUI or web) to a client and runs offline against the fake.

### Out of Scope
- The Wails/React/Three desktop surface; the `Surface` interface keeps it addable.
- The authenticated local-socket (JSON-RPC) transport and the control-plane **server** implementation and its composition root; a fake client and `httptest` stand in.
- Generated TypeScript bindings and any browser tooling/build pipeline.
- Authentication/authorization beyond carrying a loopback bearer token; pairing, device keys, and mTLS land in phase 11.
- Reimplementing orchestration, memory, or learning logic; the console only presents and commands.
- Long-running hosted processes, installers, and packaging.

### Required End State
- [ ] A `Surface` interface with a deterministic text surface and a local web surface over one view-model layer.
- [ ] A control-plane client typed on `contracts` with a deterministic fake and an `httptest` integration test.
- [ ] Read models for projects, work, graph, issues, strategies, assignments, nodes, tools, and learning reports.
- [ ] Typed commands with validation, idempotent writes, and an audit trail.
- [ ] Approvals and gates state consequences and require explicit decisions; nothing is auto-resolved.
- [ ] Workflow and tool/plugin/MCP configuration are edited as data through the control plane.
- [ ] The text surface output is deterministic and covered by golden tests; the web surface is accessible (semantic HTML, text alternative).
- [ ] The console imports only `contracts`; no module internals and no inter-module RPC.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and boundary checks pass.

## Architecture Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P10-001 | The console is thin: it depends only on `contracts` and the external control-plane HTTP API | Avoids coupling UX to volatile internals; enforced by `archtest` (D5, D22) |
| ADR-P10-002 | Surfaces are pluggable behind one `Surface` interface | A desktop surface must be addable without changing other modules (D22) |
| ADR-P10-003 | The control-plane client is a seam with a deterministic fake | Offline, deterministic tests and an offline-capable console without a running host |
| ADR-P10-004 | Views are pure, deterministic projections of client responses | Golden-testable rendering; ordering from ids/sequences, never wall clock |
| ADR-P10-005 | Every write is idempotent and audited | Repeated commands must not duplicate work; the control plane records intent |
| ADR-P10-006 | Approvals and gates state consequences and require an explicit decision | Human consent is runtime-enforced and never auto-resolved (D18) |
| ADR-P10-007 | One view-model layer renders to both surfaces | The TUI and web must not diverge; surfaces are views, not logic |
| ADR-P10-008 | Workflow and tool/plugin configuration are data, edited through the control plane | Configurability without console code changes (D16, D17) |
| ADR-P10-009 | The text surface is stable and golden-tested; the web surface is accessibility-first | Deterministic verification now; usable by keyboard/screen readers |
| ADR-P10-010 | No desktop/Wails surface and no generated TS in this phase | Keeps the surface thin; the interface preserves the extension point |

Diagrams: [plan.uml](./plan.uml).

## Slices

Each slice is independently validatable and maps to exactly one commit.

### Slice 1 — Phase plan and UML
**Goal** — Plan the phase and its intended infrastructure. **Inputs** — architecture, `contracts` (schemas + OpenAPI), the console directive, prior phase reports. **Expected Output** — `docs/phases/ph10-console/plan.md` + `plan.uml`. **Model Class** — Strong. **Commit Message** — `add ph10 console phase plan`. **Validation** — docs link check. **Dependencies** — none.

### Slice 2 — Core types and errors
**Goal** — The `Surface` interface, shared view/command identifiers, sentinel errors, and a deterministic clock. **Inputs** — slice 1, `contracts` v1. **Expected Output** — `console/types.go`, `console/surface.go`, `console/errors.go`, `console/clock.go` + tests. **Model Class** — General. **Commit Message** — `add console core types and errors`. **Validation** — `go test ./...`. **Dependencies** — slice 1. **Documentation** — `console/README.md`.

### Slice 3 — Control-plane client
**Goal** — An HTTP/JSON control-plane client typed on `contracts`, a transport seam, a deterministic fake, and an `httptest` integration test against the OpenAPI paths. **Inputs** — slices 1–2, `contracts/openapi/control-plane-v1.yaml`. **Expected Output** — `console/client/*.go` + tests. **Model Class** — Strong. **Commit Message** — `add console control-plane client`. **Validation** — `go test ./...`; request/response conformance, auth header, error mapping, offline fake. **Dependencies** — slice 2. **Documentation** — `console/README.md`.

### Slice 4 — Command model
**Goal** — Typed commands (register project, submit work, approve/reject, update config) with validation, idempotency keys, and an audit record routed through the client. **Inputs** — slice 3. **Expected Output** — `console/command/*.go` + tests. **Model Class** — Strong. **Commit Message** — `add console command model`. **Validation** — `go test ./...`; validation, idempotency, and audit tests. **Dependencies** — slice 3. **Documentation** — `console/README.md`.

### Slice 5 — Read models and browsing
**Goal** — Deterministic view models for projects, work packages, task graph, issues, strategies, assignments, nodes, tools, and learning reports. **Inputs** — slice 3. **Expected Output** — `console/view/*.go` + tests. **Model Class** — Strong. **Commit Message** — `add console read models`. **Validation** — `go test ./...`; ordering, empty, and partial-response tests. **Dependencies** — slice 3. **Documentation** — `console/README.md`.

### Slice 6 — Approvals and gates
**Goal** — Explicit approval/gate interaction: consequences are rendered, decisions are explicit, and nothing is auto-resolved. **Inputs** — slices 2–4. **Expected Output** — `console/approval/*.go` + tests. **Model Class** — Strong. **Commit Message** — `add console approvals and gates`. **Validation** — `go test ./...`; consequence, explicit-decision, and no-auto-resolve tests. **Dependencies** — slice 4. **Documentation** — `console/README.md`.

### Slice 7 — Workflow and tool configuration
**Goal** — Browse and edit workflows and tool/plugin/MCP manifests as data through the control plane, with validation before write. **Inputs** — slices 3–5. **Expected Output** — `console/config/*.go` + tests. **Model Class** — General. **Commit Message** — `add console workflow and tool configuration`. **Validation** — `go test ./...`; validation, round-trip, and idempotency tests. **Dependencies** — slice 5. **Documentation** — `console/README.md`.

### Slice 8 — Text (TUI) surface
**Goal** — A deterministic text surface rendering the view models and driving commands, with stable golden output. **Inputs** — slices 5–7. **Expected Output** — `console/tui/*.go` + tests. **Model Class** — Strong. **Commit Message** — `add console tui surface`. **Validation** — `go test ./...`; golden render, input handling, and no-network tests. **Dependencies** — slice 7. **Documentation** — `console/README.md`.

### Slice 9 — Local web surface
**Goal** — A standard-library HTTP surface serving the same view models as semantic HTML (with a plain-text/JSON alternative) and the same commands. **Inputs** — slices 5–7. **Expected Output** — `console/web/*.go` + tests. **Model Class** — Strong. **Commit Message** — `add console web surface`. **Validation** — `go test ./...`; handler, accessibility, and parity-with-TUI tests. **Dependencies** — slice 7. **Documentation** — `console/README.md`.

### Slice 10 — Application and end-to-end flow
**Goal** — Wire a chosen surface to a client and a runnable entry point, and prove one surface can drive the whole workflow offline. **Inputs** — slices 3, 8, 9. **Expected Output** — `console/app/*.go`, `console/cmd/aa/main.go` + tests. **Model Class** — Strong. **Commit Message** — `add console app and end-to-end flow`. **Validation** — `go test ./...`; scripted end-to-end flow and surface-parity tests. **Dependencies** — slice 9. **Documentation** — `console/README.md`.

### Slice 11 — Phase summary and UML
**Goal** — Record the as-built phase. **Inputs** — all prior slices. **Expected Output** — `summary.md` + `summary.uml` + updated `console/README.md`. **Model Class** — Strong. **Commit Message** — `add ph10 console phase summary`. **Validation** — link check. **Dependencies** — slice 10.

## Exit Criteria

- [ ] A pluggable `Surface` interface with a text surface and a local web surface over one view-model layer.
- [ ] A control-plane client typed on `contracts` with a deterministic fake and an `httptest` integration test.
- [ ] Read models cover projects, work, graph, memory, nodes, tools, and learning reports.
- [ ] Commands are validated, idempotent, and audited.
- [ ] Approvals and gates state consequences, require explicit decisions, and never auto-resolve.
- [ ] Workflow and tool configuration are edited as data.
- [ ] Rendering is deterministic and golden-tested; the web surface is accessible.
- [ ] The console imports only `contracts`; no module internals and no inter-module RPC.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and boundary checks pass.
- [ ] Phase PR merged to `main`; branch retained.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | view models, command validation, idempotency, approvals, config rendering, surface input handling |
| contract | control-plane request/response payloads conform to `contracts` v1 and the OpenAPI paths; error-code mapping |
| integration | client against an in-process `httptest` control-plane server; TUI and web surfaces over one fake client |
| boundary | `console` imports only `contracts` (archtest); no module internals, no inter-module RPC |
| e2e | a scripted run drives the whole workflow from one surface (register → submit → browse → approve → configure) |
| security | bearer token is supplied but never logged; approvals explicit; metadata-only rendering; no secrets |
| accessibility | semantic HTML with a plain-text/JSON alternative; stable, keyboard-navigable text surface |
| determinism | golden renders and view orderings independent of input order and wall clock |
