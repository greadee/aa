# ADR-0144 — RPC v1 boundary hardening

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Source** | [issue-impl-sep28 Issue 4](../issues/imp-4-rpc-boundary-hardening.md) |
| **Scope** | update phase · issue-impl-sep28 (Stage 1) |

## Context

The inter-module RPC envelope (`contracts/rpc/rpc-v1.md`) defined envelope,
security, errors, idempotency, and methods, but left end-to-end versioning,
compatibility, capability negotiation, unsupported-version behavior, and
retry/deadline semantics under-specified. The Python `inference` service already
implemented deadline, idempotency, identity, framing, and error mapping; the Go
adapters (`forge`, `toolbox`, `sync`) only checked the RPC major and did not
carry `data.retryable`. The model-routing namespace was also renamed `sifter` →
`inference` within envelope v1.

## Decision

- **The envelope is frozen at major `1`.** The `sifter` → `inference` rename is a service rename **within** v1, not an envelope change; the former `sifter.*` namespace returns `aa.method_not_found`.
- **Contract-version gate.** A callee rejects a present, unsupported contract major (currently `2`) with `aa.incompatible`, exactly as it does an unsupported RPC major. Minor differences are compatible in both directions; an absent version is compatible.
- **Capability negotiation.** A service advertises supported versions and methods (distributed: handshake/`inference.health`; in-process: `Capabilities()`); a caller uses the highest common version and calls only advertised methods. Unadvertised method → `aa.method_not_found`; missing capability → `aa.unauthorized`.
- **Deadlines fail closed.** `aa.timeoutMs` is enforced server-side; expiry cancels the handler, leaves no partial state, and returns `aa.unavailable` with `data.retryable: true`.
- **Retries are conditional and idempotent.** A caller retries only when `data.retryable` is `true`; a mutating retry reuses the same `idempotencyKey`, and reusing a key for a different request is `aa.conflict`.
- **No new network boundary.** Only boundaries with a real process/machine/language split are RPC (`inference`, `obsv` host, `sync`, control-plane API). `forge`, `toolbox`, `kernel`, and `memory` keep transport-agnostic in-process adapters that speak the same envelope, so a future socket host is a transport change, not a contract change.
- **Uniform error data.** Every governed error carries `data.retryable` (`true`/`false`).

## Rationale

- A frozen envelope with a version gate gives forward/backward compatibility without a new envelope.
- Failing closed before dispatch prevents partially applied work on version skew.
- Advertised capabilities and conditional retries let a caller degrade safely instead of guessing.
- Keeping in-process adapters socket-free avoids inventing boundaries (and failure modes) the architecture does not need.

## Alternatives and consequences

- **Bump the envelope for the rename:** rejected — the rename changes a method namespace, not the envelope.
- **Socket every module:** rejected — modules without a process/machine split gain no isolation and only failure surface.
- **Enforce deadlines only at callers:** rejected — a callee must not keep working after the caller's deadline or leave partial state.
- Consequence: the Go adapters now validate the contract major and attach `data.retryable`; the spec carries the boundary audit and the rules; each versioned boundary tests fail-closed behavior.

## Related

- [contracts/rpc/rpc-v1.md](../../contracts/rpc/rpc-v1.md)
- [ISS-IMP-4](../issues/imp-4-rpc-boundary-hardening.md)
- [ADR index](./README.md)
