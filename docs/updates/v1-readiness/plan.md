# v1 Scope & Composition — Plan

> **Type:** update phase (`v1-readiness`). **Branch:** `dev`. **Purpose:** settle the v1 definition of
> done and the composition/structure required **before** phase 10 (`ph10-console`) and phase 11
> (`ph11-release`).
> **Diagram:** [plan.uml](./plan.uml).

## Objective

Decide, up front, what **v1** is and what has to be **composed and running** for it, so phase 10 (the
console) and phase 11 (release hardening) are built against an agreed scope — not assumed to be
"everything". This plan is documentation only; no behavior changes.

## Starting state

- Sprints complete and merged to `main`: `architecture-refactor-1`, `issue-impl-sep28` (PR #21),
  `revu-findings`, `iss-revu` (PR #41). All ten ISS-IMP issues and the four ISS-REVU exceptions are closed.
- `app/` exists as the composition root but wires only a **slice**: contracts, `forge/memsync`,
  `kernel/{allocator/routing,context,joblearn,observability}`, `memory/{store,repo,projection,retrieval,retention}`,
  `registry/models`, `runtime/sandbox`, `toolbox/{policy,computeruse}`. `aa selfcheck` builds and
  exercises the graph; there is no runnable product loop.
- There is **no control-plane server**: `kernel/api` is an in-process service, disabled by default;
  `ph10-console` is client-side only and stands in a fake client + `httptest`.
- `revu` whole-repo audit shows ~44 "only test callers" exports — a proxy for built-but-unwired libraries.

## Decisions (resolved)

| # | Decision | Rationale |
|---|---|---|
| D1 | **Desktop surface is out of v1 and out of phase 10.** Phase 10 is the **console only** (CLI + TUI + local web). The Wails/React desktop surface is removed from the 10a/10b PR and deferred to **v2**. | Phase 10's exit criterion is only that the desktop is "addable"; it is not required for a releasable v1. |
| D2 | **The control-plane server is an `app/` daemon.** `app/` (the composition root) hosts the HTTP/JSON server exposing the OpenAPI control plane, composing kernel + memory + toolbox + runtime. | One composition root owns all wiring; `app/` already wires the seams. |
| D3 | **v1 composition floor** = the wired slice **plus** `kernel` scheduler/gate/planner, `runtime` worker/intake, `toolbox/host`, `kernel/joblearn`, the `runtime/inference` path, the control-plane server, and **`forge/lifecycle` + `forge/template`**. | The work loop + tools host + learning + forge lifecycle are needed for an end-to-end v1. |
| D4 | **`sync`, `obsv`, `visualizer`, `registry/{teams,routines,policies}` and `runtime/lifecycle` are deferred to v2.** | Single-node v1 (D6); the console covers inspection; the reserveds have no behavior. |
| D5 | **The v1 review gate (`revu`) runs static-only.** | Deterministic, no provider key required; the semantic pass stays available but off. |

> **Superseded in part — 2026-10-30 (Sprint 3 docs foundation).** D4 was the
> decision at the time. Phase 11 subsequently delivered `sync`, `obsv`,
> `visualizer`, `registry/{teams,routines,policies}`, and `runtime/lifecycle`,
> and the `app` daemon composes them. The desktop surface, the Wails/React/Three
> visualizer surface, and multi-machine execution authority remain deferred.
> The original decision and table are preserved as history; see
> [current-state.md](../../current-state.md) for today's boundary.

## v1 definition of done

**Surfaces in v1:**
- **Phase 10 — console**: CLI + TUI + local web over the OpenAPI control plane (Go stdlib only).
- The desktop surface is **v2**.

**A product is "v1" when:** a single entry point (the console) drives one composed daemon over the
control plane end-to-end — submit work, run it through scheduler → runtime → sandbox, learn from
results, and inspect memory/observability — with deterministic fallbacks, no model required for the
core path, and a documented release artifact.

## Composition coverage

| Module / area | Composed today | v1 |
|---|---|---|
| contracts, registry/models | yes | **in v1** |
| memory (store/repo/projection/retrieval/retention) | yes | **in v1** |
| kernel: context, allocator/routing, observability | yes | **in v1** |
| kernel: scheduler, gate, planner | no | **in v1** (D3) |
| runtime: sandbox, worker, intake | sandbox yes; worker/intake no | **in v1** (D3) |
| toolbox: policy, computeruse, host | policy/computeruse yes; host no | **in v1** (D3) |
| kernel/joblearn (experience/features/promote) | yes | **in v1** |
| runtime/inference (Python) | not composed | **in v1** (model path; optional at runtime) |
| forge: memsync, lifecycle, template | memsync yes; lifecycle/template no | **in v1** (D3) |
| control-plane server (`kernel/api` → HTTP) | no | **in v1** (D2) |
| sync | no | **v2** (D4) |
| obsv (module), visualizer, ui | no | **v2** (D4) |
| registry: teams, routines, policies; runtime/lifecycle | reserved | **v2** (D4) |

> **Composition update — 2026-10-30:** the rows above record the Sprint 2
> starting state. The core modules in this table are now composed in `app/`;
> the deferred desktop/3D surfaces remain out. Forge lifecycle/template and the
> Python inference sidecar exist as libraries/optional services but are not all
> wired into the default daemon. See [current-state.md](../../current-state.md).

## Control-plane server

The console talks to the OpenAPI control plane; that server must exist for v1.
- **Home:** built from **`app/`** (D2), exposing `contracts/openapi/control-plane-v1.yaml` over loopback
  HTTP/JSON; composes kernel (scheduler/allocator/context/observability) + memory + toolbox (host/policy)
  + runtime (worker/sandbox). `kernel/api` provides the in-process service to expose.
- **Auth posture:** loopback bearer token carrying only; real authn/authz (pairing, device keys, mTLS)
  stays in phase 11. Document that "loopback alone is not authentication".
- **Determinism:** the core path runs with the `fake`/no-model provider; models are opt-in.

## Deferred to v2

- **Desktop surface** — Wails/React, relocated under `ui/surfaces` (was `ph10-ui`).
- **`sync`**, **`obsv` (module)**, **`visualizer`**.
- **`registry/{teams,routines,policies}`**, **`runtime/lifecycle`** (reserved, no behavior).
- **A8 routing/budget residual** — pass the allocator's routing decision into `runtime/inference` and
  retire the duplicate Python routing policy.

## Release-readiness (phase 11, for reference)

Threat model & security review; performance/load; chaos/recovery; packaging/installers; release gates
& versioning (tags, CHANGELOG); docs completion; repository audits (use `revu` whole-repo); migration
guides. v1 has **no tags, no release workflow, no install story** today.

## Validation gates

| Gate | Command |
|---|---|
| Docs | fence-aware link scan; `docs-link-check` workflow |
| Review | `revu --profile pr` on the change (static-only) |
| Boundaries | `tools/archtest -root ..` (unchanged; docs-only change) |

## Exit criteria

- [x] v1 surface scope confirmed (console in; desktop → v2).
- [x] Composition coverage table agreed (in-v1 vs v2).
- [x] Control-plane server home agreed (`app/` daemon).
- [ ] Deferred register captured as issues.
- [ ] This plan merged on `dev`.
