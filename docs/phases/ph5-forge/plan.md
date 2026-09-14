# Phase 5 — Forge: Plan

> Authored at the **start** of the phase. Branch and folder: `ph5-forge`.

## Objective

Deliver `aa-forge`'s Git/GitHub orchestration: a forge abstraction with a deterministic in-memory fake, repository/project and branch management, milestones and phase tracking, issues, child and Draft phase pull requests, checkpoints, audits with classified findings, releases/tags, data-driven template rendering from `docs/github-os/templates`, forge-to-memory synchronization, a phase-lifecycle orchestrator that never merges without a human gate, and a transport-agnostic RPC adapter. The entire lifecycle is testable offline.

## Starting State

- Starting ref: `main @ 41de261` (after phase 4 merge and record).
- Available:
  - `aa-contracts` v1 schemas and Go bindings, including `Issue`, `MemoryRecord`, `Reference`, and the `forge.*` RPC method names (`contracts/go/v1`, `contracts/rpc/rpc-v1.md`).
  - The `docs/github-os/` process docs and templates (`docs/github-os/templates/`), including phase, issue, PR, and audit templates.
  - A scaffolded `forge/` Go module (`go.mod`, `doc.go`, `README.md`, `aa-forge.md`).
  - The architecture boundary harness allowing `forge -> contracts, toolbox` (`tools/archtest`).
- Missing:
  - Any forge implementation; `forge` holds `doc.go` only.
  - A forge interface, a fake, template rendering, and memory sync.
  - The phase-lifecycle orchestrator and the RPC adapter.
  - Forge coverage beyond the scaffold.
- Known constraints:
  - The forge must never merge to `main` without a human gate.
  - The forge must never store secrets or credentials.
  - The fake path must work offline; the default test suite makes no network or `gh` calls.
  - Every forge operation is idempotent and auditable.
  - The phase branch name equals the phase folder name (`ph{N}-{scope}`) and the phase PR exists as a Draft from phase start.

## Scope

### In Scope
- Capability interfaces and domain types for repositories, branches, milestones, issues, pull requests, checkpoints, audits, and releases.
- A deterministic in-memory `fake` implementing every capability, with idempotency keys and an audit log.
- A human-gated merge operation that fails closed without an approval.
- A template engine that renders the versioned process templates.
- Deterministic mapping from forge objects to `contracts` records, synced through a `Sink`.
- A phase-lifecycle orchestrator: start phase, tracking issue, child issue, child PR, Draft phase PR, checkpoint, audit, release, and completion.
- A transport-agnostic RPC adapter for `forge.createIssue`, `forge.openPullRequest`, `forge.checkpoint`, and `forge.release`.

### Out of Scope
- Live GitHub API calls and credential handling; the fake and an interface boundary stand in. A `gh`-backed adapter is a follow-up.
- Real git object manipulation and worktrees; branch objects are recorded, not created on disk.
- Automated merging; merges always pass through the human gate.
- Auto-fixing audit findings; audits classify and report only.
- Control-plane transport hosting (named pipe/socket); the RPC adapter is in-process.

### Required End State
- [ ] A deterministic fake drives a full phase lifecycle with no network.
- [ ] Every mutating operation is idempotent under repeated calls.
- [ ] A merge without an approval fails closed.
- [ ] Templates render from `docs/github-os/templates` deterministically.
- [ ] Forge issues and checkpoints map to valid `contracts` records.
- [ ] The phase branch name equals the phase folder name; the phase PR starts Draft.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and boundary checks pass.

## Architecture Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P5-001 | A forge abstraction with capability sub-interfaces and a deterministic fake | Provider independence and offline end-to-end tests |
| ADR-P5-002 | Every mutating operation takes an idempotency key and is safe to repeat | Retries and reconnects must not duplicate issues, PRs, or releases |
| ADR-P5-003 | Merging is a distinct, human-gated operation; the forge never auto-merges | Safety and the architecture's human-authority principle |
| ADR-P5-004 | Templates are data loaded from `docs/github-os/templates` | The process docs and the tool share one versioned source |
| ADR-P5-005 | The phase branch name equals the phase folder name; the phase PR is Draft from start | The phase checkpoint is visible from the beginning |
| ADR-P5-006 | Forge objects map deterministically to `contracts` records via a `Sink` | One schema source; the forge never writes canonical memory directly |
| ADR-P5-007 | The fake is the reference implementation and the e2e substrate | No network, no credentials, fully deterministic |
| ADR-P5-008 | Audits classify findings by severity and never auto-fix | Review is separate from change; findings become issues |
| ADR-P5-009 | The forge never stores secrets; credential access is out of scope | Security boundary |
| ADR-P5-010 | The RPC adapter is transport-agnostic and reuses the idempotency keys | The kernel can host it over the local socket without coupling |

Diagrams: [plan.uml](./plan.uml).

## Slices

Each slice is independently validatable and maps to exactly one commit.

