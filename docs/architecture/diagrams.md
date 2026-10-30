# Current architecture diagrams

These diagrams describe the **as-built** v1 composition, not the target roadmap.
The target design and decision record is [README.md](README.md); today's behavior
boundary is [current-state.md](../current-state.md).

Source of truth for wiring: [`app/main.go`](../../app/main.go).

## 1. Current composition map

The module-level composition is the shared map used by the README. The detailed
wiring lives in [`app/main.go`](../../app/main.go) and the module directives.

```mermaid
flowchart TB
    ui["aa-ui<br/>TUI + local web control plane"]
    kernel["aa-kernel<br/>control plane"]
    sync["aa-sync<br/>remote/offline/parallel"]
    forge["aa-forge<br/>git/github orchestration"]
    visualizer["aa-visualizer<br/>3D + time travel"]
    toolbox["aa-toolbox<br/>plugins/MCP/workflows"]
    contracts["aa-contracts<br/>schemas · RPC · generated types"]
    obsv["aa-obsv<br/>work observation"]
    memory["aa-memory<br/>system of record"]
    registry["aa-registry<br/>durable definitions"]
    runtime["aa-runtime<br/>worker/lifecycle/sandbox/inference"]

    ui --> kernel
    ui --> visualizer
    ui --> forge
    ui --> sync
    kernel --> contracts
    sync --> contracts
    forge --> contracts
    visualizer --> contracts
    toolbox --> contracts
    obsv --> contracts
    memory --> contracts
    registry --> contracts
    runtime --> contracts
    kernel -.hosts.-> obsv
    kernel --> registry
    kernel --> runtime
    kernel -.RPC.-> sync
    kernel -.RPC.-> forge
    visualizer -.subscribes.-> obsv
    visualizer -.queries.-> memory
```

Text fallback: ui sits over the kernel and the world adapters; kernel hosts
observation; memory and obsv back visualization; every module depends on
contracts; kernel dispatches to runtime and registry.

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
    serve -.->|optional, not wired| inference
    serve -.->|forge adapters, not wired| gh
    fake -.->|offline| tui
```

Text fallback: one local daemon owns state and serves loopback HTTP; both console
surfaces use the same client; the fake client bypasses the daemon for offline
development; inference and forge are adapters/optional services, not authorities; they are
not wired into the default v1 daemon.

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
