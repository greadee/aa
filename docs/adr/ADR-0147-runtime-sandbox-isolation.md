# ADR-0147 — Runtime sandbox isolation

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [issue-impl-sep28 Issue 5](../issues/imp-5-sandbox-execution.md) |
| **Scope** | update phase · issue-impl-sep28 (Stage 3) |

## Context

`runtime/sandbox` was a reserved package: work ran with no process/workload
isolation, filesystem or environment boundaries, resource limits, or
tool-permission enforcement. The architecture places isolation behind the worker
adapter, and the module layering lets `runtime` import only `contracts` and
`registry` — so the sandbox cannot import `toolbox`.

## Decision

Implement `runtime/sandbox`:

- **Process isolation behind the worker adapter.** `Sandbox.Run(ctx, Spec)` runs a
  command through a `Runner` (`ExecRunner` by default, terminated on context
  done; a fake in tests). `Adapter` implements `worker.Adapter` over a `Sandbox`
  plus a caller-supplied `Planner`, so the scheduler and roles never see isolation.
- **Filesystem boundary.** An explicit `Spec.Dir` must resolve inside
  `Config.Root` (`ErrPathEscape` otherwise); an empty `Dir` creates an ephemeral
  workspace cleaned up on every return (success, failure, cancel).
- **Environment boundary.** Only `Config.EnvAllowlist` keys survive; output is
  sorted for determinism.
- **Resource limits.** `Limits.Timeout` (wall clock) and `Limits.MaxOutputBytes`
  (per stream). Timeout fails closed with `ErrTimeout`; over-cap output returns
  `ErrLimitExceeded` with a truncated result.
- **Capabilities via a caller-supplied policy.** `Spec.Permissions` are checked
  against a `Policy` interface implemented by the toolbox policy engine at
  composition time; denial is `ErrDenied`. This preserves the module boundary.
- **Observability.** Lifecycle events are emitted to an `Observer`; the kernel
  bridges them to `obsv`.
- **Typed errors.** `ErrInvalid`, `ErrDenied`, `ErrPathEscape`, `ErrTimeout`,
  `ErrLimitExceeded`, `ErrExecution`.
- **Execution stays disabled by default** (ADR-0039): the scheduler still uses
  the `Disabled` adapter unless execution is explicitly enabled and a sandbox is
  wired.

## Rationale

- One isolation owner behind one adapter keeps roles and the scheduler agnostic.
- Delegating capability decisions to the policy engine avoids a `runtime` →
  `toolbox` dependency while keeping one policy authority.
- Fail-closed timeouts/limits and mandatory cleanup prevent partial state and
  resource leaks.

## Alternatives and consequences

- **Import `toolbox` for permissions:** rejected — violates layering; the sandbox
  depends on the `Policy` seam instead.
- **No filesystem confinement:** rejected — a run could escape its workspace.
- **Real subprocesses in unit tests:** rejected — tests inject a deterministic
  `Runner`; `ExecRunner` is exercised through the adapter tests.
- Consequence: `runtime/sandbox` is no longer reserved; `runtime/lifecycle`
  remains reserved for Issue 8.

## Related

- [runtime/sandbox module doc](../modules/runtime/sandbox.md)
- [Architecture §4.6](../architecture/README.md#46-boundaries-and-integration-points)
- [ADR-0039](./ADR-0039-execution-is-disabled-by-default-and-enabled-explicitly.md)
- [ADR index](./README.md)
