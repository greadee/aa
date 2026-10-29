# Toolbox metrics

`toolbox.Metrics` is the instrumentation seam for tool invocations, policy
denials, workflow steps, host cache activity, and RPC calls. The default is
`toolbox.NoopMetrics`, so enabling metrics never changes behavior.

## Implementations

| Type | Use |
|---|---|
| `toolbox.NoopMetrics` | Default. Discards every observation. |
| `toolbox.MemMetrics` | In-process counters with `Snapshot()`. Intended for tests and local diagnostics; it is not an external exporter. |
| `toolbox.LogMetrics` | Structured JSON lines written to any `io.Writer`. Use this as the external exporter (log shipper, Promtail, Fluent Bit, OTel collector with a JSON receiver). |

An embedder may implement `toolbox.Metrics` directly. Implementations must be
safe for concurrent use.

## Configuration

The composed `app` daemon keeps the no-op default. Set
`AA_TOOLBOX_METRICS=log` to emit structured JSON to stderr:

```sh
AA_TOOLBOX_METRICS=log aa serve
```

`toolbox.LogMetrics` emits one object per observation:

```json
{"capability":"write_workspace","event":"toolbox.denial","reason":"missing_capability","toolId":"tool_screen"}
{"event":"toolbox.invocation","kind":"tool","latencyMs":12,"success":true,"toolId":"tool_screen"}
{"event":"toolbox.cache_hit","key":"tool_screen:ec_1:k1"}
{"attempts":2,"event":"toolbox.workflow_step","kind":"tool","latencyMs":5,"stepId":"step_1","success":false}
```

The event names are stable: `toolbox.invocation`, `toolbox.denial`,
`toolbox.workflow_step`, `toolbox.cache_hit`, `toolbox.cache_miss`,
`toolbox.rpc`, `toolbox.rpc_error`.

## Snapshot shape

`MemMetrics.Snapshot()` returns `MetricsSnapshot`. Counters are lower-camel JSON
keys: `invocations`, `failures`, `denials`, `workflowSteps`, `cacheHits`,
`cacheMisses`, `rpcRequests`, `avgLatencyMs`, plus per-label maps such as
`denialsByReason`, `denialsByCapability`, and `rpcErrorsByCategory`.

## Compatibility

The former `toolbox/rpc.Metrics` and `toolbox/rpc.MetricsSnapshot` names remain
as aliases, but this is not a full source or wire compatibility guarantee:

- Snapshot JSON keys changed from Go field names (`Invocations`, `Denials`,
  `AvgLatencyMS`) to lower-camel tags (`invocations`, `denials`,
  `avgLatencyMs`).
- `MetricsSnapshot` gained fields, so unkeyed struct literals no longer compile.
- The `toolbox.Metrics` interface gained `RecordRPC` and `RecordRPCError`, so
  external implementations must add both methods.

These APIs were introduced during the same unreleased sprint, so no external
consumers are assumed. New code should use `toolbox.MemMetrics` or
`toolbox.LogMetrics` directly.

## Tests

`toolbox/metrics_test.go` covers the zero value, snapshot JSON keys, counters,
denial/capability labels, workflow attempts, and the log exporter.
