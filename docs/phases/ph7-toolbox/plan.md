# Phase 7 — Toolbox: Plan

> Authored at the **start** of the phase. Branch and folder: `ph7-toolbox`.

## Objective

Deliver `aa-toolbox`: a tool, plugin, and MCP registry with declared manifests; a policy engine that grants capabilities only by explicit intersection with the caller's execution contract; a deterministic sandbox decision; a provider-agnostic invocation host with idempotent replay and an audited denial log; a workflow compiler that validates a declarative workflow into a deterministic plan; a resumable workflow runtime with retries, gates, and per-step failure policy; and a transport-agnostic RPC adapter for `toolbox.invoke`, `toolbox.compileWorkflow`, and `toolbox.registry`. The entire capability is testable offline.

## Starting State

- Starting ref: `main @ a934498` (after phase 6 merge and record).
- Available:
  - `aa-contracts` v1 schemas and Go bindings for `tool_manifest` (`contracts/go/v1/tool.go`) and `workflow` (`contracts/go/v1/workflow.go`), plus `Capability`, `ExecutionContract`, `Budget`, and the shared validators.
  - The `toolbox.*` RPC method names and envelope conventions (`contracts/rpc/rpc-v1.md`).
  - A scaffolded `toolbox/` Go module (`go.mod`, `doc.go`, `README.md`, `aa-toolbox.md`).
  - The architecture boundary harness allowing `toolbox -> contracts` (`tools/archtest`).
  - The phase documentation conventions and templates (`docs/github-os/phase-documentation.md`).
- Missing:
  - Any toolbox implementation; `toolbox` holds `doc.go` only.
  - The registry, the manifest lifecycle, and the provider seam.
  - The policy engine, capability grants, and sandbox decisions.
  - The invocation host, idempotent replay, and the audit log.
  - The workflow compiler and the resumable workflow runtime.
  - The RPC adapter.
- Known constraints:
  - Capabilities are never granted implicitly or by default; a denial is explicit and audited.
  - A plugin may never reach outside its declared permissions.
  - Adding a tool or MCP server must not require a core change.
  - Workflows are data: versioned, resumable, and testable; ordering derives from step IDs and the compiled plan, never the wall clock.
  - The default test suite makes no network calls; live MCP transport is a follow-up.
  - The phase branch name equals the phase folder name (`ph{N}-{scope}`) and the phase PR exists as a Draft from phase start.

## Scope

### In Scope
- Tool/plugin/MCP registration from validated `tool_manifest` contracts, with lookup, listing, replacement, and removal.
- A small provider seam (builtin, plugin, MCP, and a deterministic in-memory fake) so a new tool kind is registration, not a core change.
- A policy engine that intersects a manifest's declared capabilities with a caller's execution contract, honors explicit denials, and returns a typed decision.
- Sandbox decisions: required isolation and network access are enforced against the contract; missing enforcement fails closed.
- An append-only, ordered audit log of grants and denials.
- A provider-agnostic invocation host with idempotent replay and deterministic results.
- A workflow compiler: structural validation, duplicate/dangling dependency checks, cycle detection, and a deterministic topological execution plan.
- A resumable workflow runtime: bounded retries, gate steps, human-approval steps, and `fail`/`retry`/`skip`/`compensate` failure policy over a runner seam.
- A transport-agnostic RPC adapter for `toolbox.invoke`, `toolbox.compileWorkflow`, and `toolbox.registry`.

### Out of Scope
- Real OS sandbox primitives (job objects, AppContainer, seccomp) and process/container isolation; the sandbox is a deterministic decision and a provider seam. A live enforcer is a follow-up.
- Live MCP transport (stdio/HTTP) and a long-running host process; a provider seam and a deterministic fake stand in.
- Model routing and budgets on tool invocation; those belong to `aa-sifter` and `aa-kernel`.
- Persisting the audit log or invocation results to `aa-memory`; the log is in-memory and ordered. Memory promotion is a kernel/console concern.
- Scheduling workflows; `aa-kernel` compiles and drives them.

