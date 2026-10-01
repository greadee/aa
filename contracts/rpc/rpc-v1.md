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
  "method": "inference.route",
  "params": { "contractVersion": "2.0", "kind": "route_request", "id": "req_01H..." },
  "aa": {
    "rpcVersion": "1.0",
    "contractVersion": "2.0",
    "caller": "kernel",
    "callerInstanceId": "inst_01H...",
    "timeoutMs": 30000
  }
}
```

Responses are either `result` or `error`. Errors use a namespaced `code` and a `data` object that may carry a contract error object.

## Boundary audit

Inter-module communication is RPC only where a process, machine, or language
boundary actually exists. Logical separation alone does **not** create a network
boundary.

| Boundary | Kind | Versioned on the wire | Enforced at |
|---|---|---|---|
| `inference` (Python) | Distributed (local socket) | `aa.rpcVersion`, `aa.contractVersion` | `runtime/inference/rpc` (server, deadline, idempotency, identity, framing) |
| `obsv` host | Distributed (local socket) | protocol v1 (`hello`) | `obsv/transport` |
| `sync` | Distributed (machine to machine) | RPC envelope + work-package contract | `sync/rpc` |
| Control-plane API | External (HTTP/JSON) | OpenAPI `1.0.0` | `contracts/openapi/control-plane-v1.yaml` |
| `forge`, `toolbox`, `kernel`, `memory` | In-process, transport-agnostic | RPC envelope when hosted | the module's `rpc` package, hosted by the kernel |

In-process adapters expose the same envelope and error set, so adding a socket
host later is a transport change, not a contract change. No socket is created for
a module that does not need one.

## Security

1. The socket is owner-only (current user). Services verify the peer's user where the OS allows it (Windows pipe server SID, Unix `SO_PEERCRED` / directory ownership).
2. A service binds to one store identity; attaching to a different store is an error (`aa.store_mismatch`).
3. Requests never carry credentials in the body; identity is established by the transport and the `aa.caller`/`aa.callerInstanceId` fields.
4. Capability checks happen at the callee using the caller's execution contract, not by trusting the caller.

## Versioning and compatibility

- `aa.rpcVersion` and `aa.contractVersion` are `MAJOR.MINOR`.
- The RPC envelope is **frozen at major `1`**. A callee rejects a different RPC major with `aa.incompatible`.
- A callee rejects a different **contract** major (currently `2`) with `aa.incompatible`. An absent version is treated as compatible.
- **Minor differences are compatible in both directions.** A `1.7`/`2.3` caller is served by a `1.0`/`2.0` callee and vice versa; consumers ignore unknown fields and unknown enum members they do not act on.
- Unknown methods return `aa.method_not_found`; unknown params fields are ignored.
- Unsupported-version behavior **fails closed**: the request is rejected before any handler runs and leaves no partial state.
- **Contracts generation.** `v1` was sunset (ten-issue sprint, Issue 1 / A4); `v2` is the only active generation, so an unsupported contract major is always `aa.incompatible`.
- **Service rename (migration).** The model/compute routing service was renamed
  `sifter` → `inference` (module update `architecture-refactor-1`), so its
  methods are now `inference.*`. The former `sifter.*` namespace is retired and
  returns `aa.method_not_found`. This is a service rename within RPC envelope
  version `1.0`, not an envelope change; it is recorded here because it changes
  the method namespace. Callers must use `inference.*`.

## Capability negotiation

- A service advertises the versions and methods it supports. A distributed service returns them from a handshake (`inference.health` returns `rpcVersion`, `contractVersion`, and `capabilities[]`); an in-process adapter exposes the equivalent `Capabilities()`.
- A caller negotiates before calling: it uses the highest common major/minor and calls **only advertised methods**.
- Calling an unadvertised method returns `aa.method_not_found`; calling an advertised method without the required capability returns `aa.unauthorized`.
- Capability checks happen at the callee using the caller's execution contract, never by trusting the caller's metadata.

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

Every governed error carries `data.retryable` (`true` or `false`). A caller must not retry unless it is `true`; JSON-RPC numeric codes (parse/invalid request/params, method not found, internal) are never retryable unchanged.

## Deadlines, retries, and idempotency

- Every request carries `aa.timeoutMs`. The server **enforces** it: a request that exceeds the deadline is cancelled, fails closed with `aa.unavailable` and `data.retryable: true`, and leaves no partial state.
- A caller retries **only** when `error.data.retryable` is `true`, with bounded backoff. Non-retryable errors (`aa.incompatible`, `aa.unauthorized`, `aa.budget_exceeded`, `aa.approval_required`, `aa.conflict`, invalid params) must not be retried unchanged.
- Every mutating method takes an `idempotencyKey` derived from the request contract `id`. A retry reuses the same key; the callee returns the original result.
- The callee records applied keys for at least the retention window of the affected aggregate. Reusing a key with a **different** request is `aa.conflict`.

## Methods

### inference (model/compute routing; formerly `sifter`)

| Method | Params | Result |
|---|---|---|
| `inference.route` | `route_request` | `route_response` |
| `inference.generate` | `{tier, messages, budget, idempotencyKey}` | `{text, usage}` |
| `inference.health` | `{}` | `{available, providers[]}` |

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
