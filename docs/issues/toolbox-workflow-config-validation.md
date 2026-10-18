# Toolbox — Untyped WorkflowStep.Config

**Type:** technical-debt
**Severity:** P2 (medium)
**Status:** planned
**Branch:** `dev` (update phase)

## Goal

Add validation for `WorkflowStep.Config` per step kind, so step-specific configuration is validated at compile time or at least at compile time, not at runtime.

## Problem

`WorkflowStep.Config` is declared as `map[string]any` in the contract:

```go
type WorkflowStep struct {
    // ...
    Config    map[string]any `json:"config,omitempty"`
}
```

The workflow compiler (`workflow.Compile`) validates step IDs, kinds, dependencies, retries, gates, and tool IDs — but never validates the `Config` field. This means:

- Step-specific config is silently accepted with any shape.
- Typos in config keys are not caught.
- Required config fields per step kind are not enforced.
- No schema validation for step-specific configurations.

## Expected Behavior

The workflow compiler should validate `Config` per step kind, rejecting invalid or missing required config fields.

## Actual Behavior

`Config` is accepted as-is with no validation.

## Impact

- **Silent failures:** Misconfigured steps execute with wrong/missing config and fail at runtime.
- **No schema enforcement:** Step-specific config schemas (inputSchema, outputSchema on the manifest) are not cross-checked against step config.

## Evidence

`toolbox/workflow/compile.go`: `Compile()` validates step IDs, kinds, dependencies, retries, gates, tool IDs — but never touches `step.Config`.

`contracts/go/v2/workflow.go`: `WorkflowStep.Config` is `map[string]any` with no validation in `Validate()`.

## Suspected Area

`toolbox/workflow/compile.go` — `Compile` function; `contracts/go/v2/workflow.go` — `WorkflowStep.Validate`.

## Acceptance Criteria

- [ ] observable behavior — valid workflows behave identically
- [ ] tests added or updated
- [ ] error cases handled — invalid config produces ErrInvalid
- [ ] documentation updated where required

## Recommendation

Add per-step-kind config validation in `Compile`:

```go
func validateStepConfig(step toolbox.WorkflowStep) error {
    if step.Config == nil || len(step.Config) == 0 {
        return nil // optional, no config needed
    }
    switch step.Kind {
    case "tool":
        return validateToolConfig(step)
    case "parallel":
        return validateParallelConfig(step)
    case "conditional":
        return validateConditionalConfig(step)
    default:
        return nil // gate, task, human_approval: config is optional
    }
}
```

Call this from `Compile` for each step after existing validations.

## Scope

`toolbox/workflow/compile.go` — `Compile` function; `contracts/go/v2/workflow.go` — `WorkflowStep.Validate`.

## Suggested Phase

Update phase (pre-hardening); targets `ph7-toolbox`.
