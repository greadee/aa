# aa-memory.md

Agent directive for the `memory` module (aa-memory).

## Responsibility

Be the canonical, portable, rebuildable system of record. Everything durable about work, projects, issues, and strategies lives here. The SQLite database is a projection; the records are the truth.

## Owns

- Canonical records and the portable `.aa-project/` directory layout.
- Rebuildable SQLite projections and rebuild-equivalence.
- Work history, git history, job history, and project history.
- The issue repository (see [aa-issue.md](aa-issue.md)).
- The strategy repository (see [aa-strgy.md](aa-strgy.md)).
- The memory hierarchy and deterministic promotion lifecycle.
- Provenance and retention.
- Deterministic query API.

## Must Not

- Execute work, schedule, or call models.
- Treat the SQLite projection as authoritative.
- Store raw prompts, source, credentials, or terminal transcripts.

## Interfaces

- Record read/write.
- Query API consumed by `kernel`, `visualizer`, `ui`, and `forge`.
- Projection rebuild.

## Rules

1. Directory records are canonical; projections are rebuildable and must prove rebuild-equivalence.
2. Ingestion is idempotent; identities are stable and versioned.
3. Validation fails closed.
4. Promotion is deterministic and evidence-based; `EPHEMERAL → CANDIDATE → VALIDATED → ACTIVE → SUPERSEDED → ARCHIVED`.
5. History is append-oriented; supersede, never silently overwrite.

## Canonical references

- [Architecture](../docs/architecture/README.md)
- [Terminology and memory hierarchy](../docs/reference/terminology.md)
