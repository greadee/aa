# Toolbox Pre-Hardening — Sprint Plan

## Phase

Update phase (unnumbered) — pre-hardening for `ph7-toolbox`.

## Sprint Objective

Address all findings from the toolbox module code review in a single pre-hardening sprint before ph7-toolbox is merged to main.

## Issues

- #51 — RPC TOCTOU race condition (P0)
- #51 — Stale contract version in tests (P1)
- #51 — RPC layer uses map[string]any (P1)
- #51 — No timeout honored in RPC (P2)
- #51 — RPC error responses lack structured data (P2)
- #51 — Untyped WorkflowStep.Config (P2)
- #51 — No metrics/observability (P2)
- #51 — Large contract files (P3)
- #51 — Python/TypeScript binding parity (P3)

## Slices

### Slice 1 — Fix RPC TOCTOU race

**Issues:** #51 (P0)

**Goal:** Fix the TOCTOU race condition in `Service.Handle` that allows concurrent duplicate executions.

**Inputs:**
- `toolbox/rpc/rpc.go` — `Service.Handle` method, lines 124–131 and 149–153

**Expected Output:**
- Concurrent identical requests execute the provider exactly once
- Pending-flag pattern implemented (store pending entry with WaitGroup, subsequent requests wait)
- Regression test: concurrent identical requests test

**Model Class:** General

**Commit Message:**
```
fix RPC TOCTOU race in Service.Handle
```

**Validation:**
- `go test ./toolbox/rpc/...` passes
- Concurrent identical requests test passes

---

### Slice 2 — Update test fixtures to v2

**Issues:** #51 (P1)

**Goal:** Update all toolbox test fixtures to use `ContractVersion: "2.0"`.

**Inputs:**
- `toolbox/host/host_test.go`
- `toolbox/policy/policy_test.go`
- `toolbox/registry/registry_test.go`
- `toolbox/rpc/rpc_test.go`
- `toolbox/workflow/compile_test.go`
- `toolbox/workflow/runtime_test.go`

**Expected Output:**
- All test fixtures use `"2.0"`
- All toolbox tests pass

**Model Class:** Lightweight

**Commit Message:**
```
update toolbox test fixtures to contract v2
```

**Validation:**
- `go test ./toolbox/...` passes

---

### Slice 3 — Add typed RPC request structs

**Issues:** #51 (P1)

**Goal:** Replace `map[string]any` deserialization with typed request/response structs.

**Inputs:**
- `toolbox/rpc/rpc.go`

**Expected Output:**
- Typed `InvokeRequest`, `CompileWorkflowRequest`, `RegistryRequest` structs
- Manual field extraction removed
- All existing tests pass

**Model Class:** General

**Commit Message:**
```
add typed RPC request structs
```

**Validation:**
- `go vet ./toolbox/rpc/...` passes
- `go test ./toolbox/rpc/...` passes

---

### Slice 4 — Honor TimeoutMS in RPC

**Issues:** #51 (P2)

**Goal:** Honor the `TimeoutMS` field from `AA` request metadata.

**Inputs:**
- `toolbox/rpc/rpc.go` — `Service.Handle`

**Expected Output:**
- Requests with `TimeoutMS > 0` are bounded by context.WithTimeout
- Timeout returns proper JSON-RPC error

**Model Class:** Lightweight

**Commit Message:**
```
honor TimeoutMS in RPC handler
```

**Validation:**
- `go test ./toolbox/rpc/...` passes
- Timeout test added

---

### Slice 5 — Add structured RPC error data

**Issues:** #51 (P2)

**Goal:** Add structured diagnostic data to RPC error responses.

**Inputs:**
- `toolbox/rpc/rpc.go` — `errResponseFrom`, `errResponseData`

**Expected Output:**
- `ErrorData` struct with `toolboxErrorName`, `toolId`, `executionContractId`
- All error paths include structured data

**Model Class:** Lightweight

**Commit Message:**
```
add structured data to RPC error responses
```

**Validation:**
- `go test ./toolbox/rpc/...` passes

---

### Slice 6 — Validate WorkflowStep.Config

**Issues:** #51 (P2)

**Goal:** Add validation for `WorkflowStep.Config` per step kind.

**Inputs:**
- `toolbox/workflow/compile.go`
- `contracts/go/v2/workflow.go`

**Expected Output:**
- `validateStepConfig` called for each step in `Compile`
- Invalid config returns `ErrInvalid`

**Model Class:** Lightweight

**Commit Message:**
```
validate WorkflowStep.Config per step kind
```

**Validation:**
- `go test ./toolbox/workflow/...` passes

---

### Slice 7 — Add metrics interface

**Issues:** #51 (P2)

**Goal:** Add metrics collection interface to toolbox.

**Inputs:**
- `toolbox/policy/policy.go`
- `toolbox/host/host.go`
- `toolbox/workflow/runtime.go`
- `toolbox/rpc/rpc.go`

**Expected Output:**
- `Metrics` interface defined
- No-op implementation provided by default
- Optional prometheus/OpenTelemetry implementation stubbed

**Model Class:** General

**Commit Message:**
```
add metrics interface to toolbox
```

**Validation:**
- `go test ./toolbox/...` passes
- No functional change to existing behavior

---

### Slice 8 — Split large contract files

**Issues:** #51 (P3)

**Goal:** Split large contract files into smaller, more navigable units.

**Inputs:**
- `contracts/go/v2/records.go`
- `contracts/go/v2/event.go`
- `contracts/go/v2/telemetry.go`

**Expected Output:**
- Domain-specific files (planning.go, routing.go, execution.go, etc.)
- Conformance suite still passes

**Model Class:** Lightweight

**Commit Message:**
```
split large contract files into domain-specific units
```

**Validation:**
- `go test ./contracts/...` passes

---

### Slice 9 — Review Python/TypeScript bindings

**Issues:** #51 (P3)

**Goal:** Review Python and TypeScript bindings for parity with Go.

**Inputs:**
- `contracts/typescript/v2/index.ts`
- `contracts/python/aa_contracts/v2.py`
- `contracts/go/v2/conformance_test.go`

**Expected Output:**
- Audit report identifying missing types
- Missing types added to Python/TypeScript bindings

**Model Class:** Lightweight

**Commit Message:**
```
review Python/TypeScript bindings for parity with Go
```

**Validation:**
- Audit report documented
- Missing types added (if any)

---

## Sprint Exit Criteria

- [ ] all 9 slices complete
- [ ] no open P0/P1 bugs
- [ ] toolbox test suite passes
- [ ] contracts conformance suite passes
- [ ] documentation updated
- [ ] audit complete
- [ ] ready for main
