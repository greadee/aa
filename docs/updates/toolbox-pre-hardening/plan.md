# Toolbox Pre-Hardening — Sprint Plan

## Sprint Metadata

- **Phase:** Update phase (pre-hardening for ph7-toolbox)
- **Milestone:** #2 — `v1 — Toolbox Pre-Hardening`
- **Tracking Issue:** #50 (umbrella)
- **Sprint PR:** #60 (branch `update-toolbox-pre-hardening` → `main`)
- **Modules affected:** `toolbox/`, `contracts/go/v2/`
- **Model class:** General (P0/P1), Lightweight (P2/P3)
- **Slices:** 7 (slices 1-2 already done, 5 active slices)
- **Estimated commits:** 14–20 (2 commits per active slice: implementation + tests)

## Sprint Objective

Address all findings from the toolbox module code review in a single pre-hardening sprint. Fix critical and high-severity issues (TOCTOU race, stale contract versions, RPC fragility) and medium-severity improvements (timeouts, error structure, config validation, observability) before ph7-toolbox is merged to main.

## Sprint Exit Criteria

- [ ] all 7 active slices complete
- [ ] no open P0/P1 bugs
- [ ] `go test ./toolbox/...` passes
- [ ] `go test ./contracts/...` passes (conformance suite)
- [ ] `go vet ./toolbox/...` passes
- [ ] documentation updated where required
- [ ] audit complete (P0: 0, P1: 0, open P2: 0)
- [ ] ready for main merge

## Issue Reference (Active Only)

Issues #58 and #59 are **already resolved** (see Slice 0 below). Remaining slices:

| Issue # | Severity | Slice | Title |
|---|---|---|---|
| #52 | P1 (high) | 1 | Update toolbox test fixtures to contract v2 |
| #56 | P2 (medium) | 2 | Validate WorkflowStep.Config per step kind |
| #51 | P0 (critical) | 3 | Fix RPC TOCTOU race condition in Service.Handle |
| #53 | P1 (high) | 4 | Replace map[string]any with typed RPC request structs |
| #54 | P2 (medium) | 5 | Honor TimeoutMS in RPC handler |
| #55 | P2 (medium) | 6 | Add structured data to RPC error responses |
| #57 | P2 (medium) | 7 | Add metrics interface to toolbox |

## Execution Order and Dependencies

```
#52 (stale version)  — independent, do first (small, builds confidence)
  ↓
#56 (config validation) — independent, can be parallel
  ↓
#51 (TOCTOU race) — independent, do before #53 (P0 critical)
  ↓
#53 (typed structs) — depends on #51 (cleaner after race fix)
  ↓
#54 (timeout) — depends on #53 (typed structs make timeout easier)
  ↓
#55 (error data) — depends on #53 (typed structs make error data easier)
  ↓
#57 (metrics) — touches everything, do last
```

## Slice 0 — Already Done (Pre-Check Before Starting)

### Slice 0a — Split Large Contract Files (Issue #58, P3) — ✅ RESOLVED

The contracts/go/v2 directory has **22 domain-specific files**, all under 231 lines. No large files exist. File: `ls /Users/prool/aa/contracts/go/v2/*.go` shows: common.go (140), records.go (172), event.go (112), telemetry.go (50), workflow.go (63), executioncontract.go (92), executionplan.go (46), workplan.go (45), taskgraph.go (53), route.go (85), routingdecision.go (42), result.go (92), tool.go (48), trace.go (135), validate.go (183), conformance_test.go (231), plus modelspec.go, rolespec.go, teamspec.go, routine.go, workpackage.go, issue, strategy.

**Action:** Mark issue #58 as closed. No implementation needed.

### Slice 0b — Binding Parity Review (Issue #59, P3) — ✅ RESOLVED

The Go conformance test handles **21 kinds**: event, work_package, task_graph, execution_contract, result_envelope, telemetry, trace, project_record, memory_record, issue, strategy, route_request, route_response, tool_manifest, workflow, role_spec, model_spec, team_spec, work_plan, execution_plan, routine.

Both Python (`contracts/python/aa_contracts/v2.py`) and TypeScript (`contracts/typescript/v2/index.ts`) bindings have **all 21 kinds** defined with TypedDict/Interface types and validation logic. The TypeScript bindings are actually more detailed (with union types, specific kind fields, etc.).

**Action:** Mark issue #59 as closed. No implementation needed.

---

## Prerequisites

Before starting, verify the repository state:

```bash
cd /Users/prool/aa
git checkout main
git pull origin main
go work sync
go test ./toolbox/...  # baseline: all tests must pass
go test ./contracts/...  # baseline: conformance suite must pass
```

The repository root `/Users/prool/aa` contains:
- `contracts/` — JSON schemas, Go/TS/Python bindings, RPC spec
- `toolbox/` — registry, policy, host, workflow, rpc, computeruse packages
- `go.work` — workspace file listing all modules

---

## Slice 1 — Update Test Fixtures to Contract v2 (Issue #52, P1)

**Goal:** Update all toolbox test fixtures to use the active contract version `"2.0"` instead of the stale `"1.0"`.

**Inputs:**
- `contracts/go/v2/common.go` line 8: `const Version = "2.0"`
- 6 test files contain `ContractVersion: "1.0"` (confirmed by grep)

