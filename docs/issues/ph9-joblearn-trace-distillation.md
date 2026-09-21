# ISS-LEARN-1 — joblearn: trace distillation and evaluation

**Type:** feature / technical debt
**Status:** in progress (joblearn module update — planning)
**Branch:** `ph9-joblearn`
**Phase PR:** (linked from the module summary when opened)
**Parent umbrella:** `ISS-TRACE-LOOP` — the observation → trace → learning substrate is under-delivered (`docs/issues/trace-learning-substrate.md`)
**Module plan:** [../../kernel/joblearn/module-upd-plan.md](../../kernel/joblearn/module-upd-plan.md)
**Module summary:** `kernel/joblearn/module-upd-summary.md` (authored at the end of the implementation update)

## Goal

Make the learning loop consume bounded per-step traces and turn them into improved subagent artifacts. After this update `kernel/joblearn` reads traces through a pure source seam, derives step-level attribution and features from `v1.Trace`, distills role- and trade-scoped candidate artifacts with trace provenance, and evaluates them by backtest over trace-derived datasets against a deterministic baseline whose production arm is the governed `sifter` recommender. The plan for this work is delivered first; no code lands in the planning pull request.

## Problem

Phase 9 delivered a deterministic learning engine, but it consumes per-attempt `telemetry` records, not the step-level `trace` the umbrella records as missing (deviation 4: "Observation evidence from `obsv`"; deviation 3: the recommender baseline is a fake). The loop therefore cannot see where an attempt spent its steps, which phase failed, or which error class recurred, so it cannot distill or evaluate an improved subagent artifact. The product goal — improving subagents from evidence of prior work sessions — is unimplemented. The trace contract (`ISS-TRACE-1`), the canonical trace store (`ISS-TRACE-2`), and the live observation substrate (`ISS-OBSV-1`) supply the evidence; this sub-issue supplies the consumer.

## Requirements

- A pure `TraceSource` seam that yields bounded `v1.Trace` values (or their deterministic `TraceSummary` projections) with fakes for tests; the engine stays free of store and I/O.
- Step-level feature extraction from `TraceStep` (phase, operation, target, outcome, error class, duration, tokens, cost, redaction/truncation), deterministic and content-free.
- Trace-derived attribution and scoring that link steps to the work package, role, trade, and worker through the existing `Attribution`/`Score` types.
- Artifact distillation: synthesize role-, trade-, and task-scoped candidate artifacts as existing `Candidate` kinds (strategy/pattern/pitfall) carrying trace provenance.
- Evaluation over traces: a deterministic trace-derived backtest dataset, a bounded and order-independent `backtest.Run`, and a production baseline arm backed by the governed `sifter` recommender behind the existing `backtest.Predictor` seam.
- Evidence gates fed by trace-derived outcome counts; capabilities remain disabled by default and never bypass a gate or contract.
- Candidates stay candidates; promotion remains explicit through the existing `memory` lifecycle.
- Determinism and metadata-only evidence: the same traces yield the same features, candidates, datasets, and reports; no wall clock, no randomness, no raw content.

## Acceptance Criteria

- [ ] A trace fixture flows through the source seam to deterministic attributed scores.
- [ ] Step-level features are derived without content and honour `redacted`/`truncated`.
- [ ] At least one distilled artifact candidate is produced per role/trade scope with trace provenance.
- [ ] A trace-derived backtest runs within budget and is order-independent.
- [ ] The production baseline is reached through the `sifter` RPC seam; fakes remain test-only.
- [ ] Capabilities stay disabled without evidence; no gate or execution contract is bypassed; a fallback remains.
- [ ] Distilled candidates persist as `CANDIDATE` through the existing promotion `Sink`.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and the boundary check pass.
- [ ] tests added or updated
- [ ] documentation updated where required

## Affected branches

Depends on the trace contract (`ISS-TRACE-1`, `ph1-contracts`), the canonical trace store (`ISS-TRACE-2`, `ph2-memory`), the live observation substrate that produces traces (`ISS-OBSV-1`, `ph3-kernel`), and the governed model-routing boundary that supplies the production baseline (`ISS-SIFTER-1`, `ph4-sifter`). It unblocks console surfaces over learned artifacts and any later cross-project learning. Their solutions are not specified here.

## Notes

This document and `kernel/joblearn/module-upd-plan.md` are planning artifacts. Implementation slices are listed in the module plan and proceed only after the plan is reviewed. Reference counterparts: `contracts/module-upd-plan.md` (`ISS-TRACE-1`, PR #12), `memory/module-upd-plan.md` (`ISS-TRACE-2`, PR #13), `obsv/module-upd-plan.md` (`ISS-OBSV-1`, PR #15), and `sifter/module-upd-plan.md` (`ISS-SIFTER-1`, PR #16).
