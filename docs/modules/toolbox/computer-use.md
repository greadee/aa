# toolbox/computer-use

> Project history for the `toolbox/computeruse` submodule of [aa-toolbox](./README.md).

**Responsibility** — Provide computer-use as a toolbox tool capability: declare
the authority it needs, run actions against a driver seam, and fail closed when a
permission is not declared. It carries no orchestration policy.

**Source** — [`toolbox/computeruse/`](../../../toolbox/computeruse/)

**Parent module** — [aa-toolbox](./README.md) · [docs/modules](../README.md)

## Capability and permissions

- **Capabilities** (closed execution-contract vocabulary): `read_project`,
  `create_artifact`.
- **Permissions** (free-form manifest strings): `screen_capture`,
  `input_injection`.
- **Sandbox**: `required: true`, `kind: process` — the toolbox policy engine
  allows a call only when the runtime sandbox can enforce process isolation
  (`runtime/sandbox.Enforcer`).

`Manifest(id)` returns a validated `builtin` tool manifest. `Provider` implements
`registry.Provider`: it decodes and validates the action, denies an action whose
permission the manifest does not declare (`ErrDenied`), propagates cancellation,
and returns output plus artifact references. `Driver` is the OS-interaction seam;
`Fake` is the deterministic test driver.

## Wiring

Execution is composed by the kernel, which imports both `toolbox` and `runtime`:

```text
kernel scheduler → runtime adapter → toolbox host → policy (sandbox gate) → computer-use provider → driver
```

The `runtime` module never imports `toolbox`, and `toolbox` never imports
`runtime`; they meet at the policy engine's `SandboxEnforcer` seam and the worker
adapter. Roles and the scheduler carry no sandbox or driver knowledge.

Decision record: [ADR-0148](../../adr/ADR-0148-computer-use-capability-and-runtime-wiring.md).
