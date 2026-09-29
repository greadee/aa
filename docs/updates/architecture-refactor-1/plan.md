# Architecture Refactor 1 — Update Phase Plan

> **Type:** update phase (unnumbered — refactors/edits/updates are not numbered phases).
> **Branch:** `dev`. **Draft PR:** opened from `dev` into `main`.
> **Diagrams:** [plan.uml](./plan.uml) (lightweight, change-scoped). The top-level
> [architecture README](../../architecture/README.md) is updated with **every** change.
> **Predecessor handoff:** `AA Architecture Refactor — Planning Handoff`.
> **Successor handoff:** `AA Ten-Issue Plan — Post-Refactor Planning Handoff`
> (`aa-downloads/AA_TEN_ISSUE_POST_REFACTOR_HANDOFF.md`).

## Objective

Perform a **behavior-preserving** structural refactor that establishes the target
boundaries; moves/renames/splits packages; defines the new contracts and vocabulary;
retires `trade`; reorganizes the documentation system (ADRs, module/project history,
update phases); and leaves the repository ready for the ten-issue implementation
phase — **without implementing any of it**.

## Starting State

- Starting ref: `main @ 5197ca8` (after the phase 8 module update merged). Branch `dev` created from `main`.
- Available: Go workspace modules `contracts, obsv, kernel, memory, sync, forge, toolbox, visualizer, console`; Python `sifter`; `tools/archtest`; phase/module/issue/architecture docs; CI (`ci.yml`, `docs-link-check.yml`).
- Missing (reserve, do not build): model registry; team/crew/workforce entities; routine engine; retrieval ranking; scheduler policy/queue; sandbox; apprenticing/studying behavior; allocation beyond today's selection.
- Constraints (verbatim):
  - **Critical rule:** behavior-preserving structural refactor. Functionality may be moved, renamed, split, or have interfaces cleaned up, but **missing functionality must NOT be implemented** because it appears in the target.
  - Sandboxing does not exist and **must not** be implemented here.
  - Existing tests stay green, or change only where package/import boundaries legitimately move.
  - The refactor merges and is stable **before** the ten-issue phase begins.

## Scope

### In scope
- Move/rename/split packages and modules to the target boundaries.
- Delete `kernel/registry`; create `registry/`, top-level `runtime/`, `ui/`.
- Introduce `kernel/allocator` (`planner`, `role_allocator`, `model_allocator`, `compute_allocator`) and `kernel/scheduler`.
- Contracts **v2.0**: retire `trade`; add `RoleSpec`, `ModelSpec`, `TeamSpec`, `WorkPlan`, `ExecutionPlan`, `Routine`; update Go/TS/Python bindings, schemas, fixtures, conformance.
- Rename Python `sifter` → `runtime/inference` (rename/rehome only).
- `memory/query` → `memory/retrieval`.
- Documentation architecture: global ADRs, module/project-history docs, update phases, relocation of module-update plans/summaries into `docs/`.
- Update `go.work`, `tools/archtest`, CI matrices, indexes, `MANIFEST.json`, diagrams.

