# ISS-SIFTER-1 — sifter: production RPC wiring

**Type:** feature / technical debt
**Status:** complete (sifter module update)
**Branch:** `ph4-sifter`
**Phase PR:** [#16](https://github.com/greadee/aa/pull/16)
**Parent umbrella:** `ISS-TRACE-LOOP` — the observation → trace → learning substrate is under-delivered (`docs/issues/trace-learning-substrate.md`)
**Module plan:** [../modules/sifter/updates/production-rpc-wiring/plan.md](../modules/sifter/updates/production-rpc-wiring/plan.md)
**Module summary:** [../modules/sifter/updates/production-rpc-wiring/summary.md](../modules/sifter/updates/production-rpc-wiring/summary.md)

## Goal

Host `aa_sifter.rpc.SifterService` over the real aa inter-module RPC v1 transport — an owner-only local socket carrying newline-delimited JSON-RPC 2.0 — with verified peer identity, enforced deadlines, the full error mapping, durable idempotency, schema validation at the boundary, and a typed client the kernel's production runtime adapter can use. The plan for this work is delivered first; no code lands in the planning pull request.

## Problem

Phase 4 delivered the sifter as a transport-agnostic JSON-RPC service with one redaction chokepoint, but explicitly deferred hosting and the kernel adapter (deviations #6; follow-up "Host the RPC service over the named pipe / Unix socket transport and wire the kernel's production runtime adapter"). As merged, the sifter is only reachable in-process, the kernel's runtime adapter still uses a scripted fake, and the v1 transport guarantees in `contracts/rpc/rpc-v1.md` — owner-only socket, peer and store identity, deadlines, idempotency, namespaced errors — are unimplemented. The sifter is therefore not yet the governed, single path to every model that the architecture requires.

## Requirements

- NDJSON JSON-RPC 2.0 framing over a per-user local socket (Windows named pipe / Unix domain socket), per `contracts/rpc/rpc-v1.md`.
- Owner-only socket creation and attach-or-own single ownership; verified peer identity; store-identity binding (`aa.store_mismatch`).
- Server-side deadline enforcement from `aa.timeoutMs` with fail-closed cancellation and no partial state.
- The full v1 error set, including `aa.unauthorized`, `aa.budget_exceeded`, `aa.approval_required`, `aa.unavailable`, and `aa.conflict`, with `data.retryable` where applicable.
- Durable idempotency for mutating methods across the retention window.
- Full JSON Schema validation at the RPC boundary, not only the Python binding.
- A typed client and the seam for the kernel's production runtime adapter.
- Security tests proving redaction and identity behavior over the transport.

## Acceptance Criteria

- [x] `aa-sifter serve` binds the v1 socket and serves `sifter.route`, `sifter.generate`, `sifter.health`, and `sifter.recommend`.
- [x] The socket is owner-only; a peer or store mismatch is rejected.
- [x] Timeouts fail closed with no partial state.
- [x] Boundary objects validate against the `aa_contracts` JSON Schema.
- [x] Idempotent repeats return the original result for the retention window.
- [x] A spy provider sees no secret over the transport.
- [x] The default test suite makes no network calls; the Go checks stay green.

## Affected branches

The sifter server and client land on `ph4-sifter`. The kernel-side production runtime adapter that consumes the client is a dependent sub-problem on the kernel branch; the memory-backed `MemorySink` writer is a separate `aa-memory` integration. Their solutions are not specified here.

## Notes

The implementation slices are listed in `docs/modules/sifter/updates/production-rpc-wiring/plan.md` and delivered on this branch; the module summary records what shipped, the decisions affirmed, and the deviations.
