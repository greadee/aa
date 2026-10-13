# v1 Scope & Composition — Plan

> **Type:** update phase (`v1-readiness`). **Branch:** `dev`. **Purpose:** settle the v1 definition of
> done and the composition/structure required **before** phase 10 (`ph10-console`/`ph10-ui`) and
> phase 11 (`ph11-release`).
> **Diagram:** [plan.uml](./plan.uml).

## Objective

Decide, up front, what **v1** is and what has to be **composed and running** for it, so phase 10a/10b
(the console/desktop surfaces) and phase 11 (release hardening) are built against an agreed scope —
not assumed to be "everything". This plan is documentation only; no behavior changes.

## Starting state

- Sprints complete and merged to `main`: `architecture-refactor-1`, `issue-impl-sep28` (PR #21),
  `revu-findings`, `iss-revu` (PR #41). All ten ISS-IMP issues and the four ISS-REVU exceptions are
  closed.
- `app/` exists as the composition root but wires only a **slice**: contracts, `forge/memsync`,
  `kernel/{allocator/routing,context,joblearn,observability}`, `memory/{store,repo,projection,retrieval,retention}`,
  `registry/models`, `runtime/sandbox`, `toolbox/{policy,computeruse}`. `aa selfcheck` builds and
  exercises the graph; there is no runnable product loop.
- There is **no control-plane server**: `kernel/api` is an in-process service, disabled by default;
  `ph10-console` is client-side only and stands in a fake client + `httptest`.
- `revu` whole-repo audit shows ~44 "only test callers" exports — a proxy for built-but-unwired libraries.

## v1 definition of done (proposed)

**Surfaces in v1:**
- **10a — console**: CLI + TUI + local web, over the OpenAPI control plane (Go stdlib only per the console plan).
- **10b — desktop (Wails/React)**: **optional / deferred**. Phase 10's own exit criterion is only that
  the desktop is *"addable"*. Confirm: ship v1 without the desktop surface?

**A product is "v1" when:** a single entry point (the console) drives one composed daemon over the
control plane end-to-end — submit work, run it through scheduler → runtime → sandbox, learn from
results, and inspect memory/observability — with deterministic fallbacks, no model required for the
core path, and a documented release artifact.

## Composition coverage (proposed: in v1 vs deferred)

| Module / area | Composed today | Proposed |
|---|---|---|
| contracts, registry/models | yes | **in v1** |
| memory (store/repo/projection/retrieval/retention) | yes | **in v1** |
| kernel: context, allocator/routing, observability | yes | **in v1** |
| kernel: scheduler, gate, planner | no | **in v1** (the work loop) |
| runtime: sandbox, worker, intake | sandbox yes; worker/intake no | **in v1** |
| toolbox: host, policy, registry, computeruse | policy/computeruse yes; host no | **in v1** |
| control-plane server (`kernel/api` → HTTP) | no | **in v1** (see below) |
| runtime/inference (Python) | not composed | **in v1** for the model path (optional at runtime) |
| kernel/joblearn (experience/features/promote) | yes | **in v1** |
| forge: memsync yes; lifecycle, template | partial | **deferred** unless the workflow needs it |
| sync (distribute/parallel/queue/identity/... ) | no | **deferred** (single-node v1 per D6) |
| obsv (module), visualizer, ui | no | **deferred** (console covers inspection; visualizer post-v1) |
| registry: teams, routines, policies; runtime/lifecycle | reserved | **deferred** (decide implement-or-drop) |

## Control-plane server (proposed)

The console talks to the OpenAPI control plane; that server **must exist** for v1.
- **Home:** a daemon built from `app/` (the composition root owns wiring), exposing the
  `contracts/openapi/control-plane-v1.yaml` paths over loopback HTTP/JSON, composing
  kernel (scheduler/allocator/context/observability) + memory + toolbox (host/policy) + runtime
  (worker/sandbox). `kernel/api` provides the in-process service to expose.
- **Auth posture:** loopback bearer token carrying only; real authn/authz (pairing, device keys, mTLS)
  stays in phase 11. Document that "loopback alone is not authentication".
- **Determinism:** the core path runs with the `fake`/no-model provider; models are opt-in.

## Deferred register (carried from prior sprints)

- **A8 routing/budget residual:** pass the allocator's routing decision into `runtime/inference` and
  retire the duplicate Python routing policy.
- **A8 desktop → `ui/surfaces`:** owned by `ph10-ui`.
- **Reserved packages:** `registry/{teams,routines,policies}`, `runtime/lifecycle` — implement or drop.
- **Cross-module relocation of execution-local code:** ownership stays control-plane; Python surface retained.
- **Cadence:** whether one-commit-per-slice becomes repo-wide (AA.md rule 5) remains open.

## Release-readiness (phase 11, for reference)

Threat model & security review; performance/load; chaos/recovery; packaging/installers; release gates
& versioning (tags, CHANGELOG); docs completion; repository audits (use `revu` whole-repo); migration
guides. v1 has **no tags, no release workflow, no install story** today.

## Validation gates

| Gate | Command |
|---|---|
| Docs | fence-aware link scan; `docs-link-check` workflow |
| Review | `revu --profile pr` on the change (static-only until a model is configured) |
| Boundaries | `tools/archtest -root ..` (unchanged; docs-only change) |

## Exit criteria

- [ ] v1 surface scope confirmed (console in; desktop in/out).
- [ ] Composition coverage table agreed (in-v1 vs deferred).
- [ ] Control-plane server home and minimal composition agreed.
- [ ] Deferred register captured as issues.
- [ ] This plan merged on `dev`.

## Open decisions (for review)

1. **Desktop in v1?** Ship 10a (console) + web only, desktop post-v1? *(proposed: yes)*
2. **Control-plane server home:** `app/` daemon vs a new module? *(proposed: `app/`)*
3. **v1 composition floor:** the "in v1" rows above — confirm or trim.
4. **`revu` model:** configure `provider.model` + a key so audits run the semantic pass, or keep static-only for v1?
