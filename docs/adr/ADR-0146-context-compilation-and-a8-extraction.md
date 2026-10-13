# ADR-0146 — Context compilation ownership and the A8 extraction

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [issue-impl-sep28 Issue 3](../issues/imp-3-context-compilation.md) |
| **Scope** | update phase · issue-impl-sep28 (Stage 2) |

## Context

`kernel/context` had a pure, bounded, digest-stable compiler but no needs
discovery, selection, relevance filtering, compression, or retrieval
integration. The architecture-refactor audit (A8) recorded that the inference
service still carried context/handoff handling that "belongs in
`kernel/context`". The inference service runs in Python; the control plane runs
in Go.

## Decision

- **`kernel/context` owns control-plane context.** Add needs discovery
  (`DeriveNeeds`), retrieval-backed selection (`CompileFromRetrieval` consuming
  the ISS-IMP-2 `memory/retrieval.Retriever`), token budgeting and rune-boundary
  compression, provenance preservation (`Section.Provenance` and the bundle
  `Digest`), and final assembly. Overflow **fails closed** with
  `ErrBudgetExceeded` (returning the partial, `Truncated` bundle).
- **Context never chooses roles, models, or compute** — that is allocation.
- **The inference `context` package stays, reduced to execution-local concerns:**
  fitting an already-decided provider message list to a model window, and the
  outbound secret-redaction chokepoint plus the escalation packet
  ([ADR-0050](./ADR-0050-one-outbound-redaction-chokepoint-in-call-tier-for-every-expert.md)).
  A literal cross-language relocation of these is neither possible nor desirable:
  they run where the provider call and the cloud egress happen.

## Rationale

- The control plane owns what the worker sees; the runtime owns how a provider
  call is shaped and secured. Splitting on that line keeps one owner per concern.
- Staying deterministic and provenance-preserving makes the bundle replayable.
- Keeping redaction at the call tier preserves the single outbound chokepoint.

## Alternatives and consequences

- **Move the Python `context` package into Go:** rejected — it operates on
  provider `Message`s and the redaction chokepoint, both execution-local.
- **Assemble context inside retrieval:** rejected — duplicates `kernel/context`.
- **Truncate silently on overflow:** rejected — a caller must know the budget was
  not met; `ErrBudgetExceeded` makes that explicit.
- Consequence: `kernel/context` depends on `memory/retrieval` (allowed by the
  architecture tests); the inference module docs record the boundary.

## Related

- [kernel/context module doc](../modules/kernel/context.md)
- [inference context module doc](../modules/runtime/inference/context.md)
- [ADR-0035](./ADR-0035-the-context-compiler-is-pure-and-bounded-taking-explicit-inputs.md)
- [ADR index](./README.md)
