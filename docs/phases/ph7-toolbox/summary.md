# Phase 7 — Toolbox: Summary

> Authored at the **end** of the phase. Derived from [plan.md](./plan.md). As-built diagram: [summary.uml](./summary.uml).

## Delivered

| Slice | Status | Commit | Notes |
|---|---|---|---|
| Phase plan and UML | Complete | `4dc5081` | plan.md + plan.uml |
| Core types, errors, helpers | Complete | `c185049` | `toolbox/{types,errors,clock,idempotency}.go` |
| Registry and manifests | Complete | `7249881` | `toolbox/registry` (validated manifests, provider seam) |
| Policy engine and grants | Complete | `f78b3b2` | `toolbox/policy` (capability intersection, sandbox, audit) |
| Invocation host and fake | Complete | `0876d2e` | `toolbox/host` (idempotent invocation, deterministic fake) |
| Workflow compiler | Complete | `1fb77b3` | `toolbox/workflow/compile.go` (validate + topo plan) |
| Workflow runtime | Complete | `4fe11a0` | `toolbox/workflow/runtime.go` (resume, retries, gates) |
| RPC adapter | Complete | `dcdf45a` | `toolbox/rpc` (`invoke`, `compileWorkflow`, `registry`) |
| Phase summary and UML | Complete | this commit | summary.md + summary.uml + README |

