# sifter — Module Update Summary: production RPC wiring

> Authored at the **end** of the module update, on branch `ph4-sifter`. Derived from [plan.md](plan.md).
> Phase PR: [#16](https://github.com/greadee/aa/pull/16)
> Sub-problem: `ISS-SIFTER-1` · Parent umbrella: `ISS-TRACE-LOOP`

## Delivered

| Slice | Status | Commit message | Notes |
|---|---|---|---|
| Issue documentation and module plan | Complete | `add ph4 sifter rpc wiring plan and issue documentation` | `docs/issues/`, this plan |
| NDJSON framing | Complete | `add sifter rpc framing` | `rpc/framing.py` |
| Local socket listener | Complete | `add sifter rpc socket listener` | `rpc/endpoint.py`, `rpc/listener.py` |
| Peer and store identity | Complete | `add sifter rpc peer and store identity` | `rpc/identity.py` |
| Deadlines and cancellation | Complete | `add sifter rpc deadlines and cancellation` | `rpc/deadline.py` |
| Error mapping | Complete | `add sifter rpc error mapping` | `rpc/errors.py`, `rpc/envelope.RpcError` |
| Idempotency window | Complete | `add sifter rpc idempotency window` | `rpc/idempotency.py` |
| Server loop | Complete | `add sifter rpc server loop` | `rpc/server.py` |
| Client and runtime seam | Complete | `add sifter rpc client and runtime seam` | `rpc/client.py` |
| `serve` command | Complete | `add sifter serve command` | `cli/serve_commands.py`, `cli/main.py` |
| Schema validation | Complete | `add sifter rpc schema validation` | `contract_schema.py`, bundled `schemas/v1/` |
| Security tests | Complete | `add sifter rpc security tests` | `tests/unit/test_rpc_security.py` |
| Module update summary | Complete | `add ph4 sifter rpc wiring summary` | this file |
| Phase PR link | Complete | `link ph4 sifter rpc wiring pull request` | follow-up commit |

## What was added

- `rpc/framing` — newline-delimited JSON-RPC 2.0 framing: bounded incremental `FrameReader`, `encode_frame`/`decode_frame`, and `FramingError` carrying the parse/invalid-request code.
- `rpc/endpoint` — per-user addressing per `contracts/rpc/rpc-v1.md`: `$XDG_RUNTIME_DIR/aa-sifter-v1.sock` (per-user temp fallback) and the Windows named-pipe form.
- `rpc/listener` — an asyncio owner-only Unix socket listener with attach-or-own: a live owner means the second server attaches instead of binding twice; a stale socket file is recovered; only the owner unlinks it.
- `rpc/identity` — `SO_PEERCRED`/`getpeereid` peer verification (owner-only directory as the fallback gate) and store-identity binding in the additive `aa.storeId`, rejecting a foreign peer (`aa.unauthorized`) or store (`aa.store_mismatch`).
- `rpc/deadline` — server-side `aa.timeoutMs` enforcement: an expired request is cancelled and answered with `aa.unavailable` and `data.retryable`, never leaving partial state.
- `rpc/errors` — the full v1 error vocabulary: `ContractError`, `BudgetExceeded`, `ApprovalRequiredError`, `ProviderError` status mapping, `ConflictError`, `UnavailableError`, and coded errors (`IdentityError`/`DeadlineError`/`FramingError`), with `data.retryable` where applicable.
- `rpc/idempotency` — a content-fingerprinted, TTL-bounded, durable JSONL `IdempotencyStore`: identical repeats replay, a reused key with a different request raises `aa.conflict`, and expired entries are evicted and compacted.
- `rpc/server` — the NDJSON server loop: read frames, dispatch under deadline, write one framed response per request, report readiness, and shut down gracefully (stop accepting, drain in-flight, cancel the rest).
- `rpc/client` — `SifterClient`, a typed, serialized client that attaches caller identity and deadline, plus the `RuntimeGateway` Protocol seam the kernel's production runtime adapter is written against.
- `cli/serve_commands` — `aa-sifter serve`: assembles the service (store identity, durable idempotency), binds and reports readiness, refuses to start when already served, and handles SIGINT/SIGTERM.
- `contract_schema` + bundled `schemas/v1/` — a standard-library draft 2020-12 subset validator applied at the boundary independently of the generated `aa_contracts` binding, with `common`, `route`, and `memory-record` schemas bundled and a drift test.

## What this changes about the module and the app

- `aa-sifter` is no longer in-process only. It can host `sifter.route`, `sifter.generate`, `sifter.health`, and `sifter.recommend` over the real aa inter-module RPC v1 transport, behind verified identity, enforced deadlines, the full error set, and durable idempotency.
- The kernel can reach the service through a typed client behind the existing `runtime.Adapter` seam; the scripted fake remains test-only. Wiring the kernel-side adapter is the dependent sub-problem recorded in the plan.
- Contract validation no longer depends on the generated binding being importable: the bundled JSON Schema is authoritative at the boundary, and a previously latent schema violation (missing `producedAt`) was fixed in the emitted `memory_record` provenance.
- No new runtime dependencies; the sifter still imports no other aa module except the optional `aa_contracts` binding.

## Is this part of a larger change?

Yes. This is the sifter sub-problem (`ISS-SIFTER-1`) of the umbrella change `ISS-TRACE-LOOP` (the observation → trace → learning substrate was under-delivered; phase 4 deviation #6). It closes the "Host the RPC service over the named pipe / Unix socket transport" follow-up. It depends at the substrate level on `ISS-TRACE-1` (`ph1-contracts`) and `ISS-TRACE-2` (`ph2-memory`) and unblocks the learning sub-issue `ISS-LEARN-1` (`ph9-joblearn`), whose production baseline is the governed `sifter` recommender reached through this boundary.

## Validation

| Gate | Result | Evidence |
|---|---|---|
| `go` checks | N/A | module update is Python-only |
| `ruff format --check` | Pass | 158 files clean |
| `ruff check` | Pass | clean |
| `mypy` | Pass | 79 source files, no issues |
| `pytest` | Pass | 528 tests, with and without `aa_contracts` on the path |
| Transport round-trip | Pass | `sifter.route`/`generate`/`health`/`recommend` over a real socket |
| Deadlines | Pass | expired requests fail closed with `aa.unavailable`, no partial idempotency state |
| Idempotency | Pass | replay across instances from the durable log; conflict on reuse |
| Identity | Pass | foreign peer never reaches the service; store mismatch is `aa.store_mismatch` |
| Security | Pass | owner-only socket; expert egress redacts secrets before the provider; `local_only` yields zero expert traffic |
| Boundaries | Pass | `sifter` imports no aa module except the optional `aa_contracts` binding |
| Default suite offline | Pass | no network calls in the default suite |
| Docs link check | Pending | runs on CI after push |

## Decisions affirmed

| # | Decision | Outcome |
|---|---|---|
| ADR-P4U-001 | NDJSON JSON-RPC 2.0 over a per-user local socket | Affirmed — framing and listener match `contracts/rpc/rpc-v1.md` |
| ADR-P4U-002 | One owner per socket; attach or fail rather than bind twice | Affirmed — `Ownership.OWNED`/`ATTACHED` |
| ADR-P4U-003 | Owner-only socket and verified peer identity; credentials never in the body | Affirmed — `SO_PEERCRED`/`getpeereid` with the directory-ownership fallback |
| ADR-P4U-004 | Server-side deadlines, fail closed, no partial state | Affirmed — cancelled handler, `aa.unavailable` + retryable |
| ADR-P4U-005 | Validate the full JSON Schema at the boundary | Affirmed — bundled schemas, binding applied additionally when importable |
| ADR-P4U-006 | Redaction stays a single chokepoint, proven over the RPC path | Affirmed — spy provider sees no secret on an expert-tier call over the transport |
| ADR-P4U-007 | Idempotency keys recorded for the affected aggregate's retention window | Affirmed — bounded window with eviction and compaction |
| ADR-P4U-008 | Kernel reaches sifter through a typed client behind the runtime-adapter seam | Affirmed — `SifterClient` behind `RuntimeGateway`; fake stays test-only |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | Windows named pipe and Unix domain socket | Unix domain socket implemented; the named pipe is specified and returns a clear `NotImplementedError` from its platform backend | CI runs Linux; the endpoint and identity shapes are specified, the Windows SID backend is a follow-up |
| 2 | "Full JSON Schema validation" | A standard-library draft 2020-12 subset validator plus the three schemas the boundary uses, rather than a new `jsonschema` dependency | Keeps the default suite offline and dependency-free; unsupported schema keywords fail closed instead of silently passing |
| 3 | Kernel-side production runtime adapter | Client and seam only; the kernel adapter remains a dependent sub-problem on `ph3-kernel` | Out of scope per the plan's scope boundary |
| 4 | Emitted `memory_record` provenance | Added the required `producedAt` | Boundary schema validation surfaced the latent contract violation |

## Follow-Up

- Kernel production runtime adapter: replace the scripted fake with the typed client (dependent sub-problem on `ph3-kernel`).
- Memory-backed `MemorySink` writer: persist the `memory_record` candidates the sifter already emits (separate `aa-memory` integration).
- Windows named-pipe backend for the transport.
- Live, paid provider smoke stays manual and opt-in, never in the default suite.
