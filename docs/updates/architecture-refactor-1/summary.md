# Architecture Refactor 1 — Update Phase Summary

> Authored at the **end** of the update phase. Derived from [plan.md](./plan.md). As-built diagram: [summary.uml](./summary.uml).
> Branch: `dev` (PR #20). This was a **behavior-preserving** structural refactor; no new functionality was implemented.

## Delivered

| Slice | Sub | Status | Commit | Notes |
|---|---|---|---|---|
| 1. Update phase plan and UML | — | Complete | `65f6b64` | `docs/updates/architecture-refactor-1/{plan.md,plan.uml}` |
| 2. ADR store and historical backfill | — | Complete | `ee470bd` | 130 backfilled ADRs + index; `docs/adr/` |
| 3. Module documentation structure | — | Complete | `c29b3ba` + `0ba30ac` | `docs/modules/**`; system rules; update phases |
| 4. Move module update docs into docs | — | Complete | `abd3b6a` | 7 module-update pairs relocated |
| 5. registry module skeleton | — | Complete | `a4d3862` | roles + capabilities moved; models/teams/routines/policies reserved |
| 6. runtime module (top level) | — | Complete | `f4c14cf` | `kernel/runtime` → `runtime/worker`; lifecycle/sandbox/inference reserved |
| 7. ui module | — | Complete | `25deaec` | `console` → `ui`; components/views/state/surfaces |
| 8. workspace/archtest/CI | — | Complete | `e85164f` | registry/runtime/ui in `go.work`, archtest, CI |
| 9. kernel/allocator/planner | — | Complete | `d02efdf` | `kernel/plan` → `kernel/allocator/planner` |
| 10. allocators; retire kernel/registry | — | Complete | `e1a29b2` | role/model/compute allocators; worker instances → `runtime/worker` |
| 11. kernel/scheduler | — | Complete | `8121452` | orchestrator + control merged |
| 12. kernel transitional boundaries | — | Complete | `7b36d81` | intended ownership documented |
| 13. contracts v2 | — | Complete | `8fde39d` | new major generation; specs/plans/routine |
| 14. retire trade | — | Complete | `f98d10d` | removed from contracts v2, registry, joblearn |
| 15. wire contracts | — | Complete | `3e06886` | consumers on v2; registry/allocator aliases |
| 16. sifter → inference under runtime | — | Complete | `69a3c4b` | Python service renamed and relocated |
| 17. `memory/query` → `memory/retrieval` | — | Complete | `4840b9a` | rename only |
| 18. docs/directives/indexes sweep | — | Complete | `2fd4ccf` | canonical docs reconciled (ADR-0142) |
| 19. diagrams | — | Complete | `a3aa9da` | architecture maps + `summary.uml` |
| 20. future register and readiness | — | Complete | this commit | this file |

- Starting ref: `main @ 5197ca8`. Branch `dev` created from it; **Draft PR [#20](https://github.com/greadee/aa/pull/20)**.
- No force-push; each slice is exactly one commit.

## Validation

| Gate | Result | Evidence |
|---|---|---|
| Go build / vet / test | Pass | all 11 modules (`contracts, registry, runtime, obsv, kernel, memory, sync, forge, toolbox, visualizer, ui`) |
| Formatting | Pass | `gofmt -l .` clean |
| Boundaries | Pass | `tools/archtest` ok; registry/runtime/visualizer layering enforced |
| Contracts Go conformance | Pass | `go/v1` and `go/v2` decode/round-trip/reject |
| Contracts Python | Pass | `contracts/python` unittest (v2 fixtures) |
| Contracts TypeScript | Pass | `tsc --noEmit` (v1 + v2) |
| Inference service | Pass | `runtime/inference` pytest (unit/integration), ruff format/check, mypy |
| Docs links | Pass | fence-aware scan of docs + module READMEs → 0 broken |

## Decisions affirmed

| # | Decision | Outcome | ADR |
|---|---|---|---|
| D-1/D-2 | sifter reduced to provider/execution, renamed inference under runtime | Affirmed | ADR-0132, ADR-0140 |
| D-3 | trade retired | Affirmed | ADR-0138 |
| D-4/D-5 | registry module; definitions only | Affirmed | ADR-0131 |
| D-6 | obsv name kept | Affirmed | — |
| D-7 | contracts major v2 | Affirmed | ADR-0137, ADR-0139 |
| D-9/D-10 | runtime Go module + nested Python inference; kernel → runtime | Affirmed | ADR-0132 |
| D-11/D-17 | kernel allocator (planner/role/model/compute); worker instances in runtime | Affirmed | ADR-0134 |
| D-13 | scheduler from orchestrator + control | Affirmed | ADR-0135 |
| D-14 | console → ui (cli/tui surfaces) | Affirmed | ADR-0133 |
| D-15 | kernel transitional placement documented | Affirmed | ADR-0136 |
| D-18 | docs architecture (ADR store, modules, update phases) | Affirmed | ADR-0142 |
| D-8 | `memory/query` → `memory/retrieval` | Affirmed | ADR-0141 |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | Branch `dev` presumed present | Created `dev` from `main @ 5197ca8`; the phase-8 `obsv` dependency was already merged | Normal branch setup for the update phase |
| 2 | Slice 13 "bump to v2.0" ambiguous (in-place vs new directory) | New major generation under `schemas/v2`/`go/v2`/`typescript/v2`/`python v2`, per `contracts/POLICY.md`; `v1` retained | Policy requires a new major directory for breaking changes |
| 3 | Slice 14 "retire trade ... joblearn" | Per-trade learned routing replaced by per-role routing | `trade` was the only cross-role grouping dimension; capability-based routing returns with the ten-issue Sifter work |
| 4 | Slice 19 "update diagrams" | Architecture mermaid refreshed and `summary.uml` added; `plan.uml` largely as-built | Diagrams are change-scoped |

## Future register

Reserved — **not built by this refactor** (all ten-issue features remain to be
implemented):

- **Model allocation** behavior (`kernel/allocator/model_allocator`) and the model
  registry population (`registry/models` wiring to the inference catalog).
- **Compute allocation** behavior (`kernel/allocator/compute_allocator`):
  worker count, parallelism, budgets, placement.
- **Workforce / team / crew** logic; `registry/policies` remains a catalogue only.
- **Routine engine** and **workflow learning** (routine evolution).
- **Retrieval ranking/similarity** (`memory/retrieval` is deterministic reads only).
- **Sandbox** implementation (`runtime/sandbox` is a reserved boundary).
- **Inference capability extraction**: routing → allocator, budgets →
  allocator/contract, verification → runtime, context → `kernel/context`, catalog →
  `registry/models`, desktop → `ui/surfaces`.
- **Apprenticing / studying** learning behavior; observability identity-chain wiring.
- A **path-level v1 sunset** (v1 directories remain as the previous generation).
- Kernel moved-slice **(2a/2b)** summaries and PR-link slices beyond this update.

## Ten-issue translation table

The ten-issue handoff uses pre-refactor names. Translate against the as-built repo:

| Ten-issue name | As-built name |
|---|---|
| `aa-kernel/sifter` | `aa-kernel/allocator` (`planner`, `role_allocator`, `model_allocator`, `compute_allocator`) |
| Capability (allocation dimension) | Model allocation |
| Parallelism (allocation dimension) | Compute allocation |
| `aa-kernel/runtime` | top-level `runtime/` |
| `aa-agents/roles` | `registry/roles` |
| `aa-observability` | `obsv` |
| `aa-memory/retrieval` | `memory/retrieval` |
| `aa-sifter` (Python) | `runtime/inference` |
| `aa-console` | `ui` |
| `trade` | retired (role + capability) |

## Readiness for the ten-issue phase

- [x] Boundaries established and enforced (`tools/archtest`).
- [x] The new generation (contracts v2) exists and consumers use it.
- [x] The kernel allocator/scheduler and top-level runtime exist; execution mechanics moved out of the kernel.
- [x] The registry owns durable definitions; runtime owns instances.
- [x] Documentation architecture established (ADR store, module docs, update phases) and canonical docs reconciled.
- [x] No missing functionality was implemented; the Future register is explicit.
- [x] All gates green.

**Ready for the ten-issue implementation phase.**

## Metrics

- Commits: 20 slices (+ this summary) on `dev`; 1 Draft PR (#20).
- Modules: 11 (`contracts, registry, runtime, obsv, kernel, memory, sync, forge, toolbox, visualizer, ui`).
- ADRs: ADR-0131 … ADR-0142 (12 new); 130 historical ADRs backfilled.
- Contracts: v2 generation (21 schemas) + v1 retained.

## Retrospective

### Repeat
- Slice-per-commit with a stop for review kept the refactor reviewable despite its size.
- Recording every decision as an ADR made the trade retirement, contracts v2, and renames auditable.
- A fence-aware docs link scan at each slice caught every relative-link break from the moves.

### Avoid
- Start a refactor on a branch that predates its declared dependency; verify the dependency before slice 1.
- Do not defer "current-state" doc updates to a final sweep if the names change mid-refactor; update them with each move.