### Required End State
- [ ] A manifest that fails `ToolManifest.Validate` is rejected at registration.
- [ ] A new tool or MCP server is added by registration alone; no core file changes.
- [ ] Invocation with a contract missing a declared capability is denied with an explicit reason.
- [ ] An explicitly denied capability always loses, even when otherwise granted.
- [ ] A sandbox-required tool with no available sandbox fails closed; network access requires `network_access`.
- [ ] Grants and denials are recorded in order in an auditable log.
- [ ] Repeated invocation with the same idempotency key returns the original result.
- [ ] The compiler rejects cycles, duplicate IDs, and dangling dependencies, and plans the same workflow identically.
- [ ] The runtime resumes from a serialized state without re-running completed steps and honors per-step failure policy.
- [ ] `toolbox.invoke`, `toolbox.compileWorkflow`, and `toolbox.registry` dispatch and replay idempotently.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and boundary checks pass.

## Architecture Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P7-001 | The registry is keyed by manifest ID and validates every manifest against the `tool_manifest` contract | One validated identity per tool; invalid declarations never enter the registry |
| ADR-P7-002 | Tool kinds (builtin/plugin/mcp/fake) sit behind one small `Provider` seam | Adding a tool or MCP server is registration, not a core change |
| ADR-P7-003 | A grant is the intersection of the manifest's declared capabilities and the caller's execution contract, minus explicit denials | Capabilities are never implicit; least privilege is mechanical |
| ADR-P7-004 | Denials are typed, explicit, and appended to an ordered audit log | Auditable refusals; no silent no-ops |
| ADR-P7-005 | Sandbox and network access are decided from the manifest and the contract; absent required enforcement fails closed | Security is a boundary, not a prompt |
| ADR-P7-006 | The compiler validates structure, rejects cycles, and produces a deterministic topological plan | Workflows are data; the same document always compiles to the same plan |
| ADR-P7-007 | The runtime resumes from a serialized state of completed steps and attempts | Resumable workflows must not repeat finished work |
| ADR-P7-008 | Per-step failure policy (`fail`/`retry`/`skip`/`compensate`) is honored with bounded retries | Failure handling is declared, not improvised |
| ADR-P7-009 | Invocation is idempotent by a derived key; replay returns the original result | Retries and reconnects must not duplicate side effects |
| ADR-P7-010 | The RPC adapter is transport-agnostic and reuses the host and compiler | The kernel can host it over the local socket; tests stay offline |

Diagrams: [plan.uml](./plan.uml).

## Slices

Each slice is independently validatable and maps to exactly one commit.

### Slice 1 — Phase plan and UML
**Goal** — Plan the phase and its intended infrastructure. **Inputs** — architecture, contracts, the toolbox directive. **Expected Output** — `docs/phases/ph7-toolbox/plan.md` + `plan.uml`. **Model Class** — Strong. **Commit Message** — `add ph7 toolbox phase plan`. **Validation** — docs link check. **Dependencies** — none.

### Slice 2 — Core types, errors, and helpers
**Goal** — Domain types, sentinel errors, a deterministic clock, and idempotency keys. **Inputs** — the toolbox directive. **Expected Output** — `toolbox/types.go`, `toolbox/errors.go`, `toolbox/clock.go`, `toolbox/idempotency.go` + tests. **Model Class** — General. **Commit Message** — `add toolbox core types and errors`. **Validation** — `go test ./...`. **Dependencies** — slice 1. **Documentation** — `toolbox/README.md`.

### Slice 3 — Registry and manifests
**Goal** — Register, validate, look up, list, and remove tool/plugin/MCP manifests; the provider seam. **Inputs** — slice 2, `contracts/go/v1/tool.go`. **Expected Output** — `toolbox/registry/*.go` + tests. **Model Class** — General. **Commit Message** — `add toolbox registry and manifests`. **Validation** — `go test ./...`; invalid manifests rejected. **Dependencies** — slice 2. **Documentation** — `toolbox/README.md`.

