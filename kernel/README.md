# aa-kernel

The deterministic control plane: planning, roles, orchestration, execution, gates, and job learning.

## What it does

- **allocator/planner** — work packages, a validated dependency graph, cycle detection, and readiness (the allocator's planning stage).
- **allocator/role_allocator** — deterministic worker selection by role, capability, and (transitionally) trade, with rejection reasons.
- **allocator/model_allocator**, **allocator/compute_allocator** — reserved allocation boundaries (no behavior yet).
- **control** — the assignment state machine and expiring leases.
- **contract** — least-privilege execution contracts: the intersection of requested and permitted capabilities, with denials recorded.
- **context** — a pure, bounded, digest-stable context compiler.
- **gate** — tests, human, and composite gates.
- **intake** — validation and idempotent deduplication of untrusted results.
- **telemetry** — bounded execution evidence and deterministic learning candidates.
- **orchestrator** — the supervised cycle: ready → select → contract → context → lease → run → intake → gates → accept → telemetry, recorded through a `Sink`.
- **joblearn** — deterministic job learning: attribution and versioned scoring, candidate generation with provenance, feature-based similarity and conflict, evidence-gated capabilities, baseline backtests, non-authoritative routing hints with a fallback, and candidate persistence as `CANDIDATE` memory records through a `Sink`. A module update adds trace distillation: a pure `tracesource` seam, content-free step features, trace-derived attribution, scoped artifact distillation, trace-derived backtests against the governed `sifter` recommender seam, and trace-fed evidence gates.
- **api** — an in-process control-plane service (disabled by default).

## Boundaries

The kernel imports `contracts`, `registry`, `runtime`, `obsv`, `memory`, and `toolbox`. Execution mechanics live in `aa-runtime`; the worker adapter it uses is `runtime/worker`. It reaches `sync`, `forge`, and `sifter` over RPC. No model is called in the control path.

## Status

Phase 9 (`ph9-joblearn`) adds deterministic job learning whose candidates persist as `CANDIDATE` memory records through a memory-backed promotion `Sink`. Execution is disabled by default; real runtime adapters, the `obsv` observation service, and the orchestrator's memory-backed `Sink` remain deferred. See `docs/phases/ph9-joblearn/summary.md` and `docs/phases/ph3-kernel/summary.md`.

Module update: [plan.md](../docs/modules/kernel/updates/joblearn-trace-distillation/plan.md) · [summary.md](../docs/modules/kernel/updates/joblearn-trace-distillation/summary.md) — trace distillation and evaluation (`ISS-LEARN-1`).

## Directive

[aa-kernel.md](aa-kernel.md) · [aa-joblearn.md](aa-joblearn.md)
Architecture: [../docs/architecture/README.md](../docs/architecture/README.md)
