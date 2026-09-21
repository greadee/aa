# contracts — Module Update Summary: trace contract

> Authored at the **end** of the module update, on branch `ph1-contracts`. Derived from [module-upd-plan.md](module-upd-plan.md).
> Issue: [ISS-TRACE-1](../docs/issues/ph1-contracts-trace-contract.md) · Parent umbrella: [ISS-TRACE-LOOP](../docs/issues/trace-learning-substrate.md)

## Delivered

| Slice | Status | Commit message | Notes |
|---|---|---|---|
| Issue documentation and module plan | Complete | `add trace learning issue documentation and module update plan` | `docs/issues/`, this plan |
| Trace schema | Complete | `add trace contract schema` | `schemas/v1/trace.schema.json` (contract 1.1) |
| Go binding and fixtures | Complete | `add trace go binding and fixtures` | `go/v1/trace.go`, conformance wiring, valid + invalid fixtures |
| TypeScript binding | Complete | `add trace typescript binding` | `TraceStep`, `Trace`, union member |
| Python binding | Complete | `add trace python binding` | validator, TypedDicts, exports |
| Module update summary | Complete | `add ph1 contracts module update summary` | this file |
| Phase PR link | Complete | `link phase 1 module update pull request` | tracking PR |

## What was added

- `schemas/v1/trace.schema.json` — a `trace` object: `attemptId` (+ optional `assignmentId`, `workPackageId`), `redactionVersion`, `truncated`, and `steps[]`. Each step carries a required `sequence`, `phase`, and `outcome`, with optional actor, operation, target, input/output hashes, timing, tokens, cost, redaction flag, and evidence references.
- Go: `Trace`, `TraceStep`, `TracePhase`, `TraceOutcome`, and validators in `contracts/go/v1`.
- TypeScript: `TracePhase`, `TraceOutcome`, `TraceStep`, `Trace`, added to `ContractObject`.
- Python: `TRACE_PHASES`, `TRACE_STEP_OUTCOMES`, `Trace`, `TraceStep`, runtime validation, and exports.
- Conformance: `testdata/trace.json` (valid) and `testdata/trace-invalid.json` (rejected), wired into `decodeByKind`.
- Documentation: schema index, module README status, and a `POLICY.md` additive change-log entry.

## What this changes about the module and the app

- `aa-contracts` grows from 14 to 15 v1 objects. Nothing existing changes; the addition is additive and backward compatible.
- The contract set gains a shared, versioned, per-step evidence shape that earlier phases lacked. Observation capture, storage, and learning can now agree on one trace without redefining the `obsv` observation protocol.
- The app is otherwise unchanged: `contracts` remains dependency-free, and no module behavior is added here.

## Is this part of a larger change?

Yes. This is the first sub-issue of an umbrella change documented in full at [ISS-TRACE-LOOP](../docs/issues/trace-learning-substrate.md): the observation → trace → learning substrate was under-delivered across phases 0–9. This module update unblocks the remaining sub-issues on `ph2-memory`, `ph3-kernel`, `ph4-sifter`, and `ph9-joblearn`; their solutions are deliberately not specified here.

## Validation

| Gate | Result | Evidence |
|---|---|---|
| Go build / vet / test | Pass | `contracts/go/v1` conformance (trace valid, round-trip, invalid rejected) |
| Go formatting | Pass | `gofmt -l .` clean |
| Python bindings | Pass | `compileall` + 4 unittest cases |
| TypeScript bindings | Pass | `tsc --noEmit` clean |
| Schemas | Pass | `trace.schema.json` parses with draft 2020-12 and a versioned `$id` |
| Boundaries | Pass | `contracts` still depends on nothing |

## Decisions affirmed

| # | Decision | Outcome |
|---|---|---|
| ADR-P1U-001 | New `trace` object, not an extension of `telemetry` | Affirmed — the two have different authority and retention |
| ADR-P1U-002 | Explicit `sequence` ordering | Affirmed — conformance and round-trip are order-stable |
| ADR-P1U-003 | Redaction-first with hashes and flags | Affirmed — no content-bearing required field |
| ADR-P1U-004 | Per-document minor 1.1; shared constants unchanged | Affirmed — no ripple into `forge`, `memory`, or `sifter` |
| ADR-P1U-005 | Bounded steps with a `truncated` flag | Affirmed |
| ADR-P1U-006 | Observation protocol stays owned by `obsv` | Affirmed — trace is derived evidence only |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | Bump the shared contract version to 1.1 | Only the new document declares 1.1 | POLICY encodes minor per document; a package-wide bump would touch other modules' stamping, out of scope for this branch |

## Follow-Up

- `ph2-memory`: persist, project, promote, and retain traces (ISS-TRACE-2).
- `ph3-kernel`: capture real traces via live `obsv` and the kernel host (ISS-OBSV-1).
- `ph9-joblearn`: distill and evaluate subagent artifacts from traces (ISS-LEARN-1).
- Revisit a coordinated minor bump of the shared constants if a release process requires it.
