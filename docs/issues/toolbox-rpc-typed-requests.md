# Toolbox — RPC layer uses map[string]any throughout

**Type:** technical-debt
**Severity:** P1 (high)
**Status:** planned
**Branch:** `dev` (update phase)

## Goal

Replace the `map[string]any` deserialization in the RPC layer with typed request/response structs, so parameter names, types, and required fields are validated at compile time.

## Problem

The entire RPC layer (`toolbox/rpc/rpc.go`) deserializes incoming requests into `map[string]any` and extracts fields manually using helper functions like `stringField()` and `workflowParams()`. This means:

- Parameter names are strings, not compile-checked — typos silently produce wrong behavior.
- Required fields are checked at runtime only.
- Type coercion is manual (`.(string)`, `.(map[string]any)`), risking panics on malformed input.
- There is no schema-driven validation of request shapes.

Example from `invoke()`:

```go
toolID, err := stringField(req.Params, "toolId")
// ...
contractID := ""
if v, ok := req.Params["executionContractId"].(string); ok {
    contractID = v
}
```

## Expected Behavior

The RPC layer should use typed request structs with proper JSON tags, so the compiler catches missing fields, wrong types, and typos.

## Actual Behavior

All request parsing is done through `map[string]any` with manual type assertions and string-based field lookups.

## Impact

- **Fragility:** A single typo in a field name causes silent failure (empty string, wrong behavior).
- **No compile-time safety:** Schema changes require updating manual parsing code in multiple places.
- **Panic risk:** Type assertions like `.(string)` panic on unexpected input types.

## Evidence

`toolbox/rpc/rpc.go`:
- `stringField()` — manual string extraction with error wrapping
- `workflowParams()` — double marshal/unmarshal through `map[string]any`
- `invoke()` — manual field extraction for `toolId`, `executionContractId`, `input`, `idempotencyKey`
- `compileWorkflow()` — manual workflow extraction
- `listRegistry()` — manual `toolKind` extraction

## Suspected Area

`toolbox/rpc/rpc.go` — all handler methods (`invoke`, `compileWorkflow`, `listRegistry`).

## Acceptance Criteria

- [ ] observable behavior — RPC behaves identically (no functional change)
- [ ] tests added or updated
- [ ] error cases handled — malformed requests produce proper JSON-RPC errors
- [ ] documentation updated where required

## Recommendation

Define typed request structs for each method:

```go
type InvokeRequest struct {
    ToolID              string `json:"toolId"`
    ExecutionContractID string `json:"executionContractId"`
    Input               map[string]any `json:"input,omitempty"`
    IdempotencyKey      string `json:"idempotencyKey,omitempty"`
}

type CompileWorkflowRequest struct {
    Workflow Workflow `json:"workflow"`
}

type RegistryRequest struct {
    ToolKind string `json:"toolKind,omitempty"`
}
```

Update `Handle` to dispatch to typed handlers. Keep the `Request` struct for the JSON-RPC envelope but use typed params for each method.

## Scope

`toolbox/rpc/rpc.go` — request parsing and dispatch.

## Suggested Phase

Update phase (pre-hardening); targets `ph7-toolbox`.
