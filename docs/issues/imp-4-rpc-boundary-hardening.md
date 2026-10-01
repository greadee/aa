# ISS-IMP-4 — contracts/rpc: RPC / boundary / version-skew hardening

**Type:** feature / technical debt
**Status:** complete
**Branch:** `dev`
**Sprint PR:** [#21](https://github.com/greadee/aa/pull/21)
**GitHub issue:** [#26](https://github.com/greadee/aa/issues/26)
**Parent umbrella:** [ISS-IMP](post-refactor-implementation.md)
**Sprint plan:** [../updates/issue-impl-sep28/plan.md](../updates/issue-impl-sep28/plan.md) (Stage 1)

## Goal

Make inter-module communication resilient across the boundaries that already require RPC or versioned
contracts, matching the refactored architecture.

## Problem

`contracts/rpc/rpc-v1.md` defines the envelope, security, errors, idempotency, and methods, but
contract versioning, compatibility, capability negotiation, unsupported-version behavior, and
retry/timeout semantics are not specified end-to-end. The refactor renamed the model-routing service
(`sifter.*` → `inference.*`) within envelope v1, which must be handled deliberately.

## Requirements

- Audit which boundaries are actually distributed or versioned (`inference`, `sync`, `forge`,
  `obsv` host, control-plane API) before adding anything.
- Contract versioning and backward/forward compatibility rules.
- Capability negotiation.
- Serialization and error-schema consistency.
- Timeouts and retries where appropriate; unsupported-version behavior (`aa.incompatible`).
- Do **not** create network boundaries merely because modules are logically separate.
- Preserve idempotency semantics for mutating methods.

## Acceptance Criteria

- [x] Versioning and compatibility rules specified and tested at each versioned boundary.
- [x] Unsupported major versions fail closed with `aa.incompatible` (RPC **and** contract major; Go adapters + inference).
- [x] Error schemas, deadlines, and retry/idempotency behavior are consistent (`data.retryable`; server-enforced deadlines; keyed idempotency).
- [x] No new network boundary is introduced for a non-distributed module (boundary audit; in-process adapters unchanged).
- [x] `go build`, `go vet`, `go test`, `gofmt`, `tools/archtest`, and docs link check pass.
- [x] tests added or updated
- [x] documentation updated where required

## Solution

`contracts/rpc/rpc-v1.md` now carries a **boundary audit** (which boundaries are
distributed vs in-process), firm **versioning and compatibility** rules (envelope
frozen at major 1; contract major gate; minors compatible both ways; fail closed),
**capability negotiation**, and **deadline/retry/idempotency** rules, with the
error set carrying `data.retryable`.

Implemented consistently across the versioned boundaries:

- Python `inference`: `is_compatible` now also gates the contract major; `health`
  advertises `rpcVersion`/`contractVersion`/`capabilities[]` (deadline, idempotency,
  identity, framing already present).
- Go adapters (`forge`, `toolbox`, `sync`): validate the contract major, attach
  `data.retryable` to governed errors, and expose `Capabilities()`.

Decision record: [ADR-0144](../adr/ADR-0144-rpc-v1-boundary-hardening.md).

## Dependencies

Follows ISS-IMP-1. Informs ISS-IMP-8 (scheduler/runtime dispatch) and ISS-IMP-10 (observability).
