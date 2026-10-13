# ADR-0150 — Role, model, and compute allocation are independent

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [issue-impl-sep28 Issue 7](../issues/imp-7-allocation.md) |
| **Scope** | update phase · issue-impl-sep28 (Stage 4) |

## Context

Only role allocation carried behavior; `kernel/allocator/{model,compute}_allocator`
were reserved, the model registry was unpopulated, and `runtime/inference` still
owned routing and budget policy. Allocation had to stay independently testable
even when combined into one execution plan.

## Decision

- **Three independent decisions.** Role = expertise (`registry/roles`, via
  `roles.RolesFor` and the role allocator); model = model identity / reasoning
  tier (`kernel/allocator/model_allocator` over `registry/models`); compute =
  worker count, parallelism, budgets, placement (`compute_allocator`). They are
  combined only in `allocator.Allocate`.
- **Model allocation is deterministic and owns identity.** `Allocate` filters by
  locality, capability subset, and cost cap, ranks by lowest cost, then latency,
  then id, and returns a reasoning-capable **escalation** target when the primary
  is not reasoning-capable. It never decides worker count or budgets.
- **Compute allocation is opt-in and justified.** Parallelism requires an
  explicit justification (`latency`, `quality`, `coverage`, `reliability`) and is
  bounded by the independent-unit count and a concurrency ceiling. Idle agents
  are never a reason to spawn workers. Compute never changes model identity.
- **The model registry is populated.** `registry/models.Registry` stores/serves
  contracts v2 `ModelSpec`s; `DefaultCatalog` mirrors the inference service's
  advisory catalog (the runtime source) as a version-controlled snapshot.
- **One execution plan.** `allocator.Allocate` produces a contracts v2
  `ExecutionPlan` (per-unit role/model/workers, plan concurrency, budget,
  placement) for the scheduler, validated before return.
- **Routing/budget policy extraction** (A8) is the next commit: it moves policy
  out of `runtime/inference` into the allocator/contract, leaving inference
  provider/execution only.

## Rationale

- Independent allocation dimensions keep each testable and let allocation change
  without touching execution.
- Deterministic ranking and explicit parallelism justification keep plans
  replayable and resource use intentional.
- A populated registry gives the allocator real model identities without adding a
  hard dependency on the Python service.

## Alternatives and consequences

- **One combined allocator:** rejected — couples three concerns and blocks
  independent testing/evolution.
- **Read the Python catalog at runtime from Go:** rejected — cross-language,
  fragile; the mirrored snapshot is version-controlled and refreshed on change.
- **Spawn workers whenever agents are idle:** rejected — parallelism must be
  justified.
- Consequence: the reserved `model_allocator`/`compute_allocator` boundaries are
  now live; `registry/{teams,routines,policies}` and `runtime/lifecycle` remain
  reserved.

## Related

- [Issue 7](../issues/imp-7-allocation.md)
- [ADR-0032](./ADR-0032-workers-are-selected-from-a-registry-by-capability-trade-role.md)
- [ADR index](./README.md)
