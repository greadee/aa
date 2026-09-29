# ADR-0131 — Introduce a top-level registry for durable definitions

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) (D-4, D-5, D-11) |
| **Scope** | update phase · architecture-refactor-1 |

## Context

aa's durable definitions (roles, models, teams, capabilities, routines, policies)
were scattered: roles and a capability-to-role mapping lived in `kernel/registry`;
the execution-capability vocabulary lived in `kernel/contract` and `contracts`;
teams, models, routines, and policies had no home. Runtime instances (workers,
crews, assignments, executions, projects, workflows) are a different concern from
the definitions they instantiate.

## Decision

Create a top-level **`registry/`** module that owns durable definitions only:
`roles`, `models`, `teams`, `capabilities`, `routines`, and `policies`. It holds
no runtime instances and performs no allocation, scheduling, or execution. The
existing role and capability vocabularies are relocated into it behavior-
preservingly; the model/team/routine/policy specification types are reserved for
contracts v2. `kernel/registry` is retired by the refactor: definitions move to
`registry/`, selection moves to `kernel/allocator`, and worker instances move to
`runtime/worker`.

## Rationale

One owner for durable definitions; a clean division between definitions and
runtime instances; and a home for teams/models/routines/policies without the
`agents/` or `organization/` modules the handoff's first draft proposed.

## Alternatives and consequences

- **`agents/` + `organization/` modules (handoff draft):** rejected — more
  top-level concepts than needed; the registry plus runtime covers both.
- **Keep definitions in `kernel/registry`:** rejected — kernel owns coordination,
  not the vocabulary other modules also need.
- Consequence: `kernel` now depends on `registry`; architecture layering and
  `tools/archtest` were updated accordingly.

## Related

- [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) D-4, D-5, D-11
- Module documentation: [docs/modules/registry](../modules/registry/README.md)
- [ADR index](./README.md)
