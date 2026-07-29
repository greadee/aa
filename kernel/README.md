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
- **api** — an in-process control-plane service (disabled by default).

## Boundaries

The kernel imports only `contracts`, `obsv`, `memory`, and `toolbox`. It reaches `sync`, `forge`, and `sifter` over RPC. No model is called in the control path.

## Status

Phase 3 (`ph3-kernel`). Execution is disabled by default. Real runtime adapters, the `obsv` observation service, and a memory-backed `Sink` are deferred; see `docs/phases/ph3-kernel/summary.md`.

## Directive

[aa-kernel.md](aa-kernel.md) · [aa-joblearn.md](aa-joblearn.md)
Architecture: [../docs/architecture/README.md](../docs/architecture/README.md)
