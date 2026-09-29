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
| [post-refactor-implementation.md](post-refactor-implementation.md) | ISS-IMP | Umbrella: implement the ten post-refactor issues; resolve R4–R9/A4–A9 | `dev` |
| [imp-1-canonical-architecture-and-contracts.md](imp-1-canonical-architecture-and-contracts.md) | ISS-IMP-1 | Canonical architecture & contracts | `dev` |
| [imp-2-deterministic-retrieval.md](imp-2-deterministic-retrieval.md) | ISS-IMP-2 | `memory`: deterministic retrieval | `dev` |
| [imp-3-context-compilation.md](imp-3-context-compilation.md) | ISS-IMP-3 | `kernel`: context compilation, budgeting & assembly | `dev` |
| [imp-4-rpc-boundary-hardening.md](imp-4-rpc-boundary-hardening.md) | ISS-IMP-4 | `contracts`/RPC boundary & version-skew hardening | `dev` |
| [imp-5-sandbox-execution.md](imp-5-sandbox-execution.md) | ISS-IMP-5 | `runtime`: sandbox execution | `dev` |
| [imp-6-computer-use-tool-execution.md](imp-6-computer-use-tool-execution.md) | ISS-IMP-6 | `toolbox`: computer-use / tool execution | `dev` |
| [imp-7-allocation.md](imp-7-allocation.md) | ISS-IMP-7 | `kernel/allocator`, `registry`: role/model/compute allocation | `dev` |
| [imp-8-scheduling-multi-agent-execution.md](imp-8-scheduling-multi-agent-execution.md) | ISS-IMP-8 | `kernel/scheduler`, `runtime`: scheduling & multi-agent execution | `dev` |
| [imp-9-learning-experience-feedback.md](imp-9-learning-experience-feedback.md) | ISS-IMP-9 | `kernel/joblearn`, `memory`: learning / experience feedback | `dev` |
| [imp-10-observability-hardening-e2e.md](imp-10-observability-hardening-e2e.md) | ISS-IMP-10 | `obsv` + all: observability, hardening & e2e validation | `dev` |
