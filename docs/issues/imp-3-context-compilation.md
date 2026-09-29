# ISS-IMP-3 — kernel: context compilation, budgeting & assembly

**Type:** feature
**Status:** planned
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

- [ ] Needs discovery, selection, budgeting, compression, provenance, and assembly implemented.
- [ ] Deterministic given identical inputs; overflow fails closed with a typed error.
- [ ] Context consumes the ISS-IMP-2 retrieval interface; the boundary is tested.
- [ ] Context handling removed from `runtime/inference`.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and `tools/archtest` pass.
- [ ] tests added or updated
- [ ] documentation updated where required

## Dependencies

ISS-IMP-2 (retrieval interface). Unblocks ISS-IMP-7 (allocation consumes an enriched task) and
ISS-IMP-9 (experience feeds future context).

## Notes

Context must not choose roles, models, or compute quantity; that is allocation.
