# ISS-IMP-2 — memory: deterministic retrieval

**Type:** feature
**Status:** complete
**Branch:** `dev`
**Sprint PR:** [#21](https://github.com/greadee/aa/pull/21)
**GitHub issue:** [#24](https://github.com/greadee/aa/issues/24)
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

- [x] Retrieval is deterministic and order-independent, with stable ranking/tie-break.
- [x] Results carry provenance; filters and limits are honored.
- [x] Errors are typed; no panics on malformed input.
- [x] A stable interface is defined for `kernel/context` (consumed in ISS-IMP-3).
- [x] `go build`, `go vet`, `go test`, `gofmt`, and `tools/archtest` pass.
- [x] tests added or updated
- [x] documentation updated where required

## Solution

`memory/retrieval` now exposes a stable `Retriever` interface:
`Retrieve(ctx, Request) ([]Result, error)`, implemented by `*Query`.
`Request` carries `Kinds` (empty = all projected kinds), `Terms` (case-insensitive
substrings), an optional `Filter`, and `Limit` (0 = `DefaultLimit` 100; max
`MaxLimit` 10000). Every `Result` carries a `Candidate` with mandatory
`Provenance{source,kind,id,revision,hash}`, a deterministic `Score`, and a 1-based
`Rank`. Ranking is by descending score then ascending `(Kind, ID)`, independent of
input order and wall clock. Invalid requests return `ErrInvalidRequest`
(`errors.Is`); malformed record data yields empty text and never panics.
`Projection` gained `Kinds()` to support searching all kinds.

Decision record: [ADR-0145](../adr/ADR-0145-deterministic-retrieval-ranking-and-provenance.md).

## Dependencies

Follows ISS-IMP-1. Unblocks ISS-IMP-3.

## Notes

Retrieval ranking must stay deterministic; learned/vector ranking is out of scope here.
