# Toolbox — Python/TypeScript binding parity review

**Type:** audit
**Severity:** P3 (low)
**Status:** planned
**Branch:** `dev` (update phase)

## Goal

Review the Python and TypeScript contract bindings for parity with the Go bindings, ensuring all v2 contract types, validators, and test fixtures are represented.

## Problem

The contracts module has Go, TypeScript, and Python bindings. The Go bindings are well-tested (conformance suite, 20+ fixture types). The Python and TypeScript bindings have not been reviewed for parity:

- **TypeScript** (`contracts/typescript/v2/index.ts`): Check if all v2 types are exported.
- **Python** (`contracts/python/aa_contracts/v2.py`): Check if all v2 types and validators are present.

The conformance suite (`contracts/go/v2/conformance_test.go`) decodes 20+ fixture kinds into Go types. If Python/TypeScript bindings are missing types, cross-language consumers will fail silently.

## Expected Behavior

All 20+ contract types from the Go bindings are represented in the Python and TypeScript bindings, with equivalent validation logic.

## Actual Behavior

Parity has not been verified. Some types may be missing or have different validation logic.

## Impact

- **Cross-language inconsistency:** A type valid in Go may be missing or invalid in Python/TypeScript.
- **Silent failures:** Consumers using Python/TypeScript may not have all types available.
- **Conformance gap:** The conformance suite only tests Go bindings.

## Evidence

- `contracts/go/v2/conformance_test.go` decodes 20+ kinds: event, work_package, task_graph, execution_contract, result_envelope, telemetry, trace, project_record, memory_record, issue, strategy, route_request, route_response, tool_manifest, workflow, role_spec, model_spec, team_spec, work_plan, execution_plan, routine.
- `contracts/typescript/v2/index.ts` — needs review.
- `contracts/python/aa_contracts/v2.py` — needs review.

## Suspected Area

`contracts/typescript/v2/`, `contracts/python/aa_contracts/`.

## Acceptance Criteria

- [ ] observable behavior — no functional change to Go bindings
- [ ] tests added or updated — conformance suite extended to Python/TypeScript
- [ ] error cases handled
- [ ] documentation updated where required

## Recommendation

1. Audit `contracts/typescript/v2/index.ts` against `conformance_test.go`'s `decodeByKind` switch.
2. Audit `contracts/python/aa_contracts/v2.py` against the same.
3. For each missing type, add it to the binding.
4. For each type with different validation, align the logic.
5. Extend the conformance suite to test Python/TypeScript round-trips.

## Scope

`contracts/typescript/v2/`, `contracts/python/aa_contracts/`.

## Suggested Phase

Update phase (pre-hardening); targets `ph7-toolbox` (contracts dependency).
