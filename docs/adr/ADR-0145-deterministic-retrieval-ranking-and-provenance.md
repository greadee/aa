# ADR-0145 — Deterministic retrieval ranking and provenance

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [issue-impl-sep28 Issue 2](../issues/imp-2-deterministic-retrieval.md) |
| **Scope** | update phase · issue-impl-sep28 (Stage 2) |

## Context

`memory/retrieval` (formerly `memory/query`) exposed projection reads and derived
histories, but no defined ranking, tie-break, provenance, filter/limit contract,
or error behavior. `kernel/context` needs a stable interface to consume without
depending on ranking internals, and retrieval results must be reproducible for
tests and replay.

## Decision

Add a stable `Retriever` interface in `memory/retrieval`:

- `Retrieve(ctx, Request) ([]Result, error)`. `Request` carries `Kinds`, `Terms`,
  an optional `Filter`, and `Limit`; `Result` carries a `Candidate` with
  `Provenance` plus a deterministic `Score` and 1-based `Rank`.
- **Ranking is deterministic:** descending `Score` (number of distinct `Terms`
  found in a candidate's derived text), then ascending `(Kind, ID)`. Order is
  independent of input order and wall clock.
- **Provenance is mandatory:** `Provenance{source, kind, id, revision, hash}`.
- **Limits are bounded:** `0` means `DefaultLimit` (100); above `MaxLimit` (10000)
  or negative is a typed `ErrInvalidRequest`.
- **Errors are typed** (`errors.Is`); malformed record data never panics.
- **No context assembly here.** Retrieval returns candidates; budgeting and
  assembly stay in `kernel/context` (ADR-0035).

## Rationale

- Deterministic ordering makes retrieval testable and replayable and gives
  `kernel/context` a stable contract.
- Provenance lets every included section be attributed and re-derived.
- Keeping assembly out of retrieval preserves the pure, bounded context compiler
  and the single-owner-per-concern boundary.
- Learned or vector ranking is deliberately excluded until an evidence-gated
  step (Issue 9) can own it.

## Alternatives and consequences

- **Score by relevance learned from history:** rejected here — no evidence gate
  yet, and it would make retrieval non-deterministic.
- **Return assembled context:** rejected — duplicates `kernel/context`.
- Consequence: `Projection` gains `Kinds()`; `kernel/context` consumes
  `Retriever` in ISS-IMP-3.

## Related

- [memory/retrieval module doc](../modules/memory/retrieval.md)
- [ADR-0035](./ADR-0035-the-context-compiler-is-pure-and-bounded-taking-explicit-inputs.md)
- [ADR index](./README.md)
