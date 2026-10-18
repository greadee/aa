# Toolbox — RPC error responses lack structured data

**Type:** technical-debt
**Severity:** P2 (medium)
**Status:** planned
**Branch:** `dev` (update phase)

## Goal

Add structured diagnostic data to RPC error responses so callers can distinguish error types programmatically (not just by message parsing).

## Problem

The `errResponseFrom` function in `toolbox/rpc/rpc.go` maps toolbox errors to JSON-RPC error codes, but the `data` field is sparse:

```go
func errResponseFrom(id any, err error) Response {
    switch {
    case errors.Is(err, toolbox.ErrInvalid), errors.Is(err, toolbox.ErrCycle):
        return errResponseData(id, CodeInvalidParams, err.Error(), retryable(false))
    case errors.Is(err, toolbox.ErrNotFound):
        return errResponseData(id, CodeNotFound, err.Error(), retryable(false))
    case errors.Is(err, toolbox.ErrDenied), errors.Is(err, toolbox.ErrSandbox), errors.Is(err, toolbox.ErrApproval):
        return errResponseData(id, CodeUnauthorized, err.Error(), retryable(false))
    case errors.Is(err, toolbox.ErrNoProvider):
        return errResponseData(id, CodeUnavailable, err.Error(), retryable(true))
    default:
        return errResponseData(id, CodeInternalError, err.Error(), retryable(false))
    }
}
```

The `data` field only contains `{"retryable": bool}`. There's no:
- Error type/category for programmatic handling
- Original toolbox error name
- Tool ID (when available)
- Execution contract ID (when available)

## Expected Behavior

Error responses include structured diagnostic data: error category, toolbox error name, tool ID, contract ID, and retryability.

## Actual Behavior

Error responses only carry `retryable` in the `data` field. Callers must parse error messages to determine the error type.

## Impact

- **Fragile error handling:** Callers must parse error messages (strings) to determine error types.
- **No programmatic categorization:** Can't distinguish "sanbox denied" from "capability denied" without string parsing.
- **Missing context:** Tool ID and contract ID are available in the request but not included in errors.

## Evidence

`toolbox/rpc/rpc.go` lines 290–303: `errResponseFrom` maps errors to codes but `data` only contains `retryable`.

## Suspected Area

`toolbox/rpc/rpc.go` — `errResponseFrom` function.

## Acceptance Criteria

- [ ] observable behavior — error messages unchanged (backward compatible)
- [ ] tests added or updated
- [ ] error cases handled
- [ ] documentation updated where required

## Recommendation

Enhance `errResponseData` to include structured fields:

```go
type ErrorData struct {
    Retryable        bool   `json:"retryable"`
    ToolboxErrorName string `json:"toolboxError,omitempty"`
    ToolID           string `json:"toolId,omitempty"`
    ContractID       string `json:"executionContractId,omitempty"`
}
```

Pass the toolbox error name, tool ID, and contract ID from each handler to `errResponseFrom`.

## Scope

`toolbox/rpc/rpc.go` — `errResponseFrom`, `errResponseData`, and all handler methods.

## Suggested Phase

Update phase (pre-hardening); targets `ph7-toolbox`.
