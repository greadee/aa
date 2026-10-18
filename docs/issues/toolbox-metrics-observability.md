# Toolbox — No metrics or observability

**Type:** feature
**Severity:** P2 (medium)
**Status:** planned
**Branch:** `dev` (update phase)

## Goal

Add metrics collection and structured observability to the toolbox module so invocation patterns, denial rates, and latency are measurable and actionable.

## Problem

The toolbox module has no observability beyond an in-memory audit log (`policy.Engine.log`). There is no:

- Invocation count (per tool, per kind, per contract)
- Denial rate (by capability, by sandbox reason)
- Latency histograms (per provider, per step)
- Workflow execution metrics (steps completed, retries, compensations)
- Export mechanism (prometheus, OpenTelemetry, structured log)

The audit log is in-memory only, lost on process restart, and not queryable.

## Expected Behavior

The toolbox module exposes metrics for:
- Invocation count (total, per tool, per kind)
- Denial count and rate (by capability, by reason)
- Latency (p50, p95, p99 per provider/step)
- Workflow execution (steps completed, retries, compensations, failures)
- Cache hit/miss ratio (idempotency)

These metrics should be exportable via a standard interface (prometheus, OpenTelemetry, or structured logging).

## Actual Behavior

No metrics are collected. The audit log is in-memory only.

## Impact

- **No production visibility:** Cannot detect abuse, performance degradation, or policy misconfigurations.
- **No capacity planning:** No data on invocation patterns or latency distributions.
- **No alerting:** Cannot set up alerts for denial spikes, latency regressions, or failure rates.

## Evidence

- `toolbox/policy/policy.go`: `Engine.log` is `[]AuditEntry` — in-memory, append-only, not exported.
- `toolbox/host/host.go`: `Host.cache` is `map[string]toolbox.Result` — in-memory, no hit/miss tracking.
- `toolbox/workflow/runtime.go`: `Runtime.Run` returns `RunState` — no metrics on step counts, retries, or failures.

## Suspected Area

`toolbox/policy/policy.go`, `toolbox/host/host.go`, `toolbox/workflow/runtime.go`, `toolbox/rpc/rpc.go`.

## Acceptance Criteria

- [ ] observable behavior — existing functionality unchanged
- [ ] tests added or updated
- [ ] error cases handled
- [ ] documentation updated where required

## Recommendation

Define a `Metrics` interface:

```go
type Metrics interface {
    RecordInvocation(toolID ToolID, kind string, latency time.Duration, success bool)
    RecordDenial(toolID ToolID, reason string, capability string)
    RecordWorkflowStep(stepID string, kind string, attempts int, success bool, latency time.Duration)
    RecordCacheHit(key string)
    RecordCacheMiss(key string)
}
```

Wire this into `PolicyEngine`, `Host`, `Runtime`, and `Service`. Provide a no-op implementation by default, and a prometheus/OpenTelemetry implementation as an optional dependency.

## Scope

`toolbox/policy/policy.go`, `toolbox/host/host.go`, `toolbox/workflow/runtime.go`, `toolbox/rpc/rpc.go`.

## Suggested Phase

Update phase (pre-hardening); targets `ph7-toolbox`. May integrate with `obsv` module in a later phase.
