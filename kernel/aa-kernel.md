# aa-kernel.md

Agent directive for the `kernel` module (aa-kernel).

## Responsibility

Own deterministic coordination of all work: plan it, select roles and workers, dispatch it, gate it, and observe it. The kernel is the platform; models are workers.

## Owns

- The `obsv` host (journal + local socket) for current sessions.
- Planning: task aggregate, dependency graph, readiness.
- Role, trade, and worker registry plus deterministic role selection.
- The scheduler, leases, and the assignment state machine.
- The context compiler (bounded, deterministic context bundles).
- Execution contracts, permissions, and budgets.
- Coordination of execution, delegating execution mechanics to the top-level `runtime` module (the worker adapter lives in `runtime/worker`).
- Workspace and Git worktree management.
- Compute-node registry and eligibility.
- Result intake (untrusted envelopes until accepted).
- Integration and human gates.
- Operational telemetry.
- The job-learning engine (see [aa-joblearn.md](aa-joblearn.md)).
- The control-plane API.

## Must Not

- Perform file transfer (use `sync`).
- Store canonical history (use `memory`).
- Call models directly (use `sifter` over RPC).
- Open a remote shell or arbitrary remote execution (use a capability-scoped runtime adapter).
- Import `sync`, `forge`, or `sifter` code; only RPC.

## Interfaces

- Control-plane API for `ui` and `visualizer`.
- RPC clients to `sifter`, `sync`, and `forge`.
- `obsv` host APIs.
- `memory` query and record APIs.

## Rules

1. Coordination is deterministic; models supply judgment inside contracts.
2. Execution is opt-in, capability-scoped, and disabled by default.
3. Workers report; authority validates. Builders never self-certify.
4. Results are untrusted until they pass validation and gates.
5. One node holds authority per project; replicas observe only.
6. Learned or probabilistic selection never replaces a deterministic fallback.

## Canonical references

- [Architecture](../docs/architecture/README.md)
- [Terminology, state registry, events](../docs/reference/terminology.md)
