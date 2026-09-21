# aa-kernel

The deterministic control plane: planning, roles, orchestration, execution, gates, and job learning.

## What it does

- **registry** — roles, trades, and worker instances with deterministic selection and rejection reasons.
- **plan** — work packages, a validated dependency graph, cycle detection, and readiness.
- **control** — the assignment state machine and expiring leases.
- **contract** — least-privilege execution contracts: the intersection of requested and permitted capabilities, with denials recorded.
- **context** — a pure, bounded, digest-stable context compiler.
- **runtime** — a provider-neutral adapter interface with a scripted deterministic fake and a disabled adapter.
- **gate** — tests, human, and composite gates.
- **intake** — validation and idempotent deduplication of untrusted results.
- **telemetry** — bounded execution evidence and deterministic learning candidates.
- **orchestrator** — the supervised cycle: ready → select → contract → context → lease → run → intake → gates → accept → telemetry, recorded through a `Sink`.
- **joblearn** — deterministic job learning: attribution and versioned scoring, candidate generation with provenance, feature-based similarity and conflict, evidence-gated capabilities, baseline backtests, non-authoritative routing hints with a fallback, and candidate persistence as `CANDIDATE` memory records through a `Sink`.
- **api** — an in-process control-plane service (disabled by default).

## Boundaries

The kernel imports only `contracts`, `obsv`, `memory`, and `toolbox`. It reaches `sync`, `forge`, and `sifter` over RPC. No model is called in the control path.

## Status

Phase 9 (`ph9-joblearn`) adds deterministic job learning whose candidates persist as `CANDIDATE` memory records through a memory-backed promotion `Sink`. Execution is disabled by default; real runtime adapters, the `obsv` observation service, and the orchestrator's memory-backed `Sink` remain deferred. See `docs/phases/ph9-joblearn/summary.md` and `docs/phases/ph3-kernel/summary.md`.

Module update (planned): [joblearn/module-upd-plan.md](joblearn/module-upd-plan.md) — trace distillation and evaluation (`ISS-LEARN-1`).

## Directive

[aa-kernel.md](aa-kernel.md) · [aa-joblearn.md](aa-joblearn.md)
Architecture: [../docs/architecture/README.md](../docs/architecture/README.md)
