## Summary

Pre-hardening sprint for the `toolbox` module addressing all findings from a thorough code review. This sprint fixes critical and high-severity issues (TOCTOU race, stale contract versions, RPC fragility) and medium-severity improvements (timeouts, error structure, config validation, observability) before ph7-toolbox is merged to main.

## Related Issue

Refs #50 (umbrella: toolbox pre-hardening)

## Changes

### P0 — Critical
- Fix TOCTOU race condition in `Service.Handle` that allows concurrent duplicate executions of the same idempotent request

### P1 — High
- Update all toolbox test fixtures to use active contract version `"2.0"` instead of stale `"1.0"`
- Replace `map[string]any` deserialization with typed request/response structs in the RPC layer

### P2 — Medium
- Honor `TimeoutMS` field from `AA` request metadata (currently ignored)
- Add structured diagnostic data to RPC error responses
- Validate `WorkflowStep.Config` per step kind
- Add metrics collection interface (invocation count, denial rate, latency)

### P3 — Low
- Split large contract files (`records.go`, `event.go`, `telemetry.go`) into domain-specific units
- Review Python/TypeScript bindings for parity with Go bindings

## Validation

- [ ] unit tests
- [ ] integration tests
- [ ] lint/type checks
- [ ] manual verification if relevant

## Notes

This sprint is self-contained within the `toolbox` module. No cross-module changes are required. All fixes are backward-compatible (no API changes to existing callers).

## Phase

Part of #50 (umbrella: toolbox pre-hardening)

Milestone: v1 — Toolbox Pre-Hardening
