# aa

**aa** is an end-to-end agentic terminal: a modular, local-first platform for observing, planning, executing, verifying, and continuously learning from software engineering work performed by humans and AI agents across one or many machines.

The platform separates concerns into independently versioned modules that communicate through a single, versioned contract layer. A deterministic control plane owns state and coordination; models are workers behind a router; every unit of work is observable, replayable, and attributable.

> **Migration note.** This repository is a clean import of the **aa** project from an earlier repository. The `ph0`–`ph3` history was re-created during the migration as reviewed pull requests gated by CI, so commit chronology reflects the migration timeline rather than the original development dates. The architecture, code, and phase plan are unchanged. See [docs/architecture/README.md](docs/architecture/README.md).

## Modules

| Module | Responsibility |
|---|---|
| `contracts` | Single source of truth for schemas, RPC, and generated bindings |
| `obsv` | Product-neutral work observation: protocol, emitter, transport, journal, reports |
| `kernel` | Control plane: planning, roles, orchestration, execution, gates, job learning |
| `memory` | System of record: work/git/job/project history, issue and strategy repositories, promotion |
| `sync` | Remote sync, offline file sharing, and parallelization across machines |
| `sifter` | Prompt decoding and compute/model routing across local and cloud models |
| `forge` | Git and GitHub orchestration: projects, issues, PRs, checkpoints, audits, releases |
| `toolbox` | Tool, plugin, and MCP registry plus configurable workflow runtime |
| `visualizer` | Time-travel work history, debugging, and 3D graph visualization |
| `ui` | Unified interface: CLI and TUI surfaces (pluggable) |

## Documentation

The durable knowledge record lives in [`docs/`](docs/index/README.md). The optimized architecture and end-to-end plan is [docs/architecture/README.md](docs/architecture/README.md). Process, planning, and GitHub workflow conventions live in [docs/github-os/](docs/github-os/README.md).

## Status

Pre-implementation. The architecture and phase plan are complete; work proceeds phase by phase. See [docs/index/phases.md](docs/index/phases.md).
