# Phase 9 — Job Learning: Summary

> Authored at the **end** of the phase. Derived from [plan.md](./plan.md). As-built diagram: [summary.uml](./summary.uml).

## Delivered

| Slice | Status | Commit | Notes |
|---|---|---|---|
| Phase plan and UML | Complete | `3e763d5` | plan.md + plan.uml |
| Core types and errors | Complete | `934d9fe` | `kernel/joblearn/{types,errors,clock}.go` |
| Attribution and scoring | Complete | `1ca775f` | `kernel/joblearn/attribution` (versioned scores) |
| Summary determinism fix | Complete | `c9b6bee` | canonical accumulation order (extra fix commit) |
| Candidate generation | Complete | `67c115d` | `kernel/joblearn/candidates` (patterns, pitfalls, strategies, hints) |
| Similarity and conflict | Complete | `ae9c380` | `kernel/joblearn/similarity` (feature clusters + conflict) |
| Evidence gates | Complete | `a7b0059` | `kernel/joblearn/gate` (capability registry, disabled by default) |
| Backtest evaluation | Complete | `ab3fa43` | `kernel/joblearn/backtest` + benchmark |
| Routing hints | Complete | `c51c70c` | `kernel/joblearn/route` (hint + deterministic fallback) |
| Memory promotion wiring | Complete | `2b39870` | `memory/repo/memory.go` + `kernel/joblearn/promote` |
| Phase summary and UML | Complete | this commit | summary.md + summary.uml + README |

