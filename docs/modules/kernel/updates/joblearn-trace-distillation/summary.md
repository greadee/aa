# kernel/joblearn — Module Update Summary: trace distillation and evaluation

> Authored at the **end** of the module update, on branch `ph9-joblearn`. Derived from [plan.md](plan.md).
> Phase PR: [#18](https://github.com/greadee/aa/pull/18)
> Sub-problem: `ISS-LEARN-1` · Parent umbrella: `ISS-TRACE-LOOP`

## Delivered

| Slice | Status | Commit message | Notes |
|---|---|---|---|
| Issue documentation and module plan | Complete | `add ph9 joblearn trace distillation plan and issue documentation` | `docs/issues/`, this plan |
| Trace source seam | Complete | `add joblearn trace source seam` | `joblearn/tracesource` |
| Trace features | Complete | `add joblearn trace features` | `joblearn/features` |
| Trace attribution | Complete | `add joblearn trace attribution` | `attribution/trace.go` |
| Artifact distillation | Complete | `add joblearn artifact distillation` | `joblearn/distill` |
| Trace backtest dataset | Complete | `add joblearn trace backtest dataset` | `backtest/trace.go` |
| Sifter baseline adapter | Complete | `add joblearn sifter baseline adapter` | `joblearn/baseline` |
| Trace evidence gates | Complete | `add joblearn trace evidence gates` | `gate/evidence.go` |
| Trace store integration | Complete | `add joblearn trace store integration` | `tracesource/memory.go` |
| Module update summary | Complete | `add ph9 joblearn trace distillation summary` | this file |
| Phase PR link | Complete | `link ph9 joblearn trace distillation pull request` | follow-up commit |

## What was added

- `joblearn/tracesource` — the pure `Source` seam over `v1.Trace`, a deterministic in-memory `Static` source and a scripted `Fake`, a `Bounds`/`Bound`/`Load` read path that keeps whole traces and validates each one, and the store adapter `MemorySource` over `memory/repo.TraceRepository`.
- `joblearn/features` — step-level, content-free feature extraction: `Extract` yields phase, operation, outcome, error class, duration, tokens, cost, and redaction per step plus deterministic totals; `Set.Key` is a stable content-free key. Input/output hashes and targets are deliberately excluded.
- `attribution/trace.go` — `TraceOutcome` derives the attempt outcome from a bounded trace (failed > blocked > truncated-partial > all-skipped-unknown > succeeded); `AttributeTrace`, `ScoreTrace`, and `DeriveTrace`/`DeriveTraces` link traces to work package, role, trade, and worker through the existing `Attribution`/`Score` types, recording the trace id as evidence.
- `joblearn/distill` — deterministic artifact distillation: records are canonicalized, grouped by role, trade, and work package, and synthesized into `strategy` and `pitfall` candidates with trace evidence and provenance. Titles and content are built from phase and error-class metadata only.
- `backtest/trace.go` — `TraceDataset` builds content-free, outcome-labeled samples from attributed traces; `ConstantBaseline`, `MajorityBaseline`, and `ScopeBaseline` are deterministic baselines (ties broken lexically).
- `joblearn/baseline` — the `Recommender` seam and `SifterPredictor`, which adapts the governed sifter recommender to a `backtest.Predictor` and always falls back deterministically when it is unavailable.
- `gate/evidence.go` — `CountOutcomes` and `EvidenceFromTraces` derive gate evidence from trace outcomes and a measured backtest improvement.
- The end-to-end path, proven by an integration test: canonical trace store → `MemorySource` → attribution/features → distillation → promotion `Sink` → `memory/repo`, candidate-only.

## What this changes about the module and the app

- `kernel/joblearn` learns from step-level traces, not only per-attempt telemetry summaries: it can see where an attempt spent its steps, which phase or error class recurred, and distill scoped artifacts from that evidence. The telemetry input is retained (ADR-P9U-003), so existing callers keep working.
- Capabilities stay disabled until trace-derived evidence satisfies their gate; the production baseline is a seam the kernel host fills with the governed sifter RPC client, and a deterministic fallback always remains.
- Candidates remain candidates: promotion stays explicit through the existing `memory` lifecycle.
- `kernel` still imports only `contracts`, `obsv`, `memory`, and `toolbox`; `sifter` is reached over RPC, never imported.

## Is this part of a larger change?

Yes. This is the learning sub-problem (`ISS-LEARN-1`) of the umbrella change `ISS-TRACE-LOOP`. It consumes the trace contract (`ISS-TRACE-1`, `ph1-contracts`) and the canonical trace store (`ISS-TRACE-2`, `ph2-memory`) and, in production, the observation substrate (`ISS-OBSV-1`, `ph3-kernel`) and the governed sifter boundary (`ISS-SIFTER-1`, `ph4-sifter`). Because the trace contract and store land on `ph1`/`ph2`, this branch is stacked on that substrate; the PR is merge-ordered after them.

## Validation

| Gate | Result | Evidence |
|---|---|---|
| `go build` | Pass | `kernel` builds under the workspace |
| `go vet` | Pass | clean |
| `go test` | Pass | `joblearn` and its subpackages, including the new suites |
| `gofmt -l` | Pass | clean |
| Boundaries | Pass | `tools/archtest` reports ok; no sifter/forge/sync import |
| Determinism | Pass | identical traces yield identical features, candidates, datasets, and reports regardless of input order |
| Privacy | Pass | features and candidate content exclude hashes, targets, and any recoverable input |
| Gate safety | Pass | capabilities stay disabled without evidence; a missing fallback disables the capability |
| Store integration | Pass | store → source → engine → promotion `Sink` → `memory/repo`, candidate-only and idempotent |
| Docs link check | Pending | runs on CI after push |

## Decisions affirmed

| # | Decision | Outcome |
|---|---|---|
| ADR-P9U-001 | Consume traces through a pure `TraceSource` seam; the store adapter lives beside the engine | Affirmed — `tracesource.MemorySource`, mirroring `promote/memory.go` |
| ADR-P9U-002 | Distilled artifacts use the existing `Candidate` kinds and levels | Affirmed — `strategy`/`pitfall` at role/task levels; no contract change |
| ADR-P9U-003 | Traces are additive; telemetry input retained | Affirmed — `Score`/`Derive` unchanged; `DeriveTraces` added alongside |
| ADR-P9U-004 | Fixed, content-free step feature vector | Affirmed — hashes and targets excluded from `features.Step` |
| ADR-P9U-005 | Evaluation on trace-derived datasets; production baseline behind the `Predictor` seam | Affirmed — `baseline.SifterPredictor` over a `Recommender` seam with a deterministic fallback |
| ADR-P9U-006 | Redaction and truncation respected and surfaced | Affirmed — `TraceOutcome` treats truncation as partial; `features.Set` surfaces both flags |
| ADR-P9U-007 | Scoring stays versioned | Affirmed — `MetricVersion` unchanged |
| ADR-P9U-008 | Gates consume trace-derived counts; enabled only with a fallback and no bypass | Affirmed — `gate.EvidenceFromTraces` + policy |
| ADR-P9U-009 | Candidates stay candidates; promotion stays explicit | Affirmed — integration test ends at `CANDIDATE` |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | Seam over `v1.Trace` or `TraceSummary` projections | Seam over `v1.Trace` only | Keeps the engine package free of `memory`; the store adapter and its projections stay in the kernel-host adapter layer |
| 2 | Feature vector listed `target` | Omitted target and hashes | Guarantees a content-free vector; target is a content-bearing reference |
| 3 | A production sifter baseline | The `Recommender` seam plus a deterministic fallback; the RPC client is the kernel-host adapter | Keeps `sifter` out of the kernel import graph; fakes remain test-only |
| 4 | Distillation could extend `candidates` | A separate `distill` package | Keeps the telemetry-based generator unchanged and additive |

## Follow-Up

- Kernel host and orchestrator wiring that produces traces (`ISS-OBSV-1`) and the kernel-side production runtime/sifter client (`ISS-SIFTER-1`), so the baseline runs over the real RPC.
- Console surfaces over learned artifacts.
- Enabling learned routing once a capability passes its gate; cross-project and workforce-level learning.
