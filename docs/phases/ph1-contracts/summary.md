# Phase 1 — Contracts: Summary

> Authored at the **end** of the phase. Derived from [plan.md](./plan.md). As-built diagram: [summary.uml](./summary.uml).

## Delivered

| Slice | Status | Commit | Notes |
|---|---|---|---|
| Phase plan and UML | Complete | `d0a09e4` | plan.md + plan.uml |
| Contract versioning policy | Complete | `200e610` | `contracts/POLICY.md` |
| Core schemas v1 | Complete | `1ef39be` | 14 JSON Schemas |
| RPC and control-plane API specs | Complete | `497f989` | `rpc/rpc-v1.md`, `openapi/control-plane-v1.yaml` |
| Go contract types and validators | Complete | `f60f71b` | `contracts/go/v1` |
| Go conformance tests and fixtures | Complete | `825f991` | 14 fixtures + 1 invalid |
| TypeScript bindings | Complete | `b945a0e` | `contracts/typescript/v1` |
| Python bindings | Complete | `f4c0791` | `contracts/python/aa_contracts` |
| Architecture boundary harness | Complete | `44ef73f` | `tools` module, CI job |
| Branch retention policy | Complete | `7433baa` | docs/github-os + AA.md |
| Phase summary and UML | Complete | (this commit) | summary.md + summary.uml |

- Tracking issue: (created with the phase PR)
- Phase PR: (see GitHub)
- Branch: `ph1-contracts` (retained after merge)

## Validation

| Gate | Result | Evidence |
|---|---|---|
| Go build / vet / test | Pass | `contracts/go/v1` conformance suite; `tools` tests |
| Go formatting | Pass | `gofmt -l .` clean |
| Architecture boundaries | Pass | `go run ./cmd/archtest -root ..` reports ok |
| Python bindings | Pass | `compileall` + 4 unittest cases |
| TypeScript bindings | Pass | `tsc --noEmit` (local and CI) |
| Schemas | Pass | 14 schemas parse with draft and `$id` |
| OpenAPI | Pass | parses; 13 paths |
| Docs links | Pending | runs on CI after push |
| Phase audit | Not yet run | planned before merge to `main` |

## Decisions Affirmed

| # | Decision | Outcome | Evidence |
|---|---|---|---|
| ADR-P1-001 | JSON Schema is the source of truth; bindings match it | Affirmed | Go, TS, and Python bindings all decode the same fixtures |
| ADR-P1-002 | Schemas versioned under `schemas/v1` with shared common defs | Affirmed | 14 schemas under one directory with a versioned `$id` |
| ADR-P1-003 | Dependency-light bindings | Affirmed | Go stdlib only; Python stdlib only; TS types only |
| ADR-P1-004 | Conformance proven by golden fixtures | Affirmed | Fixtures decode, round-trip, and invalid input is rejected |
| ADR-P1-005 | Boundaries enforced by a `tools` harness | Affirmed | Harness detects a synthetic violation and passes the real workspace |
| ADR-P1-006 | Separate external control-plane (OpenAPI) from internal RPC | Affirmed | Two specs, one for console/visualizer, one for module calls |

## Deviations

| # | Planned | Actual | Reason |
|---|---|---|---|
| 1 | Python bindings as dataclass models | Validated dict objects with `TypedDict` shape hints | Standard-library-only and no serialization pitfalls; the schemas remain the source of truth |
| 2 | Ten product modules only | Added a non-product `tools` Go module (`archtest`) | Boundary enforcement is cross-module tooling, not a product module; excluded from the layering rules |
| 3 | A fixed nine-slice plan | Ten implementation slices plus a branch-retention docs slice | The retention/merge policy was requested explicitly and recorded as its own commit |
| 4 | Route routing defined as one schema per object | One `route.schema.json` holding `route_request` and `route_response` | The two objects are a request/response pair; kept together, tested with two fixtures |
| 5 | Conformance includes a full JSON Schema validator | Structural conformance (metadata, required fields, fixtures, round-trip) | Avoided adding a validation dependency to `contracts`; a schema-validator job can be added later |

## Deferred / Follow-Up

- Full JSON Schema validation in CI (a validator such as `check-jsonschema`).
- A schema-codegen pipeline so bindings are generated rather than authored (revisit when contract volume grows).
- Swift/Rust bindings if additional surfaces are added.
- Observation protocol integration: `obsv` adopts the shared event taxonomy in `ph3-kernel`.
- Phase audit before merge to `main`.

## Metrics

- Commits: 11 (10 slices + branch-retention docs)
- Child PRs: 0 (single phase PR)
- Issues completed: 1 (contract spine)
- ADRs: ADR-P1-001 … ADR-P1-006
- Pitfalls recorded: 0
- Tests added: 5 Go (4 conformance + 1 version) + 4 tools + 4 Python
- Schemas: 14
- Go fixtures: 15 (14 valid + 1 invalid)

## Retrospective

### Repeat
- Contract-first sequencing: policy → schemas → specs → bindings → conformance.
- One commit per slice, each independently validatable.
- Boundary harness early; it now guards every later phase.

### Avoid
- Authoring three bindings by hand is expensive; schedule codegen when the contract surface stabilizes.
- Keep the Python binding honest about being dict-based; do not imply runtime type guarantees it does not have.

### As-Built Diagram

[summary.uml](./summary.uml)
