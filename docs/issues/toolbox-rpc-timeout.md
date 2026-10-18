# Toolbox — No timeout honored in RPC handler

**Type:** bug
**Severity:** P2 (medium)
**Status:** planned
**Branch:** `dev` (update phase)

## Goal

Honor the `TimeoutMS` field from the `AA` request metadata in `Service.Handle`, so long-running provider calls are bounded and don't block indefinitely.

## Problem

The `AA` struct on every RPC request carries a `TimeoutMS` field:

```go
type AA struct {
    RPCVersion      string `json:"rpcVersion,omitempty"`
    ContractVersion string `json:"contractVersion,omitempty"`
    Caller          string `json:"json:"caller,omitempty"`
    TimeoutMS       int    `json:"timeoutMs,omitempty"`
}
```

But `Handle` never reads or uses `TimeoutMS`. Requests can execute indefinitely, potentially blocking the RPC server's goroutine pool.

## Expected Behavior

Requests with a `TimeoutMS` value should be bounded: if the provider execution exceeds the timeout, the request should return a timeout error.

## Actual Behavior

`TimeoutMS` is parsed from the request but never used. Provider execution runs unbounded.

## Impact

- **Resource exhaustion:** A slow or hung provider can hold an RPC handler goroutine indefinitely.
- **No backpressure:** Clients have no way to request a timeout; the server ignores their request.
- **DoS risk:** An attacker could send many long-running requests and exhaust server resources.

## Evidence

`toolbox/rpc/rpc.go` line 52: `TimeoutMS int` declared but never referenced in `Handle` or any handler.

## Suspected Area

`toolbox/rpc/rpc.go` — `Service.Handle` method.

## Acceptance Criteria

- [ ] observable behavior — requests with TimeoutMS are bounded
- [ ] tests added or updated
- [ ] error cases handled — timeout returns proper JSON-RPC error
- [ ] documentation updated where required

## Recommendation

In `Handle`, after resolving the contract, create a context with `context.WithTimeout`:

```go
var ctx context.Context
if req.AA != nil && req.AA.TimeoutMS > 0 {
    var cancel context.CancelFunc
    ctx, cancel = context.WithTimeout(context.Background(), time.Duration(req.AA.TimeoutMS)*time.Millisecond)
    defer cancel()
} else {
    ctx = context.Background()
}
```

Pass `ctx` to the appropriate handler. Return a timeout error code when `ctx.Err() == context.DeadlineExceeded`.

## Scope

`toolbox/rpc/rpc.go` — `Service.Handle` method.

## Suggested Phase

Update phase (pre-hardening); targets `ph7-toolbox`.
