# aa-memory — module documentation

> Project history for the `aa-memory` module. Part of [docs/modules](../README.md).

**Directive** — [`aa-memory.md`](../../../memory/aa-memory.md) · **Source** — [`memory/`](../../../memory/) · **Updates** — [`updates/`](./updates/)

## Responsibility

Be the canonical, portable, rebuildable system of record. Everything durable about work, projects, issues, and strategies lives here. The SQLite database is a projection; the records are the truth.

## Owns

- Canonical records and the portable `.aa-project/` directory layout.
- Rebuildable SQLite projections and rebuild-equivalence.
- Work history, git history, job history, and project history.
- The issue repository (see [aa-issue.md](../../../memory/aa-issue.md)).
- The strategy repository (see [aa-strgy.md](../../../memory/aa-strgy.md)).
- The memory hierarchy and deterministic promotion lifecycle.
- Provenance and retention.
- Deterministic query API.

## Must not

- Execute work, schedule, or call models.
- Treat the SQLite projection as authoritative.
- Store raw prompts, source, credentials, or terminal transcripts.

## Interfaces

- Record read/write.
- Query API consumed by `kernel`, `visualizer`, `ui`, and `forge`.
- Projection rebuild.

## Submodules

| Submodule | Responsibility |
|---|---|
| [`projection`](./projection.md) | Package projection provides aa-memory's derived, disposable views over canonical records. |
| [`query`](./query.md) | Package query provides deterministic read access to aa-memory's projection and derives work, project, and job histories from the event log. |
| [`repo`](./repo.md) | Package repo provides typed repositories over the canonical store and the memory lifecycle promotion state machine. |
| [`retention`](./retention.md) | Package retention trims a projected trace view deterministically. |
| [`store`](./store.md) | Package store implements aa-memory's canonical, portable record store. |

