# Architecture index

| Doc | Description |
|---|---|
| [Current state](../current-state.md) | What is composed and working today |
| [Current diagrams](../architecture/diagrams.md) | As-built composition, contracts, data flow, deployment, and E2E sequence |
| [Optimized architecture](../architecture/README.md) | Full end-to-end architecture, module map, layering, contracts, memory model, tooling, testing, and the phase plan |
| [Decisions](../architecture/README.md#12-architectural-decisions) | Architectural decision records with rationale |
| [Terminology](../reference/terminology.md) | Canonical glossary and state/event registry |
| [Module map](modules.md) | Current modules and submodules |
| [Contracts](../../contracts/README.md) | Cross-module schemas and RPC |

Architecture decisions are recorded as ADRs in the global [ADR store](../adr/README.md)
with `ADR-NNNN` ids; historical `ADR-Px-NNN` ids are backfilled and mapped in the
[ADR index](../adr/README.md). The core platform decisions **D1–D22** are summarized
in the architecture document (§12.1).

The [architecture document](../architecture/README.md) is the target design and
decision record. The [current state](../current-state.md) page is the as-built
boundary; when they differ, the current-state page and the code win for today's
behavior.