- Starting ref: `main @ f9063fd` (after phase 8 merge and record).
- Tracking issue: none (phase executed via the phase PR).
- Milestone: none (matches phases 0–8 on this repository).
- Phase PR: [#10](https://github.com/greadee/aa/pull/10) — merged
- Merge commit: `8be33f4`
- Branch: `ph9-joblearn` (retained after merge on `aa`)

## Validation

| Gate | Result | Evidence |
|---|---|---|
| Go build / vet | Pass | `kernel/...` and `memory/...` build and vet clean |
| Go tests | Pass | 79 tests + 1 benchmark across 9 packages (`joblearn` ×8, `memory/repo`) |
| Formatting | Pass | `gofmt -l .` clean |
| Boundaries | Pass | `archtest` ok; `kernel` imports `contracts`, `obsv`, `memory`, `toolbox` only; `sifter`/`sync`/`forge` never imported |
| Attribution / scoring | Pass | `TestAttributeLinks`, `TestScoreComponents`, `TestScoreClampsOverLimit`, `TestScoreZeroLimitsClamp`; scores versioned via `MetricVersion` |
| Score determinism | Pass | `TestScoreIsDeterministic`, `TestSummarizeOrderIndependent`; canonical accumulation order |
| Candidate generation | Pass | `TestGenerateCandidates`, `TestGenerateBelowThreshold`, `TestGenerateNoRoutingWithoutMargin`; candidates carry provenance, level, applicability |
| Candidate determinism | Pass | `TestGenerateDeterministicAcrossCalls`, `TestGenerateOrderIsStable` |
| Similarity / conflict | Pass | `TestSimilarity`, `TestClusterGroups`, `TestConflictsDetectsOpposingCandidates`, `TestConflictsOrderIndependent` |
| Evidence gates | Pass | `TestDefaultDisabled`, `TestEvaluateEnablesOnEvidence`, `TestEvaluateDisabledReasons`, `TestSetPolicyOverrideResets` |
| Backtest | Pass | `TestRunPolicyBeatsBaseline`, `TestRunDeterministic`, `TestRunOrderIndependent`, `TestRunBudget`; `BenchmarkRun` |
| Routing fallback / no-bypass | Pass | `TestDisabledWithholdsButFallsBack`, `TestFallbackWhenNoMargin`, `TestApprovalWithheld`, `TestContractIsNotBypassed` |
| Memory candidate-only | Pass | `TestMemoryRecordProposeIsCandidateOnly`, `TestMemoryRecordProposeRejectsPromoted`, `TestProposeNeverPromotes` |
| Promotion idempotency | Pass | `TestMemoryRecordProposeIsIdempotent`, `TestProposeIsIdempotent`, `TestIdentityIsDeterministic`, `TestProposeDeduplicatesWithinBatch` |
| Explicit transitions | Pass | `TestMemoryRecordTransitionStaysExplicit`, `TestPromoteIsExplicitAndValidated`, `TestMemoryPromotionEndToEnd` |
| Integration | Pass | `TestMemoryPromotionEndToEnd` runs candidate → `MemorySink` → `memory/repo` → VALIDATED → ACTIVE against a real store/projection |
| Security / privacy | Pass | metadata-only evidence; no model calls anywhere in the loop; gates/contracts never bypassed |
| Docs links | Pass | runs on CI after push |
| Phase audit | Complete | see Audit below |

## Decisions Affirmed

| # | Decision | Outcome | Evidence |
|---|---|---|---|
| ADR-P9-001 | Learning produces candidates only; `memory` owns canonical state and promotion | Affirmed | `MemoryRecordRepository.Propose` is CANDIDATE-only; `promote.Proposer` never promotes; `TestProposeNeverPromotes`, `TestMemoryRecordProposeRejectsPromoted` |
| ADR-P9-002 | Attribution and scoring are deterministic functions of evidence | Affirmed | `attribution.Summarize` accumulates in canonical order; `TestScoreIsDeterministic`, `TestSummarizeOrderIndependent` |
| ADR-P9-003 | Similarity is feature-based and deterministic (no embeddings) | Affirmed | `similarity` clusters by features; `TestClusterGroups`, `TestConflictsOrderIndependent` |
| ADR-P9-004 | Every capability sits behind an explicit gate and is disabled by default | Affirmed | `gate.NewRegistry` registers all known capabilities disabled; `TestDefaultDisabled` |
| ADR-P9-005 | Enablement requires enough outcomes and a backtest win over a baseline | Affirmed | `gate.Evaluate` + `backtest.Report.Improvement`; `TestEvaluateDisabledReasons`, `TestEvaluateEnablesOnEvidence` |
| ADR-P9-006 | Learned routing is a hint; a deterministic fallback is always returned | Affirmed | `route.Hint.Fallback`/`Choice`; `TestDisabledWithholdsButFallsBack`, `TestFallbackWhenNoMargin` |
| ADR-P9-007 | Learned routing never bypasses a human gate or execution contract | Affirmed | `TestApprovalWithheld`, `TestContractIsNotBypassed`; `Hint.Authoritative` always false |
| ADR-P9-008 | Candidates carry provenance and scope (`level`, `applicability`) | Affirmed | `Candidate` fields + `promote` mapping; `TestGenerateCandidates`, `TestProposeMapsCandidateToCandidateRecord` |
| ADR-P9-009 | The engine is pure; persistence and promotion go through seams | Affirmed | `promote.Sink` with a fake in unit tests; `MemorySink` adapts `memory/repo`; `TestMemoryPromotionEndToEnd` |
| ADR-P9-010 | Observability is deterministic data reports | Affirmed | `backtest.Report` and gate `Decision` structs; `TestRunDeterministic`, `TestDecisionsSorted` |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | One commit per slice, exactly as listed | An extra fix commit (`c9b6bee`) between slices 3 and 4 | `attribution.Summarize` needed a canonical, content-derived accumulation order so scores, candidates, and reports are order-independent; recorded as its own commit rather than amending slice 3 |
| 2 | Candidate provenance flows to memory unchanged | `promote` normalizes the source (`aa-kernel/joblearn` → `aa-kernel-joblearn`) | The generator's source contains `/`, which is not a valid contract identifier, and `memory_record` envelope validation requires one; slice 4's constant was left untouched to preserve its provenance contract |
| 3 | Compare against the sifter recommender baseline | A `backtest.Predictor` seam with deterministic fakes | Production kernel↔sifter RPC wiring is out of scope; `sifter` is reached over RPC, and the plan already specified a seam + fake stand-in |
| 4 | Observation evidence from `obsv` | Evidence arrives as telemetry and history | `obsv` remains scaffolded with no exported API; adoption is a follow-up |

## Deferred / Follow-Up

- Production kernel↔sifter RPC wiring behind the backtest baseline seam; the deterministic fakes then become test-only.
- `obsv` observation evidence behind the attribution inputs (live stream instead of telemetry/history).
- Console surfaces over the deterministic gate and backtest reports.
- Enabling learned routing in production once a capability passes its evidence gate; capabilities remain disabled by default.
- Cross-project or workforce-level learning beyond the project/session scope.

## Metrics

- Commits: 11 (10 implementation + this summary)
- Packages: 9 (`joblearn`, `attribution`, `candidates`, `similarity`, `gate`, `backtest`, `route`, `promote`, `memory/repo`)
- Tests: 79 tests + 1 benchmark
- ADRs: ADR-P9-001 … ADR-P9-010
- Pitfalls recorded: 0
- PRs: 1 phase PR (#10)
- Issues: none

## Audit

| Severity | Count | Notes |
|---|---|---|
| P0 | 0 | — |
| P1 | 0 | — |
| P2 | 0 | — |
| P3 | 3 | sifter baseline RPC deferred (deviation 3); `obsv` evidence deferred (deviation 4); console reports surface deferred |

Audit result: no blocking findings. Boundary rule holds: `kernel` imports only `contracts`, `obsv`, `memory`, and `toolbox`; `sifter`, `sync`, and `forge` are never imported. Learning produces candidates only and never promotes; capabilities are disabled by default and enable only on attributed evidence plus a backtest win; every learned routing hint carries a deterministic fallback and never bypasses a gate or execution contract. No model is called anywhere in the loop and evidence is metadata-only.

## Retrospective

### Repeat
- Keep the whole loop pure and deterministic: because attribution, scoring, similarity, gates, and backtests have no wall-clock or random state, order-independence tests were cheap and caught real issues.
- Make identity content-derived; hashing candidate kind/level/scope/title made promotion idempotent and deterministic without a coordination table.
- Prove the memory seam against a real `store`/`projection`/`repo`, not only a fake, so "persists as CANDIDATE" is tested end to end.
- Keep promotion behind a `Sink` seam so the engine never imports the store and unit tests need no fixture.

### Avoid
- Do not let a producer's provenance source drift from the contract identifier grammar; normalize at the boundary or emit a valid value upstream.
- Do not accumulate floating-point scores in input order; canonicalize first or the "same evidence yields the same score" guarantee is order-dependent.
- Do not treat learned signals as authoritative; the fallback and no-bypass checks must be asserted, not assumed.

### As-Built Diagram

[summary.uml](./summary.uml)
