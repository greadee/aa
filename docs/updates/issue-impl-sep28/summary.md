# Issue Implementation Sprint (Sep 28) — Summary

> Authored at the **end** of the update phase (`issue-impl-sep28`). Derived from [plan.md](./plan.md). As-built diagram: [summary.uml](./summary.uml).
> Branch: `dev`. Sprint PR: [#21](https://github.com/greadee/aa/pull/21) (`dev` → `main`). Issues: [#23–#32](https://github.com/greadee/aa/issues); umbrella [#22](https://github.com/greadee/aa/issues/22).

## Delivered

| Stage | Issue | Status | Commit(s) | Notes |
|---|---|---|---|---|
| Objective 0 | spine | Complete | `f43695d`, `8baff82`, `c056649` | sprint plan/UML, ten issue docs, GitHub issues + milestone |
| 0.1 | R4 | Complete | `33f4995` | contracts v2 import aliases consistent |
| 0.2 | R5–R8, R10 | Complete | `ed1991e` | scheduler comments, §5.6/§5.12, phase-4 label, OpenAPI `ui`, reserved table |
| — | cadence | Complete | `6a46704` | sprint scoped to one commit per issue (SD-7) |
| 1.1 | Issue 1 + A5 decision | Complete | `3fe6664` | boundaries §4.6, terminology §1.1 (`role ≠ capability ≠ parallelism`), ADR-0143 |
| 1.2 | A4 | Complete | `f51b0bb` | contracts v1 removed from the active path |
| 1.3 | Issue 4 | Complete | `4f3a49c` | RPC v1 hardening (versioning, contract-major gate, capability negotiation, deadlines/retries/idempotency) |
| 2.1 | Issue 2 | Complete | `682005d` | deterministic retrieval (`Retriever`, ranking, provenance, limits) |
| 2.2 | Issue 3 + A8(context) | Complete | `0159ef2` | needs/selection/budget/compression/assembly; `kernel/context` ownership |
| 3.1 | Issue 5 / A7 | Complete | `4a558f2` | `runtime/sandbox` isolation |
| 3.2 | Issue 6 | Complete | `42008bd` | `toolbox/computeruse` + runtime sandbox gate |
| 3.2a | A8(desktop) | Deferred | `8772167` | moved out of Issue 6 to `ph10-ui` |
| 3.3 | Issue 8 | Complete | `9e4e710` | scheduler concurrency/retries/cancellation/aggregation |
| 3.4 | A5 execution | Complete | `c1d2313` | `kernel/intake` → `runtime/intake` |
| 4.1 | Issue 7 | Complete | `9444085` | role/model/compute allocation + populated `registry/models` |
| 4.2 | A8(routing/budgets) | Complete | `b7c4339` | `kernel/allocator/routing` owns policy |
| 5.1 | Issue 9 + A9 | Complete | `3e4b890` | experience feedback loop + apprenticing/studying |
| 5.2 | Issue 10 + A9 | Complete | `c8b1506` | seam observability, identity chain, e2e scenarios |
| 6 | closeout | Complete | this commit | summary + UML; deferred work recorded |

- Starting ref: `main @ bf8e09d`. Branch `dev`; all work committed on `dev` and pushed. No force-push.
- Cadence: **one commit per issue** (SD-7, scoped to this sprint), with a stop at each issue boundary for review.

## Validation

| Gate | Result | Evidence |
|---|---|---|
| Formatting | Pass | `gofmt -l .` clean |
| Go | Pass | all 11 modules `go build`/`go vet`/`go test ./...` |
| Boundaries | Pass | `tools/archtest` → architecture boundaries ok |
| Contracts Go | Pass | `contracts` tests |
| Contracts Python | Pass | `contracts/python` unittest (`OK`) |
| Contracts TypeScript | Pass | `tsc --noEmit` |
| Inference | Pass | `runtime/inference` pytest (unit), ruff, mypy |
| Docs links | Pass | fence-aware scan → 0 broken |

## Decisions (ADRs)

| ADR | Decision |
|---|---|
| [ADR-0143](../../adr/ADR-0143-decide-intake-and-contract-ownership.md) | `intake` → `runtime`; `contract` stays a shared kernel package |
| [ADR-0144](../../adr/ADR-0144-rpc-v1-boundary-hardening.md) | RPC v1 boundary hardening |
| [ADR-0145](../../adr/ADR-0145-deterministic-retrieval-ranking-and-provenance.md) | Deterministic retrieval ranking and provenance |
| [ADR-0146](../../adr/ADR-0146-context-compilation-and-a8-extraction.md) | `kernel/context` owns control-plane context (A8) |
| [ADR-0147](../../adr/ADR-0147-runtime-sandbox-isolation.md) | Runtime sandbox isolation |
| [ADR-0148](../../adr/ADR-0148-computer-use-capability-and-runtime-wiring.md) | Computer-use capability + runtime wiring |
| [ADR-0149](../../adr/ADR-0149-scheduler-concurrency-retries-and-aggregation.md) | Scheduler concurrency/retries/aggregation |
| [ADR-0150](../../adr/ADR-0150-role-model-compute-allocation-separation.md) | Role/model/compute allocation separation |
| [ADR-0151](../../adr/ADR-0151-routing-and-budget-policy-in-the-allocator.md) | Routing/budget policy in the allocator |
| [ADR-0152](../../adr/ADR-0152-experience-feedback-and-learning-states.md) | Experience feedback and learning states |
| [ADR-0153](../../adr/ADR-0153-observability-identity-chain-and-accounting.md) | Observability identity chain and accounting |

## R/A resolution

| Finding | Status | Where |
|---|---|---|
| R4 alias | Resolved | `33f4995` |
| R5 comments | Resolved | `ed1991e` |
| R6 §5.6/§5.12 | Resolved | `ed1991e` |
| R7 phase-4 label | Resolved | `ed1991e` |
| R8 OpenAPI console | Resolved | `ed1991e` |
| R9 package names | No-op | set by the refactor |
| R10 reserved packages | Recorded/resolved | architecture §4.1; Future register |
| A4 v1 sunset | Resolved | `f51b0bb` |
| A5 intake/contract | Resolved | `3fe6664`, `c1d2313` |
| A6 model/compute | Resolved | `9444085` |
| A7 sandbox | Resolved | `4a558f2` |
| A8 inference extraction | Context resolved (`0159ef2`); routing/budget policy resolved (`b7c4339`); desktop deferred to `ph10-ui` (`8772167`) | ADR-0146, ADR-0151 |
| A9 apprenticing/identity | Resolved | `3e4b890`, `c8b1506` |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | One commit per slice | **One commit per issue** (sprint-scoped) | User directive to reduce commits to distinct units; plan SD-7 updated |
| 2 | A8(desktop→ui) inside Issue 6 | Deferred to `ph10-ui` | UI work is out of sprint scope (`8772167`) |
| 3 | A8 context/routing as literal cross-language moves | **Ownership** moved to the control plane; Python execution surfaces retained | A cross-language relocation of execution-local code is neither safe nor desired; residual wiring recorded below |
| 4 | Issues closed during the sprint | Issues linked to PR #21 (`Closes #n`) and left open until it merges | User directive to close at PR completion |

## Future register (deferred)

- **A8 routing/budget residual:** wire the execution path to pass the allocator's routing decision into inference and retire the duplicate Python `runtime/inference/routing` policy.
- **A8 `desktop → ui/surfaces`:** relocate the inference desktop surface under `ui` (owned by `ph10-ui`).
- **Reserved packages:** `registry/{teams,routines,policies}`, `runtime/lifecycle`.
- **UI surface:** the Wails/React surface (tracked with `ui`; `ph10-ui`).
- **Repository-wide conventions:** consider whether the one-commit-per-issue cadence should become repo-wide (currently sprint-scoped; `AA.md` rule 5 unchanged).

## Metrics

- Sprint commits on `dev`: 3 (objective 0) + 16 (stages 0–5) + 1 (closeout) = 20.
- Issues implemented: 10 (#23–#32); umbrella #22.
- ADRs: ADR-0143 … ADR-0153 (11 new).
- Modules touched: `contracts, registry, runtime, obsv, kernel, memory, toolbox, tools`.

## Exit criteria

- [x] All ten issues implemented and closed (docs; GitHub closes on PR merge), or explicitly deferred with a record.
- [x] R4–R9 and A4–A9 resolved or recorded.
- [x] Boundaries and contracts coherent; `archtest` and all module suites green.
- [x] Architecture/terminology/directives/indexes/MANIFEST/diagrams current; docs link check green.
- [x] Sprint summary + UML authored. (Sprint PR #21 merges to `main`; branch retained.)
- [x] Review/audit comments posted per issue; a closeout comment is posted on PR #21.

## Retrospective

### Repeat
- One commit per issue kept a large sprint reviewable and mapped cleanly to GitHub issues.
- Each issue closed with a justification comment plus a `Closes #n` link, so the audit trail is complete without private context.
- Writing an ADR per decision made the staged A8 extractions defensible and explicit.

### Avoid
- Do not fold an out-of-scope item (UI/desktop) into an in-scope issue; scope it out early.
- Cross-language "moves" of execution-local code should be planned as ownership changes with a follow-up, not as file relocations.
