# aa documentation

This is the entry point to the durable knowledge record for **aa**. Every document is reachable from an index; indexes orient and link, they do not duplicate.

GitHub is the durable **execution** record. This `docs/` tree is the durable **knowledge** record. Agent conversations are temporary context — knowledge that is expensive to rediscover graduates into this tree.

## Audience paths

| Reader | Path |
|---|---|
| New engineer | [Current state](../current-state.md) → [Architecture](../architecture/README.md) → [Modules](modules.md) → [README quickstart](../../README.md#quickstart) |
| Operator | [Current state](../current-state.md) → [UI module](../modules/ui/README.md) → [Reference](../reference/README.md) → [Control-plane OpenAPI](../../contracts/openapi/control-plane-v1.yaml) |
| Contributor | [CONTRIBUTING.md](../../CONTRIBUTING.md) → [Definitions of Done](../github-os/definitions-of-done.md) → [GitHub OS](../github-os/README.md) → the module directive beside the code |
| Auditor | [Repository audits](../github-os/repository-audits.md) → [Phase index](phases.md) → [ADR index](../adr/README.md) → [Current state](../current-state.md) |

## Start here

| Index | Purpose |
|---|---|
| [Current state](../current-state.md) | What works today, what is partial, and what is deferred |
| [architecture.md](architecture.md) | Current architecture maps, decisions, and the target design |
| [modules.md](modules.md) | Project history: per-module and per-submodule documentation |
| [phases.md](phases.md) | Phase roadmap and phase documentation |
| [features.md](features.md) | Cross-cutting and per-module feature documentation |
| [sprints.md](sprints.md) | Sprint and slice conventions |
| [../issues/README.md](../issues/README.md) | Durable issue documents and umbrella changes |

## Canonical vs historical

**Canonical** documents describe the system as it is today. If they disagree with
code, they are bugs and must be fixed.

- [README.md](../../README.md), [CONTRIBUTING.md](../../CONTRIBUTING.md)
- [Current state](../current-state.md)
- [Architecture](../architecture/README.md) and the [ADR store](../adr/README.md)
- [Module documentation](../modules/README.md)
- [Reference](../reference/README.md)
- [GitHub OS](../github-os/README.md) and [Definitions of Done](../github-os/definitions-of-done.md)

**Historical** documents record what was planned or true at a point in time.
They are preserved as evidence and must not be read as current-state claims.

- [Phases](../phases/README.md) — numbered major additions
- [Update phases](../updates/README.md) — refactors, edits, corrections
- [Issue documents](../issues/README.md) — durable problem records
- Sprint plans and summaries linked from the [sprint index](sprints.md)

When a historical decision is superseded, add a dated note to that document and
link the replacement; do not rewrite the record.

## Canonical documents

- **Optimized architecture:** [../architecture/README.md](../architecture/README.md)
- **Architecture Decision Records:** [../adr/README.md](../adr/README.md)
- **Module documentation (project history):** [../modules/README.md](../modules/README.md)
- **Terminology and state registry:** [../reference/terminology.md](../reference/terminology.md)
- **Process and GitHub OS:** [../github-os/README.md](../github-os/README.md)
- **Definitions of Done:** [../github-os/definitions-of-done.md](../github-os/definitions-of-done.md)
- **Contributing:** [../../CONTRIBUTING.md](../../CONTRIBUTING.md)
- **Phase plans and summaries:** [../phases/README.md](../phases/README.md)
- **Update phases:** [../updates/](../updates/)

## Documentation tree

```
docs/
├── README.md        short entry that points here
├── current-state.md current behavior and known limitations
├── index/          entry points and cross-cutting indexes
├── architecture/   system architecture and decisions
├── adr/            global Architecture Decision Records (ADR-NNNN)
├── modules/        project history: per-module and submodule documentation
├── github-os/      process, planning, review, audit, and GitHub workflow
├── phases/         numbered major additions: plans, diagrams, summaries
├── updates/        unnumbered update phases: refactors, edits, corrections
├── issues/         durable issue documents and umbrella changes
├── features/       feature-level documentation
├── media/          repository media and asset manifest
└── reference/      terminology, schemas, CLI, APIs, configuration
```

## Related

- Repository readme: [../../README.md](../../README.md)
- Primary agent directive: [../../AA.md](../../AA.md)
