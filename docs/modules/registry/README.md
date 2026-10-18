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
| [`models`](./models.md) | Model definition registry (contracts v2 `ModelSpec`) with a mirrored default catalog |
| [`teams`](./teams.md) | Durable team (role grouping) definition registry |
| [`routines`](./routines.md) | Durable organization routine definition registry |
| [`policies`](./policies.md) | Durable policy-definition catalogue (definitions, not applicability) |
