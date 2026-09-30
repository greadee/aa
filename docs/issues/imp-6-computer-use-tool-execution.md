# ISS-IMP-6 — toolbox: computer-use / tool execution integration

**Type:** feature
**Status:** planned
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

- [ ] Computer-use capability declared with explicit permissions/capabilities.
- [ ] Execution flows through runtime and (where appropriate) the sandbox.
- [ ] Cancellation, failures, artifacts, and observability events handled.
- [ ] No orchestration policy in the capability.
- [ ] N/A — the inference desktop surface relocation (A8) is deferred to `ph10-ui`.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, `tools/archtest`, `runtime/inference` checks, and docs link check pass.
- [ ] tests added or updated
- [ ] documentation updated where required

## Dependencies

ISS-IMP-5 (sandbox). Related to ISS-IMP-8 (scheduling) and ISS-IMP-10 (observability).
