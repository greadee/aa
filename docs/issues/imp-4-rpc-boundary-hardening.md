# ISS-IMP-4 — contracts/rpc: RPC / boundary / version-skew hardening

**Type:** feature / technical debt
**Status:** planned
**Branch:** `dev`
**Sprint PR:** [#21](https://github.com/greadee/aa/pull/21)
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

- [ ] Versioning and compatibility rules specified and tested at each versioned boundary.
- [ ] Unsupported major versions fail closed with `aa.incompatible`.
- [ ] Error schemas, deadlines, and retry/idempotency behavior are consistent.
- [ ] No new network boundary is introduced for a non-distributed module.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, `tools/archtest`, and docs link check pass.
- [ ] tests added or updated
- [ ] documentation updated where required

## Dependencies

Follows ISS-IMP-1. Informs ISS-IMP-8 (scheduler/runtime dispatch) and ISS-IMP-10 (observability).
