# ADR-0143 — Decide kernel `intake` and `contract` ownership (A5)

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [issue-impl-sep28 Issue 1](../issues/imp-1-canonical-architecture-and-contracts.md) (A5) |
| **Scope** | update phase · issue-impl-sep28 (Stage 1) |
| **Supersedes** | [ADR-0136](./ADR-0136-kernel-transitional-package-placement.md) for `intake` and `contract` only |
| **Executed** | [issue-impl-sep28 Issue 8](../issues/imp-8-scheduling-multi-agent-execution.md) (stage 3.4): `kernel/intake` → `runtime/intake` |

## Context

The architecture refactor left `context`, `contract`, `gate`, `telemetry`,
`joblearn`, `api`, and `intake` in `kernel/` and recorded their *intended* homes
in [kernel transitional boundaries](../modules/kernel/transitional-boundaries.md)
([ADR-0136](./ADR-0136-kernel-transitional-package-placement.md)). Two of those
intents were never decided: `intake` is intended to move to `runtime`, and
`contract` "may move under the allocator or become a shared kernel package".

The ten-issue sprint needs the decision, not the possibility, so later issues
plan against a fixed ownership. Today both packages are leaves with a single
caller: `kernel/scheduler` imports `kernel/contract` and `kernel/intake`
(`kernel/scheduler/scheduler.go`). The layering rules in
`tools/archtest` allow `kernel` to import `runtime`, but **not** the reverse, so
a package that `runtime` must import cannot live in `kernel`.

## Decision

- **`intake` → `runtime`.** Result intake is execution mechanics: validating
  untrusted result envelopes and deduplicating attempts. Its home is the
  top-level `runtime` module. The move is executed in [ISS-IMP-8](../issues/imp-8-scheduling-multi-agent-execution.md)
  (stage 3.4), where the scheduler caller is in scope. Until the move,
  `kernel/intake` stays a leaf with **no dependents other than the scheduler**.
- **`contract` stays a shared kernel package.** The execution-contract builder
  is a control-plane, leaf concern: it grants the intersection of requested and
  permitted capabilities and carries budget/authority ([ADR-0034](./ADR-0034-execution-contracts-grant-the-intersection-of-requested-and.md)).
  It is consumed by `kernel/scheduler` (and the allocator's budget step); the
  runtime receives the **built contract as data** across the worker-adapter seam
  ([ADR-0036](./ADR-0036-runtime-access-is-behind-an-adapter-interface-the-fake-is.md))
  and never imports the builder. `kernel/contract` is therefore finalized as
  `kernel/contract` (no longer transitional).

## Rationale

- Intake validates results produced by execution; grouping it with worker
  execution keeps the untrusted-input boundary in one module.
- The contract builder encodes control-plane policy (authority, budgets,
  capability intersection). Moving it to `runtime` would either export policy
  into the execution module or force a `runtime → kernel` dependency that the
  architecture forbids.
- Both packages are leaves with one caller, so the change is a relocation, not a
  redesign.

## Alternatives and consequences

- **Move `contract` under `allocator`:** rejected — the builder is also the
  authority seam for the scheduler, and `allocator` is the planning stage, not
  the contract authority.
- **Move `intake` immediately:** rejected here — the move is behaviour-preserving
  only if done with its caller; it is scheduled in stage 3.4.
- Consequence: [kernel transitional boundaries](../modules/kernel/transitional-boundaries.md)
  records `contract` as final and `intake` as decided-to-move; the move slice
  updates the module docs and this ADR's index entry.

## Related

- [ADR-0136](./ADR-0136-kernel-transitional-package-placement.md) (superseded for `intake` and `contract`)
- [Kernel transitional boundaries](../modules/kernel/transitional-boundaries.md)
- [Architecture §4.6 boundaries and integration points](../architecture/README.md#46-boundaries-and-integration-points)
- [ADR index](./README.md)