### Files to modify (exact line numbers):

1. **`toolbox/host/host_test.go`** (lines 18, 28):
   - Line 18: `Envelope: toolbox.Envelope{ContractVersion: "1.0", Kind: "tool_manifest", ID: id}`
   - Line 28: `Envelope: toolbox.Envelope{ContractVersion: "1.0", Kind: "execution_contract", ID: "ec_1"}`

2. **`toolbox/policy/policy_test.go`** (lines 15, 25):
   - Line 15: `Envelope: toolbox.Envelope{ContractVersion: "1.0", Kind: "tool_manifest", ID: id}`
   - Line 25: `Envelope: toolbox.Envelope{ContractVersion: "1.0", Kind: "execution_contract", ID: "ec_1"}`

3. **`toolbox/registry/registry_test.go`** (line 17):
   - `ContractVersion: "1.0"` in manifest helper

4. **`toolbox/rpc/rpc_test.go`** (line 29):
   - `Envelope: toolbox.Envelope{ContractVersion: "1.0", Kind: "tool_manifest", ID: "tool_a"}`

5. **`toolbox/rpc/rpc_test.go`** (lines 108, 131):
   - `"contractVersion": "1.0"` in workflow JSON strings

6. **`toolbox/workflow/compile_test.go`** (line 13):
   - `Envelope: toolbox.Envelope{ContractVersion: "1.0", Kind: "workflow", ID: id}`

7. **`toolbox/workflow/runtime_test.go`** — the `wf()` helper uses `"1.0"`

### Implementation — Replace every occurrence of `"1.0"` as `ContractVersion` with `"2.0"`:

```bash
# In each file, search for: ContractVersion: "1.0"
# Replace with: ContractVersion: "2.0"
```

**Important:** Do NOT change the `Version` field (e.g., `"1.0"` as the manifest version string). Only change `ContractVersion` fields.

### Acceptance Criteria

- [ ] All 6 test files updated
- [ ] `go test ./toolbox/...` passes after changes
- [ ] `go test ./contracts/...` still passes (conformance suite)

### Validation

```bash
cd /Users/prool/aa
go test ./toolbox/... -v
go test ./contracts/... -v
```

### Pitfalls

