# Issue Implementation Sprint (Sep 28) — Plan

> **Type:** update phase (`issue-impl-sep28`). **Branch:** `dev`. **Sprint PR:** #21 (Draft, `dev` → `main`).
> **Predecessor:** [architecture-refactor-1](../architecture-refactor-1/summary.md) (merged `bf8e09d`).
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
- The ten issues (below), delivered as stages/substages, one slice = one commit.
- Resolution of R4–R9 (review) and A4–A9 (audit deferrals).
- Sprint plan/summary/UML, one ADR per decision, and full doc/index/manifest/diagram upkeep.
- GitHub issue creation (objective 0) and closeout (`Closes #n`).

### Out of scope
- The Wails/React UI surface (tracked with `ui`); anything not in the ten issues or R/A lists — raise as a new issue instead.

## Decisions

| # | Decision | Reasoning |
|---|---|---|
| SD-1 | One sprint branch `dev`, one sprint PR #21 (`dev` → `main`); the refactor is already merged, so #21 is sprint-only | Keeps the refactor and the issue work reviewable separately; #20 is the findings source |
| SD-2 | Group issues into **stages by module area**; substages are the issues; slices are commits | Minimises jumping across the codebase within a stage |
| SD-3 | Fold R4–R9 and A4–A9 into the stages where they belong (see map); **R9 is a no-op** (allocator names already set by the refactor) | Resolve findings where the relevant code is already open |
| SD-4 | Create the ten GitHub issues + milestone in objective 0, after the plan | Execution is visible and closeable per issue |
| SD-5 | Every issue closes with history: tick `docs/issues/<slug>.md`, project history (`docs/modules/**`), work history (this folder), `Closes #n` | Durable, auditable record |
| SD-6 | Post a **code-review-format** PR comment at each stage end and a **repository-audit-format** comment per issue/phase | The refactor's comments were high-value; keep the practice |
| SD-7 | One slice = one commit; stop at each substage boundary for review | Bounded, reviewable progress |

## Issue → stage map

| Issue | Stage | Notes |
|---|---|---|
| 1 Canonical architecture & contracts | Stage 1 | + A4 (v1 sunset), R6/R7/R8 docs |
| 2 Deterministic retrieval | Stage 2 | `memory/retrieval` |
| 3 Context compilation/budgeting/assembly | Stage 2 | `kernel/context`; + A8(context) |
| 4 RPC/boundary/version-skew hardening | Stage 1 | contracts/rpc; + A5 decision |
| 5 Sandbox execution | Stage 3 | `runtime/sandbox` (A7) |
| 6 Computer-use/tool execution | Stage 3 | `toolbox` + runtime; + A8(desktop→ui) |
| 7 Role/capability/model/compute allocation | Stage 4 | `kernel/allocator`, `registry`; + A6, A8(routing/budgets) |
| 8 Scheduling & multi-agent execution | Stage 3 | `kernel/scheduler` + runtime; + A5 execution |
| 9 Learning/experience feedback | Stage 5 | `kernel/joblearn` + memory; + A9(apprenticing/studying) |
| 10 Observability/hardening/e2e | Stage 5 | `obsv` + all; + A9(identity chain) |

## Stages and slices

Each slice maps to exactly one commit. **DoD per slice:** tests green; `gofmt`/`go vet`/`archtest` clean; affected architecture/terminology/docs + ADRs updated; docs link check valid.

### Stage 0 — Cleanup & stabilization (no behavior change)

**Substage 0.1 — contracts consumer aliasing (R4)**
- Slice 0.1.1 — Goal: replace the confusing `v1 "…/contracts/go/v2"` alias with `v2` across consumers. **Commit** `alias contracts v2 imports consistently`. **Validation** build/vet/test all modules.

**Substage 0.2 — comments and current-state docs (R5–R8, R10)**
- Slice 0.2.1 — Goal: fix `kernel/scheduler` stale `orchestrator` comments (R5). **Commit** `fix scheduler comments after the rename`.
- Slice 0.2.2 — Goal: consolidate architecture §5.6/§5.12 (R6), fix phase-4 label (R7), update OpenAPI `console`→`ui` (R8), record reserved-package status (R10). **Commit** `reconcile current-state docs and openapi names`.

### Stage 1 — Canonical architecture, contracts, boundaries (Issues 1, 4)

**Substage 1.1 — Issue 1: canonical architecture & contracts**
- Slice 1.1.1 — reconcile architecture README + terminology with the as-built state; document context/retrieval/allocation/scheduler/runtime boundaries and the sandbox integration point. **Commit** `reconcile canonical architecture and terminology`.
- Slice 1.1.2 — define/refresh only the contracts the upcoming issues require; document `role ≠ capability ≠ parallelism`; update indexes/MANIFEST. **Commit** `define canonical boundaries and required contracts`.
- Slice 1.1.3 — close Issue 1 (docs + GitHub). **Commit** `close issue 1 canonical architecture and contracts`.

