# contracts — Module Update Plan: trace contract

> Authored at the **start** of the module update, on branch `ph1-contracts`.
> Issue: [ISS-TRACE-1](../../../../issues/ph1-contracts-trace-contract.md) · Parent umbrella: [ISS-TRACE-LOOP](../../../../issues/trace-learning-substrate.md)

## Objective

Add the bounded, redacted, per-step **trace** object to `aa-contracts`, so that observation capture (`ph3-kernel`), storage (`ph2-memory`), and learning (`ph9-joblearn`) share one versioned evidence shape. This is an additive update to the phase 1 contract spine; no phase is added.

## Starting State

- Starting ref: `main @ 79b495d` (after the phase 9 merge), branch `ph1-contracts` fast-forwarded to it.
- Available: 14 schemas under `schemas/v1`, Go/TypeScript/Python bindings, the conformance suite, the versioning policy, the RPC and control-plane specs.
- Missing: any per-step evidence object; the only execution evidence object is the per-attempt `telemetry` aggregate.
- Known constraints:
  - `contracts` depends on nothing and carries no business logic.
  - The observation protocol is owned by `aa-obsv`; `contracts` references it and must not redefine it.
  - Bindings are authored to match the schemas (no generator); the conformance suite guards drift.

## Scope

### In Scope
- A new `trace` JSON Schema (contract minor 1.1).
- Go, TypeScript, and Python bindings for `trace`.
- A valid fixture and an invalid fixture; conformance wiring.
- Schema index, module README, and versioning policy updates.

### Out of Scope
- Trace persistence, projection, promotion, and retention (`ph2-memory`).
- Live observation capture and the `obsv` protocol (`ph3-kernel`).
- Model routing and redaction execution (`ph4-sifter`).
- Distillation and evaluation (`ph9-joblearn`).
- A schema code generator.

## Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P1U-001 | Add a new `trace` object rather than extend `telemetry` | `telemetry` is a per-attempt aggregate with a different authority and retention; D8 keeps operational telemetry, work history, and canonical memory distinct |
| ADR-P1U-002 | Ordering is an explicit `sequence`; `at` is optional | Matches the event contract; ordering must never depend on the wall clock |
| ADR-P1U-003 | Redaction-first: content is withheld or hashed, with `redacted` and `redactionVersion` | Metadata-only by default; hashes support dedup/correlation without carrying recoverable input |
| ADR-P1U-004 | Bounded steps (`minItems` 1, `maxItems` 10000) plus a `truncated` flag | Keeps evidence bounded without silently pretending a truncated trace is complete |
| ADR-P1U-005 | New document declares `x-contract-version` 1.1; shared package constants are unchanged | POLICY encodes minor per document; bumping `v1.Version`/`CONTRACT_VERSION` would ripple into `forge`, `memory`, and `sifter` stamping and is not part of this branch |
| ADR-P1U-006 | `trace` is derived evidence, not the observation protocol | `aa-obsv` owns observation v1; `contracts` references it and stays product-neutral |

## Slices

Each slice maps to exactly one commit.

| Slice | Goal | Commit message |
|---|---|---|
| 1 | Issue documentation and this module plan | `add trace learning issue documentation and module update plan` |
| 2 | Trace schema, policy note, schema and module indexes | `add trace contract schema` |
| 3 | Go binding, conformance wiring, fixtures | `add trace go binding and fixtures` |
| 4 | TypeScript binding | `add trace typescript binding` |
| 5 | Python binding and exports | `add trace python binding` |
| 6 | Module update summary | `add ph1 contracts module update summary` |
| 7 | Link the phase PR | `link phase 1 module update pull request` |

## Required End State

- [ ] `contracts/schemas/v1/trace.schema.json` exists and parses as draft 2020-12 with a versioned `$id`.
- [ ] Go, TypeScript, and Python bindings mirror the schema.
- [ ] Go conformance decodes, validates, and round-trips a valid trace, and rejects an invalid one.
- [ ] Python and TypeScript validate the same fixtures.
- [ ] `contracts/README.md`, `schemas/v1/README.md`, and `POLICY.md` updated; `summary.md` authored.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, Python tests, and `tsc --noEmit` pass.

## Exit Criteria

- [ ] Trace object delivered in all three bindings and the conformance suite.
- [ ] Additive change recorded in `POLICY.md`; no breaking change to existing objects.
- [ ] Validation gates pass.
- [ ] Phase PR opened on `ph1-contracts`.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | Go `TraceStep`/`Trace` validators accept valid and reject malformed steps |
| contract | `trace.json` decodes, validates, and round-trips through Go and Python; schema metadata is well formed |
| negative | `trace-invalid.json` is rejected by Go and Python |
| types | `tsc --noEmit` over the TypeScript binding |
| boundary | `tools/archtest` unaffected (`contracts` still depends on nothing) |
| docs | link check |
