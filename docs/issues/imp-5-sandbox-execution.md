# ISS-IMP-5 — runtime: sandbox execution

**Type:** feature / security
**Status:** planned
**Branch:** `dev`
**Sprint PR:** [#21](https://github.com/greadee/aa/pull/21)
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

- [ ] Sandbox isolates process, filesystem, environment, and resources; limits and timeouts are enforced.
- [ ] Tool permissions integrate with the `toolbox` policy engine.
- [ ] Cleanup on success, failure, and cancellation; errors propagate typed.
- [ ] Escape/failure tests exist and pass.
- [ ] Runtime runs work through the sandbox without leaking sandbox details to roles.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, `tools/archtest`, and docs link check pass.
- [ ] tests added or updated
- [ ] documentation updated where required

## Dependencies

Follows ISS-IMP-1/4. Unblocks ISS-IMP-6 (tool execution) and ISS-IMP-10 (sandbox observability).