**Substage 1.2 — A4: contracts v1 sunset**
- Slice 1.2.1 — mark v1 deprecated in `POLICY.md`/README; confirm no consumers. **Commit** `deprecate contracts v1`.
- Slice 1.2.2 — remove v1 schemas/bindings/fixtures from the active path (retain in git history); update CI/docs. **Commit** `remove contracts v1 from the active path`.

**Substage 1.3 — Issue 4: RPC/boundary/version-skew hardening**
- Slice 1.3.1 — audit which boundaries are actually distributed/versioned; specify contract versioning + compatibility. **Commit** `specify rpc versioning and compatibility`.
- Slice 1.3.2 — error schemas, timeouts, retries, idempotency, unsupported-version behavior at the RPC boundary. **Commit** `harden rpc errors deadlines and idempotency`.
- Slice 1.3.3 — close Issue 4. **Commit** `close issue 4 rpc boundary hardening`.

**Substage 1.4 — A5: boundary decision**
- Slice 1.4.1 — record the decision for `intake → runtime` and `contract` placement (execute in 3.4). **Commit** `decide kernel transitional ownership`.

### Stage 2 — Knowledge path (Issues 2, 3)

**Substage 2.1 — Issue 2: deterministic retrieval**
- Slice 2.1.1 — deterministic query behavior + stable ranking/tie-break + provenance. **Commit** `add deterministic retrieval ranking and provenance`.
- Slice 2.1.2 — filtering, retrieval limits, error behavior, fixtures/tests. **Commit** `add retrieval filters limits and tests`.
- Slice 2.1.3 — close Issue 2. **Commit** `close issue 2 deterministic retrieval`.

**Substage 2.2 — Issue 3: context compilation, budgeting, assembly**
- Slice 2.2.1 — context-needs discovery + selection/relevance from retrieval. **Commit** `add context needs discovery and selection`.
- Slice 2.2.2 — token budgeting, compression, provenance, assembly, overflow behavior. **Commit** `add context budgeting compression and assembly`.
- Slice 2.2.3 — explicit retrieval/context boundary tests. **Commit** `test the retrieval context boundary`.
- Slice 2.2.4 — close Issue 3. **Commit** `close issue 3 context compilation`.

**Substage 2.3 — A8(context): extract from inference**
- Slice 2.3.1 — move context/handoff compression from `runtime/inference` into `kernel/context`. **Commit** `move inference context handling into kernel context`.

### Stage 3 — Execution path (Issues 5, 6, 8)

**Substage 3.1 — Issue 5 / A7: sandbox**
- Slice 3.1.1 — `runtime/sandbox`: isolation, filesystem/env, limits, timeout/termination, cleanup, error propagation, observability. **Commit** `implement runtime sandbox execution`.
- Slice 3.1.2 — tool permissions + failure/escape tests; runtime executes through the sandbox without roles knowing. **Commit** `add sandbox permissions and failure tests`.
- Slice 3.1.3 — close Issue 5. **Commit** `close issue 5 sandbox execution`.

**Substage 3.2 — Issue 6: computer-use / tool execution**
- Slice 3.2.1 — `toolbox/computer-use` capability + permission/capability declarations. **Commit** `add toolbox computer use capability`.
- Slice 3.2.2 — runtime↔tool execution lifecycle, cancellation, results/artifacts, audit events. **Commit** `wire computer use through the runtime boundary`.
- Slice 3.2.3 — A8(desktop→ui): relocate any inference desktop surface into `ui/surfaces` (reserve). **Commit** `reserve ui surface for inference desktop`.
- Slice 3.2.4 — close Issue 6. **Commit** `close issue 6 computer use integration`.

**Substage 3.3 — Issue 8: scheduling & multi-agent execution**
- Slice 3.3.1 — DAG/dependencies, dispatch, concurrency limits, synchronization, aggregation. **Commit** `harden scheduler dispatch and concurrency`.
- Slice 3.3.2 — cancellation, retries, partial failures, worker lifecycle, serial vs parallel. **Commit** `add scheduler cancellation and retries`.
- Slice 3.3.3 — close Issue 8. **Commit** `close issue 8 scheduling and multi agent execution`.

**Substage 3.4 — A5: move intake to runtime**
- Slice 3.4.1 — move `kernel/intake` → `runtime` per the 1.4 decision; update archtest/docs. **Commit** `move result intake to runtime`.

### Stage 4 — Allocation (Issue 7)

**Substage 4.1 — role/capability allocation**
- Slice 4.1.1 — role allocation from `registry/roles` + `capabilities` (deterministic). **Commit** `add role and capability allocation`.
- Slice 4.1.2 — close the role/capability sub-scope of Issue 7. **Commit** `close issue 7 role allocation`.

**Substage 4.2 — model allocation + registry/models (A6)**
- Slice 4.2.1 — `registry/models` population from the inference catalog. **Commit** `populate the model registry`.
- Slice 4.2.2 — model allocation (identity selection) + escalation representation. **Commit** `add model allocation and escalation`.

