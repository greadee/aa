# Phase 3 — Kernel: Summary

> Authored at the **end** of the phase. Derived from [plan.md](./plan.md). As-built diagram: [summary.uml](./summary.uml).

## Delivered

| Slice | Status | Commit | Notes |
|---|---|---|---|
| Phase plan and UML | Complete | `4f4dba9` | plan.md + plan.uml |
| Role and trade registry | Complete | `490a882` | `kernel/registry` |
| Task graph and readiness | Complete | `5065ebc` | `kernel/plan` |
| Assignment state machine and leases | Complete | `0c510a1` | `kernel/control` |
| Execution contracts | Complete | `55f6db2` | `kernel/contract` |
| Context compiler | Complete | `5ee57fa` | `kernel/context` |
| Runtime adapter and fake | Complete | `1782b76` | `kernel/runtime` |
| Gates and result intake | Complete | `ca5e5b7` | `kernel/gate`, `kernel/intake` |
| Telemetry and learning candidates | Complete | `9f7fa6b` | `kernel/telemetry` |
| Orchestrator | Complete | `2e8d21e` | `kernel/orchestrator` |
| Control-plane API | Complete | `ccce81c` | `kernel/api` |
| Phase summary and UML | Complete | (this commit) | summary + module docs |

- Tracking issue: none (phase executed via the phase PR)
- Phase PR: (see GitHub)
- Branch: `ph3-kernel` (retained after merge)

## Validation

| Gate | Result | Evidence |
|---|---|---|
| Go build / vet | Pass | all `kernel/...` packages |
| Go tests | Pass | 44 tests across 11 packages |
| Supervised run | Pass | two-work-package run with the fake runtime and a human gate |
| Execution disabled by default | Pass | disabled config never calls the runtime |
| Least privilege | Pass | unpermitted `deploy` denied and never sent to the runtime |
| Determinism | Pass | selection ordering and context digests are stable |
| Boundaries | Pass | `archtest` reports kernel imports only contracts/obsv/memory/toolbox |
| Formatting | Pass | `gofmt -l .` clean |
| Docs links | Pending | runs on CI after push |
| Phase audit | Not yet run | before merge to `main` |

## Decisions Affirmed

| # | Decision | Outcome | Evidence |
|---|---|---|---|
| ADR-P3-001 | Deterministic coordination; no model in the control path | Affirmed | the whole cycle runs with a scripted fake and no model |
| ADR-P3-002 | Registry selection with stable tie-breaks | Affirmed | selection prefers lower cost then id; rejections carry reasons |
| ADR-P3-003 | Assignment transitions fail closed with leases | Affirmed | illegal transitions and retry-from-running rejected |
| ADR-P3-004 | Contracts grant only the intersection | Affirmed | `deploy` denied and absent from the runtime request |
| ADR-P3-005 | Context compiler is pure and bounded | Affirmed | order-independent digest; budget respected |
| ADR-P3-006 | Runtime behind an adapter with a fake | Affirmed | end-to-end run without a model |
| ADR-P3-007 | Results untrusted until intake and gates | Affirmed | succeeded-but-untested results are rejected by the tests gate |
| ADR-P3-008 | Telemetry feeds learning candidates | Affirmed | success/pitfall/cost candidates derived deterministically |
| ADR-P3-009 | Execution disabled by default | Affirmed | disabled service returns `ErrDisabled`; adapter not called |
| ADR-P3-010 | The orchestrator records through a `Sink` | Affirmed | events emitted through the sink; no memory/RPC coupling |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | Single-node supervised pilot on a real repo | In-memory two-work-package supervised run with the scripted fake | No production runtime adapter exists yet; the fake proves the control plane end to end |
| 2 | `obsv` host and live observation socket | Kernel emits events through a `Sink` interface; the `obsv` service migration is deferred | The `obsv` module is scaffold-only; migrating it is a separate, larger effort |
| 3 | Memory-backed recording | `Sink` interface with `NopSink`; a memory adapter is deferred | Keeps control logic free of `memory` coupling; a memory sink lands with pilot wiring |
| 4 | Context compiled from live memory queries | Compiler takes explicit inputs via `ContextInputs` | Purity and deterministic digests; retrieval is the caller's concern |
| 5 | Control-plane HTTP/socket surface | In-process `api.Service` only | Transport belongs with the console phase; the service shape is fixed now |
| 6 | Role selection from organizational heuristics | Deterministic `RolesFor(capabilities)` plus registry requirements | No LLM in the control path; heuristics remain a later evidence-gated feature |

## Deferred / Follow-Up

- `obsv` service migration and the kernel observation host.
- A memory-backed `Sink` adapter recording canonical events and records.
- A production runtime adapter (Codex or hosted), capability-scoped and opt-in.
- The pilot on a real repository with the runtime adapter enabled.
- Git worktree/workspace provisioning (deferred; not required for the in-memory pilot).
- Control-plane transport (HTTP/socket) with the console.
- Phase audit before merge to `main`.

## Metrics

- Commits: 12 (11 slices + this summary)
- Packages: 11 (`registry`, `plan`, `control`, `contract`, `context`, `runtime`, `gate`, `intake`, `telemetry`, `orchestrator`, `api`)
- ADRs: ADR-P3-001 … ADR-P3-010
- Pitfalls recorded: 0
- Tests added: 44

## Retrospective

### Repeat
- Split the control plane into small pure packages; each is independently testable and the orchestrator composes them.
- The scripted fake made an end-to-end supervised run possible before any model integration.
- Fail-closed tests for transitions, permissions, and gates.

### Avoid
- The `context` package name collides with the standard library `context`; import it aliased (`kcontext`) at call sites.
- Wire `Sink` to `memory` early next time so history exists from the first pilot.

### As-Built Diagram

[summary.uml](./summary.uml)
