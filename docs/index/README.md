# aa documentation

This is the entry point to the durable knowledge record for **aa**. Every document is reachable from an index; indexes orient and link, they do not duplicate.

GitHub is the durable **execution** record. This `docs/` tree is the durable **knowledge** record. Agent conversations are temporary context — knowledge that is expensive to rediscover graduates into this tree.

## Start here

| Index | Purpose |
|---|---|
| [architecture.md](architecture.md) | System architecture, module boundaries, and decisions |
| [modules.md](modules.md) | Project history: per-module and per-submodule documentation |
| [phases.md](phases.md) | Phase roadmap and phase documentation |
| [features.md](features.md) | Cross-cutting and per-module feature documentation |
| [sprints.md](sprints.md) | Sprint and slice conventions |
| [../issues/README.md](../issues/README.md) | Durable issue documents and umbrella changes |

## Canonical documents

- **Optimized architecture:** [../architecture/README.md](../architecture/README.md)
- **Architecture Decision Records:** [../adr/README.md](../adr/README.md)
- **Module documentation (project history):** [../modules/README.md](../modules/README.md)
- **Terminology and state registry:** [../reference/terminology.md](../reference/terminology.md)
- **Process and GitHub OS:** [../github-os/README.md](../github-os/README.md)
- **Phase plans and summaries:** [../phases/README.md](../phases/README.md)
- **Update phases:** [../updates/](../updates/)

## Documentation tree

```
docs/
├── index/          entry points and cross-cutting indexes
├── architecture/   system architecture and decisions
├── adr/            global Architecture Decision Records (ADR-NNNN)
├── modules/        project history: per-module and submodule documentation
├── github-os/      process, planning, review, audit, and GitHub workflow
├── phases/         numbered major additions: plans, diagrams, summaries
├── updates/        unnumbered update phases: refactors, edits, corrections
├── issues/         durable issue documents and umbrella changes
├── features/       feature-level documentation
└── reference/      terminology, schemas, CLI, APIs, configuration
```

## Related

- Repository readme: [../../README.md](../../README.md)
- Primary agent directive: [../../AA.md](../../AA.md)
