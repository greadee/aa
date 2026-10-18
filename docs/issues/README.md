# Issues

Durable issue documentation. GitHub issues are the execution record; these documents are the knowledge record for work that must survive a session.

## Convention

- One document per issue or sub-issue. Do not collapse distinct problems into a single document.
- A larger change that spans several modules and branches gets an **umbrella** document that lists its distinct sub-issues and the branches they affect. The umbrella states the problem and the dependency order; it does not specify the solution for a branch whose work has not started.
- Each sub-issue's solution plan is authored with the pull request on the branch that owns it, and is linked from that sub-issue's document.
- Module-level plans and summaries live under the module's project history (`docs/modules/<module>/updates/<slug>/plan.md` and `.../summary.md`) and are linked from the owning sub-issue document.

## Index

| Document | Issue | Scope | Branches |
|---|---|---|---|
| [trace-learning-substrate.md](trace-learning-substrate.md) | ISS-TRACE-LOOP | Umbrella: the observation → trace → learning substrate is under-delivered | `ph1-contracts`, `ph2-memory`, `ph3-kernel`, `ph4-sifter`, `ph9-joblearn` |
| [ph1-contracts-trace-contract.md](ph1-contracts-trace-contract.md) | ISS-TRACE-1 | `contracts`: bounded per-step trace object | `ph1-contracts` |
| [ph2-memory-trace-store.md](ph2-memory-trace-store.md) | ISS-TRACE-2 | `memory`: canonical trace store, summaries, retention | `ph2-memory` |
| [post-refactor-implementation.md](post-refactor-implementation.md) | ISS-IMP (#22) | Umbrella: implement the ten post-refactor issues; resolve R4–R9/A4–A9 | `dev` |
| [imp-1-canonical-architecture-and-contracts.md](imp-1-canonical-architecture-and-contracts.md) | ISS-IMP-1 (#23) | Canonical architecture & contracts | `dev` |
| [imp-2-deterministic-retrieval.md](imp-2-deterministic-retrieval.md) | ISS-IMP-2 (#24) | `memory`: deterministic retrieval | `dev` |
| [imp-3-context-compilation.md](imp-3-context-compilation.md) | ISS-IMP-3 (#25) | `kernel`: context compilation, budgeting & assembly | `dev` |
| [imp-4-rpc-boundary-hardening.md](imp-4-rpc-boundary-hardening.md) | ISS-IMP-4 (#26) | `contracts`/RPC boundary & version-skew hardening | `dev` |
| [imp-5-sandbox-execution.md](imp-5-sandbox-execution.md) | ISS-IMP-5 (#27) | `runtime`: sandbox execution | `dev` |
| [imp-6-computer-use-tool-execution.md](imp-6-computer-use-tool-execution.md) | ISS-IMP-6 (#28) | `toolbox`: computer-use / tool execution | `dev` |
| [imp-7-allocation.md](imp-7-allocation.md) | ISS-IMP-7 (#29) | `kernel/allocator`, `registry`: role/model/compute allocation | `dev` |
| [imp-8-scheduling-multi-agent-execution.md](imp-8-scheduling-multi-agent-execution.md) | ISS-IMP-8 (#30) | `kernel/scheduler`, `runtime`: scheduling & multi-agent execution | `dev` |
| [imp-9-learning-experience-feedback.md](imp-9-learning-experience-feedback.md) | ISS-IMP-9 (#31) | `kernel/joblearn`, `memory`: learning / experience feedback | `dev` |
| [imp-10-observability-hardening-e2e.md](imp-10-observability-hardening-e2e.md) | ISS-IMP-10 (#32) | `obsv` + all: observability, hardening & e2e validation | `dev` |
| [toolbox-pre-hardening.md](toolbox-pre-hardening.md) | (umbrella) | Pre-hardening sprint: TOCTOU race, stale contracts, RPC fragility, missing timeouts, untyped config, observability, file split, binding parity | `dev` |
| [toolbox-rpc-toctou-race.md](toolbox-rpc-toctou-race.md) | (P0) | RPC TOCTOU race in `Service.Handle` — concurrent identical requests can execute provider twice | `dev` |
| [toolbox-stale-contract-version.md](toolbox-stale-contract-version.md) | (P1) | Test fixtures use stale `ContractVersion: "1.0"` instead of active `"2.0"` | `dev` |
| [toolbox-rpc-typed-requests.md](toolbox-rpc-typed-requests.md) | (P1) | RPC layer uses `map[string]any` — no compile-time safety | `dev` |
| [toolbox-rpc-timeout.md](toolbox-rpc-timeout.md) | (P2) | `TimeoutMS` field on `AA` struct is never honored | `dev` |
| [toolbox-rpc-error-structure.md](toolbox-rpc-error-structure.md) | (P2) | RPC error responses lack structured diagnostic data | `dev` |
| [toolbox-workflow-config-validation.md](toolbox-workflow-config-validation.md) | (P2) | `WorkflowStep.Config` is untyped — no per-step-kind validation | `dev` |
| [toolbox-metrics-observability.md](toolbox-metrics-observability.md) | (P2) | No metrics or observability beyond in-memory audit log | `dev` |
| [toolbox-contract-file-split.md](toolbox-contract-file-split.md) | (P3) | Large contract files should be split into domain-specific units | `dev` |
| [toolbox-contracts-parity-review.md](toolbox-contracts-parity-review.md) | (P3) | Python/TypeScript bindings not reviewed for parity with Go | `dev` |
