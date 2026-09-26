# aa inter-module RPC v1

Modules never import each other's internals. They call each other over an authenticated local socket using JSON-RPC 2.0. This document defines the transport envelope, security, versioning, errors, and the initial method set.

## Transport

- **Framing:** newline-delimited JSON-RPC 2.0 messages; each frame is one JSON object.
- **Addressing:** a per-user named pipe on Windows (`\\.\pipe\aa-<service>-v1`) or a per-user Unix domain socket elsewhere (`$XDG_RUNTIME_DIR/aa-<service>-v1.sock`), created owner-only.
- **Reference transport:** the `aa-obsv` local transport (`hello` / `append` / `replay` / `subscribe`) is the canonical example of an attach-or-own, identity-bound service. Other services follow the same pattern.
- **Deadlines:** every request carries `timeout_ms`. Servers must fail closed on timeout and must not leave partial state.

## Envelope

```json
{
  "jsonrpc": "2.0",
  "id": "rpc_01H...",
  "method": "sifter.route",
  "params": { "contractVersion": "1.0", "kind": "route_request", "id": "req_01H..." },
  "aa": {
    "rpcVersion": "1.0",
    "contractVersion": "1.0",
    "caller": "kernel",
    "callerInstanceId": "inst_01H...",
    "timeoutMs": 30000
  }
}
```

Responses are either `result` or `error`. Errors use a namespaced `code` and a `data` object that may carry a contract error object.

## Security

1. The socket is owner-only (current user). Services verify the peer's user where the OS allows it (Windows pipe server SID, Unix `SO_PEERCRED` / directory ownership).
2. A service binds to one store identity; attaching to a different store is an error (`aa.store_mismatch`).
3. Requests never carry credentials in the body; identity is established by the transport and the `aa.caller`/`aa.callerInstanceId` fields.
4. Capability checks happen at the callee using the caller's execution contract, not by trusting the caller.

## Versioning

- `aa.rpcVersion` is `MAJOR.MINOR`. A callee rejects an unsupported major with `aa.incompatible`.
- `aa.contractVersion` is the contract version of `params`/`result`; additive changes are minor.
- Unknown methods return `aa.method_not_found`; unknown params fields are ignored.

## Error codes

| Code | Meaning |
|---|---|
| `-32700` | Parse error |
| `-32600` | Invalid request |
| `-32601` | Method not found |
| `-32602` | Invalid params |
| `-32603` | Internal error |
| `aa.incompatible` | Unsupported RPC or contract major |
| `aa.store_mismatch` | Attached to a different store identity |
| `aa.unauthorized` | Caller lacks a required capability |
| `aa.budget_exceeded` | Call exceeded its budget |
| `aa.approval_required` | Operation needs human approval |
| `aa.unavailable` | Service or runtime unavailable |
| `aa.conflict` | Optimistic-concurrency or duplicate conflict |

Errors that are retryable set `data.retryable: true`.

## Idempotency

- Every mutating method takes an `idempotencyKey` derived from the request contract `id`. Repeated calls with the same key return the original result.
- The callee records applied keys for at least the retention window of the affected aggregate.

## Methods

### sifter (model/compute routing)

| Method | Params | Result |
|---|---|---|
| `sifter.route` | `route_request` | `route_response` |
| `sifter.generate` | `{tier, messages, budget, idempotencyKey}` | `{text, usage}` |
| `sifter.health` | `{}` | `{available, providers[]}` |

### memory (system of record)

| Method | Params | Result |
|---|---|---|
| `memory.record` | a canonical record (`event`, `issue`, `strategy`, `memory_record`, …) | `{id, revision}` |
| `memory.query` | `{kind, filter, cursor, limit}` | `{items[], cursor}` |
| `memory.rebuild` | `{projectId}` | `{rebuilt, digest}` |

### kernel (control plane)

| Method | Params | Result |
|---|---|---|
| `kernel.submit` | `{objective, projectId, budget?}` | `{projectId, workPackageIds[]}` |
| `kernel.status` | `{projectId}` | project/work-package/assignment states |
| `kernel.approve` | `{gateId, decision, actor}` | `{accepted}` |

### sync (transport)

| Method | Params | Result |
|---|---|---|
| `sync.distribute` | `{workPackageId, nodeId}` | `{accepted, transferId}` |
| `sync.fetchArtifact` | `{artifactId, nodeId}` | `{transferId, hash}` |
| `sync.status` | `{nodeId?}` | peer and transfer state |

### forge (git/github)

| Method | Params | Result |
|---|---|---|
| `forge.createIssue` | `{projectId, issue}` | `{forgeRef}` |
| `forge.openPullRequest` | `{branch, base, title, body}` | `{forgeRef}` |
| `forge.checkpoint` | `{projectId, phase, commit}` | `{forgeRef}` |
| `forge.release` | `{projectId, tag, notes}` | `{forgeRef}` |

### toolbox (tools, workflows)

| Method | Params | Result |
|---|---|---|
| `toolbox.invoke` | `{toolId, input, executionContractId, idempotencyKey}` | `{output}` |
| `toolbox.compileWorkflow` | `workflow` | `{plan}` |
| `toolbox.registry` | `{toolKind?}` | `{tools[]}` |

### obsv (observation)

The observation service uses its own protocol (`hello` / `append` / `replay` / `subscribe`). See `../../obsv/`. It is listed here only for completeness.

## Control plane vs RPC

The RPC surface is **internal** (module to module). The **external** surface for `ui` and `visualizer` is the versioned HTTP/JSON control plane described in `../openapi/control-plane-v1.yaml`. The ui never calls RPC methods directly.
