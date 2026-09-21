# ISS-OBSV-1 — Kernel (+ obsv): live observation substrate

**Type:** feature / technical debt
**Status:** in progress (kernel module update)
**Branch:** `ph3-kernel`
**Phase PR:** (linked from the module summary when opened)
**Parent umbrella:** `ISS-TRACE-LOOP` — the observation → trace → learning substrate is under-delivered (`docs/issues/trace-learning-substrate.md`)
**Module plan:** [../../obsv/module-upd-plan.md](../../obsv/module-upd-plan.md)
**Module summary:** [../../obsv/module-upd-summary.md](../../obsv/module-upd-summary.md)

## Goal

Deliver the `aa-obsv` observation substrate that phase 3 deferred: protocol v1, a bounded failure-open emitter, a journal with replay/subscribe/retention, an attach-or-own local transport, deterministic work reports, and the Codex `exec` translator, behind an `Ensure`/`Runtime`, `Send`, `Replay`, and `Subscribe` facade.

## Problem

The umbrella `ISS-TRACE-LOOP` records that the observation → trace → learning substrate was under-delivered. Phase 1 added the trace contract and phase 2 the trace store, but the source of every trace — the `obsv` work-observation module — is still scaffold only. Phase 3 explicitly deferred it (deviation #2): the kernel emits through an `orchestrator.Sink` and there is no live observation protocol, journal, or transport. Without `obsv`, trace capture has no producer.

## Requirements

- Observation protocol v1: session, tool, file, and work-delta source types with explicit `source_confidence` (`exact`, `correlated`, `observed`, `inferred`).
- Metadata-only by default: an allowlist sanitizer that drops content, prompt, source, and credential keys before journaling.
- A bounded, failure-open emitter that never blocks or panics the caller and accounts for drops.
- A journal with cursor replay, live subscription, and projected-view retention; an in-memory and a durable JSONL file backend.
- An in-process local transport exposing `hello` / `append` / `replay` / `subscribe` with attach-or-own semantics.
- Deterministic work reports and a Codex `exec` JSONL translator.
- A facade: `Ensure`/`Runtime`, `Send`, `Replay`, `Subscribe`.

## Acceptance Criteria

- [x] `obsv/protocol`, `obsv/emit`, `obsv/journal`, `obsv/transport`, `obsv/report`, and `obsv/codex` are implemented and tested.
- [x] `obsv.Ensure` is idempotent per session; `Send`/`Replay`/`Subscribe` work end to end.
- [x] `obsv` still imports no other aa module; the architecture boundary is unchanged.
- [x] `go build`, `go vet`, `go test`, `gofmt`, and the boundary check pass.

## Affected branches

Depends on the trace contract (`ISS-TRACE-1`, `ph1-contracts`) and the trace store (`ISS-TRACE-2`, `ph2-memory`) only at the substrate level: `obsv` produces the evidence those two store. This update does not require them to be merged, because the protocol is owned by `obsv`. It unblocks live capture on the kernel host and the learning sub-issue (`ISS-LEARN-1`).

## Notes

Kernel orchestrator wiring (an `obsv`-backed `Sink`) and the live kernel host socket remain follow-up work under `ISS-OBSV-1`; this document records the module itself.
