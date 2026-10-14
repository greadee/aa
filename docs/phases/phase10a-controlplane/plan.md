# Phase 10a — Control Plane — Plan

> **Type:** phase (`phase10a-controlplane`). **Branch:** `phase10a-controlplane` → `main`.
> **Base:** `main @ 8d2295a` (v1 scope & composition merged; see [v1-readiness](../../updates/v1-readiness/plan.md)).
> **Predecessor:** `v1-readiness` (PR #42). **Sibling:** `phase10b-console` (the client).
> **Diagram:** [plan.uml](./plan.uml).

## Objective

Deliver the **real control-plane server**: one `app/` daemon that composes the platform modules and
serves the external, versioned **HTTP/JSON control plane** (`contracts/openapi/control-plane-v1.yaml`)
over loopback. The core path is deterministic and model-free; models stay opt-in. This is the substrate
the console (phase 10b) is a client of.

## Starting state

- `app/` exists as the composition root (`aa selfcheck` builds and exercises the graph but serves nothing).
- `kernel/api` is an **in-process** control-plane service, disabled by default.
- `contracts/openapi/control-plane-v1.yaml` defines the paths (health/projects/work/memory/nodes/tools).
- `contracts/rpc/rpc-v1.md` defines the inter-module RPC envelope/errors/idempotency (design precedent).
- v1 floor composed by `app/`: contracts, `forge/memsync`, kernel (context/allocator/routing/observability),
  memory (store/repo/projection/retrieval/retention), `registry/models`, `runtime/sandbox`, toolbox (policy/computeruse).

## Scope

**In:** the `app/` HTTP/JSON daemon; loopback bind + bearer-token carrying; the v1 control-plane endpoints
over the composed slice; typed errors/envelope parity with the RPC spec; deterministic (no model on the
core path); integration tests.

**Out:** real authn/authz (pairing/keys/mTLS) → phase 12; the console/UI (→ phase 10b); post-console
modules (sync/obsv/visualizer/reserveds/lifecycle) → phase 11; desktop surface (→ v2).

## Decisions

| # | Decision | Reasoning |
|---|---|---|
| C1 | Server home is **`app/`**; `kernel/api` provides the in-process service it exposes | One composition root owns wiring (per D2 of v1-readiness) |
| C2 | Transport is **HTTP/JSON over loopback**, path/response shapes exactly per `control-plane-v1.yaml` | The console is specified against this spec; no drift |
| C3 | **Loopback bearer token carrying only**; no authn claim | Loopback is not authentication; real authn is phase 12 |
| C4 | Core path runs with the **fake/no-model provider**; models opt-in | Deterministic, testable, no external calls in the default suite |
| C5 | Errors reuse the **`rpc-v1` error codes/envelope** (governed `aa.*` codes) | One error vocabulary across boundaries |

## Endpoint map (v1)

| Path | Source module | Notes |
|---|---|---|
| `GET /health`, `GET /ready` | daemon | liveness/readiness of the composed graph |
| `/projects` | memory (project records) | list/get/create project |
| `/work` (+ graph, assignments) | kernel (planner/scheduler/allocator) | submit work; inspect graph/assignments |
| `/memory` | memory (repo/retrieval) | browse/search records |
| `/tools` | toolbox (registry/host/policy) | list manifests; invoke under a contract |
| `/nodes` | single-node registry (sync-scoped) | node status (single node for v1) |

## Stages and commits

DoD per commit: `gofmt`/build/vet/test green; `tools/archtest` clean; docs current.

- **10a.1 — Plan** (`add phase10a control plane plan`).
- **10a.2 — Daemon + health**: `app/` serves `kernel/api` over loopback HTTP/JSON; config (bind/port/token); `GET /health`,`/ready`. (`add control-plane daemon and health endpoints`)
- **10a.3 — Projects + work**: compose memory + kernel planner/scheduler/allocator; `/projects`, `/work` (submit + inspect graph/assignments). (`serve projects and work over the control plane`)
- **10a.4 — Memory + retrieval**: `/memory` browse/search over `memory/repo` + `retrieval`. (`serve memory and retrieval`)
- **10a.5 — Tools + nodes**: toolbox registry/host/policy `/tools`; single-node `/nodes` read model. (`serve tools and nodes`)
- **10a.6 — Errors + determinism**: unify envelope/error codes with `rpc-v1`; bearer-token middleware; wire the fake/no-model path. (`harden control-plane errors and determinism`)
- **10a.7 — Tests + docs**: `httptest` integration against the OpenAPI paths; module docs; phase summary. (`add control-plane integration tests and docs`)

**Commit budget:** ~7 (phase total 10a+10b target 10–15).

## Validation gates

| Gate | Command |
|---|---|
| Formatting | `gofmt -l .` |
| Go | per-module `go build/vet/test ./...` (`app`, `kernel`, `memory`, `toolbox`, `runtime`) |
| Boundaries | `tools/archtest -root ..` |
| API | `httptest` integration over `control-plane-v1.yaml` paths |
| Review | `revu --profile pr` on each PR (static-only) |

## Exit criteria

- [ ] `app/` runs a daemon serving the v1 control-plane paths over loopback.
- [ ] The composed slice (kernel/memory/toolbox/runtime) is reachable through the API; core path is model-free.
- [ ] Envelope/errors match `rpc-v1`; bearer-token carrying enforced (documented as not authentication).
- [ ] Integration tests pass; phase summary authored; `phase10b-console` can build its client against this.

## Open items

- Console **UI/TUI appearance and component design** are decided with the maintainer separately and are
  intentionally not specified here (see phase 10b).
