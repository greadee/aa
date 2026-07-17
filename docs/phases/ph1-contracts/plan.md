# Phase 1 — Contracts: Plan

> Authored at the **start** of the phase. Branch and folder: `ph1-contracts`.

## Objective

Make `aa-contracts` the single, versioned source of truth for every object and protocol that crosses a module boundary, with generated bindings for Go, TypeScript, and Python, a conformance suite, and an architecture boundary harness that keeps modules honest.

## Starting State

- Starting ref: `main @ 8464e35`
- Available:
  - Repository scaffold, canonical docs tree, adapted GitHub OS.
  - Optimized architecture and phase plan (`docs/architecture/README.md`).
  - Ten module skeletons (`contracts`, `obsv`, `kernel`, `memory`, `sync`, `sifter`, `forge`, `toolbox`, `visualizer`, `console`), nine Go modules in `go.work`, Python `sifter`.
  - CI (`ci.yml`) and docs link-check.
- Missing:
  - Any schema, type, or protocol.
  - Conformance and boundary tests (D5 not yet realized).
  - Versioning/compatibility policy.
- Known constraints:
  - `contracts` must depend on nothing.
  - No business logic in `contracts`.
  - Python binding must not force a heavy runtime dependency on `sifter`; `sifter` already uses Pydantic, so Pydantic is acceptable there but the binding should stay importable with only the standard library where practical.
  - The observation protocol is owned by `obsv`; `contracts` carries the orchestration/execution event taxonomy and references observation v1 rather than redefining it.

## Scope

### In Scope
- A versioning and compatibility policy for contracts.
- JSON Schema (draft 2020-12) v1 for the core cross-module objects.
- An RPC specification for inter-module calls and an OpenAPI document for the control plane.
- Generated/hand-written bindings in Go, TypeScript, and Python.
- A dependency-free conformance test suite and golden fixtures.
- An architecture boundary test harness enforced in CI.

### Out of Scope
- Implementing any module behavior (kernel, memory, etc.).
- A code generator toolchain; bindings are authored to match the schemas with conformance tests guarding them.
- Observation protocol changes (owned by `obsv`).
- Persistence, networking, or runtime code.

### Required End State
- [ ] Every cross-module object has exactly one v1 schema.
- [ ] Go, TypeScript, and Python bindings exist and decode the golden fixtures.
- [ ] Conformance tests pass and fail closed on malformed input.
- [ ] The boundary harness runs in CI and fails on a layering violation.
- [ ] No module defines an ad-hoc cross-module DTO.

## Architecture Decisions

| # | Decision | Reasoning |
|---|---|---|
| ADR-P1-001 | JSON Schema 2020-12 is the source of truth; bindings are authored to match it | Language-neutral, tooling-rich, and readable without a generator |
| ADR-P1-002 | Schemas are versioned under `schemas/v1/` with a shared `common` envelope | Additive evolution; one place to reason about compatibility |
| ADR-P1-003 | Bindings are dependency-light: Go stdlib only, Python stdlib-first (Pydantic optional for `sifter`) | Keeps `contracts` trivially embeddable and avoids dependency cycles |
| ADR-P1-004 | Conformance is proven by golden fixtures decoded through each binding | Catches drift between schema and implementations without a full validator dependency |
| ADR-P1-005 | Boundaries are enforced by a dedicated `tools/archtest` module that inspects `go list -deps` | Mechanical enforcement of D5; language-independent of product modules |
| ADR-P1-006 | The control plane is specified as OpenAPI; inter-module RPC as a small JSON-RPC spec | Separate the external/console surface from the internal module surface |

Diagrams: [plan.uml](./plan.uml).

## Slices

Each slice maps to exactly one commit.

### Slice 1 — Phase plan and UML

**Issues** — (phase tracking issue)

**Goal** — Record the plan and intended contract infrastructure.

**Inputs** — Architecture document, phase documentation convention.

**Expected Output** — `plan.md`, `plan.uml`.

**Model Class** — General

**Commit Message**

```text
add ph1 contracts phase plan
```

**Validation** — Docs link check.

**Dependencies** — none.

**Documentation** — this file, `plan.uml`.

### Slice 2 — Contract versioning and compatibility policy

**Goal** — Define how contracts evolve.

**Inputs** — D2, D20, terminology state registry.

**Expected Output** — `contracts/POLICY.md`.

**Model Class** — General

**Commit Message**

```text
add contract versioning policy
```

**Validation** — Docs link check.

**Dependencies** — Slice 1.

**Documentation** — `contracts/README.md`.

### Slice 3 — Core schemas v1

**Goal** — Define the v1 JSON Schemas for all cross-module objects.

