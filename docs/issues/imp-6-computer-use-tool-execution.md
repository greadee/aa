# ISS-IMP-6 — toolbox: computer-use / tool execution integration

**Type:** feature
**Status:** complete
**Branch:** `dev`
**Sprint PR:** [#21](https://github.com/greadee/aa/pull/21)
**GitHub issue:** [#28](https://github.com/greadee/aa/issues/28)
**Parent umbrella:** [ISS-IMP](post-refactor-implementation.md)
**Sprint plan:** [../updates/issue-impl-sep28/plan.md](../updates/issue-impl-sep28/plan.md) (Stage 3)
**Resolves:** — (the A8 `desktop → ui/surfaces` relocation is deferred to `ph10-ui`; not in this sprint)

## Goal

Provide computer-use and tool execution as a `toolbox` capability, coordinated through the kernel
runtime and the sandbox.

## Problem

`toolbox` owns tool/plugin/MCP manifests, policy, and workflows, but there is no computer-use
capability, and the execution boundary between runtime, a tool capability, and a computer-use
implementation is unspecified.

The inference service's desktop surface (A8 `desktop → ui/surfaces`) belongs with the UI phase and
is **out of scope here** (see the sprint Out-of-scope); it is tracked for `ph10-ui`.

## Requirements

- `toolbox/computer-use` capability with permission/capability declarations.
- Execution coordinated through the runtime boundary:
  `kernel scheduler → runtime → tool capability → computer-use implementation`.
- Cancellation, failures, sandbox interaction, result/artifact handling, and audit/observability events.
- No orchestration policy embedded in the computer-use implementation.

## Acceptance Criteria

- [x] Computer-use capability declared with explicit permissions/capabilities.
- [x] Execution flows through runtime and (where appropriate) the sandbox (via the policy `SandboxEnforcer` seam + worker adapter; kernel composes).
- [x] Cancellation, failures, artifacts, and observability events handled.
- [x] No orchestration policy in the capability.
- [x] N/A — the inference desktop surface relocation (A8) is deferred to `ph10-ui`.
- [x] `go build`, `go vet`, `go test`, `gofmt`, `tools/archtest`, `runtime/inference` checks, and docs link check pass.
- [x] tests added or updated
- [x] documentation updated where required

## Solution

`toolbox/computeruse` provides the computer-use capability:

- `Manifest(id)` — a validated `builtin` tool manifest: capabilities
  `read_project`/`create_artifact`, permissions `screen_capture`/
  `input_injection`, and a **required** `process` sandbox.
- `Provider` (`registry.Provider`) — validates the action, denies any action
  whose permission the manifest does not declare (`ErrDenied`), propagates
  cancellation, wraps driver failures (`ErrFailed`), and returns output plus
  artifact references. No orchestration policy.
- `Driver` seam + deterministic `Fake` (records calls, honors cancellation).

Runtime wiring: `runtime/sandbox.Enforcer` implements the toolbox policy
engine's `SandboxEnforcer` seam, so the computer-use manifest's required process
sandbox is allowed only when the runtime sandbox can enforce it; the kernel
composes `toolbox` and `runtime` (neither imports the other). An integration test
in `kernel` proves the flow and the fail-closed path without an enforcer.

Decision record: [ADR-0148](../adr/ADR-0148-computer-use-capability-and-runtime-wiring.md).

## Dependencies

ISS-IMP-5 (sandbox). Related to ISS-IMP-8 (scheduling) and ISS-IMP-10 (observability).
