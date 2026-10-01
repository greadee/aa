# ISS-IMP-5 — runtime: sandbox execution

**Type:** feature / security
**Status:** complete
**Branch:** `dev`
**Sprint PR:** [#21](https://github.com/greadee/aa/pull/21)
**GitHub issue:** [#27](https://github.com/greadee/aa/issues/27)
**Parent umbrella:** [ISS-IMP](post-refactor-implementation.md)
**Sprint plan:** [../updates/issue-impl-sep28/plan.md](../updates/issue-impl-sep28/plan.md) (Stage 3)
**Resolves:** A7

## Goal

Implement execution isolation at the runtime boundary. This is where sandboxing is introduced; it was
intentionally reserved (not implemented) by the architecture refactor.

## Problem

`runtime/sandbox` is a reserved package with no implementation. Work runs without process/workload
isolation, filesystem/environment boundaries, resource limits, or tool-permission enforcement.

## Requirements

- Process/workload isolation.
- Filesystem boundaries and environment handling.
- Resource limits.
- Timeout/termination behavior.
- Tool-permission enforcement (with `toolbox` policy).
- Cleanup and error propagation.
- Observability of sandbox events.
- Escape/failure tests appropriate to the project's threat model.
- Runtime executes work through the sandbox **without agent roles knowing the sandbox implementation**.

## Acceptance Criteria

- [x] Sandbox isolates process, filesystem, environment, and resources; limits and timeouts are enforced.
- [x] Tool permissions integrate with the `toolbox` policy engine (via the `Policy` seam; the toolbox engine implements it at composition time, preserving the module boundary).
- [x] Cleanup on success, failure, and cancellation; errors propagate typed.
- [x] Escape/failure tests exist and pass.
- [x] Runtime runs work through the sandbox without leaking sandbox details to roles (`sandbox.Adapter` over `worker.Adapter`).
- [x] `go build`, `go vet`, `go test`, `gofmt`, `tools/archtest`, and docs link check pass.
- [x] tests added or updated
- [x] documentation updated where required

## Solution

`runtime/sandbox` is implemented:

- `Sandbox.Run(ctx, Spec)` runs a command through a `Runner` (`ExecRunner`
  default, terminated on context done; deterministic fake in tests).
- **Filesystem:** an explicit `Spec.Dir` must resolve inside `Config.Root`
  (`ErrPathEscape`); an empty `Dir` gets an ephemeral workspace cleaned up on
  every return.
- **Environment:** only `Config.EnvAllowlist` keys survive, sorted.
- **Limits:** `Limits.Timeout` (wall clock, `ErrTimeout` fail-closed) and
  `Limits.MaxOutputBytes` (`ErrLimitExceeded` with truncated result).
- **Capabilities:** `Spec.Permissions` checked against a caller-supplied `Policy`
  (toolbox policy engine); denial is `ErrDenied`. No `runtime` → `toolbox` import.
- **Observability:** lifecycle events to an `Observer` (kernel bridges to `obsv`).
- **Typed errors:** `ErrInvalid`, `ErrDenied`, `ErrPathEscape`, `ErrTimeout`,
  `ErrLimitExceeded`, `ErrExecution`.
- `Adapter` implements `worker.Adapter` over a `Sandbox` + `Planner`, so roles
  and the scheduler never see isolation. Execution remains disabled by default
  ([ADR-0039](../adr/ADR-0039-execution-is-disabled-by-default-and-enabled-explicitly.md)).

Decision record: [ADR-0147](../adr/ADR-0147-runtime-sandbox-isolation.md).

## Dependencies

Follows ISS-IMP-1/4. Unblocks ISS-IMP-6 (tool execution) and ISS-IMP-10 (sandbox observability).
