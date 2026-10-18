# Toolbox — Large contract files should be split

**Type:** refactor
**Severity:** P3 (low)
**Status:** planned
**Branch:** `dev` (update phase)

## Goal

Split large contract files in `contracts/go/v2/` into smaller, more navigable units without changing functionality.

## Problem

Several files in `contracts/go/v2/` are large and contain many unrelated types:

- `records.go` — contains `WorkPackage`, `ProjectRecord`, `MemoryRecord`, `ResultEnvelope`
- `event.go` — contains `Event`, `Telemetry`
- `telemetry.go` — contains `Telemetry` (possibly duplicate)

These files mix different conceptual domains (planning, execution, observation, memory) making navigation and maintenance difficult.

## Expected Behavior

Each file should cover a single conceptual domain, making it easy to find and modify related types.

## Actual Behavior

Multiple unrelated types are grouped in single files, making navigation harder.

## Impact

- **Maintenance burden:** Finding related types requires scanning large files.
- **Merge conflicts:** Unrelated changes to different domains in the same file cause unnecessary conflicts.
- **Onboarding friction:** New developers must read large files to understand the contract space.

## Evidence

`contracts/go/v2/` file sizes (approximate):
- `records.go` — ~200 lines, 5+ types from different domains
- `event.go` — ~150 lines, 3+ types
- `telemetry.go` — ~100 lines

## Suspected Area

`contracts/go/v2/` — file organization.

## Acceptance Criteria

- [ ] observable behavior — no functional change
- [ ] tests added or updated — conformance suite still passes
- [ ] error cases handled
- [ ] documentation updated where required

## Recommendation

Split into domain-specific files:

```
contracts/go/v2/
├── common.go          — Envelope, Budget, states (existing)
├── validate.go        — Validators (existing)
├── execution.go       — ExecutionContract, RuntimeSpec, WorkspaceSpec
├── workflow.go        — Workflow, WorkflowStep (existing)
├── routing.go         — RouteRequest, RouteResponse
├── planning.go        — WorkPlan, ExecutionPlan, TaskGraph
├── execution.go       — ExecutionContract, ExecutionPlan (split from routing.go)
├── workpackage.go     — WorkPackage (split from records.go)
├── records.go         — ProjectRecord, MemoryRecord (split from records.go)
├── result.go          — ResultEnvelope (split from records.go)
├── event.go           — Event (existing)
├── telemetry.go       — Telemetry (existing)
├── trace.go           — Trace (existing)
├── tool.go            — ToolManifest, SandboxSpec (existing)
├── rolespec.go        — RoleSpec (existing)
├── modelspec.go       — ModelSpec (existing)
├── teamspec.go        — TeamSpec (existing)
├── routine.go         — Routine (existing)
├── issue.go           — Issue (existing)
├── strategy.go        — Strategy (existing)
└── records.go         — WorkPackage, ProjectRecord, MemoryRecord, ResultEnvelope (split)
```

Update `conformance_test.go` to handle the new file structure (no functional change).

## Scope

`contracts/go/v2/` — file organization only.

## Suggested Phase

Update phase (pre-hardening); targets `ph7-toolbox` (contracts dependency).
