# sifter — Module Update Plan: production RPC wiring

> Authored at the **start** of the module update, on branch `ph4-sifter` (the phase that owns `aa-sifter`).
> Sub-problem: `ISS-SIFTER-1` · Parent umbrella: `ISS-TRACE-LOOP` (`docs/issues/trace-learning-substrate.md`)

## Objective

Host the existing transport-agnostic `SifterService` over the real aa inter-module RPC v1 transport — a per-user local socket (Windows named pipe / Unix domain socket) carrying newline-delimited JSON-RPC 2.0 — and harden the boundary so the kernel can reach `sifter.route` and `sifter.generate` as a supervised production service. Phase 4 delivered the in-process service and the redaction chokepoint but explicitly deferred hosting and the kernel adapter (phase 4 deviations #6, follow-up "Host the RPC service…"). Production wiring is the remaining half of the sifter's governance contract.

This plan is planning only: **no code is implemented in this module update yet**. Implementation proceeds slice by slice after review.

## Starting State

- Starting ref: `main @ 79b495d`; branch `ph4-sifter` fast-forwarded to it.
- Available:
  - `aa_sifter.rpc.SifterService.handle` — a transport-agnostic JSON-RPC 2.0 dispatcher implementing `sifter.route`, `sifter.generate`, `sifter.health`, and the additive `sifter.recommend`.
  - `aa_sifter.rpc.envelope` — message helpers, the v1 error codes, and `is_compatible` (major-version rejection with `aa.incompatible`).
  - `aa_sifter.contracts` — the `aa_contracts` bridge that validates `route_request`/`route_response` and builds `memory_record` candidates.
  - The single expert-egress redaction chokepoint (`_call_tier`) and its spy-provider tests.
  - The v1 transport spec: `contracts/rpc/rpc-v1.md` (framing, addressing, security, deadlines, errors, idempotency, method set).
  - The `obsv` local transport (`hello`/`append`/`replay`/`subscribe`) as the canonical attach-or-own, identity-bound service.
  - The kernel's `runtime.Adapter` seam (`kernel/runtime/runtime.go`) and its scripted fake; the kernel reaches `sifter` over RPC, never by import.
- Missing:
  - Any transport binding: no socket/pipe server, no framing reader/writer, no client.
  - Server lifecycle: no `aa-sifter serve` command, readiness, logging, or graceful shutdown.
  - Peer identity and store-identity binding; deadline enforcement; the full v1 error mapping; durable idempotency.
  - Full JSON Schema validation at the boundary (only the Python binding is used today).
  - The kernel-side production runtime adapter that calls the hosted service.
- Known constraints:
  - The sifter must not hold project state or orchestrate; it answers requests.
  - No cloud egress without redaction; no default test may touch the network or a real provider.
  - The sifter imports no aa module except the optional `aa_contracts` binding (boundary rule).
  - The service shape is fixed; the transport must not change `SifterService.handle`'s contract.
  - CI runs on Linux; Windows named-pipe behavior is specified but may be exercised only by unit-level abstraction.

## Scope

### In Scope
- NDJSON JSON-RPC 2.0 framing: bounded frame reader/writer, parse and invalid-request errors.
- A per-user local socket listener (Unix domain socket; Windows named pipe) with owner-only creation and attach-or-own, single-owner semantics.
- Peer identity verification (`SO_PEERCRED` / directory ownership; Windows pipe SID) and store-identity binding (`aa.store_mismatch`).
- Deadline handling from `aa.timeoutMs`, fail-closed cancellation, and no partial state.
- A server loop that dispatches to `SifterService.handle`, plus readiness/health and graceful shutdown.
- The full v1 error mapping, including `aa.unauthorized`, `aa.budget_exceeded`, `aa.approval_required`, and `aa.unavailable` with `data.retryable`.
- Durable idempotency for mutating methods across the retention window.
- A typed client and the seam the kernel's production runtime adapter will use.
- An `aa-sifter serve` entry point and service configuration.
- Full JSON Schema validation at the RPC boundary using the `aa_contracts` schemas.
- Security tests proving redaction and capability behavior over the transport.

### Out of Scope
- Kernel-side wiring of the production runtime adapter itself (a dependent sub-problem on the kernel branch).
- A memory-backed `MemorySink` writer (a separate `aa-memory` integration).
- Live, paid provider smoke (stays manual and opt-in).
- Learned routing or model-in-the-loop decisions (routing stays deterministic).
- The external control-plane HTTP surface (that belongs to `console`).
- The obsv observation transport (already delivered by `ph3-kernel`).

## Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P4U-001 | The production transport is NDJSON JSON-RPC 2.0 over a per-user local socket, exactly as `contracts/rpc/rpc-v1.md` | One cross-module transport spec; the service stays transport-agnostic |
| ADR-P4U-002 | One owner per socket; a second server attaches or fails rather than binding twice | Mirrors the `obsv` attach-or-own precedent and prevents split-brain |
| ADR-P4U-003 | Owner-only socket permissions and verified peer identity; credentials never travel in the body | Local least privilege; identity comes from the transport |
| ADR-P4U-004 | Deadlines are enforced server-side and fail closed, never leaving partial state | Prevents hung kernel calls and half-applied work |
| ADR-P4U-005 | Validate at the full JSON Schema boundary, not only the Python bindings | The binding can drift; the schema is the contract |
| ADR-P4U-006 | Redaction stays a single chokepoint and is proven over the RPC path | Cloud egress safety must hold no matter the entry point |
| ADR-P4U-007 | Idempotency keys are recorded for the retention window of the affected aggregate | The spec requires replay-safe mutating calls |
| ADR-P4U-008 | The kernel reaches sifter through a typed client behind the existing runtime adapter seam; the fake stays for tests | Model-agnostic kernel; end-to-end tests without a model |

## Slices

Each slice maps to exactly one commit and is authored after this plan is reviewed. Slice 1 is delivered by the pull request carrying this plan.

| Slice | Goal | Commit message |
|---|---|---|
| 1 | Issue documentation and this plan | `add ph4 sifter rpc wiring plan and issue documentation` |
| 2 | NDJSON framing reader/writer and framing errors | `add sifter rpc framing` |
| 3 | Local socket listener with owner-only creation and attach-or-own | `add sifter rpc socket listener` |
| 4 | Peer identity and store-identity binding | `add sifter rpc peer and store identity` |
| 5 | Deadlines, cancellation, and fail-closed dispatch | `add sifter rpc deadlines and cancellation` |
| 6 | Full v1 error mapping and retryable flags | `add sifter rpc error mapping` |
| 7 | Durable idempotency window | `add sifter rpc idempotency window` |
| 8 | Server loop, readiness, and graceful shutdown | `add sifter rpc server loop` |
| 9 | Typed client and kernel runtime-adapter seam | `add sifter rpc client and runtime seam` |
| 10 | `aa-sifter serve` entry point and configuration | `add sifter serve command` |
| 11 | Full JSON Schema validation at the boundary | `add sifter rpc schema validation` |
| 12 | Security and redaction tests over the transport | `add sifter rpc security tests` |
| 13 | Module update summary | `add ph4 sifter rpc wiring summary` |
| 14 | Link the module update PR | `link ph4 sifter rpc wiring pull request` |

## Required End State

- [x] `aa-sifter serve` binds the v1 local socket and serves `sifter.route`, `sifter.generate`, `sifter.health`, and `sifter.recommend` over NDJSON.
- [x] The socket is owner-only; peer identity and store identity are verified; a mismatch returns `aa.store_mismatch`.
- [x] Requests honour `aa.timeoutMs`; expired requests fail closed with `aa.unavailable` and no partial state.
- [x] Errors use the full v1 code set; retryable errors set `data.retryable: true`.
- [x] Idempotent repeats of a mutating call return the original result for the retention window.
- [x] Boundary inputs and outputs validate against the `aa_contracts` JSON Schema.
- [x] A spy provider proves no secret crosses the transport on any expert-tier call.
- [x] A typed client call is exercised by a socket integration test; the default suite makes no network calls.
- [x] `ruff format --check`, `ruff check`, `mypy`, `pytest`, and the Go checks remain green.

## Exit Criteria

- [x] The hosted service answers the v1 method set over the local socket with verified identity and enforced deadlines.
- [x] The kernel can reach the service through the typed client behind the runtime adapter seam.
- [x] Redaction and capability behavior are proven over the transport.
- [x] `sifter` still imports no aa module except `aa_contracts`; boundary check passes.
- [x] A module update PR is opened on `ph4-sifter` into `main`.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | frame parsing/bounds, error mapping, idempotency window, deadline expiry |
| contract | `route_request`/`route_response` and `memory_record` validate against the schemas at the boundary |
| integration | start the server on a temp socket; client round-trips `route`, `generate`, `health`, `recommend` |
| security | owner-only permissions, peer/store identity mismatch, no secret on the wire, `local_only` yields zero expert traffic |
| failure | malformed frames, oversized frames, unknown methods, timeout, provider unavailability |
| boundary | `sifter` imports no aa module except the optional `aa_contracts` binding |
| live | optional, manual, paid provider smoke (never in default CI) |

## Dependencies And Follow-Up

- **Kernel production runtime adapter** (dependent sub-problem on `ph3-kernel`): replaces the fake with the typed client from slice 9.
- **Memory-backed `MemorySink`** (separate `aa-memory` integration): persists the `memory_record` candidates the sifter already emits.
- **Live provider smoke**: manual and paid; remains out of the default suite.