### Out of scope
See [Future Register](#future-register).

## Ground Rules (allowed / not now)

**Allowed now:** rename; relocate; split files where behavior is unchanged; define
interfaces; define contracts/types; define terminology; reserve fields; document
ownership; add compatibility adapters if needed; update tests/imports for structural
changes.

**Not now:** new allocation algorithms; new model-selection behavior; new compute
allocation; new workforce logic; new learning behavior; new retrieval algorithms;
sandbox implementation; inference redesign; ten-issue feature implementation.

## Decisions (all greenlit)

| Id | Decision |
|---|---|
| GR-1 | Behavior-preserving; missing functionality reserved, never implemented. |
| GR-2 | Sandbox boundary reserved only (`runtime/sandbox`), not implemented. |
| GR-3 | Tests stay green; updates only for legitimate path/import moves. |
| GR-4 | Merged to `main` before the ten-issue phase. |
| D-1 | Python `sifter` reduced to a model provider/execution service, renamed **`inference`**, nested under top-level **`runtime/`**; rename/rehome only. |
| D-2 | `kernel/runtime` moves to **top-level `runtime/`** (lifecycle, worker, sandbox reserved, inference). |
| D-3 | **`trade` is retired** (Role = expertise/spec; Model = inference). |
| D-4 | No `agents/` or `organization/` modules; **no instance folders** in `registry/`. |
| D-5 | New top-level **`registry/`** = durable definitions only: `roles, models, teams, capabilities, routines, policies`. Policies define *what policies exist*, not applicability. Excludes worker/crew/project/workflow/assignment/execution. |
| D-6 | **`obsv` keeps its name.** |
| D-7 | Contracts **major bump to v2.0**; remove trade; add specs/plans/routine; update all bindings/fixtures/conformance. |
| D-8 | `routines → registry/routines`; `memory/query → memory/retrieval`. |
| D-9 | `runtime/` = one Go module with Go subpackages + nested `inference/` Python subproject; **kernel → runtime**. |
| D-10 | Instance ownership: `worker/crew/execution → runtime`; `assignment → kernel/scheduler`; `project → memory+contracts`; `workflow → contracts+toolbox`. |
| D-11 | `kernel/registry` deleted: role/team/capability defs → `registry/`; selection → `kernel/allocator`; worker instances → `runtime/worker`. |
| D-12 | `team → registry/teams` (def); `crew` + `workforce → runtime` (instances/capacity). |
| D-13 | Add `WorkPlan` + `ExecutionPlan` + `Routine` contracts; workflow compiler stays toolbox; execution flows `kernel/scheduler → runtime`. |
| D-14 | All greenlit renames + full docs/CI/archtest/diagram sweep. `console/` → top-level **`ui/`** with `components/ views/ state/ surfaces/`; **surfaces = `cli` and `tui` only for now**. |
| D-15 | `intake` and the other kernel packages (`context, contract, gate, telemetry, joblearn, api, intake`) **stay in kernel for now**; intended future boundaries documented; no new dependencies entrench transitional placement. |
| D-16 | Vocabulary + reserved fields only (no behavior): apprenticing/studying, observability identity chain `org→project→subtask→crew→worker→{role→team, model}`. |
| D-17 | Allocation subsystem = `kernel/allocator/{planner, role_allocator, model_allocator, compute_allocator}`; ten-issue terms `capability→model`, `parallelism→compute`. |
| D-18 | Docs: `docs/updates/<slug>/` update phases; `docs/adr/ADR-NNNN-<slug>.md` (global, chronological; historical backfilled); `docs/modules/<module>/{README.md, <submodule>.md, updates/<slug>/{plan.md,summary.md}}`; directives stay beside modules; phases = work history, modules = project history. |
| D-19 | This refactor's plan/summary live in `docs/updates/architecture-refactor-1/`; plan/summary UML are lightweight and change-scoped; the architecture README is updated with every change. |
| D-20 | ~20 slices, one concern each; slice → exactly one commit; stop between slices for review. |

## Target Architecture

```text
aa/
├── kernel/
│   ├── context/                       # transitional placement; future boundary documented
│   ├── allocator/{planner, role_allocator, model_allocator, compute_allocator}
│   ├── scheduler/                     # from orchestrator + control
│   ├── contract/ gate/ telemetry/ joblearn/ api/ intake/   # transitional, documented
├── registry/                          # roles/ models/ teams/ capabilities/ routines/ policies/
├── runtime/                           # top-level; Go module + inference/ Python subproject
│   ├── lifecycle/
│   ├── worker/
│   ├── sandbox/                       # reserved, not implemented
│   └── inference/                     # renamed from sifter; provider/execution only
├── memory/                            # retrieval/ projection/ repo/ retention/ store/
├── ui/                                # components/ views/ state/ surfaces/{cli,tui}/
├── obsv/                              # unchanged name
├── toolbox/ forge/ sync/ visualizer/ contracts/ tools/
```

Boundaries: Context decides what information is needed; memory/retrieval finds
candidates. Planner decides what work and dependencies; allocators decide
role/model/compute. Scheduler decides order/readiness/concurrency; runtime executes and
binds `role + model + context + tools + constraints`. Registry owns durable
definitions; runtime owns instances.

## Current → Target Mapping

| Current | Disposition | Target |
|---|---|---|
| `contracts/` | Rename/expand | v2.0 schemas/bindings; add specs/plans/routine; remove trade |
| `obsv/` | Keep | `obsv/` (name unchanged) |
| `kernel/context` | Keep (transitional) | `kernel/context` (future boundary documented) |
| `kernel/plan` | Move | `kernel/allocator/planner` |
| `kernel/registry` | Split/delete | defs → `registry/`; selection → `kernel/allocator`; instances → `runtime/worker` |
| `kernel/orchestrator` | Rename/merge | `kernel/scheduler` |
| `kernel/control` | Move | `kernel/scheduler` (leases/assignment state) |
| `kernel/runtime` | Move | `runtime/` (top-level) |
| `kernel/intake` | Keep for now | future `runtime` (documented, not moved) |
| `kernel/contract` | Keep (transitional) | `kernel/contract` (future boundary documented) |
| `kernel/gate` | Keep (transitional) | `kernel/gate` |
| `kernel/telemetry` | Keep (transitional) | `kernel/telemetry` |
| `kernel/joblearn` | Keep | `kernel/joblearn` (vocabulary extended later) |
| `kernel/api` | Keep | `kernel/api` |
| `memory/query` | Rename | `memory/retrieval` |
| `memory/{projection,repo,retention,store}` | Keep | unchanged |
| `sifter/` (Python) | Rename/relocate | `runtime/inference/` (provider/execution only) |
| `console/` | Rename/restructure | `ui/` (`components/views/state/surfaces/{cli,tui}`) |
| `forge/ sync/ toolbox/ visualizer/ tools/` | Keep | unchanged (archtest/CI updated) |
| — | New | `registry/`, top-level `runtime/`, `ui/` |

## Documentation Architecture

- **Work history:** numbered `docs/phases/` (major additions only; frozen archive) and
  unnumbered `docs/updates/<slug>/` (refactors/edits/updates).
- **Project history:** `docs/modules/<module>/README.md` + per-submodule pages;
  module-update plans/summaries move to
  `docs/modules/<module>/updates/<slug>/{plan.md,summary.md}` and are linked from the
  owning update phase.
- **ADRs:** `docs/adr/ADR-NNNN-<slug>.md`, global chronological sequence + index;
  historical `ADR-Px-NNN` backfilled with an `old → new` mapping; statuses and
  supersession preserved.
- **Directives:** `aa-<module>.md` stay beside modules; `aa-obsrv.md → aa-obsv.md`,
  `console/aa-tui.md → ui/aa-ui.md`.
- Update `documentation-system.md`, `module-updates.md`, `phase-documentation.md`,
  `adrs-and-pitfalls.md`, templates, `docs/index/*`, `MANIFEST.json`.

## Slices

Each slice maps to exactly one commit and is independently validatable. **Definition of
Done for every slice:** tests green; `gofmt`/`go vet`/`archtest` clean; affected
architecture README + terminology updated; ADRs added/updated; relative links valid.

### Slice 1 — Update phase plan and diagrams
- **Goal** — Author this plan and its change-scoped UML.
- **Expected Output** — `docs/updates/architecture-refactor-1/{plan.md,plan.uml}`.
- **Model Class** — Strong. **Commit** — `add architecture refactor update phase plan`.
- **Validation** — docs link check. **Dependencies** — none.

### Slice 2 — ADR store and historical backfill
- **Goal** — Establish `docs/adr/` (global IDs, index, statuses, supersession) and
  backfill historical `ADR-Px-NNN` in phase order; add the `old → new` index.
- **Expected Output** — `docs/adr/README.md`, `docs/adr/ADR-*.md`; updated `adrs-and-pitfalls.md`.
- **Model Class** — General. **Commit** — `establish adr store and backfill historical decisions`.
- **Dependencies** — none. **Validation** — all old ADR ids mapped; link check.

### Slice 3 — Module documentation structure and system rules
- **Goal** — Add `docs/modules/<module>/{README.md,<submodule>.md}`; update the docs
  system docs and templates for the update-phase/module model.
- **Expected Output** — `docs/modules/**`; updated `documentation-system.md`,
  `module-updates.md`, `phase-documentation.md`, templates, indexes, `MANIFEST.json`.
- **Model Class** — General. **Commit** — `add module documentation structure and system rules`.
- **Dependencies** — slices 1–2.

### Slice 4 — Move module update plans/summaries into docs
- **Goal** — Relocate the six existing `module-upd-{plan,summary}.md` pairs to
  `docs/modules/<module>/updates/<slug>/` and link them from update phases.
- **Model Class** — Lightweight. **Commit** — `move module update plans and summaries into docs`.
- **Dependencies** — slice 3. **Validation** — no broken links; module folders clean.

### Slice 5 — registry module skeleton
- **Goal** — Add `registry/{roles,models,teams,capabilities,routines,policies}` with
  interfaces/types; move durable definitions out of `kernel/registry/roles.go` and
  contract capability vocabularies.
- **Model Class** — Strong. **Commit** — `add registry module skeleton`.
- **Dependencies** — slice 3. **Validation** — build/vet/test; no behavior change.

### Slice 6 — runtime module (top-level)
- **Goal** — Move `kernel/runtime` → `runtime/`; add `lifecycle/`, `worker/`; reserve
  `sandbox/`; keep `inference/` placeholder.
- **Model Class** — Strong. **Commit** — `add runtime module with lifecycle worker and reserved sandbox`.
- **Dependencies** — slices 3, 5.

### Slice 7 — ui module
- **Goal** — `console/` → `ui/` with `components/views/state/surfaces/{cli,tui}`;
  `aa-tui.md → ui/aa-ui.md`.
- **Model Class** — Lightweight. **Commit** — `add ui module with components views state and cli tui surfaces`.
- **Dependencies** — slice 3.

### Slice 8 — workspace/archtest/CI
- **Goal** — Update `go.work`, `tools/archtest/rules.go` (new modules + kernel→runtime),
  and CI matrices.
- **Model Class** — General. **Commit** — `update go.work archtest and ci for registry runtime ui`.
- **Dependencies** — slices 5–7.

### Slice 9 — kernel/allocator/planner
- **Goal** — Move `kernel/plan` → `kernel/allocator/planner`; update imports.
- **Model Class** — Strong. **Commit** — `move kernel plan into allocator planner`.
- **Dependencies** — slice 8.

### Slice 10 — allocators + delete kernel/registry
- **Goal** — Add `role_allocator`, `model_allocator`, `compute_allocator`; move
  selection from `kernel/registry`; move worker instances to `runtime/worker`; delete
  `kernel/registry`.
- **Model Class** — Strong. **Commit** — `add role model compute allocators and retire kernel registry`.
- **Dependencies** — slice 9.

### Slice 11 — kernel/scheduler
- **Goal** — Merge `kernel/orchestrator` + `kernel/control` into `kernel/scheduler`;
  rewire `api`.
- **Model Class** — Strong. **Commit** — `rename kernel orchestrator and control to scheduler`.
- **Dependencies** — slice 10.

### Slice 12 — document kernel transitional boundaries
- **Goal** — Document intended future homes for
  `context/contract/gate/telemetry/joblearn/api/intake`; avoid entrenching
  transitional dependencies.
- **Model Class** — General. **Commit** — `document kernel transitional boundaries and intended ownership`.
- **Dependencies** — slice 11.

### Slice 13 — contracts v2
- **Goal** — Bump to v2.0; add
  `RoleSpec/ModelSpec/TeamSpec/WorkPlan/ExecutionPlan/Routine`; update Go/TS/Python
  bindings, schemas, fixtures, conformance.
- **Model Class** — Strong. **Commit** — `bump contracts to v2 with spec plan and routine types`.
- **Dependencies** — slice 5.

### Slice 14 — retire trade
- **Goal** — Remove `trade` from contracts, registry, kernel, joblearn, bindings, fixtures.
- **Model Class** — General. **Commit** — `retire trade from contracts registry and joblearn`.
- **Dependencies** — slice 13.

### Slice 15 — wire the new contracts
- **Goal** — Connect `registry`/`kernel`/`runtime` to the new spec/plan/routine types;
  add compatibility adapters where needed.
- **Model Class** — Strong. **Commit** — `wire role model team routine and plan contracts`.
- **Dependencies** — slices 11–14.

### Slice 16 — rename sifter to inference
- **Goal** — `sifter/` → `runtime/inference/`; rename Python package/CLI/RPC service/env/
  socket; keep behavior; provider/execution only.
- **Model Class** — General. **Commit** — `rename sifter to inference under runtime`.
- **Dependencies** — slice 8.

### Slice 17 — memory/query → memory/retrieval
- **Goal** — Rename the package and update callers (`browse`, context contract).
- **Model Class** — General. **Commit** — `rename memory query to retrieval`.
- **Dependencies** — slice 8.

### Slice 18 — architecture/terminology/directives/indexes sweep
- **Goal** — Update architecture README, terminology, directives, `docs/index/*`,
  `MANIFEST.json`, cross-links.
- **Model Class** — General. **Commit** — `update architecture terminology directives and indexes`.
- **Dependencies** — slices 1–17.

### Slice 19 — diagrams
- **Goal** — Update update-phase and top-level architecture diagrams (lightweight,
  change-scoped).
- **Model Class** — General. **Commit** — `update update-phase and architecture diagrams`.
- **Dependencies** — slice 18.

### Slice 20 — future register and ten-issue readiness
- **Goal** — Record reserved components and the stale-name translation table; confirm
  readiness.
- **Model Class** — General. **Commit** — `add future register and ten-issue readiness mapping`.
- **Dependencies** — slices 1–19.

## Required End State

- [ ] `docs/updates/architecture-refactor-1/` holds this plan (and later its summary).
- [ ] `docs/adr/` is canonical; historical decisions backfilled and mapped.
- [ ] `docs/modules/<module>/` carries project history; module-update docs live under it.
- [ ] `registry/`, top-level `runtime/`, `ui/` exist at the target boundaries.
- [ ] `kernel/allocator` and `kernel/scheduler` exist; `kernel/registry` is gone.
- [ ] Contracts are v2.0; `trade` retired; specs/plans/routine exist and are wired.
- [ ] Python `sifter` is `runtime/inference`, provider/execution only.
- [ ] `memory/retrieval` replaces `memory/query`.
- [ ] `go.work`, archtest, CI, indexes, MANIFEST, diagrams reflect the topology.
- [ ] Tests green; `gofmt`, `go vet`, `tools/archtest`, docs link check pass.

## Exit Criteria

- [ ] Target boundaries and vocabulary documented and enforced by archtest.
- [ ] No missing functionality implemented; Future register explicit.
- [ ] Every slice = exactly one commit; each stopped for review.
- [ ] Architecture README and terminology reflect each change.
- [ ] Repo ready for the ten-issue implementation phase.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | unchanged package behavior after moves/renames |
| contract | v2 suite: specs/plans/routine validate; trade absent; bindings round-trip |
| boundary | `tools/archtest`: layering with `registry`/`runtime`/`ui`; kernel→runtime; obsv single-vocabulary guard |
| docs | all relative links resolve; ADR index maps every historical id; link check passes |
| regression | `go build/vet/test` across every module; `gofmt`; CI matrices |

## Future Register

Reserved — **not built here**: model/compute allocation behavior beyond current
selection; model-registry population and `registry/models` wiring to the inference
catalog; team/crew/workforce logic; routine engine and workflow learning; retrieval
ranking/similarity; sandbox implementation; inference capability extraction
(routing→allocator, budgets→allocator/contract, verification→runtime,
context→kernel/context, catalog→registry/models, desktop→ui/surfaces);
apprenticing/studying behavior; observability identity chain wiring; all ten-issue
features.

## Ten-Issue Translation Table

| Ten-issue name | Post-refactor name |
|---|---|
| `aa-kernel/sifter` | `aa-kernel/allocator` (`planner`, role/model/compute allocators) |
| Capability (allocation dimension) | Model allocation |
| Parallelism (allocation dimension) | Compute allocation |
| `aa-kernel/runtime` | top-level `runtime/` |
| `aa-agents/roles` | `registry/roles` |
| `aa-observability` | `obsv` |
| `aa-memory/retrieval` | `memory/retrieval` (unchanged intent) |

The ten-issue handoff document is left as-is; its **Issue 1** reconciles canonical docs.

## Risks and Tradeoffs

| Risk | Mitigation |
|---|---|
| Large mechanical churn | One concern per slice; tests green each step; stop for review |
| Contracts v2 is breaking | Compatibility adapters allowed; bindings/fixtures/conformance updated together |
| Doc reorganization breaks links | `docs-link-check` in CI; mapping index for historical links |
| Transitional kernel placement entrenching | Slice 12 documents intended boundaries; no new deps across transitional packages |
| Scope creep into ten-issue features | Ground rules + Future register enforced at every slice |