### Slice 1 — Phase plan and UML
**Goal** — Plan the phase and its intended infrastructure. **Inputs** — architecture, contracts, github-os docs. **Expected Output** — `docs/phases/ph5-forge/plan.md` + `plan.uml`. **Model Class** — Strong. **Commit Message** — `add ph5 forge phase plan`. **Validation** — docs link check. **Dependencies** — none. **Documentation** — this file and `plan.uml`.

### Slice 2 — Core types, interfaces, and errors
**Goal** — Domain types and capability interfaces, idempotency keys, and the audit log. **Inputs** — the forge directive and framework docs. **Expected Output** — `forge/forge/*.go` + tests. **Model Class** — General. **Commit Message** — `add forge core types and interfaces`. **Validation** — `go test ./forge/...`. **Dependencies** — slice 1. **Documentation** — `forge/README.md`.

### Slice 3 — Deterministic fake: repositories and issues
**Goal** — Repositories, branches, milestones, and issues with idempotency and an audit trail. **Inputs** — slice 2. **Expected Output** — `forge/fake/*.go` + tests. **Model Class** — General. **Commit Message** — `add forge deterministic fake for repos and issues`. **Validation** — `go test ./forge/...`. **Dependencies** — slice 2. **Documentation** — `forge/README.md`.

### Slice 4 — Fake pull requests, checkpoints, and releases
**Goal** — Draft/child/phase PRs with a human-gated merge, checkpoints, and releases/tags. **Inputs** — slice 3. **Expected Output** — `forge/fake/*.go` + tests. **Model Class** — Strong. **Commit Message** — `add forge fake pull requests checkpoints and releases`. **Validation** — `go test ./forge/...`; merge without approval fails closed. **Dependencies** — slice 3. **Documentation** — `forge/README.md`.

### Slice 5 — Audits and findings
**Goal** — Record audits and classified findings with severity; no auto-fix. **Inputs** — `docs/github-os/repository-audits.md`. **Expected Output** — `forge/fake/audits.go` + tests. **Model Class** — General. **Commit Message** — `add forge audits and findings`. **Validation** — `go test ./forge/...`. **Dependencies** — slice 4. **Documentation** — `forge/README.md`.

### Slice 6 — Template engine
**Goal** — Render the versioned process templates deterministically. **Inputs** — `docs/github-os/templates`. **Expected Output** — `forge/template/*.go` + tests; parameterized github templates. **Model Class** — General. **Commit Message** — `add forge template engine`. **Validation** — `go test ./forge/...`. **Dependencies** — slice 2. **Documentation** — `forge/README.md`.

### Slice 7 — Forge-to-memory synchronization
**Goal** — Map forge issues and checkpoints to `contracts` records and sync through a `Sink`. **Inputs** — `contracts/go/v1`. **Expected Output** — `forge/memsync/*.go` + tests. **Model Class** — Strong. **Commit Message** — `add forge memory synchronization`. **Validation** — `go test ./forge/...` with contract validation. **Dependencies** — slices 4, 5. **Documentation** — `forge/README.md`.

### Slice 8 — Phase lifecycle orchestrator
**Goal** — Drive a complete phase lifecycle over the forge, enforcing the naming rule, the Draft phase PR, and the human gate. **Inputs** — slices 3–7. **Expected Output** — `forge/lifecycle/*.go` + e2e tests. **Model Class** — Strong. **Commit Message** — `add forge phase lifecycle orchestrator`. **Validation** — `go test ./forge/...`; end-to-end fake lifecycle. **Dependencies** — slice 7. **Documentation** — `forge/README.md`.

### Slice 9 — RPC adapter
**Goal** — Transport-agnostic handlers for `forge.createIssue`, `forge.openPullRequest`, `forge.checkpoint`, and `forge.release`. **Inputs** — contracts RPC v1. **Expected Output** — `forge/rpc/*.go` + tests. **Model Class** — General. **Commit Message** — `add forge rpc adapter`. **Validation** — `go test ./forge/...`; idempotent replay. **Dependencies** — slice 8. **Documentation** — `forge/README.md`.

### Slice 10 — Phase summary and UML
**Goal** — Record the as-built phase. **Inputs** — all prior slices. **Expected Output** — `summary.md` + `summary.uml`. **Model Class** — Strong. **Commit Message** — `add ph5 forge phase summary`. **Validation** — link check. **Dependencies** — slice 9. **Documentation** — summary.

## Exit Criteria

- [ ] A full phase lifecycle completes against the fake with no network.
- [ ] Repeated operations are idempotent; no duplicate objects.
- [ ] A merge without an approval is rejected.
- [ ] Templates render deterministically from the versioned source.
- [ ] Forge records validate against `contracts`.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and boundary checks pass.
- [ ] Phase PR merged to `main`; branch retained.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | types, idempotency keys, branch/tag validation, audit severity ordering |
| contract | forge issues/checkpoints map to valid `v1.Issue`/`v1.MemoryRecord` |
| integration | full phase lifecycle over the fake; RPC replay idempotency |
| security | merge without approval fails closed; no secret fields exist |
| determinism | stable IDs, stable audit log, stable rendered templates |
| boundary | `forge` imports only contracts/toolbox (archtest) |
