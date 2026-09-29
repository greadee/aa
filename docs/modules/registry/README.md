# aa-registry — module documentation

> Project history for the `aa-registry` module. Part of [docs/modules](../README.md).

**Source** — [`registry/`](../../../registry/) · **Updates** — [`updates/`](./updates/)

## Responsibility

The durable definition store: reusable specifications for roles, models, teams,
capabilities, routines, and policies. It holds no runtime instances and performs
no allocation, scheduling, or execution.

## Owns

- Durable definitions: roles, models, teams, capabilities, routines, and policies.

## Must not

- Hold runtime instances (workers, crews, assignments, executions, projects, workflows).
- Execute, schedule, or allocate work.
- Define cross-module wire schemas (that is `contracts`); specification types are
  mirrored as contracts in contracts v2.

## Submodules

| Submodule | Responsibility |
|---|---|
| [`roles`](./roles.md) | Durable role definitions and the deterministic capability-to-role mapping |
| [`capabilities`](./capabilities.md) | The closed execution-capability vocabulary |
| [`models`](./models.md) | Reserved: model specification (contracts v2) |
| [`teams`](./teams.md) | Reserved: team (durable role grouping) specification |
| [`routines`](./routines.md) | Reserved: organization routine specification |
| [`policies`](./policies.md) | Reserved: policy-definition catalogue (definitions, not applicability) |