- Starting ref: `main @ a934498` (after phase 6 merge and record).
- Tracking issue: none (phase executed via the phase PR).
- Milestone: none (matches phases 0–6 on this repository).
- Phase PR: [#8](https://github.com/greadee/aa1/pull/8) — merged
- Merge commit: `74f483a`
- Branch: `ph7-toolbox` (retained after merge on `aa1`)

## Validation

| Gate | Result | Evidence |
|---|---|---|
| Go build / vet | Pass | all `toolbox/...` packages |
| Go tests | Pass | 63 tests across 6 packages |
| Formatting | Pass | `gofmt -l .` clean |
| Boundaries | Pass | `archtest` reports ok; `toolbox` imports only contracts (and its own subpackages) |
| Manifest validation | Pass | invalid `tool_manifest` and unknown kinds rejected at registration |
| Capability grants | Pass | grant is the declared/contract intersection; no implicit grant |
| Explicit denials | Pass | a denied capability loses even when also granted |
| Sandbox | Pass | network requires `network_access`; required isolation fails closed without an enforcer |
| Audit log | Pass | decisions are ordered, timestamped, and returned as a copy |
| Invocation | Pass | provider runs only on an allow; repeated keys return the original result |
| Compiler | Pass | cycles, self/duplicate/dangling deps rejected; order is input-order independent |
| Runtime | Pass | retries, `fail`/`retry`/`skip`/`compensate`, gates, and resume verified |
| RPC | Pass | `toolbox.*` methods dispatch; idempotent replay; incompatible version and bad params rejected |
| Docs links | Pass | runs on CI after push |
| Phase audit | Complete | see Audit below |

## Decisions Affirmed

| # | Decision | Outcome | Evidence |
|---|---|---|---|
| ADR-P7-001 | Registry keyed by manifest ID; validate on entry | Affirmed | `registry.Register` rejects invalid manifests; `Register`/`Replace` semantics |
| ADR-P7-002 | One `Provider` seam for all tool kinds | Affirmed | `TestRegisterProviderAndNewKindWithoutCoreChange`; MCP kind added by registration |
| ADR-P7-003 | Grant is the contract intersection minus denials | Affirmed | `policy.Evaluate`; `TestGrantIsIntersection`, `TestNoImplicitGrant` |
| ADR-P7-004 | Explicit, typed, audited denials | Affirmed | `TestExplicitDenialWins`; `Engine.Audit` order and copy |
| ADR-P7-005 | Sandbox and network fail closed | Affirmed | `TestRequiredSandboxFailsClosedWithoutEnforcer`, `TestSandboxNetworkRequiresCapability` |
| ADR-P7-006 | Deterministic topological compile | Affirmed | `TestCompileOrderIndependentOfInputOrder`, `TestCompileDetectsCycle` |
| ADR-P7-007 | Resume from serialized state | Affirmed | `TestResumeAfterApproval`, `TestResumePreservesAttempts` |
| ADR-P7-008 | Bounded retries and per-step failure policy | Affirmed | `TestRetryThenSuccess`, `TestOnFailureSkip`, `TestOnFailureCompensate` |
| ADR-P7-009 | Invocation is idempotent by derived key | Affirmed | `TestInvokeIsIdempotent`, `TestInvokeIdempotentReplay` |
| ADR-P7-010 | Transport-agnostic RPC over the host/compiler | Affirmed | `rpc.Service.Handle` takes/returns message objects; replay cache |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | Live MCP transport (stdio/HTTP) and a long-running host | The `Provider` seam plus a deterministic in-memory fake | CI has no MCP servers; the live host is a follow-up |
| 2 | Real OS sandbox primitives (job objects, seccomp, containers) | A deterministic sandbox decision behind a `SandboxEnforcer` seam | No OS primitives in CI; the enforcer is a follow-up |
| 3 | Audit log and results persisted to `aa-memory` | In-memory ordered audit log and idempotency cache | Memory promotion is a kernel/console concern |
| 4 | Enforce call/token/cost budgets on invocation | Budget is carried on the contract and workflow but not enforced here | Budgets belong to `aa-kernel`/`aa-sifter`; toolbox stays deterministic |
| 5 | Toolbox schedules and drives workflows | The runtime executes a compiled plan over a `Runner` seam; the kernel drives it | Keeps the module free of scheduling and execution authority |
| 6 | Built-in tool providers (git, filesystem, shell) | The registry, policy, host, and fake only | Built-ins are a later concern; extensibility is provider-agnostic |

## Deferred / Follow-Up

- A live MCP host (stdio/HTTP) registered as an ordinary provider behind the same permission model.
- A real `SandboxEnforcer` for process/container/OS isolation.
- Built-in providers for git, filesystem, and shell tools under explicit capability grants.
- A memory-backed audit log and invocation-result history over `memory.record`.
- Hosting `toolbox/rpc` over the local socket and wiring the kernel's toolbox client.
- Enforcing workflow and invocation budgets in `aa-kernel`/`aa-sifter`.

## Metrics

- Commits: 8 implementation + this summary
- Packages: 6 (`toolbox`, `registry`, `policy`, `host`, `workflow`, `rpc`)
- Tests: 63
- ADRs: ADR-P7-001 … ADR-P7-010
- Pitfalls recorded: 0
- PRs: 1 phase PR (#8)
- Issues: none

## Audit

| Severity | Count | Notes |
|---|---|---|
| P0 | 0 | — |
| P1 | 0 | — |
| P2 | 0 | — |
| P3 | 2 | Live MCP transport and the OS sandbox enforcer (documented above) |

Audit result: no blocking findings. Boundary rule holds: `toolbox` imports only contracts (and its own subpackages). Capabilities are never granted implicitly; every decision is audited.

## Retrospective

### Repeat
- Keep every tool kind behind one `Provider` seam; the "add an MCP without a core change" exit criterion fell out of registration alone.
- Decide policy before dispatch and audit every decision; the denial tests doubled as the security suite.
- Compile workflows to a plain data plan; determinism and cycle detection were easy to test because the compiler has no runtime dependency.

### Avoid
- Do not let the host reach for scheduling or budget authority; passing contracts in and results out kept the boundary clean.
- Make resume state a value with a deep clone; early aliasing of `Results` would have corrupted resume.
- Do not treat absence as approval; a nil gate evaluator leaves a run awaiting approval rather than proceeding.

### As-Built Diagram

[summary.uml](./summary.uml)
