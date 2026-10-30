# Phase index

Numbered phases are reserved for **major additions**. Refactors, edits, corrections,
and module updates are unnumbered **update phases**, documented under
[docs/updates/](../updates/README.md), with their module-level plans and summaries
under [docs/modules/<module>/updates/](../modules/README.md).

Each phase is a GitHub milestone with a tracking issue, a phase branch, a Draft
phase PR, child issues and PRs, and a phase-end audit.

## Documentation convention

Every phase creates a folder under `docs/phases/` named `ph{N}-{scope}` (for example `ph0-scaffold`, `ph1-contracts`, `ph2-memory`). The branch name is **identical** to the folder name. Each phase folder contains four required artifacts:

| File | Purpose |
|---|---|
| `plan.md` | Phase plan, broken into slices (each slice is an individual commit) |
| `plan.uml` | Infrastructure/sequence/class diagrams for the planned phase |
| `summary.md` | Phase summary: what was delivered, decisions affirmed, deviations and why |
| `summary.uml` | Copy of `plan.uml`, edited to the as-built end state |

Architecture decisions are recorded in the global [ADR store](../adr/README.md), not
per phase.

The plan is authored at the **start** of the phase. The summary is authored at the **end**, by copying and editing the plan diagrams to the finished state, affirming decisions that were correct, and recording any deviation with its reason.

## Phase roadmap

| Phase | Folder | Scope | Status |
|---|---|---|---|
| 0 | `ph0-scaffold` | Foundation, governance, repository and documentation scaffolding | Delivered |
| 1 | `ph1-contracts` | Contract spine (`aa-contracts`) | Delivered |
| 2 | `ph2-memory` | `aa-memory` system of record | Delivered |
| 3 | `ph3-kernel` | `aa-kernel` control plane | Delivered |
| 4 | `ph4-sifter` | `runtime/inference` model provider/execution integration (formerly `aa-sifter`; compute moved to `kernel/allocator`) | Delivered |
| 5 | `ph5-forge` | `aa-forge` git/GitHub orchestration | Delivered |
| 6 | `ph6-sync` | `aa-sync` remote/offline/parallel completion | Delivered |
| 7 | `ph7-toolbox` | `aa-toolbox` plugins/MCP and configurable workflows | Delivered |
| 8 | `ph8-visualizer` | `aa-visualizer` replay/layout/browse core (3D surface deferred) | Delivered |
| 9 | `ph9-joblearn` | Job learning and optimization loop | Delivered |
| 10 | `phase10a-controlplane`, `phase10b-console` | Control plane and console — unified interface | Delivered |
| 11 | `ph11-postreq` | Post-console modules: sync, obsv, visualizer, registry extras, lifecycle | Delivered |
| 12 | `p12-release` | Hardening, real authn/authz, security, performance, audits | Next |
| 13 | `p13-release` | Release gates, packaging/installers, `CHANGELOG.md`, final docs completion | Planned |

The docs foundation is an **update phase** (`docs-foundation`), not a numbered
phase; its plan is [../updates/docs-foundation/plan.md](../updates/docs-foundation/plan.md).
The Phase 12 and Phase 13 plans are still pending; this index does not publish
them early.

The full phase plan with goals, deliverables, exit criteria, and testing is in the [architecture document](../architecture/README.md#11-end-to-end-phase-plan).
