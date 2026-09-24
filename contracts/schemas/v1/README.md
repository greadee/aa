# Contract schemas v1

JSON Schema (draft 2020-12) definitions for all cross-module objects. These files are the source of truth; bindings in `go/`, `typescript/`, and `python/` must match them, guarded by the conformance suite.

## Objects

| Schema | Purpose |
|---|---|
| `common.schema.json` | Shared definitions: envelope, identifiers, actor, provenance, budget, state enums, memory lifecycle |
| `event.schema.json` | Canonical orchestration/execution event and event-type taxonomy |
| `work-package.schema.json` | A bounded, verifiable unit of work |
| `task-graph.schema.json` | Versioned dependency graph over work packages |
| `execution-contract.schema.json` | Immutable least-privilege execution authority and capability vocabulary |
| `result-envelope.schema.json` | Untrusted result of an attempt |
| `telemetry.schema.json` | Bounded execution evidence (operational, not canonical) |
| `trace.schema.json` | Bounded, redacted per-step learning evidence (`x-contract-version` 1.1) |
| `project-record.schema.json` | Canonical project manifest |
| `memory-record.schema.json` | Knowledge record in the memory hierarchy |
| `issue.schema.json` | Canonical issue / deficiency / finding |
| `strategy.schema.json` | Validated reusable approach |
| `route.schema.json` | Model/compute routing request and response (`route_request`, `route_response`) |
| `tool-manifest.schema.json` | Tool / plugin / MCP declaration |
| `workflow.schema.json` | Declarative workflow definition |

## Conventions

- `$id` is `https://github.com/greadee/aa/contracts/schemas/v1/<name>.schema.json`.
- `x-contract-version` is the `MAJOR.MINOR` version of the document.
- Objects are open (`additionalProperties: true`) unless deliberately closed; consumers ignore unknown properties.
- Identifiers are opaque and stable; timestamps are RFC 3339 UTC.
- Observation events are owned by `aa-obsv`; `event.schema.json` covers orchestration/execution events only. `trace.schema.json` is derived learning evidence, not the observation protocol.

## Evolution

See [../../POLICY.md](../../POLICY.md). The observation protocol is referenced, not redefined here. Minors are tracked per document, so a newly added document may declare a higher `x-contract-version` than older documents in the same major directory.
