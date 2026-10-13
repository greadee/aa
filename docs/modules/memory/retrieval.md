# memory/retrieval

> Project history for the `memory/retrieval` submodule of [aa-memory](./README.md).

**Responsibility** — The retrieval package provides deterministic read access to aa-memory's projection and derives work, project, and job histories from the event log.

**Source** — [`memory/retrieval/`](../../../memory/retrieval/)

**Parent module** — [aa-memory](./README.md) · [docs/modules](../README.md)

## Retrieval interface

`Retrieve(ctx, Request) ([]Result, error)` on `*Query` implements the stable
`Retriever` interface consumed by `kernel/context` (ISS-IMP-3). It returns
candidate information only — never an assembled context window.

- **Request** — `Kinds` (empty = every projected kind), `Terms` (case-insensitive
  substrings scored against a candidate's text), an optional `Filter`, and
  `Limit` (0 = `DefaultLimit` 100; max `MaxLimit` 10000).
- **Result** — the `Candidate` (kind, id, revision, hash, derived text) plus a
  deterministic `Score` and 1-based `Rank`.
- **Provenance** — every candidate carries `Provenance{source, kind, id,
  revision, hash}` so consumers can attribute and re-derive it.
- **Ranking** — by descending `Score` (count of distinct `Terms` present), then
  ascending `(Kind, ID)`. Results are independent of the order of `Kinds`/`Terms`
  and of wall-clock time. Learned/vector ranking is out of scope.
- **Errors** — invalid requests return a typed error (`errors.Is(err,
  ErrInvalidRequest)`); malformed record data never panics and yields empty text.

Decision record: [ADR-0145](../../adr/ADR-0145-deterministic-retrieval-ranking-and-provenance.md).
