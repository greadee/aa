# Phase 10b — Console — Plan

> **Type:** phase (`phase10b-console`). **Branch:** `phase10b-console` → `main`.
> **Base:** `main @ 8d2295a`. **Predecessor:** `phase10a-controlplane` (the server this client talks to).
> **Diagram:** [plan.uml](./plan.uml).

## Objective

Deliver `aa`'s **unified interface**: one thin entry point through which a human drives and inspects
the whole workflow. The console reads and writes **only** through the external versioned HTTP/JSON
control plane (phase 10a) — it never imports module internals. Deliver a pluggable `Surface` interface
with a deterministic text surface and a local web surface over one view-model layer, typed commands
with idempotent audited writes, read models, and explicit approval/gate interactions. Fully testable
offline against a deterministic fake client; no model calls; **no desktop surface**.

## Starting state

- Phase 10a serves the control plane (`contracts/openapi/control-plane-v1.yaml`).
- `ui/` module scaffold exists (`doc.go`, README, `aa-*.md` directive): surfaces CLI/TUI/web; must import
  only `contracts` (enforced by `tools/archtest`).
- `contracts` v2 bindings + the OpenAPI/`rpc-v1` specs; `kernel/api` in-process service.

## Scope

**In:** a `Surface` interface (text + local web) over one view-model layer; a control-plane client typed
on `contracts` with a deterministic fake + `httptest` integration; read models (projects, work, graph,
issues, strategies, assignments, nodes, tools, learning reports); typed commands (register project,
submit work, approve/reject a gate, update workflow/tool config) with validation, idempotency, audit;
approval/gate interactions; an entry point wiring a surface to a client.

**Out:** the Wails/React desktop surface (→ v2); the control-plane **server** (→ 10a); generated
TypeScript/browser tooling; authn beyond carrying a loopback bearer token (→ phase 12); post-console
modules (→ phase 11).

## Decisions

| # | Decision | Reasoning |
|---|---|---|
| U1 | Console imports **only `contracts`**; all I/O via the control plane | Boundary D5; keeps modules swappable |
| U2 | One **`Surface`** interface; a text surface and a local web surface implement it over shared view models | New surfaces (incl. desktop) are additive |
| U3 | Writes are **idempotent** (keyed by command contract id) and **audited** | Matches the `rpc-v1` precedent; safe retries |
| U4 | Approvals/gates are **explicit**; nothing is auto-resolved | Human-in-the-loop safety |
| U5 | Rendering is **metadata-only**; secrets/raw prompts/source never shown or logged | Privacy default |
| U6 | **Go standard library only** in 10b | No heavyweight UI deps; deterministic, offline-testable |

## Stages and commits

DoD per commit: `gofmt`/build/vet/test green; `tools/archtest` clean; docs current.

- **10b.1 — Plan** (`add phase10b console plan`).
- **10b.2 — Client + fake**: typed control-plane client over `contracts`; deterministic in-memory fake;
  `httptest` integration over the OpenAPI paths. (`add control-plane client and fake`)
- **10b.3 — Read models**: view models for projects, work, graph, issues, strategies, assignments, nodes,
  tools, and the phase-9 learning reports. (`add console read models`)
- **10b.4 — Text surface**: the `Surface` interface + a deterministic text surface (golden-testable).
  *Appearance/components are decided with the maintainer (see Open).* (`add console surface and text surface`)
- **10b.5 — Web surface**: a local web surface (semantic HTML + plain-text alternative) over the same
  view models. (`add console web surface`)
- **10b.6 — Commands + audit**: typed commands (register project, submit work, approve/reject gate,
  config) with validation, idempotency keys, audit trail. (`add console commands and audit`)
- **10b.7 — Entry point + tests + docs**: wire a chosen surface to a client; e2e against the fake;
  module docs; phase summary. (`add console entry point, tests and docs`)

**Commit budget:** ~7 (phase total 10a+10b target 10–15).

## Validation gates

| Gate | Command |
|---|---|
| Formatting | `gofmt -l .` |
| Go | per-module `go build/vet/test ./...` (`ui`, `contracts`) |
| Boundaries | `tools/archtest -root ..` (console/`ui` imports only `contracts`) |
| Integration | `httptest` against the control-plane spec; deterministic fake client |
| Review | `revu --profile pr` on each PR (static-only) |

## Exit criteria

- [ ] A single entry point drives the control plane through a chosen surface; no module imports.
- [ ] Text + web surfaces share one view-model layer; golden-testable, deterministic output.
- [ ] Typed commands are validated, idempotent, audited; approvals explicit.
- [ ] Integration tests pass against the fake + `httptest`; phase summary authored.

## Open (maintainer decision — intentionally not specified here)

- **Console component design** and **how the TUI should appear** (layout, panes, navigation, keybindings,
  visual language) — decided with the maintainer before/within 10b.4.
- Web surface look/feel and accessibility choices.
