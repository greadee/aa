# ADR-0148 — Computer-use capability and runtime wiring

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [issue-impl-sep28 Issue 6](../issues/imp-6-computer-use-tool-execution.md) |
| **Scope** | update phase · issue-impl-sep28 (Stage 3) |

## Context

`toolbox` owns manifests, policy, and workflows, but there was no computer-use
capability and the execution boundary between the runtime, a tool capability, and
its implementation was unspecified. The execution-contract **capability**
vocabulary is a closed enum, and the module layering forbids `toolbox` ↔ `runtime`
imports.

## Decision

- **Computer-use is a toolbox capability/service.** Add `toolbox/computeruse`:
  a validated `builtin` tool manifest (`Manifest`), a `Provider`
  (`registry.Provider`), and a `Driver` seam with a deterministic `Fake`.
- **Authority is declared, not implicit.** The closed capability vocabulary is
  reused (`read_project`, `create_artifact`); the sensitive OS authorities are
  free-form manifest **permissions** (`screen_capture`, `input_injection`), so no
  contract generation change is needed. The provider fails closed (`ErrDenied`)
  when an action's permission is not declared.
- **Isolation is required.** The manifest declares `sandbox.required: true,
  kind: process`. The toolbox policy engine allows a call only when a
  `SandboxEnforcer` can enforce that kind; the kernel wires
  `runtime/sandbox.Enforcer` into that seam.
- **Wiring is composed by the kernel.** `kernel scheduler → runtime adapter →
  toolbox host → policy (sandbox gate) → provider → driver`. `runtime` never
  imports `toolbox` and vice versa; they meet at the enforcer seam and the worker
  adapter, so roles and the scheduler carry no sandbox or driver knowledge.
- **No orchestration policy** in the capability; lifecycle (cancellation,
  failures, artifacts, audit) is handled by the host/policy and the provider
  returns typed toolbox errors.

## Rationale

- Reusing the closed capability enum avoids a breaking contract change mid-sprint.
- Expressing OS authorities as permissions keeps grant semantics explicit and
  provider-enforced.
- A required sandbox means computer-use runs only where isolation is real.
- Kernel composition preserves the enforced module boundaries.

## Alternatives and consequences

- **Add a `computer_use` capability to the closed enum:** rejected — a closed-enum
  change is a major contract bump requiring a migration plan.
- **Import `runtime/sandbox` from `toolbox`:** rejected — violates layering.
- **A dedicated tool kind:** rejected — the tool-kind enum is closed; `builtin`
  suffices and the provider routes computer-use tools.
- Consequence: `runtime/sandbox` exposes `Enforcer`; the desktop→`ui` relocation
  (A8) is separate and deferred to `ph10-ui`.

## Related

- [toolbox/computer-use module doc](../modules/toolbox/computer-use.md)
- [runtime/sandbox module doc](../modules/runtime/sandbox.md)
- [ADR-0147](./ADR-0147-runtime-sandbox-isolation.md)
- [ADR index](./README.md)
