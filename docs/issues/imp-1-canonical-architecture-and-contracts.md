# ISS-IMP-1 — contracts/kernel/runtime/registry/obsv: canonical architecture & contracts

**Type:** documentation / contracts
**Status:** complete
**Branch:** `dev`
**Sprint PR:** [#21](https://github.com/greadee/aa/pull/21)
**GitHub issue:** [#23](https://github.com/greadee/aa/issues/23)
**Parent umbrella:** [ISS-IMP](post-refactor-implementation.md)
**Sprint plan:** [../updates/issue-impl-sep28/plan.md](../updates/issue-impl-sep28/plan.md) (Stage 1)
**Resolves:** R6, R7, R8; A4; the A5 ownership decision

## Goal

Reconcile the canonical documentation and contracts with the completed refactor and the boundaries the
upcoming issues depend on, so every later issue plans against one agreed vocabulary and shape.

## Problem

The refactor left the target boundaries in place but the ten issues need explicit, current
documentation and only the contracts they actually use. Residual review/audit findings remain:
architecture §5.6/§5.12 duplicate the runtime/inference boundary (R6); `docs/index/phases.md`
still labels phase 4 `aa-sifter` (R7); `contracts/openapi/control-plane-v1.yaml` still says
`aa-console` (R8); the **v1** contract generation is retained with no sunset plan (A4); and the
intake/contract ownership decision is recorded but unexecuted (A5).

## Requirements

- Document the boundaries for `kernel/context`, `memory/retrieval`, `kernel/allocator`,
  `kernel/scheduler`, `runtime` (worker/lifecycle/sandbox/inference), `registry`, `toolbox`, and
  `obsv`/event boundaries, plus the sandbox integration point.
- Document the distinction **role ≠ capability ≠ parallelism**.
- Define or update only the contracts the upcoming issues require; avoid speculative interfaces
  nobody needs.
- Reconcile architecture §5.6/§5.12, the phase index label, and the OpenAPI console references.
- Plan and execute the **v1 sunset**: mark v1 deprecated and remove it from the active path
  (git history retains it).
- Record the `intake → runtime` / `contract` ownership decision (executed in ISS-IMP-8).
- Record that the reserved packages (`registry/{models,teams,routines,policies}`,
  `runtime/{lifecycle,sandbox}`, `kernel/allocator/{model,compute}_allocator`) have no consumers yet
  (R10).

## Acceptance Criteria

- [x] Architecture README and terminology reflect the as-built boundaries; §5.6/§5.12 consolidated.
- [x] `role ≠ capability ≠ parallelism` documented (terminology §1.1).
- [x] Required contracts audited; no speculative contract added; indexes, `MANIFEST.json`, and the
      phase index current; OpenAPI references `ui`.
- [x] Contracts v1 deprecated and removed from the active path; no live consumer (commit 1.2 / A4).
- [x] Intake/contract ownership decision recorded ([ADR-0143](../adr/ADR-0143-decide-intake-and-contract-ownership.md)).
- [x] Reserved-package status recorded in the Future register and architecture §4.1.
- [x] `go build`, `go vet`, `go test`, `gofmt`, `tools/archtest`, and docs link check pass.
- [x] tests added or updated — no code change in this slice; existing suites unaffected and green.
- [x] documentation updated where required

## Progress

| Commit | Scope | Notes |
|---|---|---|
| `ed1991e` | R5–R8, R10 reconciliation | scheduler comments, §5.6/§5.12, phase-4 label, OpenAPI, reserved-boundary table (Stage 0.2) |
| `reconcile canonical architecture, boundaries and required contracts` | Issue 1: boundaries (§4.6), terminology §1.1, ADR-0143, indexes/Future register | commit 1.1 |
| `sunset contracts v1` | A4: generation v1 removed from the active path (schemas/bindings/CI/docs → v2) | commit 1.2; closes Issue 1 |

Closes [#23](https://github.com/greadee/aa/issues/23).

## Dependencies

None; foundational. Unblocks ISS-IMP-2…10 (shared vocabulary and contracts).

## Notes

The A5 decision is recorded here and executed in [ISS-IMP-8](imp-8-scheduling-multi-agent-execution.md)
(stage 3.4), where the scheduler caller is in scope.
