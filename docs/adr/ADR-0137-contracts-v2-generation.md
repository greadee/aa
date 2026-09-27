# ADR-0137 — Contracts v2 generation

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) (D-7) |
| **Scope** | update phase · architecture-refactor-1 |

## Context

The architecture refactor introduces durable-definition objects (roles, models,
teams, routines), planning objects (WorkPlan, ExecutionPlan), and retires
`trade`. `contracts/POLICY.md` says a breaking change requires a new **major**
directory; both generations may coexist during migration.

## Decision

Add a new major generation under `contracts/schemas/v2/` with bindings
`contracts/go/v2/`, `contracts/typescript/v2/`, and
`contracts/python/aa_contracts/v2.py`. It declares `x-contract-version` `2.0`
and adds `role_spec`, `model_spec`, `team_spec`, `work_plan`, `execution_plan`,
and `routine`. The `trade` field is removed from the v2 agent ontology
(Role × Model). `v1` remains as the previous generation until consumers migrate
in a later slice of this update phase.

## Rationale

Following the repo's own versioning policy keeps migrations explicit and
reversible: v1 stays valid while consumers move, and the conformance suite
guards both generations.

## Alternatives and consequences

- **In-place breaking change in `schemas/v1`:** rejected — contradicts POLICY,
  which requires a new major directory and a migration note.
- **No new objects:** rejected — the target architecture needs them.
- Consequence: `go/v1` and `go/v2` coexist; consumers move to `v2` in the wiring
  slice. `POLICY.md` records the 2.0 migration note.

## Related

- [architecture-refactor-1 plan](../updates/architecture-refactor-1/plan.md) D-7
- [contracts README](../../contracts/README.md), [schemas/v2](../../contracts/schemas/v2/README.md)
- [ADR index](./README.md)
