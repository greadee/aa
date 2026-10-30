# Current architecture diagrams

These diagrams describe the **as-built** v1 composition, not the target roadmap.
The target design and decision record is [README.md](README.md); today's behavior
boundary is [current-state.md](../current-state.md).

Source of truth for wiring: [`app/main.go`](../../app/main.go).

## 1. Current composition map

The composition root owns the process. Modules are libraries with explicit seams;
`app` passes every dependency in `build()`.

```mermaid
flowchart TB
    subgraph Surface["Human surfaces"]
        console["ui console<br/>CLI · TUI · web"]
        fake["ui fake client<br/>offline deterministic"]
    end

    subgraph Composition["app composition root"]
        daemon["aa serve<br/>HTTP control plane"]
        state["projectstate<br/>projects · work · approvals"]
        planes["obsplane · visplane · syncplane"]
    end

    subgraph Control["Control plane"]
        kernel["kernel<br/>allocator · planner · scheduler · gates · context · joblearn"]
        registry["registry<br/>roles · models · teams · routines · policies"]
        toolbox["toolbox<br/>registry · policy · host · workflows"]
    end

    subgraph Execution["Execution"]
        runtime["runtime<br/>worker · sandbox · intake · lifecycle"]
        inference["runtime/inference<br/>Python sidecar, optional"]
    end

    subgraph Knowledge["Knowledge"]
        memory["memory<br/>store · repo · projection · retrieval"]
        obsv["obsv<br/>journal · protocol · reports"]
        vis["visualizer<br/>replay · layout · browse"]
    end

    subgraph World["World"]
        forge["forge<br/>git/GitHub lifecycle"]
        sync["sync<br/>single-node fabric in v1"]
    end

    contracts["contracts v2<br/>schemas · RPC · generated Go/TS/Python"]

    console --> daemon
    fake --> console
    daemon --> state
    daemon --> planes
    daemon --> kernel
    daemon --> memory
    daemon --> obsv
    daemon --> vis
    daemon --> sync
    daemon --> toolbox
    daemon --> registry
    kernel --> runtime
    kernel --> registry
    kernel --> memory
    kernel --> obsv
    kernel --> toolbox
    runtime --> inference
    runtime --> contracts
    state --> memory
    memory --> contracts
    obsv --> contracts
    vis --> contracts
    toolbox --> contracts
    forge --> memory
    forge --> contracts
    sync --> contracts
    registry --> contracts
    kernel --> contracts
    console --> contracts
```

Text fallback: console → app daemon → project state and planes; kernel dispatches
to runtime and toolbox; memory is canonical; obsv and visualizer are read paths;
forge and sync are world adapters; contracts is the shared boundary.

## 2. Contracts and generated bindings

```mermaid
flowchart LR
    schema["contracts/schemas/v2<br/>JSON Schema 2020-12"]
    openapi["contracts/openapi<br/>control-plane-v1.yaml"]
    go["contracts/go/v2"]
    ts["contracts/typescript/v2"]
    py["contracts/python/aa_contracts/v2.py"]
    fixtures["conformance fixtures"]
    parity["check_binding_parity.py<br/>names + fields + types"]

    schema --> go
    schema --> ts
    schema --> py
    schema --> fixtures
    go --> parity
    ts --> parity
    py --> parity
    openapi --> daemon["app control plane"]
    parity --> ci["CI contracts jobs"]
    fixtures --> ci
```

Text fallback: one schema source generates Go, TypeScript, and Python; OpenAPI
describes the control plane; fixtures and the parity gate enforce conformance.

## 3. Primary work data flow

```mermaid
flowchart LR
    user["user / console"]
    api["control plane<br/>POST work package"]
    project["projectstate<br/>durable record"]
    scheduler["kernel scheduler<br/>assignment + lease"]
    gate["human gate<br/>approve / reject"]
    worker["runtime worker"]
    sandbox["runtime sandbox"]
    intake["runtime intake<br/>validate + dedupe"]
    memory["memory store<br/>records + events"]
    obs["obsv journal"]
    vis["visualizer<br/>projection + layout"]

    user --> api
    api --> project
    project --> scheduler
    scheduler --> gate
    gate -->|approved| worker
    gate -->|rejected| project
    worker --> sandbox
    sandbox --> intake
    intake --> memory
    worker --> obs
    memory --> vis
    obs --> vis
    vis --> user
```

Text fallback: work is recorded before dispatch; the scheduler holds an assignment
and lease; a human gate decides; the runtime executes in a sandbox; intake
validates results before memory; observation feeds visualization.

## 4. Control-plane deployment

```mermaid
flowchart LR
    subgraph Host["Local host"]
        serve["aa serve<br/>127.0.0.1:7100/api/v1<br/>bearer token"]
        state["AA_ROOT/.aa<br/>durable state"]
        tui["aa console<br/>TUI"]
        web["aa console --web<br/>127.0.0.1:7101"]
        fake["aa console --fake<br/>no server"]
        inference["aa-inference<br/>optional Python sidecar"]
        gh["git / gh<br/>forge adapters"]
    end

    tui -->|HTTP + bearer| serve
    web -->|HTTP + bearer| serve
    serve --> state
    serve --> inference
    serve --> gh
    fake -.->|offline| tui
```

Text fallback: one local daemon owns state and serves loopback HTTP; both console
surfaces use the same client; the fake client bypasses the daemon for offline
development; inference and forge are adapters, not authorities.

## 5. End-to-end deterministic sequence

```mermaid
sequenceDiagram
    participant U as User
    participant C as Console
    participant A as app control plane
    participant P as projectstate
    participant S as kernel scheduler
    participant R as runtime worker/sandbox
    participant I as runtime intake
    participant M as memory
    participant O as obsv

    U->>C: create project / submit work
    C->>A: POST /projects, /work-packages
    A->>P: persist project + work package
    P->>S: seed graph
    S->>S: assign + lease
    U->>C: approve gate
    C->>A: POST /approvals/{gate}
    A->>P: record approval
    P->>S: apply approval
    S->>R: dispatch assignment
    R->>R: execute in sandbox
    R->>I: result envelope
    I->>I: validate + dedupe
    I->>M: persist result
    S->>O: observation event
    M-->>A: read models
    O-->>A: report
    A-->>C: project, work, assignment, graph, events
    C-->>U: inspect / replay
```

Text fallback: create project → submit work → approve gate → dispatch → execute →
intake → persist → observe → read back in the console.

## Related

- [Current state](../current-state.md)
- [Target architecture](README.md)
- [Control-plane OpenAPI](../../contracts/openapi/control-plane-v1.yaml)
- [Module index](../index/modules.md)
