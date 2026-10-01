# ISS-IMP-3 — kernel: context compilation, budgeting & assembly

**Type:** feature
**Status:** complete
**Branch:** `dev`
**Sprint PR:** [#21](https://github.com/greadee/aa/pull/21)
**GitHub issue:** [#25](https://github.com/greadee/aa/issues/25)
**Parent umbrella:** [ISS-IMP](post-refactor-implementation.md)
**Sprint plan:** [../updates/issue-impl-sep28/plan.md](../updates/issue-impl-sep28/plan.md) (Stage 2)
**Resolves:** A8 (extract context/handoff from `runtime/inference`)

## Goal

Own construction of the task's working context: determine needs, select and budget retrieved
information, compress where required, preserve provenance, and assemble the execution context.

## Problem

`kernel/context` has a pure, bounded, digest-stable compiler, but no needs-discovery, selection,
relevance filtering, compression, or retrieval integration. `runtime/inference` still carries
context/handoff compression (A8), which belongs here.

## Requirements

- Context-needs discovery.
- Selection and relevance filtering over `memory/retrieval` candidates.
- Token/context-window budgeting.
- Compression/summarization where required.
- Provenance preservation.
- Final context-package assembly.
- Deterministic behavior where feasible; defined overflow/failure behavior.
- Move context/handoff handling out of `runtime/inference` into `kernel/context`.
- Tests that make the retrieval/context boundary explicit.

## Acceptance Criteria

- [x] Needs discovery, selection, budgeting, compression, provenance, and assembly implemented.
- [x] Deterministic given identical inputs; overflow fails closed with a typed error (`ErrBudgetExceeded`).
- [x] Context consumes the ISS-IMP-2 retrieval interface; the boundary is tested.
- [x] Control-plane context handling is owned by `kernel/context`, not `runtime/inference` (residual execution-local fitting/redaction retained by design — see Solution and [ADR-0146](../adr/ADR-0146-context-compilation-and-a8-extraction.md)).
- [x] `go build`, `go vet`, `go test`, `gofmt`, and `tools/archtest` pass.
- [x] tests added or updated
- [x] documentation updated where required

## Solution

`kernel/context` now owns control-plane context:

- `DeriveNeeds(objective, workPackageID)` — pure needs discovery (default kinds;
  distinct lowercased terms length ≥ 3, plus the work-package id; sorted).
- `Compiler.CompileFromRetrieval(ctx, retriever, projectID, workPackageID, needs)`
  — selects through the ISS-IMP-2 `memory/retrieval.Retriever`, maps candidates
  to inputs with provenance, and assembles a bounded, digest-stable `Bundle`.
- Budgeting/compression remain in the pure `Compiler`; `Section` now carries
  `retrieval.Provenance` and the digest covers it.
- Overflow fails closed: a partial `Truncated` bundle is returned with
  `ErrBudgetExceeded`.

**A8 scope.** The A8 audit listed "context → `kernel/context`". Control-plane
context ownership moved as above. The Python `runtime/inference/context` package
is retained as **execution-local** — provider-message fitting and the outbound
redaction chokepoint plus escalation formatting — because it runs where the
provider call and cloud egress happen ([ADR-0050](../adr/ADR-0050-one-outbound-redaction-chokepoint-in-call-tier-for-every-expert.md));
a literal cross-language relocation is neither possible nor desirable
([ADR-0146](../adr/ADR-0146-context-compilation-and-a8-extraction.md)). Module
docs record the boundary.

## Dependencies

ISS-IMP-2 (retrieval interface). Unblocks ISS-IMP-7 (allocation consumes an enriched task) and
ISS-IMP-9 (experience feeds future context).

## Notes

Context must not choose roles, models, or compute quantity; that is allocation.
