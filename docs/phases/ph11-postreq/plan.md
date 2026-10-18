# Phase 11 — console-postreq — Plan

> **Type:** phase (`ph11-postreq`). **Branch:** `ph11-postreq` → `dev`. **Tracker:** milestone [`v1 — phases 11-12`](https://github.com/greadee/aa/milestone/2).
> **Predecessor:** Phase 10 — control plane (10a) & console (10b) — merged (PR #11).
> **Issues:** [#43](https://github.com/greadee/aa/issues/43)–[#48](https://github.com/greadee/aa/issues/48).
> **Diagram:** [plan.uml](./plan.uml).

## Objective

Implement the modules **deferred out of v1** — so nothing ships half-baked — composing each into the
control plane (`app/`, phase 10a) and, where relevant, the console (`ui/`, phase 10b). Progress for
this phase **and** phase 12 is tracked through the
**[`v1 — phases 11-12` milestone](https://github.com/greadee/aa/milestone/2)**.

## Starting state

- Phase 10 merged: `app/` serves the control plane; `ui/` console is a client of it; `aa serve` / `aa console`.
- Deferred, test-only or uncomposed: `sync`, `obsv`, `visualizer`, `registry/{teams,routines,policies}`,
  `runtime/lifecycle`; the A8 routing/budget residual.
- The single-node v1 posture (D6) and the reserved-package decisions come from
  [`v1-readiness`](../../updates/v1-readiness/plan.md).

## Scope

The six tracked issues, one commit each:

| Stage | Issue | Work |
|---|---|---|
| 11.2 | [#43](https://github.com/greadee/aa/issues/43) ISS-DeferPH11-1 | Wire `sync` into the control plane (single-node scope per D6). |
| 11.3 | [#44](https://github.com/greadee/aa/issues/44) ISS-DeferPH11-2 | Compose the `obsv` observation substrate. |
| 11.4 | [#45](https://github.com/greadee/aa/issues/45) ISS-DeferPH11-3 | Compose the `visualizer` (obsv consumer). |
| 11.5 | [#46](https://github.com/greadee/aa/issues/46) ISS-DeferPH11-4 | Implement `registry/{teams,routines,policies}` stores. |
| 11.6 | [#47](https://github.com/greadee/aa/issues/47) ISS-DeferPH11-5 | Implement `runtime/lifecycle` worker lifecycle. |
| 11.7 | [#48](https://github.com/greadee/aa/issues/48) ISS-DeferPH11-6 | Complete the A8 routing/budget residual. |

## Decisions

| # | Decision | Reasoning |
|---|---|---|
| P1 | One commit per issue; structured like the `issue-impl-sep28` sprint | Reviewable, per-issue closeout |
| P2 | Compose into the `app/` daemon; expose via the v1 control-plane paths where applicable | One composition root; console stays a client |
| P3 | `sync` is scoped to single-node for v1 unless the issue's decision says otherwise (D6) | Avoids a multi-machine scope jump |
| P4 | Close each issue with `Closes #n` and an "implemented" comment | Durable record |

## Stages and commits

- **11.1 — Plan** (`add ph11 console-postreq plan and scope`).
- **11.2 — [#43]** wire sync (`wire sync into the control plane`).
- **11.3 — [#44]** obsv (`compose the obsv observation substrate`).
- **11.4 — [#45]** visualizer (`compose the visualizer`).
- **11.5 — [#46]** registry (`implement registry teams, routines and policies`).
- **11.6 — [#47]** lifecycle (`implement runtime worker lifecycle`).
- **11.7 — [#48]** A8 residual (`complete the A8 routing and budget residual`).
- **11.8 — Closeout** (`add ph11 console-postreq summary and close out issues`).

DoD per commit: affected module `gofmt`/build/vet/test green; `tools/archtest` clean; docs current.

## Validation gates

| Gate | Command |
|---|---|
| Formatting | `gofmt -l .` |
| Go | per-module `go build/vet/test ./...` |
| Boundaries | `tools/archtest -root ..` |
| Control plane | `httptest` integration over the OpenAPI paths |
| Review | `revu --profile pr` on the phase PR (bot-authored, static-only) |

## Exit criteria

- [x] All six issues implemented (or explicitly removed) and composed in the daemon.
- [x] Control-plane/console surfaces updated where the issue requires a read model.
- [x] `revu` clean on the phase PR; architecture boundaries hold.
- [x] Issues #43–#48 closed; the `v1 — phases 11-12` **milestone** updated.

See [summary.md](./summary.md) for the as-built record.

## Tracking

Progress for phases 11 and 12 is tracked through the
**[`v1 — phases 11-12` milestone](https://github.com/greadee/aa/milestone/2)**.
This phase's commits land on `dev`; the phase PR (and the milestone) reflect progress toward `main`.