- The contracts package declares `Version = "2.0"` in `contracts/go/v2/common.go`. All test fixtures must match.
- The conformance test in `contracts/go/v2/conformance_test.go` validates against `"2.0"` fixtures. This is already correct.
- **Do NOT change the `Version` field on `Envelope`** (that's the manifest version, e.g., `"1.0"`). Only change `ContractVersion`.

---

## Slice 2 — Validate WorkflowStep.Config (Issue #56, P2)

**Goal:** Add validation for `WorkflowStep.Config` per step kind, validated at compile time (when the workflow is compiled), not at runtime.

**Inputs:**
- `contracts/go/v2/workflow.go` — `WorkflowStep.Config` is `map[string]any` with no validation in `Validate()`
- `toolbox/workflow/compile.go` — `Compile()` validates step IDs, kinds, dependencies, retries, gates, tool IDs — but never touches `step.Config`

### Current code (contracts/go/v2/workflow.go):

```go
type WorkflowStep struct {
    ID        string         `json:"id"`
    Kind      string         `json:"kind"`
    Role      string         `json:"role,omitempty"`
    ToolID    string         `json:"toolId,omitempty"`
    DependsOn []string       `json:"dependsOn,omitempty"`
    Retries   *int           `json:"retries,omitempty"`
    Gates     []string       `json:"gates,omitempty"`
    OnFailure string         `json:"onFailure,omitempty"`
    Config    map[string]any `json:"config,omitempty"`
}
```

The `Config` field is `map[string]any` — no validation.

### Current code (toolbox/workflow/compile.go lines 42-50):

```go
for _, s := range w.Steps {
    if s.Kind == "tool" && s.ToolID == "" {
        return Plan{}, fmt.Errorf("%w: step %q is a tool step without a toolId", toolbox.ErrInvalid, s.ID)
    }
    if s.Kind == "gate" && len(s.Gates) == 0 {
        return Plan{}, fmt.Errorf("%w: step %q is a gate step without gates", toolbox.ErrInvalid, s.ID)
    }
    if s.Retries != nil && *s.Retries < 0 {
        return Plan{}, fmt.Errorf("%w: step %q has negative retries", toolbox.ErrInvalid, s.ID)
    }
    // NO validation of s.Config here — that's the bug
}
```

### Implementation:

**Step 2.1 — Add validateStepConfig function to `toolbox/workflow/compile.go`:**

```go
func validateStepConfig(step toolbox.WorkflowStep) error {
    if step.Config == nil || len(step.Config) == 0 {
        return nil // optional, no config needed
    }
    switch step.Kind {
    case "parallel":
        return validateParallelStepConfig(step)
    case "conditional":
        return validateConditionalStepConfig(step)
    default:
        // tool, gate, task, human_approval: config is optional, no validation needed
        return nil
    }
}

func validateParallelStepConfig(step toolbox.WorkflowStep) error {
    if v, ok := step.Config["concurrency"]; ok {
        switch v.(type) {
        case int, int64, float64:
            // OK
        default:
            return fmt.Errorf("parallel step config: concurrency must be a positive integer")
        }
    }
    return nil
}

func validateConditionalStepConfig(step toolbox.WorkflowStep) error {
    if v, ok := step.Config["condition"]; ok {
        if s, ok := v.(string); !ok || s == "" {
            return fmt.Errorf("conditional step config: condition must be a non-empty string")
        }
    }
    return nil
}
```

**Step 2.2 — Call validateStepConfig in Compile()** (after line 50, before the dependency loop):

```go
// After: if s.Retries != nil && *s.Retries < 0 { ... }
// Add:
if err := validateStepConfig(s); err != nil {
    return Plan{}, fmt.Errorf("%w: step %q config: %s", toolbox.ErrInvalid, s.ID, err)
}
```

### Acceptance Criteria

- [ ] valid workflows behave identically (no functional change for workflows without Config)
- [ ] workflows with invalid parallel/conditional config are rejected with ErrInvalid
- [ ] `go test ./toolbox/workflow/...` passes

### Validation

```bash
cd /Users/prool/aa
go test ./toolbox/workflow/... -v
```

---

## Slice 3 — Fix RPC TOCTOU Race (Issue #51, P0)

**Goal:** Fix the TOCTOU race condition in `Service.Handle` that allows concurrent identical requests to both execute the underlying provider, defeating the idempotency cache.

**Inputs:**
- `toolbox/rpc/rpc.go` — `Service.Handle` method, lines 117–155
- **Exact race:** Lines 124–131 (check cache, unlock) and 149–153 (re-lock to store)

### Current code (toolbox/rpc/rpc.go lines 117–155):

```go
func (s *Service) Handle(ctx context.Context, req Request) Response {
    if req.JSONRPC != "" && req.JSONRPC != "2.0" {
        return errResponse(req.ID, CodeInvalidRequest, "jsonrpc must be \"2.0\"")
    }
    if !compatible(req.AA) {
        return errResponseData(req.ID, CodeIncompatible, "unsupported RPC or contract major", retryable(false))
    }
    // RACE WINDOW: lines 124-131
    if key, ok := idempotencyKey(req); ok {
        s.mu.Lock()
        if cached, found := s.cache[req.Method+"|"+key]; found {
            s.mu.Unlock()
            return Response{JSONRPC: "2.0", ID: req.ID, Result: cached}
        }
        s.mu.Unlock()  // <-- RACE: another goroutine passes here
    }
    var (
        result any
        err    error
    )
    switch req.Method {
    case MethodInvoke:
        result, err = s.invoke(ctx, req)
    case MethodCompileWorkflow:
        result, err = s.compileWorkflow(req)
    case MethodRegistry:
        result, err = s.listRegistry(req)
    default:
        return errResponse(req.ID, CodeMethodNotFound, fmt.Sprintf("unknown method %q", req.Method))
    }
    if err != nil {
        return errResponseFrom(req.ID, err)
    }
    // RACE WINDOW: lines 149-153
    if key, ok := idempotencyKey(req); ok {
        s.mu.Lock()
        s.cache[req.Method+"|"+key] = result
        s.mu.Unlock()
    }
    return Response{JSONRPC: "2.0", ID: req.ID, Result: result}
}
```

### Current Service struct (lines 86-92):

```go
type Service struct {
    host      *host.Host
    contracts ContractResolver

    mu    sync.Mutex
    cache map[string]any  // holds either Response (completed) or *pendingEntry (in-flight)
}
```

### Implementation:

**Step 3.1 — Add pendingEntry type (add near line 92, after Service struct):**

```go
// pendingEntry represents an in-flight request that has not yet completed.
// The cache holds either a Response (completed) or a *pendingEntry (in-flight).
type pendingEntry struct {
    ready chan Response  // buffered channel (size 1) to avoid deadlock
}
```

**Step 3.2 — Rewrite Handle() to use the pending-flag pattern:**

Replace the entire `Handle` method (lines 117-155) with:

```go
func (s *Service) Handle(ctx context.Context, req Request) Response {
    // Validate JSON-RPC version
    if req.JSONRPC != "" && req.JSONRPC != "2.0" {
        return errResponse(req.ID, CodeInvalidRequest, "jsonrpc must be \"2.0\"")
    }
    if !compatible(req.AA) {
        return errResponseData(req.ID, CodeIncompatible, "unsupported RPC or contract major", retryable(false))
    }

    // Extract idempotency key (if any).
    cacheKey := ""
    if req.Params != nil {
        if v, ok := req.Params["idempotencyKey"].(string); ok && v != "" {
            cacheKey = req.Method + "|" + v
        }
    }

    // Handle idempotent requests with pending-flag pattern.
    if cacheKey != "" {
        s.mu.Lock()
        // Check if there's already a completed result or a pending entry.
        if existing, found := s.cache[cacheKey]; found {
            switch entry := existing.(type) {
            case *pendingEntry:
                // Another request is in flight — wait for it, then return its result.
                s.mu.Unlock()
                resp := <-entry.ready
                return resp
            case Response:
                // It's a completed result — return it immediately.
                resp := entry
                s.mu.Unlock()
                return resp
            }
        }
        // No existing entry — create a pending entry and store it.
        entry := &pendingEntry{ready: make(chan Response, 1)}
        s.cache[cacheKey] = entry
        s.mu.Unlock()

        // Execute the handler (outside the lock).
        var result any
        var err error
        switch req.Method {
        case MethodInvoke:
            result, err = s.invoke(ctx, req)
        case MethodCompileWorkflow:
            result, err = s.compileWorkflow(req)
        case MethodRegistry:
            result, err = s.listRegistry(req)
        default:
            err = fmt.Errorf("unknown method %q", req.Method)
        }

        // Build the response.
        resp := Response{JSONRPC: "2.0", ID: req.ID}
        if err != nil {
            resp.Error = errResponseFrom(req.ID, err).Error
        } else {
            resp.Result = result
        }

        // Store the result (replacing the pending entry) and signal waiters.
        s.mu.Lock()
        s.cache[cacheKey] = resp
        s.mu.Unlock()
        entry.ready <- resp
        return resp
    }

    // No idempotency key — execute directly (no caching).
    var result any
    var err error
    switch req.Method {
    case MethodInvoke:
        result, err = s.invoke(ctx, req)
    case MethodCompileWorkflow:
        result, err = s.compileWorkflow(req)
    case MethodRegistry:
        result, err = s.listRegistry(req)
    default:
        return errResponse(req.ID, CodeMethodNotFound, fmt.Sprintf("unknown method %q", req.Method))
    }

    resp := Response{JSONRPC: "2.0", ID: req.ID}
    if err != nil {
        resp.Error = errResponseFrom(req.ID, err).Error
    } else {
        resp.Result = result
    }
    return resp
}
```

**Step 3.3 — Remove the old `idempotencyKey()` helper** (line 261-270) since it's no longer used (the key extraction is now inline in Handle).

Actually, keep `idempotencyKey()` for backward compatibility if any external callers use it. If it's unexported and only used internally, remove it.

### Acceptance Criteria

- [ ] root cause understood (race window between unlock and re-lock)
- [ ] race condition corrected (pending-flag pattern)
- [ ] concurrent identical requests execute provider exactly once
- [ ] `go test ./toolbox/rpc/... -v -race` passes (no data races)

### Validation

```bash
cd /Users/prool/aa
go test ./toolbox/rpc/... -v -race
```

The `-race` flag enables the Go race detector.

### Pitfalls

- The `pendingEntry` pattern requires careful handling of the channel. Use a **buffered channel** (`make(chan Response, 1)`) to avoid deadlocks.
- The cache now holds either a `Response` (completed) or a `*pendingEntry` (in-flight). Type assertions use a `switch` on type.
- The cache key is `req.Method + "|" + key` (method-scoped per-key). This is already correct.
- **Do not change the public API** of `Service.Handle` — same signature, same return type.

---

## Slice 4 — Add Typed RPC Request Structs (Issue #53, P1)

**Goal:** Replace the `map[string]any` deserialization in the RPC layer with typed request/response structs, so parameter names, types, and required fields are validated at compile time.

**Inputs:**
- `toolbox/rpc/rpc.go` — `stringField()` (line 272-278), `workflowParams()` (line 222-236), `invoke()` (line 157-198), `compileWorkflow()` (line 200-210), `listRegistry()` (line 212-220)

### Current code (toolbox/rpc/rpc.go):

**`stringField()` (lines 272-278):**
```go
func stringField(params map[string]any, field string) (string, error) {
    value, ok := params[field].(string)
    if !ok || value == "" {
        return "", fmt.Errorf("%w: %s is required", toolbox.ErrInvalid, field)
    }
    return value, nil
}
```

**`invoke()` (lines 157-198):**
```go
func (s *Service) invoke(ctx context.Context, req Request) (any, error) {
    toolID, err := stringField(req.Params, "toolId")
    if err != nil {
        return nil, err
    }
    contractID := ""
    if v, ok := req.Params["executionContractId"].(string); ok {
        contractID = v
    }
    if contractID == "" {
        return nil, fmt.Errorf("%w: executionContractId is required", toolbox.ErrInvalid)
    }
    // ... resolve contract, get input, call host.Invoke ...
}
```

**`workflowParams()` (lines 222-236):**
```go
func workflowParams(params map[string]any) (toolbox.Workflow, error) {
    raw := any(params)
    if inner, ok := params["workflow"]; ok {
        raw = inner
    }
    encoded, err := json.Marshal(raw)
    if err != nil {
        return toolbox.Workflow{}, fmt.Errorf("%w: workflow: %v", toolbox.ErrInvalid, err)
    }
    var wf toolbox.Workflow
    if err := json.Unmarshal(encoded, &wf); err != nil {
        return toolbox.Workflow{}, fmt.Errorf("%w: workflow: %v", toolbox.ErrInvalid, err)
    }
    return wf, nil
}
```

### Implementation:

**Step 4.1 — Add typed request structs (add near line 83, after ContractResolver interface):**

```go
// InvokeRequest is the typed parameters for toolbox.invoke.
type InvokeRequest struct {
    ToolID              string         `json:"toolId"`
    ExecutionContractID string         `json:"executionContractId"`
    Input               map[string]any `json:"input,omitempty"`
    IdempotencyKey      string         `json:"idempotencyKey,omitempty"`
}

// CompileWorkflowRequest is the typed parameters for toolbox.compileWorkflow.
type CompileWorkflowRequest struct {
    Workflow toolbox.Workflow `json:"workflow"`
}

// RegistryRequest is the typed parameters for toolbox.registry.
type RegistryRequest struct {
    ToolKind string `json:"toolKind,omitempty"`
}
```

**Step 4.2 — Rewrite `invoke()` (replace lines 157-198):**

```go
func (s *Service) invoke(ctx context.Context, req Request) (any, error) {
    var params InvokeRequest
    if req.Params != nil {
        encoded, err := json.Marshal(req.Params)
        if err != nil {
            return nil, fmt.Errorf("%w: invalid parameters", toolbox.ErrInvalid)
        }
        if err := json.Unmarshal(encoded, &params); err != nil {
            return nil, fmt.Errorf("%w: invalid parameters: %v", toolbox.ErrInvalid, err)
        }
    }

    if params.ToolID == "" {
        return nil, fmt.Errorf("%w: toolId is required", toolbox.ErrInvalid)
    }
    if params.ExecutionContractID == "" {
        return nil, fmt.Errorf("%w: executionContractId is required", toolbox.ErrInvalid)
    }
    if s.contracts == nil {
        return nil, fmt.Errorf("%w: no execution contract for %q", toolbox.ErrDenied, params.ExecutionContractID)
    }
    contract, ok := s.contracts.Resolve(params.ExecutionContractID)
    if !ok {
        return nil, fmt.Errorf("%w: execution contract %q", toolbox.ErrNotFound, params.ExecutionContractID)
    }

    input := params.Input
    if input == nil {
        input = map[string]any{}
    }

    result, err := s.host.Invoke(ctx, toolbox.Invocation{
        ToolID:              toolbox.ToolID(params.ToolID),
        Input:               input,
        ExecutionContractID: params.ExecutionContractID,
        IdempotencyKey:      params.IdempotencyKey,
    }, contract)
    if err != nil {
        return nil, err
    }
    return map[string]any{"output": result.Output}, nil
}
```

**Step 4.3 — Rewrite `compileWorkflow()` (replace lines 200-210):**

```go
func (s *Service) compileWorkflow(req Request) (any, error) {
    var params CompileWorkflowRequest
    if req.Params != nil {
        encoded, err := json.Marshal(req.Params)
        if err != nil {
            return nil, fmt.Errorf("%w: invalid parameters", toolbox.ErrInvalid)
        }
        if err := json.Unmarshal(encoded, &params); err != nil {
            return nil, fmt.Errorf("%w: invalid parameters: %v", toolbox.ErrInvalid, err)
        }
    }

    plan, err := workflow.Compile(params.Workflow)
    if err != nil {
        return nil, err
    }
    return map[string]any{"plan": plan}, nil
}
```

**Step 4.4 — Rewrite `listRegistry()` (replace lines 212-220):**

```go
func (s *Service) listRegistry(req Request) (any, error) {
    var params RegistryRequest
    if req.Params != nil {
        encoded, err := json.Marshal(req.Params)
        if err != nil {
            return nil, fmt.Errorf("%w: invalid parameters", toolbox.ErrInvalid)
        }
        if err := json.Unmarshal(encoded, &params); err != nil {
            return nil, fmt.Errorf("%w: invalid parameters: %v", toolbox.ErrInvalid, err)
        }
    }

    return map[string]any{"tools": s.host.ListTools(params.ToolKind)}, nil
}
```

**Step 4.5 — Remove old helper functions:**

- Remove `stringField()` (lines 272-278)
- Remove `workflowParams()` (lines 222-236)

### Acceptance Criteria

- [ ] RPC behaves identically (no functional change for valid requests)
- [ ] malformed requests produce proper JSON-RPC errors (CodeInvalidParams)
- [ ] `go test ./toolbox/rpc/...` passes

### Validation

```bash
cd /Users/prool/aa
go vet ./toolbox/rpc/...
go test ./toolbox/rpc/... -v
```

### Pitfalls

- The `json.Marshal(req.Params)` then `json.Unmarshal(..., &params)` approach requires `req.Params` to be serializable to JSON. Since it's `map[string]any`, this works for well-formed requests.
- Existing tests that pass `map[string]any` params should still work (the typed structs are unmarshaled from the same map).
- **Do not change the Response shape** — it still returns `map[string]any{"output": ...}` and `map[string]any{"plan": ...}`.

---

## Slice 5 — Honor TimeoutMS in RPC (Issue #54, P2)

**Goal:** Honor the `TimeoutMS` field from the `AA` request metadata in `Service.Handle`, so long-running provider calls are bounded and don't block indefinitely.

**Inputs:**
- `toolbox/rpc/rpc.go` line 52: `TimeoutMS int` declared in `AA` struct but never referenced in `Handle`
- `toolbox/rpc/rpc.go` line 48-53: `AA` struct definition

### Current code (toolbox/rpc/rpc.go lines 48-53):

```go
type AA struct {
    RPCVersion      string `json:"rpcVersion,omitempty"`
    ContractVersion string `json:"contractVersion,omitempty"`
    Caller          string `json:"caller,omitempty"`
    TimeoutMS       int    `json:"timeoutMs,omitempty"`
}
```

`TimeoutMS` is declared but never used in `Handle`.

### Implementation:

**Step 5.1 — Add timeout logic to Handle (after the validation block, before the switch):**

In the Handle method, after the existing validation (JSON-RPC version + compatible check) but before the dispatch switch, add:

```go
// Apply timeout if specified in the AA metadata.
var innerCtx context.Context
if req.AA != nil && req.AA.TimeoutMS > 0 {
    var cancel context.CancelFunc
    innerCtx, cancel = context.WithTimeout(ctx, time.Duration(req.AA.TimeoutMS)*time.Millisecond)
    defer cancel()
} else {
    innerCtx = ctx
}
```

**Step 5.2 — Use `innerCtx` instead of `ctx` in the dispatch switch.**

**Step 5.3 — Check for timeout after the switch:**

```go
// Check if the request timed out.
if innerCtx.Err() == context.DeadlineExceeded {
    return errResponseData(req.ID, CodeUnavailable, "request timed out", retryable(true))
}
```

**Step 5.4 — Add `time` to imports** (if not already imported).

### Acceptance Criteria

- [ ] requests with `TimeoutMS > 0` are bounded and return a timeout error on expiry
- [ ] requests without `TimeoutMS` still work (no regression)
- [ ] timeout returns proper JSON-RPC error (CodeUnavailable, retryable: true)
- [ ] `go test ./toolbox/rpc/...` passes

### Validation

```bash
cd /Users/prool/aa
go test ./toolbox/rpc/... -v
```

### Pitfalls

- **Do not break existing behavior:** requests without `TimeoutMS` (or with `TimeoutMS <= 0`) must still work exactly as before (no timeout applied).
- The timeout should apply to the entire handler execution, not just the provider call.
- Make sure `time` is imported in the `toolbox/rpc/rpc.go` file.

---

## Slice 6 — Add Structured RPC Error Data (Issue #55, P2)

**Goal:** Add structured diagnostic data to RPC error responses so callers can distinguish error types programmatically.

**Inputs:**
- `toolbox/rpc/rpc.go` lines 290-303: `errResponseFrom` maps errors to codes but `data` only contains `retryable`

### Current code (toolbox/rpc/rpc.go lines 290-303):

```go
func errResponseFrom(id any, err error) Response {
    switch {
    case errors.Is(err, toolbox.ErrInvalid), errors.Is(err, toolbox.ErrCycle):
        return errResponseData(id, CodeInvalidParams, err.Error(), retryable(false))
    case errors.Is(err, toolbox.ErrNotFound):
        return errResponseData(id, CodeNotFound, err.Error(), retryable(false))
    case errors.Is(err, toolbox.ErrDenied), errors.Is(err, toolbox.ErrSandbox), errors.Is(err, toolbox.ErrApproval):
        return errResponseData(id, CodeUnauthorized, err.Error(), retryable(false))
    case errors.Is(err, toolbox.ErrNoProvider):
        return errResponseData(id, CodeUnavailable, err.Error(), retryable(true))
    default:
        return errResponseData(id, CodeInternalError, err.Error(), retryable(false))
    }
}

func retryable(v bool) map[string]any { return map[string]any{"retryable": v} }
```

### Current Error struct (lines 65-69):

```go
type Error struct {
    Code    any            `json:"code"`
    Message string         `json:"message"`
    Data    map[string]any `json:"data,omitempty"`
}
```

### Implementation:

**Step 6.1 — Add `toolboxError` field to error data (modify `errResponseFrom`):**

Replace `errResponseFrom` (lines 290-303) with:

```go
func errResponseFrom(id any, err error) Response {
    switch {
    case errors.Is(err, toolbox.ErrInvalid), errors.Is(err, toolbox.ErrCycle):
        return errResponseData(id, CodeInvalidParams, err.Error(), map[string]any{
            "retryable":    false,
            "toolboxError": "invalid",
        })
    case errors.Is(err, toolbox.ErrNotFound):
        return errResponseData(id, CodeNotFound, err.Error(), map[string]any{
            "retryable":    false,
            "toolboxError": "not_found",
        })
    case errors.Is(err, toolbox.ErrDenied), errors.Is(err, toolbox.ErrSandbox), errors.Is(err, toolbox.ErrApproval):
        return errResponseData(id, CodeUnauthorized, err.Error(), map[string]any{
            "retryable":    false,
            "toolboxError": "denied",
        })
    case errors.Is(err, toolbox.ErrNoProvider):
        return errResponseData(id, CodeUnavailable, err.Error(), map[string]any{
            "retryable":    true,
            "toolboxError": "unavailable",
        })
    default:
        return errResponseData(id, CodeInternalError, err.Error(), map[string]any{
            "retryable":    false,
            "toolboxError": "internal",
        })
    }
}
```

**Step 6.2 — Remove the `retryable()` helper** (line 288) since it's no longer used (each case now has its own inline `map[string]any`).

### Acceptance Criteria

- [ ] error messages unchanged (backward compatible)
- [ ] `data["toolboxError"]` is set to one of: "invalid", "not_found", "denied", "unavailable", "internal"
- [ ] `data["retryable"]` still works (backward compatible)
- [ ] `go test ./toolbox/rpc/...` passes

### Validation

```bash
cd /Users/prool/aa
go test ./toolbox/rpc/... -v
```

### Pitfalls

- The `data` field in the existing `Error` struct is `map[string]any`. Adding `"toolboxError"` is backward compatible — callers that parse `data["retryable"]` will continue to work.
- **Do not change the error message strings** — only add structured data.

---

## Slice 7 — Add Metrics Interface (Issue #57, P2)

**Goal:** Add metrics collection interface to toolbox so invocation patterns, denial rates, and latency are measurable and actionable.

**Inputs:**
- `toolbox/policy/policy.go`: `Engine.log` is `[]AuditEntry` — in-memory, append-only, not exported
- `toolbox/host/host.go`: `Host.cache` is `map[string]toolbox.Result` — in-memory, no hit/miss tracking
- `toolbox/workflow/runtime.go`: `Runtime.Run` returns `RunState` — no metrics on step counts, retries, or failures
- `toolbox/rpc/rpc.go`: no metrics

### Current code:

**`toolbox/policy/policy.go` (Engine struct, lines 61-67):**
```go
type Engine struct {
    mu       sync.Mutex
    enforcer SandboxEnforcer
    clock    toolbox.Clock
    seq      int
    log      []AuditEntry
}
```

**`toolbox/host/host.go` (Host struct, lines 20-26):**
```go
type Host struct {
    registry *registry.Registry
    policy   *policy.Engine

    mu    sync.Mutex
    cache map[string]toolbox.Result
}
```

**`toolbox/workflow/runtime.go` (Runtime struct, lines 110-113):**
```go
type Runtime struct {
    runner Runner
    gates  GateEvaluator
}
```

**`toolbox/rpc/rpc.go` (Service struct, lines 86-92):**
```go
type Service struct {
    host      *host.Host
    contracts ContractResolver

    mu    sync.Mutex
    cache map[string]any
}
```

### Implementation:

**Step 7.1 — Create `toolbox/metrics.go` (new file):**

```go
package toolbox

import "time"

// Metrics defines the interface for toolbox metrics collection.
type Metrics interface {
    // RecordInvocation records a tool invocation.
    RecordInvocation(toolID ToolID, kind string, latency time.Duration, success bool)

    // RecordDenial records a policy denial.
    RecordDenial(toolID ToolID, reason string, capability string)

    // RecordWorkflowStep records a workflow step execution.
    RecordWorkflowStep(stepID string, kind string, attempts int, success bool, latency time.Duration)

    // RecordCacheHit records an idempotency cache hit.
    RecordCacheHit(key string)

    // RecordCacheMiss records an idempotency cache miss.
    RecordCacheMiss(key string)
}

// NoopMetrics is a no-op implementation for when metrics are not needed.
type NoopMetrics struct{}

func (NoopMetrics) RecordInvocation(ToolID, string, time.Duration, bool)      {}
func (NoopMetrics) RecordDenial(ToolID, string, string)                       {}
func (NoopMetrics) RecordWorkflowStep(string, string, int, bool, time.Duration) {}
func (NoopMetrics) RecordCacheHit(string)                                   {}
func (NoopMetrics) RecordCacheMiss(string)                                  {}
```

**Step 7.2 — Wire metrics into PolicyEngine (`toolbox/policy/policy.go`):**

Add `metrics Metrics` field to `Engine` struct (after `clock toolbox.Clock`):

```go
type Engine struct {
    mu       sync.Mutex
    enforcer SandboxEnforcer
    clock    toolbox.Clock
    metrics  Metrics
    seq      int
    log      []AuditEntry
}
```

Add `NewWithMetrics` constructor (after existing `New`):

```go
func NewWithMetrics(enforcer SandboxEnforcer, clock Clock, metrics Metrics) *Engine {
    if clock == nil {
        clock = SystemClock{}
    }
    if metrics == nil {
        metrics = NoopMetrics{}
    }
    return &Engine{enforcer: enforcer, clock: clock, metrics: metrics}
}
```

In `Evaluate()` (after `e.record(decision, c.ID)`), add:

```go
if !decision.Allowed {
    e.metrics.RecordDenial(d.ToolID, d.Reason, "")
}
```

**Step 7.3 — Wire metrics into Host (`toolbox/host/host.go`):**

Add `metrics Metrics` field to `Host` struct:

```go
type Host struct {
    registry *registry.Registry
    policy   *policy.Engine
    metrics  Metrics

    mu    sync.Mutex
    cache map[string]toolbox.Result
}
```

Add `NewWithMetrics` constructor (after existing `New`):

```go
func NewWithMetrics(reg *registry.Registry, eng *policy.Engine, metrics Metrics) *Host {
    return &Host{registry: reg, policy: eng, metrics: metrics, cache: map[string]toolbox.Result{}}
}
```

In `Invoke()` (after provider execution, before caching), add:

```go
latency := time.Since(start)  // Add start := time.Now() at the beginning of Invoke
m.metrics.RecordInvocation(inv.ToolID, manifest.ToolKind, latency, err == nil)
```

**Step 7.4 — Wire metrics into Runtime (`toolbox/workflow/runtime.go`):**

Add `metrics Metrics` field to `Runtime` struct:

```go
type Runtime struct {
    runner  Runner
    gates   GateEvaluator
    metrics Metrics
}
```

Add `NewWithMetrics` constructor (after existing `New`):

```go
func NewWithMetrics(runner Runner, gates GateEvaluator, metrics Metrics) *Runtime {
    return &Runtime{runner: runner, gates: gates, metrics: metrics}
}
```

In `runWithPolicy()` (after each successful step execution), add:

```go
latency := time.Since(start)  // Add start := time.Now() at the beginning of runWithPolicy
rt.metrics.RecordWorkflowStep(step.ID, step.Kind, attempt, err == nil, latency)
```

**Step 7.5 — Wire metrics into RPC Service (`toolbox/rpc/rpc.go`):**

Add `metrics Metrics` field to `Service` struct:

```go
type Service struct {
    host      *host.Host
    contracts ContractResolver
    metrics   Metrics

    mu    sync.Mutex
    cache map[string]any
}
```

Add `NewWithMetrics` constructor (after existing `New`):

```go
func NewWithMetrics(h *host.Host, contracts ContractResolver, metrics Metrics) *Service {
    return &Service{host: h, contracts: contracts, metrics: metrics, cache: map[string]any{}}
}
```

In the pending-flag section of `Handle()` (after storing the result), add:

```go
// Record cache hit/miss.
if cacheKey != "" {
    // Check if this was a cache hit (completed entry, not pending).
    s.metrics.RecordCacheMiss(cacheKey)  // We know it was a miss (we just created the pending entry)
}
```

Note: In the pending-flag pattern, every request with a cacheKey that creates a new pending entry is a cache miss. Requests that find an existing completed result are cache hits.

### Acceptance Criteria

- [ ] existing functionality unchanged (NoopMetrics does nothing)
- [ ] metrics interface is exported and usable by callers who want real metrics
- [ ] `go test ./toolbox/...` passes
- [ ] `go vet ./toolbox/...` passes

### Validation

```bash
cd /Users/prool/aa
go vet ./toolbox/...
go test ./toolbox/... -v
```

### Pitfalls

- The `NoopMetrics` implementation is critical — it ensures existing code continues to work without changes.
- **Do not break existing tests** — they should pass with `NoopMetrics` (which does nothing).
- The metrics interface should be defined in the `toolbox` package (not a sub-package) so all sub-packages can import it.

---

## Summary of All Active Changes

| Issue | Severity | Slice | Files Changed |
|---|---|---|---|
| #52 | P1 | 1 | `toolbox/host/host_test.go`, `toolbox/policy/policy_test.go`, `toolbox/registry/registry_test.go`, `toolbox/rpc/rpc_test.go`, `toolbox/workflow/compile_test.go`, `toolbox/workflow/runtime_test.go` |
| #56 | P2 | 2 | `toolbox/workflow/compile.go`, `toolbox/workflow/compile_test.go` |
| #51 | P0 | 3 | `toolbox/rpc/rpc.go`, `toolbox/rpc/rpc_test.go` |
| #53 | P1 | 4 | `toolbox/rpc/rpc.go`, `toolbox/rpc/rpc_test.go` |
| #54 | P2 | 5 | `toolbox/rpc/rpc.go`, `toolbox/rpc/rpc_test.go` |
| #55 | P2 | 6 | `toolbox/rpc/rpc.go`, `toolbox/rpc/rpc_test.go` |
| #57 | P2 | 7 | `toolbox/metrics.go` (new), `toolbox/policy/policy.go`, `toolbox/host/host.go`, `toolbox/workflow/runtime.go`, `toolbox/rpc/rpc.go`, tests |

## Post-Sprint Checklist

After all slices are complete:

```bash
cd /Users/prool/aa
go build ./...
go vet ./...
go test ./toolbox/... -v -race
go test ./contracts/... -v
# Verify all issues are closed on GitHub
# Update milestone #2 with final status
# Merge PR #60 to main
```

## Notes for the Next Session

1. **Slices 1 and 2 (already resolved):** Issues #58 and #59 are marked as resolved. The contract files are already split into 22 domain-specific files (all under 231 lines). The Python and TypeScript bindings have full parity with Go (all 21 kinds present).

2. **Execution order matters:** Start with Slice 1 (stale version, smallest change), then Slice 2 (config validation), then Slice 3 (TOCTOU race, the P0 critical fix), then Slice 4 (typed structs, benefits from race fix), then Slices 5-7.

3. **Test-first approach is recommended** for each slice: write the failing test, implement the fix, verify the test passes.

4. **The `-race` flag is essential** for verifying the TOCTOU fix (Slice 3).

5. **All changes are backward compatible:** No public API signatures change. Existing callers continue to work.

6. **GitHub PR #60** is the tracking PR. Branch is `update-toolbox-pre-hardening`. Target is `main`.

7. **Milestone #2** is `v1 — Toolbox Pre-Hardening`.

8. **All issue numbers:** Umbrella #50, sub-issues #51-#59.
