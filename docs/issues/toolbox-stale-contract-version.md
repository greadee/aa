# Toolbox — Stale contract version in test fixtures

**Type:** bug
**Severity:** P1 (high)
**Status:** planned
**Branch:** `dev` (update phase)

## Goal

Update all toolbox test fixtures to use the active contract version `"2.0"` instead of the stale `"1.0"`, so tests accurately reflect the current contract generation.

## Problem

The active contracts are generation **v2** (`"2.0"`), as established by `contracts/go/v2/common.go` (`Version = "2.0"`) and documented in the contracts README. However, most toolbox test fixtures hardcode `ContractVersion: "1.0"`:

- `toolbox/host/host_test.go` — `manifest()` and `contract()` helpers
- `toolbox/policy/policy_test.go` — `manifest()` and `contract()` helpers
- `toolbox/registry/registry_test.go` — `manifest()` helper
- `toolbox/rpc/rpc_test.go` — `newService()` helper
- `toolbox/workflow/compile_test.go` — `wf()` helper
- `toolbox/workflow/runtime_test.go` — `wf()` helper

This creates a disconnect between tests and production code. A test using `"1.0"` envelopes could pass even if `"2.0"` validation fails, giving false confidence.

## Expected Behavior

All test fixtures should use `ContractVersion: "2.0"` to match the active contract generation.

## Actual Behavior

Test fixtures use `ContractVersion: "1.0"` while the contracts package declares `Version = "2.0"`.

## Impact

- Tests may pass against stale contract versions while production code fails.
- Developers may be confused about which contract version is active.
- The conformance suite validates v2, but toolbox tests validate v1 — inconsistent validation.

## Evidence

`contracts/go/v2/common.go` line 8: `const Version = "2.0"`.

Test fixtures across `host_test.go`, `policy_test.go`, `registry_test.go`, `rpc_test.go`, `compile_test.go`, and `runtime_test.go` all use `"1.0"`.

## Suspected Area

`toolbox/` — all test files (`*_test.go`).

## Acceptance Criteria

- [ ] root cause understood
- [ ] bug corrected — all test fixtures use `"2.0"`
- [ ] regression test added where practical
- [ ] related tests pass
- [ ] observable behavior — tests validate against active contract version
- [ ] tests added or updated
- [ ] documentation updated where required

## Recommendation

Update the helper functions in each test file to use `"2.0"`:
- `host_test.go`: `manifest()` and `contract()` helpers
- `policy_test.go`: `manifest()` and `contract()` helpers
- `registry_test.go`: `manifest()` helper
- `rpc_test.go`: `newService()` helper
- `compile_test.go`: `wf()` helper
- `runtime_test.go`: `wf()` helper

## Scope

`toolbox/` — all `*_test.go` files.

## Suggested Phase

Update phase (pre-hardening); targets `ph7-toolbox`.
