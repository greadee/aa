# Phase 3 — Kernel: Plan

> Authored at the **start** of the phase. Branch and folder: `ph3-kernel`.

## Objective

Deliver `aa-kernel`'s deterministic control plane: a role/trade registry with deterministic selection, a task graph with readiness, assignment state machines with leases, least-privilege execution contracts, a deterministic context compiler, a runtime adapter interface with a scripted fake, result intake with integration and human gates, telemetry with learning candidates, an orchestrator that runs a full supervised cycle on a real graph, and a control-plane API. Execution is disabled by default and models are never called directly.

## Starting State

- Starting ref: `main @ 67ce6c7` (after phase 2 merge).
- Available:
  - `aa-contracts` v1 schemas and Go bindings (phase 1), including work packages, task graphs, execution contracts, result envelopes, and telemetry.
  - `aa-memory` canonical store, projection, query, and repositories (phase 2).
  - Architecture boundary harness enforcing `kernel -> contracts, obsv, memory, toolbox`.
  - State registry and event taxonomy in `docs/reference/terminology.md`.
- Missing:
  - Any kernel implementation; `kernel` holds `doc.go` only.
  - `obsv` implementation (the module is scaffolded only); the observation host is deferred.
- Known constraints:
  - The kernel must not import `sync`, `forge`, or `sifter`; those are reached over RPC.
  - Coordination is deterministic; models supply judgment inside contracts.
  - Execution is opt-in and disabled by default.
  - Workers report; authority validates. Builders never self-certify.

## Scope

### In Scope
- Role, trade, and worker registry with deterministic selection and rejection reasons.
- Work packages, a validated dependency graph, and readiness computation.
- Assignment state machine and leases.
- Execution contracts with capability intersection and budgets.
- A deterministic, bounded context compiler.
- A runtime adapter interface with a scripted fake and a no-op adapter.
- Result intake and gates (tests gate, human gate, composite).
- Telemetry and learning-candidate derivation.
- An orchestrator running a full supervised cycle: ready -> select -> contract -> context -> lease -> run -> collect -> gate -> accept.
- A control-plane API for submit, status, dispatch, and approve.

### Out of Scope
- Real model/runtime execution (Codex or hosted); the fake adapter stands in.
- `obsv` service migration and the live observation socket (deferred slice; kernel emits through a `Sink` interface).
- Context compilation from live `memory` queries beyond a source interface (a memory-backed source lands with the orchestrator wiring).
- Production HTTP server, TLS, and browser surface (console phase).
- Learned routing or selection; only deterministic rules.

### Required End State
- [ ] Deterministic registry selection with reasons.
- [ ] Graph validation (missing deps, cycles) and readiness.
- [ ] Assignment transitions fail closed; leases expire.
- [ ] Contracts grant only the intersection of requested and permitted capabilities.
- [ ] Context bundles are deterministic and bounded.
- [ ] The orchestrator completes a two-work-package supervised run with the fake runtime and a human gate.
- [ ] Execution is disabled unless explicitly enabled.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and boundary checks pass.

## Architecture Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P3-001 | Coordination is deterministic; no model is called in the control path | Reproducibility, testability, and safety |
| ADR-P3-002 | Workers are selected from a registry by capability/trade/role, with stable tie-breaks | Provider independence and deterministic dispatch |
| ADR-P3-003 | Assignment transitions are an explicit state machine with leases | Fail-closed control and safe recovery |
| ADR-P3-004 | Execution contracts grant the intersection of requested and permitted capabilities; the rest are denied and recorded | Least privilege, auditable |
| ADR-P3-005 | The context compiler is pure and bounded, taking explicit inputs | Deterministic bundles and a stable digest |
| ADR-P3-006 | Runtime access is behind an `Adapter` interface; the fake is deterministic | Model-agnostic kernel; end-to-end tests without a model |
| ADR-P3-007 | Results are untrusted until intake validates and gates pass | Authority validates; workers never self-certify |
| ADR-P3-008 | Telemetry feeds learning candidates, never authoritative decisions | Evidence-gated learning |
| ADR-P3-009 | Execution is disabled by default and enabled explicitly | Security boundary |
| ADR-P3-010 | The orchestrator records through a `Sink` interface | Keeps `memory`/RPC wiring out of the control logic |

Diagrams: [plan.uml](./plan.uml).

## Slices

Each slice maps to exactly one commit.

### Slice 1 — Phase plan and UML
**Commit** — `add ph3 kernel phase plan`. **Validation** — docs link check.

### Slice 2 — Role and trade registry
**Goal** — Roles, trades, workers, and deterministic selection with rejection reasons. **Output** — `kernel/registry/*.go` + tests. **Commit** — `add kernel role and trade registry`.

### Slice 3 — Task graph and readiness
**Goal** — Work packages, a validated dependency graph, cycle detection, and readiness. **Output** — `kernel/plan/*.go` + tests. **Commit** — `add kernel task graph and readiness`.

### Slice 4 — Assignment state machine and leases
**Goal** — Assignment lifecycle transitions and expiring leases. **Output** — `kernel/control/*.go` + tests. **Commit** — `add kernel assignment state machine and leases`.

### Slice 5 — Execution contracts
**Goal** — Capability intersection, denial recording, budgets, and contract digests. **Output** — `kernel/contract/*.go` + tests. **Commit** — `add kernel execution contracts`.

### Slice 6 — Context compiler
**Goal** — Deterministic, bounded, digest-stable context bundles. **Output** — `kernel/context/*.go` + tests. **Commit** — `add kernel context compiler`.

### Slice 7 — Runtime adapter and fake
**Goal** — A runtime interface plus a scripted deterministic fake and a no-op. **Output** — `kernel/runtime/*.go` + tests. **Commit** — `add kernel runtime adapter and fake`.

### Slice 8 — Gates and result intake
**Goal** — A tests gate, a human gate, a composite gate, and idempotent result intake. **Output** — `kernel/gate/*.go`, `kernel/intake/*.go` + tests. **Commit** — `add kernel gates and result intake`.

### Slice 9 — Telemetry and learning candidates
**Goal** — Telemetry records and deterministic learning candidates. **Output** — `kernel/telemetry/*.go` + tests. **Commit** — `add kernel telemetry and learning candidates`.

### Slice 10 — Orchestrator
**Goal** — End-to-end supervised cycle over a graph using the registry, contracts, context, runtime, gates, and telemetry, recorded through a `Sink`. **Output** — `kernel/orchestrator/*.go` + tests. **Commit** — `add kernel orchestrator`.

### Slice 11 — Control-plane API
**Goal** — In-process control-plane service for submit, status, dispatch, and approve, disabled by default. **Output** — `kernel/api/*.go` + tests. **Commit** — `add kernel control plane api`.

### Slice 12 — Phase summary and UML
**Commit** — `add ph3 kernel phase summary`. **Validation** — link check.

## Exit Criteria

- [ ] A two-work-package supervised run completes with the fake runtime and a human gate.
- [ ] Execution is disabled unless enabled.
- [ ] All control decisions are deterministic and tested.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, boundary checks pass.
- [ ] Phase PR merged to `main`; branch retained.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | selection, readiness, cycles, transitions, leases, contract intersection, context budgets, gates |
| contract | constructs validate against `contracts` shapes |
| integration | orchestrator end-to-end; API submit/dispatch/approve |
| boundary | `kernel` imports only contracts/obsv/memory/toolbox (archtest) |
| determinism | context digests and selection ordering are stable |
| fail-closed | illegal transitions, expired leases, missing permissions, and gated results are rejected |