**Inputs** — Architecture sections 5, 6, 8; terminology.

**Expected Output** — `contracts/schemas/v1/*.schema.json` (common, event, work-package, task-graph, execution-contract, result-envelope, telemetry, project-record, memory-record, issue, strategy, route, tool-manifest, workflow).

**Model Class** — Strong

**Commit Message**

```text
add contract schemas v1
```

**Validation** — Schemas parse as JSON; each declares draft 2020-12 and a versioned `$id`; docs link check.

**Dependencies** — Slice 2.

**Documentation** — `contracts/schemas/v1/README.md`.

### Slice 4 — RPC and control-plane API specs

**Goal** — Specify inter-module RPC and the console/visualizer control plane.

**Inputs** — Architecture section 6.

**Expected Output** — `contracts/rpc/rpc-v1.md`, `contracts/openapi/control-plane-v1.yaml`.

**Model Class** — General

**Commit Message**

```text
add rpc and control-plane api specs
```

**Validation** — OpenAPI parses as YAML/JSON; link check.

**Dependencies** — Slice 3.

**Documentation** — `contracts/README.md`.

### Slice 5 — Go contract types and validators

**Goal** — Provide Go types and validation for v1 objects.

**Inputs** — Schemas v1.

**Expected Output** — `contracts/go/v1/*.go` plus `contracts/go/v1/validate.go`.

**Model Class** — Strong

**Commit Message**

```text
add go contract types
```

**Validation** — `go build`, `go vet`, unit tests.

**Dependencies** — Slice 3.

**Documentation** — package doc comments.

### Slice 6 — Go conformance tests and fixtures

**Goal** — Prove bindings match the schemas and fail closed.

**Inputs** — Go types, schemas.

**Expected Output** — `contracts/go/v1/testdata/*.json`, `contracts/go/v1/conformance_test.go`.

**Model Class** — Strong

**Commit Message**

```text
add go contract conformance tests
```

**Validation** — `go test ./...` in `contracts`.

**Dependencies** — Slice 5.

**Documentation** — `contracts/README.md`.

### Slice 7 — TypeScript bindings

**Goal** — Provide TypeScript types for UI/console consumers.

**Inputs** — Schemas v1.

**Expected Output** — `contracts/typescript/v1/*.ts`.

**Model Class** — General

**Commit Message**

```text
add typescript contract bindings
```

**Validation** — `tsc --noEmit` is not yet wired (no Node project); types are self-contained and reviewed against schemas.

**Dependencies** — Slice 3.

**Documentation** — `contracts/typescript/README.md`.

### Slice 8 — Python bindings

**Goal** — Provide Python types for `sifter`.

**Inputs** — Schemas v1.

**Expected Output** — `contracts/python/aa_contracts/*.py`.

**Model Class** — General

**Commit Message**

```text
add python contract bindings
```

**Validation** — `python -m compileall`; a stdlib round-trip test.

**Dependencies** — Slice 3.

**Documentation** — `contracts/python/README.md`.

### Slice 9 — Architecture boundary test harness

**Goal** — Mechanically enforce layering (D5).

**Inputs** — Architecture section 4.2; module list.

**Expected Output** — `tools/` Go module (`archtest` package + `cmd/archtest`), `go.work` update, CI step.

**Model Class** — Strong

**Commit Message**

```text
add architecture boundary test harness
```

**Validation** — `go test ./...` in `tools`; harness run against the workspace passes; a unit test proves a violation is detected.

**Dependencies** — Slices 3–6.

**Documentation** — `tools/README.md`.

### Slice 10 — Phase summary and UML

**Goal** — Record delivered, affirmed, and deviated.

**Expected Output** — `summary.md`, `summary.uml`.

**Model Class** — General

**Commit Message**

```text
add ph1 contracts phase summary
```

**Validation** — link check.

**Dependencies** — all slices.

**Documentation** — `summary.md`, `summary.uml`.

## Exit Criteria

- [ ] Every cross-module object has a v1 schema.
- [ ] Go, TypeScript, and Python bindings exist and pass conformance/fixtures.
- [ ] RPC and control-plane API are specified.
- [ ] Boundary harness runs in CI and detects violations.
- [ ] `go build`, `go vet`, `go test`, `gofmt`, and link checks pass.
- [ ] Phase PR merged to `main`; branch retained.

## Test Plan

| Layer | What is tested |
|---|---|
| unit | Go validators accept valid and reject invalid objects; Python round-trip |
| contract | Schema metadata; golden fixtures decode through Go and Python bindings |
| boundary | `tools/archtest` unit rules and a live workspace scan |
| docs | link check |
| e2e | none this phase (no runtime yet) |
