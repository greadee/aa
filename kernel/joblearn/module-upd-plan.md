# kernel/joblearn — Module Update Plan: trace distillation and evaluation

> Authored at the **start** of the module update, on branch `ph9-joblearn` (the phase that owns `aa-joblearn`).
> Sub-problem: `ISS-LEARN-1` · Parent umbrella: `ISS-TRACE-LOOP` (`docs/issues/trace-learning-substrate.md`)

## Objective

Make `kernel/joblearn` learn from bounded per-step traces instead of per-attempt telemetry summaries, and close the loop the product promises: distill improved subagent artifacts from traces and evaluate them against a deterministic baseline before any capability is enabled. After this update the engine reads `v1.Trace` through a pure source seam, derives step-level attribution and features, synthesizes role/trade-scoped candidate artifacts with trace provenance, and backtests them over trace-derived datasets whose production baseline is the governed `sifter` recommender.

This plan is planning only: **no code is implemented in this module update yet**. Implementation proceeds slice by slice after review.

## Starting State

- Starting ref: `main @ d9f8477`; branch `ph9-joblearn` fast-forwarded to it.
- Available:
  - The pure phase 9 engine: `attribution` (attribute/score/summarize/group from `telemetry.Record`), `candidates` (pattern/pitfall/strategy/routing generation with provenance), `similarity`/`conflict`, `gate` (capabilities disabled by default, `Evaluate`), `backtest` (`Predictor` seam, bounded `Run`), `route` (non-authoritative hint with fallback), and `promote` (`Sink`, `MemorySink` to `memory/repo`).
  - The trace contract `v1.Trace`/`v1.TraceStep` (contract 1.1, `ISS-TRACE-1`, `ph1-contracts`): explicit `sequence`, `phase`, `actor`, `operation`, `target`, `inputHash`/`outputHash`, `outcome`, `errorClass`, `durationMs`, `tokens`, `costUsd`, `redacted`, and trace-level `truncated`/`redactionVersion`.
  - The canonical trace store (`ISS-TRACE-2`, `ph2-memory`): `memory/repo.TraceRepository` (`Ingest`/`Get`/`List`/`ListByAttempt`/`ListByProject`) and the deterministic `memory/query` `TraceSummary` read surface.
  - The live observation substrate (`ISS-OBSV-1`, `ph3-kernel`) as the trace producer, and the sifter RPC plan (`ISS-SIFTER-1`, `ph4-sifter`) for the production baseline arm.
  - Phase 9 plan/summary and `kernel/aa-joblearn.md`; the architecture boundary (`kernel` imports only `contracts`, `obsv`, `memory`, `toolbox`).
- Missing:
  - Any trace-consuming input: `attribution` accepts only `telemetry.Record`; there is no trace seam.
  - Step-level features and trace-derived attribution; candidates come from aggregate outcome statistics.
  - Artifact distillation: no synthesis of improved subagent artifacts per role/trade/task.
  - Evaluation over traces: `backtest` runs on synthetic `Sample` datasets with fake `Predictor`s; no trace-derived dataset and no real baseline.
  - Gate evidence derived from traces; production wiring of the baseline.
- Known constraints:
  - Learning produces candidates only; `memory` owns canonical state and promotion.
  - No model call in the control path; the `sifter` is reached over RPC, never imported.
  - Evidence is metadata-only; traces are redacted and may be truncated.
  - Determinism: no wall clock, no randomness, no input-order dependence.
  - Capabilities stay behind evidence gates, always with a deterministic fallback and never bypassing a gate or execution contract.

## Scope

### In Scope
- A pure `TraceSource` seam returning bounded traces (or `TraceSummary` projections) plus deterministic fakes.
- Step-level feature extraction from `TraceStep`, content-free and deterministic.
- Trace-derived `Attribution`/`Score`, linking steps to work package, role, trade, and worker.
- Artifact distillation into existing `Candidate` kinds, scoped by role/trade/task and carrying trace provenance.
- A trace-derived backtest dataset, a bounded order-independent `backtest.Run`, and a production baseline arm behind the `backtest.Predictor` seam.
- Gate evidence and capability enablement derived from trace outcomes.
- A store adapter (mirroring `promote/memory.go`) and an end-to-end store → engine → promotion integration test.
- Module documentation, the sub-issue document, and the module summary.

### Out of Scope
- The trace contract, the trace store, the observation substrate, and the sifter RPC transport themselves (`ISS-TRACE-1`, `ISS-TRACE-2`, `ISS-OBSV-1`, `ISS-SIFTER-1`).
- Changing the trace or memory contracts; evolution stays additive and is requested from `contracts` if ever needed.
- Kernel host and orchestrator `Sink` wiring that produces traces (`ISS-OBSV-1`).
- Console surfaces over learned artifacts.
- Model training or fine-tuning; learned signals stay non-authoritative.
- Cross-project or workforce-level learning beyond the project/session scope.

## Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P9U-001 | Consume traces through a pure `TraceSource` seam; the store adapter lives beside the engine like `promote/memory.go` | Keeps the engine pure and unit-testable while `kernel` remains the only importer of `memory` |
| ADR-P9U-002 | Distilled artifacts use the existing `Candidate` kinds and levels; no contract change | The candidate shape already carries provenance, scope, and confidence; contract evolution belongs to `contracts` |
| ADR-P9U-003 | Traces are first-class but the telemetry input is retained | Existing callers and tests keep working; traces are additive evidence, not a breaking replacement |
| ADR-P9U-004 | Extract a fixed, content-free step feature vector (phase, operation, outcome, error class, duration, tokens, cost) | Deterministic and privacy-safe; hashes stay for correlation only |
| ADR-P9U-005 | Evaluation runs on trace-derived datasets; the production baseline is the `sifter` recommender behind the `Predictor` seam | Closes phase 9 deviation 3 without importing `sifter` or calling a model in the engine |
| ADR-P9U-006 | Redaction and truncation are respected and surfaced in reports | Evidence must never carry recoverable input; bounded traces must not distort comparisons silently |
| ADR-P9U-007 | Scoring stays versioned; `MetricVersion` changes only if the semantics change | Scores remain comparable within a version, as phase 9 established |
| ADR-P9U-008 | Gates consume trace-derived counts; capabilities remain disabled by default with a fallback and no bypass | The evidence-gate guarantee is unchanged by the new input |
| ADR-P9U-009 | Candidates remain candidates; promotion stays explicit through the memory lifecycle | Phase 9 ADR-P9-001 still holds |

## Slices

Each slice maps to exactly one commit and is authored after this plan is reviewed. Slice 1 is delivered by the pull request carrying this plan.

| Slice | Goal | Commit message |
|---|---|---|
| 1 | Issue documentation and this plan | `add ph9 joblearn trace distillation plan and issue documentation` |
| 2 | Pure trace-source seam and deterministic fakes | `add joblearn trace source seam` |
| 3 | Step-level, content-free feature extraction | `add joblearn trace features` |
| 4 | Trace-derived attribution and scoring | `add joblearn trace attribution` |
| 5 | Artifact distillation into scoped candidates | `add joblearn artifact distillation` |
| 6 | Trace-derived backtest dataset and deterministic baseline | `add joblearn trace backtest dataset` |
| 7 | `sifter` recommender production baseline behind the RPC seam | `add joblearn sifter baseline adapter` |
| 8 | Trace-derived evidence gates | `add joblearn trace evidence gates` |
| 9 | Trace store adapter and end-to-end promotion integration | `add joblearn trace store integration` |
| 10 | Module update summary | `add ph9 joblearn trace distillation summary` |
| 11 | Link the module update pull request | `link ph9 joblearn trace distillation pull request` |

## Required End State

- [ ] `joblearn` consumes `v1.Trace` through a pure source seam; a trace fixture yields deterministic attributed scores.
- [ ] Step-level features are content-free and honour `redacted`/`truncated`.
- [ ] Distillation yields role/trade-scoped artifact candidates carrying trace provenance.
- [ ] A trace-derived backtest runs within budget, is order-independent, and compares against a deterministic baseline.
- [ ] The production baseline is reached through the `sifter` RPC seam; fakes remain test-only.
- [ ] Gates consume trace evidence; capabilities stay disabled without it; no gate or contract is bypassed; a fallback remains.
- [ ] Distilled candidates persist as `CANDIDATE` through the existing promotion `Sink`.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and the boundary check remain green.

## Exit Criteria

- [ ] Traces are the learning input; telemetry remains supported.
- [ ] A trace fixture produces deterministic scores and at least one distilled artifact candidate.
- [ ] A trace-derived backtest shows improvement over the deterministic baseline and is deterministic.
- [ ] Capabilities remain disabled without evidence; fallback and no-bypass checks are asserted.
- [ ] `kernel` still imports only `contracts`, `obsv`, `memory`, and `toolbox`; `sifter` is reached over RPC.
- [ ] A module update PR is opened on `ph9-joblearn` into `main`.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | step features, trace attribution, distillation, gate decisions, backtest budget and ordering |
| contract | a `v1.Trace` fixture decodes/validates; distilled candidates map to `memory_record` envelopes |
| integration | store → `TraceSource` → engine → promotion `Sink` → `memory/repo`, candidate-only |
| determinism | identical traces yield identical features, candidates, datasets, and reports regardless of input order |
| boundary | `kernel` imports only `contracts`, `obsv`, `memory`, `toolbox`; `sifter` never imported |
| privacy | no content in features or evidence; redaction and truncation respected |
| docs | link check |

## Dependencies And Follow-Up

- **`ISS-TRACE-1` (`ph1-contracts`)** — the `v1.Trace` contract this update consumes.
- **`ISS-TRACE-2` (`ph2-memory`)** — the canonical trace store and `TraceSummary` read surface.
- **`ISS-OBSV-1` (`ph3-kernel`)** — the live observation substrate that produces traces; until it lands, tests use fixtures and the store adapter.
- **`ISS-SIFTER-1` (`ph4-sifter`)** — the governed RPC boundary that supplies the production baseline arm; until it lands, the deterministic fake remains test-only.
- **Follow-up** — console surfaces over learned artifacts, enabling learned routing once a capability passes its gate, and cross-project/workforce-level learning.
