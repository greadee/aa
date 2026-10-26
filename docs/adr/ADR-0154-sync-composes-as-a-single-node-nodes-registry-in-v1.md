# ADR-0154 — Sync composes as a single-node nodes registry in v1

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [ISS-DeferPH11-1](https://github.com/greadee/aa/issues/43) |
| **Scope** | phase 11 · console-postreq (Stage 11.2) |

## Context

The `sync` module is implemented but had no production caller. The v1 scope
decision (D4) deferred it, leaving the console's `nodes` read model backed by a
hard-coded `local` entry. Issue #43 asked for the open decision D6 to be
resolved before composing it:

- (a) full multi-machine sync — pairing, revisions, offline queue, parallel
  placement across machines; or
- (b) a minimal single-node `nodes` registry sufficient for the console's
  `nodes` read model.

D6 already fixes v1 as single-authority, single-node execution with read-only
replicas, so full multi-machine sync would exceed the v1 posture.

## Decision

**Option (b): compose sync as a single-node nodes registry in v1.**

- The daemon owns one local Ed25519 identity. Its seed is persisted at
  `<root>/node.seed` with `0600` permissions so the node keeps a stable ID
  across restarts.
- `GET /api/v1/nodes` is served from the composed sync plane: the local node is
  the only authority node; declared peers appear with a non-authority state.
- `GET /api/v1/sync/status` reports the local node ID, the node registry, and
  transfer receipts through the `sync.status` RPC adapter.
- Distribution, revision reconciliation, and the offline queue are composed
  over the transport seam and exercised by the daemon's tests, but no remote
  peer session is exposed in v1. Enabling paired multi-machine sync is a later
  phase, behind the same seam.
- Explicit peer registration is local bookkeeping, **not** authentication.
  Pairing, mutual authentication, and live network transport stay out of v1.
- Sync carries payloads only. It gains no execution authority, starts no
  runtime, and exposes no remote shell or hidden control channel.

## Rationale

- Matches D6: one authority node, no split-brain or multi-writer risk.
- Gives the console a real backend instead of a placeholder with a small,
  reviewable surface.
- Keeps the sync module's verified transfer/reconcile pieces composed and
  tested, so a later multi-node phase adds transport and pairing rather than
  re-plumbing the daemon.

## Alternatives and consequences

- **Full multi-machine sync now (a):** rejected — it requires pairing, live
  transport, and multi-writer reconciliation policy, which contradict the v1
  single-node posture and would ship an unexercised surface.
- **Leave the hard-coded node:** rejected — the console read model would stay a
  placeholder and `sync` would remain uncalled.
- Consequence: v1 does not authenticate peers or move work across machines.
  The `sync` node registry must not be presented as a trust boundary.

## Related

- [sync README](../../sync/README.md)
- [v1 readiness plan](../updates/v1-readiness/plan.md) (D4, D6)
- [control-plane contract](../../contracts/openapi/control-plane-v1.yaml)
- [ADR index](./README.md)