### Slice 4 — Policy engine and capability grants
**Goal** — Intersect declared capabilities with the execution contract, honor denials, decide sandbox/network, and append to the audit log. **Inputs** — slices 2–3, `contracts/go/v1/executioncontract.go`. **Expected Output** — `toolbox/policy/*.go` + tests. **Model Class** — Strong. **Commit Message** — `add toolbox policy engine and grants`. **Validation** — `go test ./...`; permission-denial tests. **Dependencies** — slice 3. **Documentation** — `toolbox/README.md`.

### Slice 5 — Invocation host and fake provider
**Goal** — Provider-agnostic invocation with idempotent replay, deterministic results, and an in-memory fake provider. **Inputs** — slices 3–4. **Expected Output** — `toolbox/host/*.go`, `toolbox/host/fake.go` + tests. **Model Class** — Strong. **Commit Message** — `add toolbox invocation host and fake`. **Validation** — `go test ./...`; replay and denial tests. **Dependencies** — slice 4. **Documentation** — `toolbox/README.md`.

### Slice 6 — Workflow compiler
**Goal** — Validate a workflow, reject duplicate IDs, dangling dependencies, and cycles, and produce a deterministic topological plan. **Inputs** — slices 2, `contracts/go/v1/workflow.go`. **Expected Output** — `toolbox/workflow/compile.go` + tests. **Model Class** — Strong. **Commit Message** — `add toolbox workflow compiler`. **Validation** — `go test ./...`; cycle and determinism tests. **Dependencies** — slice 5. **Documentation** — `toolbox/README.md`.

### Slice 7 — Workflow runtime
**Goal** — Resumable execution over a runner seam with bounded retries, gates, human approval, and per-step failure policy. **Inputs** — slice 6. **Expected Output** — `toolbox/workflow/runtime.go` + tests. **Model Class** — Strong. **Commit Message** — `add toolbox workflow runtime`. **Validation** — `go test ./...`; resume, retry, and gate tests. **Dependencies** — slice 6. **Documentation** — `toolbox/README.md`.

### Slice 8 — RPC adapter
**Goal** — Transport-agnostic handlers for `toolbox.invoke`, `toolbox.compileWorkflow`, and `toolbox.registry`. **Inputs** — `contracts/rpc/rpc-v1.md`. **Expected Output** — `toolbox/rpc/*.go` + tests. **Model Class** — General. **Commit Message** — `add toolbox rpc adapter`. **Validation** — `go test ./...`; idempotent replay. **Dependencies** — slice 7. **Documentation** — `toolbox/README.md`.

### Slice 9 — Phase summary and UML
**Goal** — Record the as-built phase. **Inputs** — all prior slices. **Expected Output** — `summary.md` + `summary.uml` + updated `toolbox/README.md`. **Model Class** — Strong. **Commit Message** — `add ph7 toolbox phase summary`. **Validation** — link check. **Dependencies** — slice 8.

## Exit Criteria

- [ ] Invalid manifests are rejected at registration.
- [ ] A tool or MCP server is addable without a core change.
- [ ] Capability intersection and explicit denials behave as specified; denials are audited.
- [ ] Sandbox and network decisions fail closed when required.
- [ ] Invocation is idempotent and deterministic.
- [ ] The compiler rejects cycles/duplicates/dangling deps and is deterministic.
- [ ] The workflow runtime resumes and honors failure policy.
- [ ] `toolbox.*` methods dispatch and replay idempotently.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and boundary checks pass.
- [ ] Phase PR merged to `main`; branch retained.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | manifest validation, idempotency keys, capability intersection, sandbox decisions, cycle detection, retry math |
| contract | `toolbox.*` RPC params/results; `tool_manifest` and `workflow` conformance |
| integration | registry + policy + host invocation; compile + run a workflow over the fake runner |
| security | implicit grants refused; explicit denial wins; sandbox-required fails closed; no credential fields |
| determinism | stable plan order, stable invocation results, stable audit order |
| recovery | workflow resume after a partial run; retries then success; retries exhausted |
| boundary | `toolbox` imports only contracts (archtest) |
