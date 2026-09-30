# runtime/sandbox

> Project history for the `runtime/sandbox` submodule of [aa-runtime](./README.md).

**Responsibility** — Execution isolation behind the worker adapter: a run is
confined to a working directory under a configured root, receives only
allowlisted environment variables, is bounded by a wall-clock timeout and an
output cap, and must hold every capability it declares. Delivered by the
ten-issue sprint (Issue 5 / A7).

**Source** — [`runtime/sandbox/`](../../../runtime/sandbox/)

**Parent module** — [aa-runtime](./README.md) · [docs/modules](../README.md)

## Design

- **Process isolation** — commands run through a `Runner`; `ExecRunner` runs an
  OS process and is terminated when the context is done. Tests inject a fake.
- **Filesystem boundary** — an explicit `Spec.Dir` must resolve inside
  `Config.Root` (else `ErrPathEscape`); an empty `Dir` creates an ephemeral
  workspace that is always removed on return (success, failure, or cancel).
- **Environment boundary** — only keys in `Config.EnvAllowlist` survive, emitted
  sorted for determinism.
- **Limits** — `Limits.Timeout` (wall clock) and `Limits.MaxOutputBytes`
  (per-stream cap); exceeding a cap returns `ErrLimitExceeded` with a truncated
  result.
- **Tool permissions** — `Spec.Permissions` are checked against a caller-supplied
  `Policy` (the toolbox policy engine at composition time). This package does not
  import `toolbox` (the runtime module may not); denial is `ErrDenied`.
- **Observability** — run lifecycle events (`SANDBOX_STARTED/COMPLETED/FAILED/
  TIMEOUT/CANCELLED/DENIED/LIMIT_EXCEEDED`) are emitted to an `Observer`.
- **Typed errors** — `ErrInvalid`, `ErrDenied`, `ErrPathEscape`, `ErrTimeout`,
  `ErrLimitExceeded`, `ErrExecution`.

`Adapter` implements `worker.Adapter` over a `Sandbox` plus a `Planner`, so work
runs through isolation without roles or the scheduler knowing the sandbox.

Decision record: [ADR-0147](../../adr/ADR-0147-runtime-sandbox-isolation.md).
