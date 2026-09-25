# Phase 10b — Surface: Plan

> Part of [Phase 10 — Interface](../../plan.md). Branch: `ph10-console`; this plan lives at `docs/phases/ph10-console/parts/b-surface/plan.md`.
> Read with [Phase 10a — Terminal](../a-terminal/plan.md). Authored at the **start** of part 10b, before implementation.

## Objective

Generalize the console into a pluggable surface architecture. Part 10b extracts the shared, surface-agnostic view-model layer proven by the terminal in [Phase 10a](../a-terminal/plan.md), defines the public `Surface` interface, and adds the surface registration, selection, and conformance contract that future surfaces implement. After this part a new UI/TUI/GUI surface can be added by implementing one interface against the shared view models, without changing the client, commands, approvals, or configuration. This is the extension point future UI work (tracked by PR #17) builds on.

## Starting State

- Starting ref: `ph10-console @ <sha>` (after part 10a).
- Available:
  - Part 10a's console: control-plane client, typed commands, read models, approvals, configuration, deterministic text/terminal surface, and application entry point.
  - `contracts` v1 bindings and the OpenAPI control-plane specification.
  - The architecture boundary test (`tools/archtest`) permitting `console -> contracts` only.
- Missing:
  - A published `Surface` interface; the terminal currently consumes its view models directly.
  - A shared view-model layer that is not specific to the text surface.
  - Surface registration/selection and a conformance contract for new surfaces.
- Known constraints:
  - `console` imports only `contracts` (archtest D5); no module internals and no inter-module RPC.
  - Surfaces are views, not logic: no surface may call the control plane directly or bypass commands/approvals.
  - The default test suite makes no external network calls and is deterministic; ordering derives from ids and sequences, never the wall clock.
  - Metadata-only: secrets, raw prompts, and source are never rendered or logged.
  - Go standard library only: the abstraction must not pull in a UI framework; a future GUI adds its own dependencies on its own branch (PR #17).

## Scope

### In Scope
- The shared, surface-agnostic view-model layer extracted from part 10a.
- The public `Surface` interface (render a view model, receive an intent/decision).
- Surface registration and explicit, deterministic selection.
- A conformance test-kit that any surface must pass against the shared view models.
- Refactoring the terminal from 10a onto the shared layer with no behavior change.
- The documented extension contract and guidance for future UI/TUI/GUI surfaces.

### Out of Scope
- Any concrete new surface (local web, desktop); those are future UI work (PR #17).
- The control-plane server and composition root (Phase 10 core).
- Changes to the client, commands, approvals, or configuration semantics; 10b refactors the rendering seam only.
- Generated TypeScript bindings and browser tooling.
- Authentication/authorization beyond carrying a loopback bearer token (phase 11).

## Architecture Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P10B-001 | Surfaces are pluggable behind one `Surface` interface | New surfaces must be addable without changing other modules (D22) |
| ADR-P10B-002 | One shared view-model layer renders to every surface; surfaces are views, not logic | Surfaces must not diverge, and no surface may bypass commands, approvals, or the control-plane client |
| ADR-P10B-003 | The surface extension contract is documented and conformance-tested | A new surface is correct by construction, not by convention |
| ADR-P10B-004 | Surface selection is explicit and deterministic | No hidden global registry; the chosen surface is configuration, reproducible in tests |
| ADR-P10B-005 | The terminal from part 10a is refactored onto the shared layer with no behavior change | Proves the abstraction against a real surface and keeps one rendering path |
| ADR-P10B-006 | The abstraction stays standard-library only | Keeps `console` dependency-free; a GUI adds its own framework on its own branch (PR #17) |

Diagrams: [plan.uml](./plan.uml).

## Slices

Each slice is independently validatable and maps to exactly one commit on `ph10-console`.

### Slice 1 — Part plan and UML
**Goal** — Plan part 10b and its intended infrastructure. **Expected Output** — this `plan.md` + `plan.uml`. **Commit Message** — `add ph10 console 10b surface plan`. **Validation** — docs link check. **Dependencies** — part 10a.

### Slice 2 — Shared view-model layer
**Goal** — Extract the surface-agnostic view models and intents from part 10a into one shared layer. **Expected Output** — `console/viewmodel/*.go` + tests. **Commit Message** — `add console shared view models`. **Validation** — deterministic projection and ordering tests.

### Slice 3 — Surface interface
**Goal** — The public `Surface` interface: render a view model and receive an explicit intent/decision, with no access to the control plane. **Expected Output** — `console/surface/*.go` + tests. **Commit Message** — `add console surface interface`. **Validation** — interface contract and no-bypass tests.

### Slice 4 — Surface registry and selection
**Goal** — Explicit registration and deterministic selection of a surface from configuration. **Expected Output** — `console/surface/registry.go` + tests. **Commit Message** — `add console surface registry`. **Validation** — selection determinism and unknown-surface errors.

### Slice 5 — Terminal on the shared surface layer
**Goal** — Refactor the 10a terminal to implement `Surface` over the shared view models with no behavior change. **Expected Output** — `console/tui/*.go` updated + tests. **Commit Message** — `add console terminal on shared surface`. **Validation** — golden renders unchanged.

### Slice 6 — Surface conformance test-kit
**Goal** — A reusable kit asserting any surface against the shared view models and intents. **Expected Output** — `console/surface/conformance` + tests. **Commit Message** — `add console surface conformance kit`. **Validation** — kit passes for the terminal; a stub surface demonstrates the failure modes.

### Slice 7 — Extension contract and documentation
**Goal** — Document the surface extension contract and record how future UI/TUI/GUI surfaces plug in. **Expected Output** — `console/README.md`, `console/aa-tui.md` updates + docs. **Commit Message** — `add console surface extension docs`. **Validation** — link check; boundary check.

### Slice 8 — Part summary and UML
**Goal** — Record the as-built part. **Expected Output** — `summary.md` + `summary.uml`. **Commit Message** — `add ph10 console 10b surface summary`. **Validation** — link check.

## Required End State

- [ ] A shared, surface-agnostic view-model layer used by the terminal.
- [ ] A public `Surface` interface with no control-plane access.
- [ ] Explicit, deterministic surface registration and selection.
- [ ] A conformance test-kit that validates a surface against the shared view models.
- [ ] The terminal from part 10a implements `Surface` with unchanged golden output.
- [ ] The extension contract is documented for future surfaces.
- [ ] `console` still imports only `contracts`; boundary checks pass.
- [ ] `go build`, `go vet`, `go test`, and `gofmt` pass.

## Exit Criteria

- [ ] The terminal is one implementation of `Surface`; another surface can be added without touching the client, commands, approvals, or configuration.
- [ ] The conformance kit passes for the terminal and fails a non-conforming stub.
- [ ] Golden renders are unchanged from part 10a.
- [ ] `console` imports only `contracts` (archtest).
- [ ] Part summary and UML authored.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | shared view models, intent/decision handling, registry selection |
| contract | the `Surface` interface conformance kit; no surface can reach the control plane |
| integration | the terminal re-run over one fake client with unchanged golden output |
| boundary | `console` imports only `contracts` (archtest); surfaces are views, not logic |
| determinism | identical inputs yield identical view models and renders regardless of order |
| docs | link check |
