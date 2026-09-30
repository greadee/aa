# aa-kernel — module documentation

> Project history for the `aa-kernel` module. Part of [docs/modules](../README.md).

**Directive** — [`aa-kernel.md`](../../../kernel/aa-kernel.md) · **Source** — [`kernel/`](../../../kernel/) · **Updates** — [`updates/`](./updates/)

## Responsibility

Own deterministic coordination of all work: plan it, select roles and workers, dispatch it, gate it, and observe it. The kernel is the platform; models are workers.

## Owns

- The `obsv` host (journal + local socket) for current sessions.
- Planning: task aggregate, dependency graph, readiness.
- The allocator: planning, and role/model/compute allocation (worker instances live in `runtime/worker`).
- The scheduler, leases, and the assignment state machine.
- The context compiler (bounded, deterministic context bundles).
- Execution contracts, permissions, and budgets.
- Coordination of execution, delegating execution mechanics to the top-level `runtime` module (`runtime/worker`).
- Workspace and Git worktree management.
- Compute-node registry and eligibility.
- Integration and human gates.
- Operational telemetry.
- The job-learning engine (see [aa-joblearn.md](../../../kernel/aa-joblearn.md)).
- The control-plane API.

## Must not

- Perform file transfer (use `sync`).
- Store canonical history (use `memory`).
- Call models directly (use `inference` over RPC).
- Open a remote shell or arbitrary remote execution (use a capability-scoped runtime adapter).
- Import `sync`, `forge`, or `inference` code; only RPC.

## Interfaces

- Control-plane API for `ui` and `visualizer`.
- RPC clients to `inference`, `sync`, and `forge`.
- `obsv` host APIs.
- `memory` query and record APIs.

## Boundaries

Current placement is not final ownership. The intended home of each package, and
the constraints on transitional packages, are recorded in
[transitional-boundaries.md](./transitional-boundaries.md).

## Submodules

| Submodule | Responsibility |
|---|---|
| [`api`](./api.md) | Package api exposes the kernel control plane as an in-process service. |
| [`context`](./context.md) | Package context compiles deterministic, bounded context bundles. |
| [`contract`](./contract.md) | Package contract builds immutable, least-privilege execution contracts. |
| [`gate`](./gate.md) | Package gate evaluates deterministic and human gates before work is accepted. |
| [`joblearn`](./joblearn.md) | Package joblearn turns completed work into evidence and evidence into learning candidates. |
| [`joblearn/attribution`](./joblearn-attribution.md) | Package attribution links completed attempts to their work packages, roles, and workers, and normalizes evidence into versioned scores. |
| [`joblearn/backtest`](./joblearn-backtest.md) | Package backtest compares a learned policy against a deterministic baseline over a bounded set of labeled historical samples. |
| [`joblearn/baseline`](./joblearn-baseline.md) | Package baseline adapts the governed inference recommender to a backtest baseline. |
| [`joblearn/candidates`](./joblearn-candidates.md) | Package candidates derives learning candidates from attributed outcomes. |
| [`joblearn/distill`](./joblearn-distill.md) | Package distill synthesizes scoped subagent artifacts from trace evidence. |
| [`joblearn/experience`](./joblearn-experience.md) | Closed learning loop: verified results → reusable lessons with learning states and mentorship. |
| [`joblearn/features`](./joblearn-features.md) | Package features extracts step-level, content-free features from a trace. |
| [`joblearn/gate`](./joblearn-gate.md) | Package gate governs learned capabilities with evidence gates. |
| [`joblearn/promote`](./joblearn-promote.md) | Package promote persists learning candidates as CANDIDATE memory records. |
| [`joblearn/route`](./joblearn-route.md) | Package route exposes learned routing as a non-authoritative hint. |
| [`joblearn/similarity`](./joblearn-similarity.md) | Package similarity clusters attributed outcomes deterministically and detects conflicting learning candidates. |
| [`joblearn/tracesource`](./joblearn-tracesource.md) | Package tracesource is the pure seam through which joblearn consumes bounded per-step traces. |
| [`scheduler`](./scheduler.md) | Runs the supervised control cycle, and owns the assignment state machine, dispatch readiness, concurrency policy, and leases. |
| [`allocator/planner`](./allocator-planner.md) | The allocator's planning stage: work packages, dependency graph, and deterministic dispatch readiness. |
| [`allocator/role_allocator`](./role-allocator.md) | Deterministic worker selection by role and capability, with rejection reasons. |
| [`allocator/model_allocator`](./model-allocator.md) | Deterministic model selection from `registry/models`, with an escalation target. |
| [`allocator/compute_allocator`](./compute-allocator.md) | Worker count/parallelism/placement, justified and bounded. |
| [`allocator/routing`](./routing.md) | Model-locality/tier routing policy and the budget gate. |
| [`telemetry`](./telemetry.md) | Package telemetry records bounded execution evidence and derives deterministic learning candidates. |

