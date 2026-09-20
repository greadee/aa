# aa-memory

The system of record: canonical, portable, rebuildable institutional knowledge.

## What it does

- **Canonical store** (`memory/store`): records write to `.aa-project/records/<kind>/<id>.json`; events append to `.aa-project/events.jsonl`. Writes are atomic and hashed; ingestion is idempotent.
- **Projection** (`memory/projection`): a derived, disposable view. `Rebuild` reconstructs it from canonical records to an identical digest.
- **Query** (`memory/query`): deterministic reads plus derived work, project, and job histories.
- **Repositories** (`memory/repo`): typed issue, strategy, and memory-record repositories with the memory lifecycle promotion state machine. Memory records enter as `CANDIDATE` and promote only through explicit, validated transitions.
- **Facade**: `memory.Open(root)` wires store, projection, repositories, and query.

## Portable layout

```
.aa-project/
├── manifest.json          project record
├── events.jsonl           append-only event log
├── records/<kind>/<id>.json
└── retention.json         event retention floor
```

## Status

Phase 2 (`ph2-memory`). Directory records are canonical; projections are disposable. A SQLite projection adapter is deferred; the `Projection` interface is ready for it.

## Directive

[aa-memory.md](aa-memory.md) · [aa-issue.md](aa-issue.md) · [aa-strgy.md](aa-strgy.md)
Architecture: [../docs/architecture/README.md](../docs/architecture/README.md)
