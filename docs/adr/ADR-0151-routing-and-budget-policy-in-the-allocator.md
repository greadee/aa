# ADR-0151 — Routing and budget policy live in the allocator

| Field | Value |
|---|---|
| **Status** | Accepted — completed in phase 11 |
| **Source** | [issue-impl-sep28 Issue 7](../issues/imp-7-allocation.md) (A8) |
| **Scope** | update phase · issue-impl-sep28 (Stage 4) |

## Context

`runtime/inference` owned model-locality/tier routing policy and the cloud budget
gate. The architecture assigns control-plane policy to `kernel` and leaves
inference as a provider/execution service (the A8 audit: "routing → allocator,
budgets → allocator/contract").

## Decision

- **The allocator owns the policy decision.** `kernel/allocator/routing.Decide`
  resolves a policy (`local_only`, `cloud_only`, `local_first`, `expert_first`,
  `adaptive`, `budget_constrained`) and a significance level (`routine`,
  `significant`, `major`, `critical`) into a locality/tier decision, deterministically
  and without calling a model.
- **The budget gate is control-plane.** The expert tier is blocked fail-closed
  when cloud is disabled or the estimated cost exceeds `Budget.maxCostUsd`.
- **Inference remains provider/execution plus verification.** The allocator's
  `Decision` is carried to inference as `v2.RoutingDecision` and over RPC as the
  `routingDecision` field on `route_request` and `inference.generate`. Inference
  executes the decided locality/tier/budget and **fails closed** when the
  decision is absent, inconsistent, or blocked (`aa.invalid_params` /
  `aa.budget_exceeded`). The duplicate Python policy engine (~~routing/policy.py~~)
  is retired; preflight assessment, cloud budget accounting, and escalation
  evaluation remain execution-time helpers only.
- **Capital/contract.** `v2.RoutingDecision` is the wire form; route
  request/response remain the cross-module contract.

## Rationale

- Routing is a cost/safety policy, not inference mechanics; the control plane is
  the authority for where work runs and within what budget.
- Deterministic, model-free routing keeps decisions replayable and auditable.
- Keeping the Python surface avoids breaking the standalone inference service and
  its tests while the ownership moves.

## Alternatives and consequences

- **Delete the Python routing package immediately in the A8 sprint:** rejected
  then as a large cross-language refactor; completed in phase 11 after the
  control plane exposed a decision contract.
- **Leave policy in inference:** rejected — it contradicts the A8 ownership split
  and duplicates policy authority.
- Consequence (resolved): the residual is complete. The standalone CLI keeps a
  local fallback when no allocator decision is supplied; the RPC execution path
  never falls back.

## Related

- [kernel/allocator/routing module doc](../modules/kernel/routing.md)
- [inference routing module doc](../modules/runtime/inference/routing.md)
- [ISS-DeferPH11-6](https://github.com/greadee/aa/issues/48)
- [ADR-0150](./ADR-0150-role-model-compute-allocation-separation.md)
- [ADR index](./README.md)
