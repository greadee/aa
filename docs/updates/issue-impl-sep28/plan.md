# Issue Implementation Sprint (Sep 28) — Plan

> **Type:** update phase (`issue-impl-sep28`). **Branch:** `dev`. **Sprint PR:** #21 (Draft, `dev` → `main`).
> **Predecessor:** [architecture-refactor-1](../architecture-refactor-1/summary.md) (merged `bf8e09d`).
> **Tracking:** umbrella [ISS-IMP](../../issues/post-refactor-implementation.md) ([#22](https://github.com/greadee/aa/issues/22)) · milestone *Issue implementation sprint (Sep 28)* · sub-issues #23–#32.
> **Source of findings:** [PR #20](https://github.com/greadee/aa/pull/20) review (R1–R10) and repository audit (A1–A9); R1–R3 are already resolved; **R4–R9 and A4–A9 are folded into this sprint**.
> **Diagram:** [plan.uml](./plan.uml).

## Objective

Implement the ten post-refactor issues (canonical architecture/contracts → observability/hardening)
in **stages that group work by module area** to minimise cross-module jumping, folding in the refactor
review findings **R4–R9** and audit deferrals **A4–A9**, and closing out every issue with full
project/work/issue history.

## Starting State

- Starting ref: `main @ bf8e09d` (architecture-refactor-1 merged). Branch `dev` fast-forwarded to it.
- Available: 11 modules (`contracts, registry, runtime, obsv, kernel, memory, sync, forge, toolbox, visualizer, ui`); contracts **v2** current; kernel `allocator` + `scheduler`; top-level `runtime` (worker/lifecycle/sandbox/inference); `registry` definitions; `memory/retrieval`; `obsv`; `docs/adr/` store; `docs/modules/**`; `docs/updates/**`.
- Missing (to be delivered here): the ten issues, their GitHub issues, and the sprint documentation.
- Known constraints (from the refactor): no missing behavior implemented under the refactor; transitional kernel placement documented; sandbox not implemented; contracts v1 retained; `trade` retired.

## Scope

### In scope
- The ten issues (below), delivered as stages/substages, **one commit per issue**.
- Resolution of R4–R9 (review) and A4–A9 (audit deferrals).
- Sprint plan/summary/UML, one ADR per decision, and full doc/index/manifest/diagram upkeep.
- GitHub issue creation (objective 0) and closeout (`Closes #n`).

### Out of scope
- The Wails/React UI surface (tracked with `ui`); anything not in the ten issues or R/A lists — raise as a new issue instead.

## Decisions

| # | Decision | Reasoning |
|---|---|---|
| SD-1 | One sprint branch `dev`, one sprint PR #21 (`dev` → `main`); the refactor is already merged, so #21 is sprint-only | Keeps the refactor and the issue work reviewable separately; #20 is the findings source |
| SD-2 | Group issues into **stages by module area**; each issue/substage is one commit | Minimises jumping across the codebase and keeps every commit a distinct, simply-describable change |
| SD-3 | Fold R4–R9 and A4–A9 into the stages where they belong (see map); **R9 is a no-op** (allocator names already set by the refactor) | Resolve findings where the relevant code is already open |
| SD-4 | Create the ten GitHub issues + milestone in objective 0, after the plan | Execution is visible and closeable per issue |
| SD-5 | Every issue closes with history: tick `docs/issues/<slug>.md`, project history (`docs/modules/**`), work history (this folder), `Closes #n` | Durable, auditable record |
| SD-6 | Post a **code-review-format** PR comment at each stage end and a **repository-audit-format** comment per issue/phase | The refactor's comments were high-value; keep the practice |
| SD-7 | **One commit per issue/substage.** Fold closely-related sub-tasks into a single commit when one simple message describes them; only genuinely distinct changes get their own commit. Stop at each substage boundary for review. Supersedes the per-slice commit rule in [sprint-planning.md](../../github-os/sprint-planning.md) **for this sprint only** | Fewer, more meaningful commits; each commit is one coherent change |

## Issue → stage map

| Issue | Stage | Notes |
|---|---|---|
| 1 Canonical architecture & contracts | Stage 1 | + A4 (v1 sunset), R6/R7/R8 docs |
| 2 Deterministic retrieval | Stage 2 | `memory/retrieval` |
| 3 Context compilation/budgeting/assembly | Stage 2 | `kernel/context`; + A8(context) |
| 4 RPC/boundary/version-skew hardening | Stage 1 | contracts/rpc; + A5 decision |
| 5 Sandbox execution | Stage 3 | `runtime/sandbox` (A7) |
| 6 Computer-use/tool execution | Stage 3 | `toolbox` + runtime |
| 7 Role/capability/model/compute allocation | Stage 4 | `kernel/allocator`, `registry`; + A6, A8(routing/budgets) |
| 8 Scheduling & multi-agent execution | Stage 3 | `kernel/scheduler` + runtime; + A5 execution |
| 9 Learning/experience feedback | Stage 5 | `kernel/joblearn` + memory; + A9(apprenticing/studying) |
| 10 Observability/hardening/e2e | Stage 5 | `obsv` + all; + A9(identity chain) |

## Stages and commits

Each issue/substage below is **one commit**; closely-related sub-tasks are folded into it. **DoD per commit:** tests green; `gofmt`/`go vet`/`archtest` clean; affected architecture/terminology/docs + ADRs updated; docs link check valid.

### Stage 0 — Cleanup & stabilization (no behavior change)

- **0.1 — contracts consumer aliasing (R4).** Replace the confusing `v1 "…/contracts/go/v2"` alias with `v2` across consumers. **Commit** `alias contracts v2 imports consistently`. *(done)*
- **0.2 — comments and current-state docs (R5–R8, R10).** Fix `kernel/scheduler` stale `orchestrator` comments (R5); consolidate architecture §5.6/§5.12 (R6); fix the phase-4 label (R7); update OpenAPI `console`→`ui` (R8); record reserved-package status (R10); R9 is a no-op. **Commit** `fix stale comments and reconcile current-state docs`.

### Stage 1 — Canonical architecture, contracts, boundaries (Issues 1, 4)

- **1.1 — Issue 1 + A5 decision.** Reconcile architecture README + terminology with the as-built state; document context/retrieval/allocation/scheduler/runtime boundaries and the sandbox integration point; define/refresh only the contracts the upcoming issues require; document `role ≠ capability ≠ parallelism`; update indexes/MANIFEST; record the `intake → runtime` and `contract` placement decision (A5, executed in 3.4); close Issue 1. **Commit** `reconcile canonical architecture, boundaries and required contracts`.
- **1.2 — A4: contracts v1 sunset.** Mark v1 deprecated in `POLICY.md`/README; confirm no consumers; remove v1 schemas/bindings/fixtures from the active path (retain in git history); update CI/docs. **Commit** `sunset contracts v1`.
- **1.3 — Issue 4: RPC/boundary/version-skew hardening.** Audit which boundaries are actually distributed/versioned; specify contract versioning + compatibility; error schemas, timeouts, retries, idempotency, unsupported-version behavior at the RPC boundary; close Issue 4. **Commit** `harden rpc boundary versioning, errors and idempotency`.

### Stage 2 — Knowledge path (Issues 2, 3)

- **2.1 — Issue 2: deterministic retrieval.** Deterministic query behavior, stable ranking/tie-break, provenance; filtering, retrieval limits, error behavior, fixtures/tests; close Issue 2. **Commit** `add deterministic retrieval`.
- **2.2 — Issue 3 + A8(context).** Context-needs discovery + selection/relevance from retrieval; token budgeting, compression, provenance, assembly, overflow behavior; explicit retrieval/context boundary tests; move context/handoff compression from `runtime/inference` into `kernel/context`; close Issue 3. **Commit** `add context compilation, budgeting and assembly`.

### Stage 3 — Execution path (Issues 5, 6, 8)

- **3.1 — Issue 5 + A7: sandbox.** `runtime/sandbox`: isolation, filesystem/env, limits, timeout/termination, cleanup, error propagation, observability; tool permissions + failure/escape tests; runtime executes through the sandbox without roles knowing; close Issue 5. **Commit** `implement runtime sandbox execution`.
- **3.2 — Issue 6.** `toolbox/computer-use` capability + permission/capability declarations; runtime↔tool execution lifecycle, cancellation, results/artifacts, audit events; close Issue 6. **Commit** `add toolbox computer use and wire it through runtime`.
- **3.3 — Issue 8: scheduling & multi-agent execution.** DAG/dependencies, dispatch, concurrency limits, synchronization, aggregation; cancellation, retries, partial failures, worker lifecycle, serial vs parallel; close Issue 8. **Commit** `harden scheduler dispatch, concurrency and cancellation`.
- **3.4 — A5: move intake to runtime.** Move `kernel/intake` → `runtime` per the 1.1 decision; update archtest/docs. **Commit** `move result intake to runtime`.

### Stage 4 — Allocation (Issue 7)

- **4.1 — Issue 7 + A6.** Role allocation from `registry/roles` + `capabilities` (deterministic); `registry/models` population from the inference catalog; model allocation (identity selection) + escalation representation; worker count, parallelism, budgets, placement (justify parallelism by expected benefit); close Issue 7. **Commit** `add role, model and compute allocation`.
- **4.2 — A8(routing/budgets).** Move routing/budget policy from `runtime/inference` into `allocator`/`contract`; inference keeps provider/execution only. **Commit** `move routing and budgets out of inference`.

### Stage 5 — Learning & observability (Issues 9, 10)

- **5.1 — Issue 9 + A9(apprenticing/studying).** Result/evaluation → reusable lesson → experience store → future retrieval/context; selective mentorship/escalation pattern (no permanent senior/junior roles); contamination/quality control; role learning states: apprenticing (project) / studying (organization) with provenance; close Issue 9. **Commit** `add experience feedback loop and learning states`.
- **5.2 — Issue 10 + A9(identity chain).** Structured events/traces across the new seams (context, retrieval, allocation, plan, scheduler, runtime, sandbox, tool, learning); org→project→subtask→crew→worker→{role→team, model} identity-chain attribution; regression + integration + e2e scenarios (simple / difficult / decomposable / tool-using); reconciliation, failure visibility, cost/token/compute accounting; close Issue 10. **Commit** `add observability identity chain and end-to-end validation`.

### Stage 6 — Sprint closeout

- **6.1 — Summary and closeout.** Sprint summary + summary.uml; close remaining issues; record deferred work as new issues; link the sprint PR. **Commit** `add issue-impl-sep28 sprint summary and close out issues`.

## R/A resolution map

| Finding | Commit | Action |
|---|---|---|
| R4 alias | 0.1 | rename to `v2` |
| R5 comments | 0.2 | fix |
| R6 §5.6/§5.12 | 0.2 | consolidate |
| R7 phase-4 label | 0.2 | fix |
| R8 OpenAPI console | 0.2 | `ui` |
| R9 package names | — | **no-op** (set by the refactor) |
| R10 reserved packages | 0.2 + stages | document; consumers in 3.1/4.1 |
| A4 v1 sunset | 1.2 | deprecate + remove |
| A5 intake/contract | 1.1 + 3.4 | decide + execute |
| A6 model/compute | 4.1 | implement |
| A7 sandbox | 3.1 | implement |
| A8 inference extraction | 2.2 (context), 4.2 (routing/budgets) | extract; `desktop → ui/surfaces` deferred to `ph10-ui` |
| A9 apprenticing/identity | 5.1/5.2 | implement |

## Directives (apply throughout)

1. **Issue closeout.** On finishing an issue/substage: tick acceptance criteria in `docs/issues/<slug>.md`, set Status `complete`, link PR/commits, add to the issue index, record in project history (`docs/modules/<module>/{README,<submodule>}.md`) and work history (this folder); close the GitHub issue with `Closes #n`.
2. **New issues.** Discovered out-of-scope work → new `docs/issues/<slug>.md` + GitHub issue, linked from the Future register; never absorbed silently.
3. **Sprint documentation.** Maintain plan/UML, per-stage summary, final summary + UML, one ADR per decision in `docs/adr/` (indexed), and update architecture/terminology/indexes/`MANIFEST.json`/diagrams; docs link check must pass.
4. **Review/audit comments.** At each stage end post a PR comment in the **code-review format** (blocking/non-blocking, severity, validation, risk, recommendation); at each issue/phase end post a **repository-audit format** comment (P0..P3, recommendation). P0/P1 block; P2 decided.
5. **Cadence.** One commit per issue/substage; fold closely-related sub-tasks into one commit when a single simple message describes them; stop at each substage boundary for review.
6. **Branch/CI.** Keep `dev`; PR #21 (`dev` → `main`). Gates: `gofmt`/build/vet/test, `archtest`, contracts Go+Python+TS, `runtime/inference` pytest/ruff/mypy, docs link check. Retain the branch on merge.

## Validation gates (per commit unless stated)

| Gate | Command |
|---|---|
| Formatting | `gofmt -l .` |
| Go | per-module `go build/vet/test ./...` |
| Boundaries | `tools/archtest -root ..` (from `tools`) |
| Contracts | `contracts` Go tests; `contracts/python` unittest; `contracts/typescript` `tsc --noEmit` |
| Inference | `runtime/inference` pytest (unit/integration), `ruff format --check`, `ruff check`, `mypy` |
| Docs | fence-aware link scan + CI `markdown links` |

## Exit criteria

- [ ] All ten issues implemented and closed (docs + GitHub), or explicitly deferred with new issues.
- [ ] R4–R9 and A4–A9 resolved or recorded.
- [ ] Boundaries and contracts coherent; `archtest` and all module suites green.
- [ ] Architecture/terminology/directives/indexes/MANIFEST/diagrams current; docs link check green.
- [ ] Sprint summary + UML authored; sprint PR merged to `main`; branch retained.
- [ ] Review and audit comments posted for each stage/issue.

## Future register

Anything discovered during the sprint that is out of scope is raised as a new issue and listed here (and in the final summary). Known seeds: AI-driven role/strategy optimisation beyond Issue 7/9, multi-writer history, cloud-cost governance, and the Wails/React UI surface.

**A8 `desktop → ui/surfaces` (deferred to `ph10-ui`).** Relocating the inference service's desktop surface under `ui/surfaces` is UI work; it is out of sprint scope (see Out-of-scope) and belongs to the UI phase. Issue 6 does not touch `ui`.

Contracts generation v1 was removed from the active path in commit 1.2 (Issue 1 / A4); the deleted files are retained only in Git history, so the earlier "v1 deletion" seed is closed.

**Reserved packages with no consumers yet (R10).** `kernel/allocator/{model,compute}_allocator`, `registry/{models,teams,routines,policies}`, and `runtime/lifecycle` are documented placeholders — no behavior, no callers. They must not be mistaken for live code; they are delivered by Issues 7 (allocation) and 8 (lifecycle) (recorded in [architecture §4.1](../../architecture/README.md#41-module-map)). `runtime/sandbox` was delivered by Issue 5 and is no longer reserved.

## Ten-issue translation table (as-built names)

| Handoff name | As-built |
|---|---|
| `aa-kernel/sifter` | `kernel/allocator` (planner, role/model/compute allocators) |
| `aa-kernel/runtime` | top-level `runtime/` |
| `aa-agents/roles` | `registry/roles` |
| `aa-observability` | `obsv` |
| `aa-memory/retrieval` | `memory/retrieval` |
| `aa-sifter` (Python) | `runtime/inference` |
| `aa-console` | `ui` |
