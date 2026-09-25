# ISS-TRACE-1 — Contracts: bounded per-step trace object

**Type:** feature
**Status:** complete (phase 1 module update)
**Branch:** `ph1-contracts`
**Phase PR:** [#12](https://github.com/greadee/aa/pull/12)
**Parent:** [ISS-TRACE-LOOP](trace-learning-substrate.md)
**Module plan:** [../modules/contracts/updates/trace-contract/plan.md](../modules/contracts/updates/trace-contract/plan.md)
**Module summary:** [../modules/contracts/updates/trace-contract/summary.md](../modules/contracts/updates/trace-contract/summary.md)

## Goal

Make `aa-contracts` the single source of truth for a bounded, redacted, per-step **trace** — the evidence the learning and evaluation loop will consume — without redefining the observation protocol owned by `aa-obsv`.

## Problem

Phase 1 defined 14 cross-module objects, but the only execution evidence object is `telemetry`, which is a per-attempt aggregate. Kernel telemetry, memory history, and job learning therefore have no shared, versioned shape for step-level evidence, so step-level learning cannot be stored or compared across modules and projects.

## Requirements

- A new `trace` object with an explicit `sequence` ordering (never the wall clock).
- Bounded steps; a `truncated` flag when the bound is reached.
- Redaction-first: content is withheld or hashed; `redacted` and `redactionVersion` record the policy applied.
- Derived evidence only — not canonical history, and not the `obsv` observation protocol.
- Additive evolution: document contract minor 1.1, no breaking change to existing objects.
- Bindings and conformance across Go, TypeScript, and Python.

## Acceptance Criteria

- [x] `contracts/schemas/v1/trace.schema.json` (contract 1.1) exists and parses.
- [x] Go, TypeScript, and Python bindings mirror the schema.
- [x] A valid fixture decodes, validates, and round-trips; an invalid fixture is rejected.
- [x] Schemas, contracts, and policy documents updated.
- [x] `go build`, `go vet`, `go test`, `gofmt`, Python tests, and `tsc --noEmit` pass.

## Affected branches

This contract is a prerequisite for later sub-issues and is therefore referenced by `ph2-memory`, `ph3-kernel`, and `ph9-joblearn`. Their solutions are not specified here.

## Notes

Trace is operational evidence. Canonical storage, promotion, capture, routing, and distillation are owned by their respective modules and branches (see the parent umbrella).