**Substage 4.3 — compute allocation (A6)**
- Slice 4.3.1 — worker count, parallelism, budgets, placement; justify parallelism by expected benefit. **Commit** `add compute allocation and parallelism`.
- Slice 4.3.2 — close Issue 7. **Commit** `close issue 7 allocation`.

**Substage 4.4 — A8(routing/budgets): extract from inference**
- Slice 4.4.1 — move routing/budget policy from `runtime/inference` into `allocator`/`contract`; inference keeps provider/execution only. **Commit** `move routing and budgets out of inference`.

### Stage 5 — Learning & observability (Issues 9, 10)

**Substage 5.1 — Issue 9: learning/experience feedback**
- Slice 5.1.1 — result/evaluation → reusable lesson → experience store → future retrieval/context. **Commit** `add experience feedback loop`.
- Slice 5.1.2 — selective mentorship/escalation pattern (no permanent senior/junior roles); contamination/quality control. **Commit** `add selective mentorship and quality gates`.
- Slice 5.1.3 — close Issue 9. **Commit** `close issue 9 learning and experience feedback`.

**Substage 5.2 — A9: apprenticing/studying**
- Slice 5.2.1 — role learning states: apprenticing (project) / studying (organization); provenance. **Commit** `add apprenticing and studying learning states`.

**Substage 5.3 — Issue 10 + A9: observability and identity chain**
- Slice 5.3.1 — structured events/traces across the new seams (context, retrieval, allocation, plan, scheduler, runtime, sandbox, tool, learning). **Commit** `add execution seam observability`.
- Slice 5.3.2 — org→project→subtask→crew→worker→{role→team, model} identity chain attribution. **Commit** `add observability identity chain`.

**Substage 5.4 — Issue 10: end-to-end validation & hardening**
- Slice 5.4.1 — regression + integration + e2e scenarios (simple / difficult / decomposable / tool-using). **Commit** `add end to end validation scenarios`.
- Slice 5.4.2 — reconciliation, failure visibility, cost/token/compute accounting. **Commit** `add failure visibility and cost accounting`.
- Slice 5.4.3 — close Issue 10. **Commit** `close issue 10 observability and hardening`.

### Stage 6 — Sprint closeout

- Slice 6.1 — Goal: sprint summary + summary.uml; close remaining issues; record deferred work as new issues. **Commit** `add issue-impl-sep28 sprint summary and close out issues`.
- Slice 6.2 — Goal: link the sprint PR. **Commit** `link issue-impl-sep28 sprint pull request`.

## R/A resolution map

| Finding | Slice | Action |
|---|---|---|
| R4 alias | 0.1.1 | rename to `v2` |
| R5 comments | 0.2.1 | fix |
| R6 §5.6/§5.12 | 0.2.2 | consolidate |
| R7 phase-4 label | 0.2.2 | fix |
| R8 OpenAPI console | 0.2.2 | `ui` |
| R9 package names | — | **no-op** (set by the refactor) |
| R10 reserved packages | 0.2.2 + stages | document; consumers in 3.1/4.2/4.3 |
| A4 v1 sunset | 1.2 | deprecate + remove |
| A5 intake/contract | 1.4 + 3.4 | decide + execute |
| A6 model/compute | 4.2/4.3 | implement |
| A7 sandbox | 3.1 | implement |
| A8 inference extraction | 2.3 (context), 3.2 (desktop), 4.4 (routing/budgets) | extract |
| A9 apprenticing/identity | 5.2/5.3 | implement |

## Directives (apply throughout)

1. **Issue closeout.** On finishing an issue/substage: tick acceptance criteria in `docs/issues/<slug>.md`, set Status `complete`, link PR/commits, add to the issue index, record in project history (`docs/modules/<module>/{README,<submodule>}.md`) and work history (this folder); close the GitHub issue with `Closes #n`.
2. **New issues.** Discovered out-of-scope work → new `docs/issues/<slug>.md` + GitHub issue, linked from the Future register; never absorbed silently.
3. **Sprint documentation.** Maintain plan/UML, per-stage summary, final summary + UML, one ADR per decision in `docs/adr/` (indexed), and update architecture/terminology/indexes/`MANIFEST.json`/diagrams; docs link check must pass.
4. **Review/audit comments.** At each stage end post a PR comment in the **code-review format** (blocking/non-blocking, severity, validation, risk, recommendation); at each issue/phase end post a **repository-audit format** comment (P0..P3, recommendation). P0/P1 block; P2 decided.
5. **Cadence.** One slice = one commit; stop at each substage boundary for review.
6. **Branch/CI.** Keep `dev`; PR #21 (`dev` → `main`). Gates: `gofmt`/build/vet/test, `archtest`, contracts Go+Python+TS, `runtime/inference` pytest/ruff/mypy, docs link check. Retain the branch on merge.

## Validation gates (per slice unless stated)

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

Anything discovered during the sprint that is out of scope is raised as a new issue and listed here (and in the final summary). Known seeds: AI-driven role/strategy optimisation beyond Issue 7/9, multi-writer history, cloud-cost governance, the Wails/React UI surface, and v1 deletion from the repository (not just the active path).

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
