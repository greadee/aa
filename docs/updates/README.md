# Update phases

Work history for **unnumbered update phases**: refactors, edits, corrections, and
module updates that are not major additions. Numbered [phases](../phases/README.md)
are reserved for major additions only.

Each update phase carries the phase discipline — `plan.md`, `plan.uml` (at start)
and `summary.md`, `summary.uml` (at end) — with lightweight, change-scoped diagrams.
Module-level plans and summaries live under
[docs/modules/<module>/updates/](../modules/README.md) and are linked from the
owning update phase.

| Update phase | Scope | Plan | Summary |
|---|---|---|---|
| [architecture-refactor-1](./architecture-refactor-1/plan.md) | Target boundaries, contracts v2, `registry`/`runtime`/`ui`, documentation restructure | [plan](./architecture-refactor-1/plan.md) | pending |

## Historical module updates

Module updates delivered on retained phase branches before update phases were
formalized. Their plans and summaries live under project history
(`docs/modules/<module>/updates/`); this is their work-history index.

| Module update | Module | Plan | Summary |
|---|---|---|---|
| Trace contract (`ISS-TRACE-1`) | contracts | [plan](../modules/contracts/updates/trace-contract/plan.md) | [summary](../modules/contracts/updates/trace-contract/summary.md) |
| Trace store (`ISS-TRACE-2`) | memory | [plan](../modules/memory/updates/trace-store/plan.md) | [summary](../modules/memory/updates/trace-store/summary.md) |
| Observation substrate (`ISS-OBSV-1`) | obsv | [plan](../modules/obsv/updates/observation-substrate/plan.md) | [summary](../modules/obsv/updates/observation-substrate/summary.md) |
| Production RPC wiring (`ISS-SIFTER-1`) | sifter | [plan](../modules/sifter/updates/production-rpc-wiring/plan.md) | [summary](../modules/sifter/updates/production-rpc-wiring/summary.md) |
| obsv transport adoption / AAV de-duplication (`ISS-OBSV-2`) | visualizer | [plan](../modules/visualizer/updates/obsv-adoption/plan.md) | [summary](../modules/visualizer/updates/obsv-adoption/summary.md) |
| Trace distillation and evaluation (`ISS-LEARN-1`) | kernel (joblearn) | [plan](../modules/kernel/updates/joblearn-trace-distillation/plan.md) | [summary](../modules/kernel/updates/joblearn-trace-distillation/summary.md) |
| Issues, sub-problems, and module updates | github-os | [plan](../modules/github-os/updates/issues-and-module-updates/plan.md) | [summary](../modules/github-os/updates/issues-and-module-updates/summary.md) |
