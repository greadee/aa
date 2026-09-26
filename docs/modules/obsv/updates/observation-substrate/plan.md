# obsv — Module Update Plan: observation substrate

> Authored at the **start** of the module update, on branch `ph3-kernel` (the phase that owns the `obsv` host).
> Sub-problem: `ISS-OBSV-1` · Parent umbrella: `ISS-TRACE-LOOP` (`docs/issues/trace-learning-substrate.md`)

## Objective

Deliver the missing `aa-obsv` observation substrate: a product-neutral protocol v1, a bounded failure-open emitter, a durable journal with replay/subscribe/retention, an attach-or-own local transport, deterministic work reports, and the Codex `exec` JSONL translator. This is the portion of the observation → trace → learning substrate that phase 3 deferred (phase 3 deviation #2), and the prerequisite for live trace capture on the kernel.

## Starting State

- Starting ref: `main @ 79b495d`; branch `ph3-kernel` fast-forwarded to it.
- Available:
  - `obsv` module scaffold (`go.mod`, `doc.go`, `README.md`, `aa-obsrv.md`) and the architecture boundary `obsv -> contracts`.
  - `contracts` v1 event taxonomy, which explicitly leaves observation events to `aa-obsv` (`contracts/go/v1/event.go`).
  - The kernel control plane that emits through an `orchestrator.Sink` and the visualizer's event seam that expects an `obsv` transport.
- Missing:
  - Any executable `obsv` code: protocol types, emitter, journal, transport, reports, translator, or facade.
- Known constraints:
  - Observation is metadata-only by default; sanitize to a durable allowlist before journaling.
  - Observation failure must never alter agent behavior (failure-open).
  - Confidence is semantic and must never be promoted (`exact`, `correlated`, `observed`, `inferred`).
  - `obsv` depends only on itself (and, if needed, `contracts`); it never calls models or owns history.
  - The kernel host wiring (using `obsv` from the orchestrator) is a separate follow-up; this update delivers the module.

## Scope

### In Scope
- `obsv/protocol`: observation protocol v1 — source types, confidence, bounded metadata, allowlist sanitization, validation.
- `obsv/emit`: bounded, failure-open emitter with panic/error isolation and drop accounting.
- `obsv/journal`: append-only journal with cursor replay, live subscription, and projected-view retention; in-memory and durable JSONL file backends.
- `obsv/transport`: in-process local transport with `hello` / `append` / `replay` / `subscribe` and attach-or-own semantics.
- `obsv/report`: deterministic work reports derived from journaled events.
- `obsv/codex`: a deterministic Codex `exec` JSONL translator.
- The `obsv` facade: `Ensure` / `Runtime`, `Send`, `Replay`, `Subscribe`.

### Out of Scope
- Kernel orchestrator wiring (a `Sink` backed by `obsv`) and the live kernel host socket.
- The control-plane server and the supervised vertical slice (tracked by `ISS-OBSV-1`).
- Visualizer adoption and the AAV compatibility profile.
- Any `contracts` schema addition; the protocol is owned by `obsv`.

## Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P3U-001 | Observation protocol v1 is owned by `obsv`; `contracts` orchestration events are untouched | `contracts/go/v1/event.go` already defers observation events to `aa-obsv`; one owner per vocabulary |
| ADR-P3U-002 | Metadata-only, allowlist-first sanitization at the protocol boundary | Privacy by default: content, prompts, source, and credentials never reach the journal |
| ADR-P3U-003 | The emitter is bounded and failure-open | A full queue or a failing sink drops observation without blocking or changing agent behavior |
| ADR-P3U-004 | The journal is append-only, sequence-ordered, and replay-equivalent; retention trims a projected view only | Deterministic replay and a canonical record mirror the `memory`/`visualizer` retention precedent |
| ADR-P3U-005 | The local transport is attach-or-own | One journal per session; a second `Ensure` attaches to the existing owner instead of opening a second journal |
| ADR-P3U-006 | Reports and translation order only by sequence, never by wall clock | Reproducible reports and fixture-stable translation |
| ADR-P3U-007 | Confidence is semantic and never promoted | Preserves the meaning of `exact`/`correlated`/`observed`/`inferred` |

## Slices

Each slice maps to exactly one commit.

| Slice | Goal | Commit message |
|---|---|---|
| 1 | Issue documentation and this module plan | `add ph3 kernel obsv module update plan and issue documentation` |
| 2 | Protocol v1 types, confidence, sanitization, validation | `add obsv observation protocol` |
| 3 | Bounded failure-open emitter | `add obsv bounded emitter` |
| 4 | Journal with replay, subscribe, retention, file backend | `add obsv journal` |
| 5 | In-process local transport with attach-or-own | `add obsv local transport` |
| 6 | Deterministic work reports | `add obsv work reports` |
| 7 | Codex `exec` JSONL translator | `add obsv codex exec translator` |
| 8 | Facade `Ensure`/`Runtime`, `Send`, `Replay`, `Subscribe`, README | `wire obsv runtime ensure send replay subscribe` |
| 9 | Module update summary | `add ph3 kernel obsv module update summary` |
| 10 | Link the module update PR | `link ph3 kernel obsv pull request` |

## Required End State

- [ ] `protocol.Event` validates; unknown source types and confidences are rejected.
- [ ] `protocol.Sanitize` keeps only allowlisted metadata and drops content-like keys.
- [ ] The emitter never panics or blocks the caller and counts drops.
- [ ] The journal replays by cursor, notifies subscribers in order, and trims only a projected view.
- [ ] The transport exposes `hello`/`append`/`replay`/`subscribe` and attaches to an existing session owner.
- [ ] Work reports and Codex translation are deterministic across runs and input order.
- [ ] `obsv.Ensure` is idempotent per session and `Send`/`Replay`/`Subscribe` work end to end.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and the architecture boundary check pass.

## Exit Criteria

- [ ] The observation substrate is delivered and tested across all packages.
- [ ] `obsv` still depends only on itself (and `contracts`); the archtest boundary is unchanged.
- [ ] `obsv/README.md`, `obsv/aa-obsrv.md`, and the module summary reflect the delivered state.
- [ ] A module update PR is opened on `ph3-kernel` into `main`.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | protocol validation/sanitization, emitter drops and panic isolation, journal ordering/retention, report determinism |
| contract | a translated Codex line and a hand-built event round-trip through the protocol |
| integration | `Ensure -> Send -> Replay/Subscribe -> Report` end to end, including a durable file journal across reopen |
| determinism | reports and translation stable across runs and input order |
| failure-open | a panicking or erroring sink never propagates and is counted |
| boundary | `obsv` imports no other aa module (`tools/archtest`) |
