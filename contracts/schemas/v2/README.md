# Contract schemas v2

JSON Schema (draft 2020-12) definitions for all cross-module objects. These files
are the source of truth; bindings in `go/`, `typescript/`, and `python/` must
match them, guarded by the conformance suite.

V2 is a major generation. It adds the durable-definition and planning objects
(`role_spec`, `model_spec`, `team_spec`, `work_plan`, `execution_plan`,
`routine`) and drops the `trade` field from the agent ontology (Role × Model).
Every document declares `x-contract-version` `2.0`.

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
| `trace.schema.json` | Bounded, redacted per-step learning evidence |
| `project-record.schema.json` | Canonical project manifest |
| `memory-record.schema.json` | Knowledge record in the memory hierarchy |
| `issue.schema.json` | Canonical issue / deficiency / finding |
| `strategy.schema.json` | Validated reusable approach |
| `route.schema.json` | Model/compute routing request and response (`route_request`, `route_response`) |
| `tool-manifest.schema.json` | Tool / plugin / MCP declaration |
| `workflow.schema.json` | Declarative workflow definition |
| `role-spec.schema.json` | Durable role specification (v2) |
| `model-spec.schema.json` | Durable model specification (v2) |
| `team-spec.schema.json` | Durable team specification (v2) |
| `work-plan.schema.json` | The Planner's output (v2) |
| `execution-plan.schema.json` | The allocator's output (v2) |
| `routine.schema.json` | Organization-owned reusable process (v2) |

## Conventions

- `$id` is `https://github.com/greadee/aa/contracts/schemas/v2/<name>.schema.json`.
- `x-contract-version` is `2.0` for every document in this generation.
- Objects are open (`additionalProperties: true`) unless deliberately closed; consumers ignore unknown properties.
- Identifiers are opaque and stable; timestamps are RFC 3339 UTC.
- Observation events are owned by `aa-obsv`; `event.schema.json` covers orchestration/execution events only.

## Evolution

See [../../POLICY.md](../../POLICY.md). Generation **v1** was removed from the
active path in the ten-issue sprint (Issue 1 / A4) and is retained only in Git
history; **v2** is the sole active generation.
