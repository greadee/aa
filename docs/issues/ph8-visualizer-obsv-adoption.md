# ISS-OBSV-2 — visualizer: adopt obsv transport and retire the AAV duplicate protocol

**Type:** feature / technical debt
**Status:** complete (visualizer module update — delivered)
**Branch:** `ph8-visualizer`
**Phase PR:** [#19](https://github.com/greadee/aa/pull/19) — open
**Parent umbrella:** none (standalone issue); dependency `ISS-OBSV-1` (`ph3-kernel`)
**Related:** [architecture](../../docs/architecture/README.md) §3.1, §5.2, §5.9, §14; decision D10
**Module plan:** [../modules/visualizer/updates/obsv-adoption/plan.md](../modules/visualizer/updates/obsv-adoption/plan.md)
**Module summary:** [../modules/visualizer/updates/obsv-adoption/summary.md](../modules/visualizer/updates/obsv-adoption/summary.md)

## Goal

Make the visualizer consume the one shared `obsv` observation protocol for both live and replay, and retire the duplicated observation protocol, IPC, and event-store carried from `agent-action-visualizer` (AAV) so that `obsv` is the only observation vocabulary.

## Problem

Phase 8 delivered the visualizer's deterministic projection, layout, replay, bridge, browsing, retention, and the explicit compatibility profile, but it could not adopt the live observation transport: `obsv` was scaffold-only, so an `EventSource` seam and a deterministic in-memory fake stood in (phase 8 deviations 1 and 2). The architecture still records the residual risk in two places: AAV and `obsv` protocol duplication with incompatible stripping (§3.1), and the `obsv`/visualizer "adoption by AAV" remaining work (§5.2, §5.9, §14, decision D10). Until the visualizer reads `obsv`, AAV's own protocol/IPC/event-store remains a second vocabulary and the "one observation protocol" claim is unmet.

This is a standalone issue, not a sub-issue of the trace-learning umbrella: the visualizer is a display consumer of observation, not part of the trace → learning path.

## Sub-problems

These are deliberately separate problems and are not one issue.

| Id | Problem | Module / branch | Why it matters | Status |
|---|---|---|---|---|
| ISS-OBSV-2a | Visualizer transport adoption: real `obsv` `Replay`/`Subscribe` behind the existing `source.EventSource` seam, an explicit `obsv/protocol.Event` → `contracts.Event` adapter, the compatibility profile over real observation metadata, and live-equals-replay preserved | `visualizer` / `ph8-visualizer` | The visualizer must show the same events `obsv` produces, not a synthetic fake | complete |
| ISS-OBSV-2b | Protocol/IPC/event-store de-duplication: AAV converges on the single `obsv` protocol plus the `compat` profile; the visualizer defines no observation protocol, transport, or event-store; enforced by a boundary test | `visualizer` / `ph8-visualizer` | A second vocabulary re-opens the concern §3.1 resolved; one owner per protocol | complete |

Both sub-problems are owned by the same branch, `ph8-visualizer`, and are delivered by one module update; their slices are listed separately in the module plan.

## Requirements

### ISS-OBSV-2a — transport adoption

- An `EventSource` implementation backed by `obsv` `Replay`/`Subscribe`; `source.Fake` becomes test-only.
- A single, explicit adapter from `obsv/protocol.Event` to the visualizer event type (`contracts` `v1.Event`), mapping session, sequence, actor, work-package/attempt identity, and observation metadata into the payload; the visualizer defines no second protocol.
- The `compat` profile normalizes real observation metadata (`secondary_paths`, `access_sequence`); unknown metadata remains ignored.
- Live and replay continue to share one projection and one render path; live equals replay is preserved over the real transport.
- Source selection lets the in-process `obsv` transport be used now and the kernel-host socket later without changing callers; the default suite stays offline and deterministic.

### ISS-OBSV-2b — de-duplication

- No package under `visualizer/` declares an observation protocol, a transport, or an event-store; `obsv` owns all three.
- A boundary test fails if a second observation vocabulary/transport/event-store appears in `visualizer`, or if live observation bypasses `obsv`.
- The AAV convergence is documented: the visualizer directive and README state that the shared protocol and the `compat` profile replace AAV's duplicated protocol/IPC/event-store.

## Acceptance Criteria

- [x] `visualizer/source` reads a session through `obsv` `Replay` and follows it through `obsv` `Subscribe`; no fake provider remains (the fake was deleted, not just made test-only).
- [x] `obsv/protocol.Event` maps to the visualizer event type through one adapter; no second protocol type exists in `visualizer`.
- [x] `secondary_paths` and `access_sequence` are normalized by `visualizer/compat` from real observation metadata; unknown metadata is ignored.
- [x] Live output equals replay output over the real transport.
- [x] Default `visualizer` tests make no network call and never import `kernel`.
- [x] A boundary test enforces the single observation protocol/transport/event-store.
- [x] `visualizer/aa-visualizer.md`, `visualizer/README.md`, and architecture §3.1/§5.2/§5.9/§14 reflect the adopted state.
- [x] `go build`, `go vet`, `go test`, `gofmt`, and `tools/archtest` pass.
- [x] tests added or updated
- [x] documentation updated where required

## Affected branches

- `ph8-visualizer` (this module update): all implementation slices.
- `ph3-kernel` / `ISS-OBSV-1` (dependency): the `obsv` module API this update consumes, and the durable metadata allowlist extension below. Its solution is not specified here.
- `ph2-memory` (adjacent, not required): session browsing continues over the memory query seam.

## Dependencies

- **`ISS-OBSV-1` (`ph3-kernel`)** delivers the `obsv` API (`Replay`/`Subscribe`, protocol types) this update consumes.
- **Allowlist extension.** The `obsv` durable metadata allowlist does not currently include the profile keys `secondary_paths`/`access_sequence`, so the `compat` profile cannot see them over the real transport. The extension is requested under `ISS-OBSV-1` on `ph3-kernel`; this issue consumes the extended allowlist and does not edit `obsv`.
- The visualizer can begin against the in-process `obsv` transport once `ISS-OBSV-1` merges; the cross-process live path additionally waits on the kernel-host socket (also `ISS-OBSV-1`).

## Notes

This document and the module plan are planning artifacts. Implementation slices are listed in `docs/modules/visualizer/updates/obsv-adoption/plan.md` and proceed only after the plan is reviewed.

**Outcome.** All implementation slices are complete on `ph8-visualizer`; the as-built record is [../modules/visualizer/updates/obsv-adoption/summary.md](../modules/visualizer/updates/obsv-adoption/summary.md) and the module update PR is [#19](https://github.com/greadee/aa/pull/19).
