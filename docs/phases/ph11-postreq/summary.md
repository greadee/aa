# Phase 11 — console-postreq — Summary

> End-of-phase summary. Derived from [plan.md](./plan.md).
> Branch: `ph11-postreq` → `dev`. Tracker: milestone [`v1 — phases 11-12`](https://github.com/greadee/aa/milestone/2).
> Phase PR: [#49](https://github.com/greadee/aa/pull/49).

## Delivered

The six deferred modules now have behavior, production callers, tests, and
control-plane surfaces. Nothing remains a behavior-less placeholder.

- **`sync` in the control plane (#43, `a49e6a8`).** A composed sync plane owns
  one persisted local Ed25519 identity, serves `/nodes` and `/sync/status`, and
  exercises distribution/revision reconciliation over the transport seam.
  ADR-0154 records D6 option (b): v1 is a single-node nodes registry. Declared
  peers are bookkeeping, not authentication; no remote execution is exposed.
- **`obsv` substrate (#44, `8d78d32`).** Durable per-session JSONL journals,
  metadata-only sanitization before journaling, failure-open emission, retained
  deterministic reports, Codex JSONL translation, replay/subscribe, and project/
  work/gate seam events. Served at `/observations/*`.
- **`visualizer` over `obsv` (#45, `14a9fb4`).** Replay and live subscription
  consume the daemon's single obsv transport; canonical browsing goes through
  the `memory` query seam; retained timelines, final graphs, and a live SSE
  stream are served at `/visualizer/*`. The single adapter and archtest boundary
  remain intact.
- **`registry/{teams,routines,policies}` (#46, `4b02ec1`).** Validated
  definition stores with duplicate rejection, persisted through `memory` and
  restored on startup. Policies add an approval/budget/routing/gate definition
  catalogue; applicability stays elsewhere. Served at `/teams`, `/routines`,
  `/policies`.
- **`runtime/lifecycle` (#47, `b529513`).** Typed worker states from creation to
  stopped/terminated/failed, cancellation and cleanup, a scheduler lifecycle
  hook plus a `worker.Adapter` implementation, failure-open events forwarded to
  `obsv`, and `/workers` + `/workers/events` read models.
- **A8 routing/budget residual (#48, `766b72a`).** `v2.RoutingDecision` is
  carried as the RPC `routingDecision`; inference executes the allocator's
  locality/tier, rejects absent or skewed decisions, and fails closed on a
  blocked decision. The duplicate Python policy engine is retired; preflight,
  budget accounting, and escalation remain execution-time helpers. ADR-0151 is
  updated to the completed state.

## Commits

| Commit | Change |
|---|---|
| `bdc09e7` | add ph11 console-postreq plan and scope |
| `4118e56` | fix v1-readiness link in ph11 plan |
| `0779f95` | track ph11 via the v1 milestone, not the merged pr 42 |
| `a49e6a8` | wire sync into the control plane |
| `8d78d32` | compose the obsv observation substrate |
| `14a9fb4` | compose the visualizer |
| `4b02ec1` | implement registry teams, routines and policies |
| `b529513` | implement runtime worker lifecycle |
| `766b72a` | complete the A8 routing and budget residual |
| `1d0b450` | mark the retired python routing policy reference as superseded |
| _(this)_ | add ph11 console-postreq summary and close out issues |

## Decisions affirmed

- **P1 — one commit per issue.** Applied for #43–#48; the revu reference fix is
  a separate review commit.
- **P2 — compose into `app/` and expose read models.** All six modules are
  composed at the daemon; the console stays a client of the control plane.
- **P3 / D6 — single-node sync.** Option (b) implemented and recorded in
  ADR-0154; multi-machine pairing and live transport remain follow-ups.
- **P4 — close each issue with evidence.** Issues #43–#48 carry implemented
  comments and close when the phase PR merges.

## Validation

| Gate | Result |
|---|---|
| Go: per-module `go build` / `go vet` / `go test ./...` | pass (`contracts`, `sync`, `obsv`, `visualizer`, `registry`, `runtime`, `kernel`, `app`) |
| Python `runtime/inference`: pytest / ruff / mypy | pass |
| `tools/archtest -root ..` | pass (boundaries incl. visualizer single-observer guard) |
| Control-plane `httptest` over the OpenAPI paths | pass (sync, observations, visualizer, registry, lifecycle) |
| `revu --profile pr` (static-only, base `origin/dev`) | clean: 0 findings, 118/118 files, recommendation Approve |
| `aa selfcheck` | pass; exercises sync, obsv/visualizer, registry, lifecycle, and the routing contract |

## Deviations / notes

- **Sync** is single-node for v1. Distribution and reconciliation are composed
  and tested, but no peer session or live network transport is exposed.
- **Registry** definitions are durable through the memory store; the registries
  themselves stay allocation-free and runtime-instance-free.
- **Lifecycle** is wired through the scheduler's dispatch hook even while the
  core execution path remains disabled; the adapter path is covered by tests.
- **A8** keeps a local fallback only for the standalone inference CLI when no
  allocator decision is supplied. The RPC execution path always requires a
  decision and never re-decides.
- The desktop surface and multi-machine sync remain v2.

## Exit criteria

- [x] All six issues implemented (or explicitly removed) and composed in the daemon.
- [x] Control-plane/console surfaces updated where the issue requires a read model.
- [x] `revu` clean on the phase PR; architecture boundaries hold.
- [x] Issues #43–#48 closed; the `v1 — phases 11-12` milestone updated.
