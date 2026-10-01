# ISS-IMP — post-refactor implementation sprint (ten issues)

**Type:** umbrella (feature program)
**Status:** complete (all ten sub-issues implemented; sprint summary authored)
**Branch:** `dev`
**Sprint PR:** [#21](https://github.com/greadee/aa/pull/21)
**GitHub issue:** [#22](https://github.com/greadee/aa/issues/22)
**Milestone:** Issue implementation sprint (Sep 28)
**Sprint plan:** [../updates/issue-impl-sep28/plan.md](../updates/issue-impl-sep28/plan.md)
**Predecessor:** [architecture-refactor-1](../updates/architecture-refactor-1/summary.md) (merged `bf8e09d`)
**Findings source:** [PR #20](https://github.com/greadee/aa/pull/20) review (R1–R10) and repository audit (A1–A9)

## Problem

The architecture refactor established the target boundaries (contracts v2, `kernel/allocator` +
`kernel/scheduler`, top-level `runtime`, `registry`, `memory/retrieval`, `ui`, `obsv`) but implemented
none of the deferred functionality. The next implementation phase is the ten issues below, plus the
unresolved refactor review findings **R4–R9** and audit deferrals **A4–A9**.

This umbrella states the problem and the dependency order; each sub-issue's solution is specified in
its own document and delivered on `dev`.

## Sub-issues

| Id | Sub-issue | Stage | Resolves |
|---|---|---|---|
| [ISS-IMP-1](imp-1-canonical-architecture-and-contracts.md) (#23) | Canonical architecture & contracts | 1 | R6, R7, R8, A4, A5 (decision) |
| [ISS-IMP-2](imp-2-deterministic-retrieval.md) (#24) | Deterministic retrieval | 2 | — |
| [ISS-IMP-3](imp-3-context-compilation.md) (#25) | Context compilation, budgeting & assembly | 2 | A8 (context) |
| [ISS-IMP-4](imp-4-rpc-boundary-hardening.md) (#26) | RPC / boundary / version-skew hardening | 1 | — |
| [ISS-IMP-5](imp-5-sandbox-execution.md) (#27) | Sandbox execution | 3 | A7 |
| [ISS-IMP-6](imp-6-computer-use-tool-execution.md) (#28) | Computer-use / tool execution | 3 | — (A8 desktop→ui deferred to `ph10-ui`) |
| [ISS-IMP-7](imp-7-allocation.md) (#29) | Role, model & compute allocation | 4 | A6, A8 (routing/budgets) |
| [ISS-IMP-8](imp-8-scheduling-multi-agent-execution.md) (#30) | Scheduling & multi-agent execution | 3 | A5 (execution) |
| [ISS-IMP-9](imp-9-learning-experience-feedback.md) (#31) | Learning / experience feedback | 5 | A9 (apprenticing/studying) |
| [ISS-IMP-10](imp-10-observability-hardening-e2e.md) (#32) | Observability, hardening & end-to-end validation | 5 | A9 (identity chain) |

Also folded into the sprint staging: **R4** (contracts `v2` import aliasing) and **R5** (scheduler
comments) in Stage 0, and **R10** (reserved-package status) as documentation.

## Dependency order

```text
ISS-IMP-4 (RPC/boundaries) ─┐
ISS-IMP-1 (canonical)       ├─> ISS-IMP-2 (retrieval) -> ISS-IMP-3 (context)
                            │
                            ├─> ISS-IMP-5 (sandbox) -> ISS-IMP-6 (computer-use)
                            │
                            ├─> ISS-IMP-8 (scheduling) <── ISS-IMP-7 (allocation)
                            │
                            └─> ISS-IMP-9 (learning) -> ISS-IMP-10 (observability/e2e)
```

Some work may overlap; the [sprint plan](../updates/issue-impl-sep28/plan.md) fixes the stage order to
minimise cross-module jumping.

## Acceptance Criteria

- [x] All ten sub-issues implemented and closed (project/work/issue history + GitHub `Closes #n`), or explicitly deferred with a new issue.
- [x] R4–R9 and A4–A9 resolved or recorded.
- [x] `go build`/`go vet`/`go test`/`gofmt`, `tools/archtest`, contracts Go+Python+TypeScript, and `runtime/inference` checks pass.
- [x] Architecture, terminology, directives, indexes, `MANIFEST.json`, and diagrams current; docs link check passes.
- [x] Sprint summary + UML authored; sprint PR opens (`dev` → `main`); branch retained. *(GitHub issues close when PR #21 merges.)*

Sprint summary: [../updates/issue-impl-sep28/summary.md](../updates/issue-impl-sep28/summary.md).

## Notes

GitHub tracking issues for the ten sub-issues are opened in sprint objective 0; this document and the
sprint plan are the knowledge record. Newly discovered, out-of-scope work is raised as a new issue and
recorded in the sprint plan's Future register.
