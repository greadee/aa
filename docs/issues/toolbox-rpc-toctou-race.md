# Toolbox — RPC TOCTOU race condition in Service.Handle

**Type:** bug
**Severity:** P0 (critical)
**Status:** planned
**Branch:** `dev` (update phase)

## Goal

Fix the TOCTOU (time-of-check to time-of-use) race condition in `Service.Handle` that allows concurrent identical requests to both execute the underlying provider, defeating the idempotency cache.

## Problem

In `toolbox/rpc/rpc.go`, the `Handle` method checks the idempotency cache, unlocks, executes the handler, then locks again to store the result (lines 124–131 and 149–153). Between the unlock and the re-lock, a second concurrent request with the same idempotency key can pass the cache check and also execute the provider. This defeats the idempotency guarantee.

The race is on the `Service.cache` map protected by `sync.Mutex`:

```go
// Lines 124-131: check cache, unlock, proceed
s.mu.Lock()
if cached, found := s.cache[req.Method+"|"+key]; found {
    s.mu.Unlock()
    return Response{...}
}
s.mu.Unlock()  // <-- RACE WINDOW: another goroutine passes here

// Lines 149-153: re-lock to store result
s.mu.Lock()
s.cache[req.Method+"|"+key] = result
s.mu.Unlock()
```

## Expected Behavior

Concurrent requests with the same idempotency key should execute the provider exactly once. The second (and subsequent) concurrent requests should wait for the first to complete, then serve the cached result.

## Actual Behavior

Concurrent identical requests can both pass the cache check and both execute the provider, producing duplicate side effects.

## Impact

- **Security/correctness failure:** Idempotency is a core guarantee of the toolbox invocation model. A TOCTOU race means tools with side effects (file writes, network calls, state mutations) can execute multiple times despite an explicit idempotency key.
- **Data integrity risk:** Any tool performing mutations (e.g., `computeruse` actions, file system operations) could produce inconsistent state.

## Evidence

`toolbox/rpc/rpc.go`, `Service.Handle` method, lines 124–131 (cache read) and 149–153 (cache write). The pattern is a classic TOCTOU: the check and the update are not atomic.

## Suspected Area

`toolbox/rpc/rpc.go` — `Service.Handle` method, idempotency cache logic.

## Acceptance Criteria

- [ ] root cause understood
- [ ] race condition corrected
- [ ] regression test added (concurrent identical requests)
- [ ] related tests pass
- [ ] observable behavior — concurrent identical requests execute provider exactly once
- [ ] tests added or updated
- [ ] error cases handled

## Recommendation

Replace the check-then-store pattern with one of:
1. **Pending-flag pattern:** When a request with a key is first seen, store a `pending` entry (with a `sync.WaitGroup` or channel). Subsequent requests with the same key wait on that entry, then serve the result.
2. **Serialize by key:** Use a per-key mutex or channel to serialize requests with the same idempotency key.

Option 1 is preferred as it avoids holding locks during long-running provider execution.

## Scope

`toolbox/rpc/rpc.go` — `Service.Handle` method, idempotency cache.

## Suggested Phase

Update phase (pre-hardening); targets `ph7-toolbox`.
