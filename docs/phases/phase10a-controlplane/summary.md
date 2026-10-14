# Phase 10a — Control Plane — Summary

> End-of-phase summary. Derived from [plan.md](./plan.md). Branch: `phase10a-controlplane`.

## Delivered

- **`app/` control-plane daemon** (`aa serve --addr --token --root`): serves the external, versioned
  HTTP/JSON control plane (`contracts/openapi/control-plane-v1.yaml`) over loopback with graceful
  shutdown.
- **Endpoints:** `GET /health` (unauthenticated), `GET /status`, `GET|POST /projects`,
  `GET|POST /projects/{id}/work-packages`, `GET /projects/{id}/graph`, `GET /projects/{id}/issues`,
  `GET /projects/{id}/strategies`, `GET /projects/{id}/events`, `GET /assignments`, `GET /nodes`,
  `GET /tools`, `POST /approvals/{gateId}`.
- **Composition:** a real kernel scheduler (`planner.Graph` + `role_allocator.Registry`) exposed
  in-process by `kernel/api`; memory records; a toolbox registry (with the computeruse manifest).
  Execution stays **disabled** on the core path (no model, no worker).
- **Errors:** governed `aa.*` codes (aligned with `contracts/rpc/rpc-v1.md`); loopback bearer-token
  carrying (documented as *not* authentication); JSON error envelope for every non-2xx, including 404.
- **Tests:** handler unit tests + an OpenAPI-path integration test; `tools/archtest` clean.

## Commits

| Commit | Change |
|---|---|
| `303be78` | add phase10a control plane plan |
| `9d6ef1d` | rename phase 10 to control plane and console; renumber phases 11–12 |
| `ffabb2e` | add control-plane daemon and health endpoints |
| `47bcac2` | serve projects and work over the control plane |
| `eb00834` | serve memory, nodes and tools over the control plane |
| `3e019b4` | harden control-plane errors and method handling |
| _(this)_ | add control-plane integration tests and phase summary |

## Validation

| Gate | Result |
|---|---|
| `gofmt` / `go build` / `go vet` / `go test` (`app`) | pass |
| `tools/archtest -root ..` | pass (`app` as composition root) |
| OpenAPI paths (`httptest`) | all pass |

## Deviations / notes

- `/events` returns a JSON array; the spec's `text/event-stream` (SSE) form is deferred.
- `/approvals/{gateId}` maps the gate id onto the scheduler's approval call (single project).
- Single project (`default`) for v1; multi-project is future work.
- Auth is bearer-token carrying on loopback only; real authn/authz is phase 12.

## Exit criteria

- [x] `app/` serves the v1 control-plane paths over loopback.
- [x] The composed slice (kernel/memory/toolbox) is reachable through the API; core path is model-free.
- [x] Envelope/errors use the `aa.*` vocabulary; bearer-token carrying enforced.
- [x] Integration tests pass; `phase10b-console` can build its client against this.
