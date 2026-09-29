# ISS-IMP-2 — memory: deterministic retrieval

**Type:** feature
**Status:** planned
**Branch:** `dev`
**Sprint PR:** [#21](https://github.com/greadee/aa/pull/21)
**Parent umbrella:** [ISS-IMP](post-refactor-implementation.md)
**Sprint plan:** [../updates/issue-impl-sep28/plan.md](../updates/issue-impl-sep28/plan.md) (Stage 2)

## Goal

Deliver deterministic, testable retrieval under `memory/retrieval` that returns candidate information
through a stable interface. It must not assemble the final prompt/context window.

## Problem

`memory/retrieval` (formerly `memory/query`) provides deterministic reads and derived histories, but
there is no defined ranking/tie-break, provenance shape, filtering/limit contract, or error behavior.
`kernel/context` needs a stable retrieval interface to consume.

## Requirements

- Deterministic query behavior (order independent of input and wall clock).
- Stable ranking with deterministic tie-breaking.
- Source/provenance metadata on results.
- Filtering and retrieval limits.
- Defined error behavior (typed, not panics).
- A stable retrieval interface returning candidate information.
- Tests and fixtures; no final context assembly here.

## Acceptance Criteria

- [ ] Retrieval is deterministic and order-independent, with stable ranking/tie-break.
- [ ] Results carry provenance; filters and limits are honored.
- [ ] Errors are typed; no panics on malformed input.
- [ ] A stable interface is consumed by `kernel/context` (ISS-IMP-3).
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and `tools/archtest` pass.
- [ ] tests added or updated
- [ ] documentation updated where required

## Dependencies

Follows ISS-IMP-1. Unblocks ISS-IMP-3.

## Notes

Retrieval ranking must stay deterministic; learned/vector ranking is out of scope here.
