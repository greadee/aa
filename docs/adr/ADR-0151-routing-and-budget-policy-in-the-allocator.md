# ADR-0151 — Routing and budget policy live in the allocator

| Field | Value |
|---|---|
| **Status** | Accepted |
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
- **Inference remains provider/execution plus verification.** Its Python
  `routing` package stays the service's execution surface (preflight assessment,
  cloud budgets, escalation evaluation); when the execution path is wired it
  consumes the allocator's decision. Inference no longer *owns* the policy.
- **Capital/contract.** The policy vocabulary is plain control-plane Go; no
  contracts v2 change is needed (route request/response remain).

## Rationale

- Routing is a cost/safety policy, not inference mechanics; the control plane is
  the authority for where work runs and within what budget.
- Deterministic, model-free routing keeps decisions replayable and auditable.
- Keeping the Python surface avoids breaking the standalone inference service and
  its tests while the ownership moves.

## Alternatives and consequences

- **Delete the Python routing package now:** rejected — a large cross-language
  refactor that would break the CLI/desktop/tests; staged instead.
- **Leave policy in inference:** rejected — it contradicts the A8 ownership split
  and duplicates policy authority.
- Consequence: the residual — wiring the execution path to pass the allocator's
  decision into inference and retiring the duplicate Python policy — is recorded
  in the sprint Future register as a follow-up.

## Related

- [kernel/allocator/routing module doc](../modules/kernel/routing.md)
- [inference routing module doc](../modules/runtime/inference/routing.md)
- [ADR-0150](./ADR-0150-role-model-compute-allocation-separation.md)
- [ADR index](./README.md)
