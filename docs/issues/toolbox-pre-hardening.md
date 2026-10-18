# Toolbox Pre-Hardening Sprint — Umbrella

**Type:** audit
**Status:** planned
**Branch:** `dev` (update phase)
**GitHub issue:** [to be created]

## Goal

Address all findings from the toolbox module code review (TOCTOU race, stale contract versions, RPC fragility, missing timeouts, untyped config, missing observability, large contract files, and binding parity) in a single pre-hardening sprint so the toolbox module meets production readiness before phase 7 (ph7-toolbox) is merged to main.

## Problem

The toolbox module (`aa-toolbox`) is architecturally sound but has several pre-hardening gaps discovered during a thorough code review:

- The RPC service has a TOCTOU race condition in its idempotency cache that allows concurrent duplicate executions.
- Most test fixtures use stale `ContractVersion: "1.0"` while the active contracts are v2 (`"2.0"`).
- The RPC layer deserializes everything into `map[string]any` with manual field extraction — no compile-time safety.
- The `TimeoutMS` field on the `AA` struct is never honored in `Handle`.
- RPC error responses lack structured diagnostic data for callers.
- `WorkflowStep.Config` is untyped with no per-step-kind validation.
- The toolbox has no metrics or observability beyond an in-memory audit log.
- Several large contract files (`records.go`, `event.go`, `telemetry.go`) should be split for navigability.
- Python and TypeScript bindings have not been reviewed for parity with Go.

These are all pre-hardening concerns that should be resolved before the toolbox phase is considered complete.

## Requirements

- Fix the TOCTOU race in `Service.Handle` to prevent concurrent duplicate executions.
- Update all toolbox test fixtures to use `ContractVersion: "2.0"`.
- Add typed request/response structs to the RPC layer (or schema-driven decoder).
- Honor `TimeoutMS` in the RPC handler.
- Add structured error data to RPC error responses.
- Add validation for `WorkflowStep.Config` per step kind.
- Add metrics collection (invocation count, denial rate, latency histograms).
- Split large contract files into smaller, more navigable units.
- Review Python and TypeScript bindings for parity with Go.

## Acceptance Criteria

- [ ] observable behavior
- [ ] tests added or updated
- [ ] error cases handled
- [ ] documentation updated where required

## Issues

- #??? — RPC TOCTOU race (P0)
- #??? — Stale contract version in tests (P1)
- #??? — RPC layer uses map[string]any (P1)
- #??? — No timeout honored in RPC (P2)
- #??? — RPC error responses lack structured data (P2)
- #??? — Untyped WorkflowStep.Config (P2)
- #??? — No metrics/observability (P2)
- #??? — Large contract files (P3)
- #??? — Python/TypeScript binding parity (P3)

## Dependencies

- None; this sprint is self-contained within the `toolbox` module.

## Risks / Open Questions

- The TOCTOU fix may require restructuring the cache pattern; a channel-based or pending-flag approach is preferred.
- Typed RPC structs increase boilerplate but improve safety — worth the tradeoff.
- Metrics integration with the `obsv` module may be deferred to a later phase if out of scope.

## Exit Criteria

- [ ] all required issues complete
- [ ] no open P0/P1 bugs
- [ ] toolbox test suite passes
- [ ] audit complete
- [ ] ready for main
