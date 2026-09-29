# ISS-IMP-10 — obsv + all: observability, hardening & end-to-end validation

**Type:** feature / hardening
**Status:** planned
**Branch:** `dev`
**Sprint PR:** [#21](https://github.com/greadee/aa/pull/21)
**GitHub issue:** [#32](https://github.com/greadee/aa/issues/32)
**Parent umbrella:** [ISS-IMP](post-refactor-implementation.md)
**Sprint plan:** [../updates/issue-impl-sep28/plan.md](../updates/issue-impl-sep28/plan.md) (Stage 5)
**Resolves:** A9 (observability identity chain)

## Goal

System-wide hardening against the refactored architecture and Issues 1–9, with observability across
the new execution seams and end-to-end validation.

## Problem

The refactor introduced new seams (context, retrieval, allocation, plan, scheduler, runtime, sandbox,
tool, learning) with no unified observability, and attribution does not yet span
`organization → project → subtask → crew → worker → {role → team, model}`. There is no end-to-end
scenario coverage for the finished pipeline.

## Requirements

- Structured events/logging across: task received, context needs, retrieval, context assembly, role
  decision, capability decision, parallelism decision, execution plan, scheduler dispatch, runtime
  start/stop, sandbox events, tool calls, agent result, evaluation/learning, failure/retry.
- Trace/correlation IDs; useful metrics; failure visibility; cost/token/compute accounting where available.
- Observability identity chain `organization → project → subtask → crew → worker → {role → team, model}`.
- Regression tests, integration tests, and end-to-end scenarios:
  simple task → single role → cheap capability → one worker;
  difficult task → correct role → capability escalation → one worker;
  decomposable task → multiple roles/workers → scheduler → parallel runtime → aggregation;
  tool-using task → runtime → sandbox → toolbox/computer-use → result.
- Documentation reconciliation.

## Acceptance Criteria

- [ ] Structured events/traces emitted across the listed seams, with correlation IDs.
- [ ] Metrics and failure visibility; cost/token/compute accounting where available.
- [ ] Identity chain attribution preserved end to end.
- [ ] E2E scenarios above pass.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, `tools/archtest`, contracts Go+Python+TypeScript, `runtime/inference` checks, and docs link check pass.
- [ ] tests added or updated
- [ ] documentation updated where required

## Dependencies

Follows ISS-IMP-1…9. Final hardening; closes the sprint.
