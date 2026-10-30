# aa

**aa** is an end-to-end agentic terminal: a modular, local-first platform for
observing, planning, executing, verifying, and continuously learning from
software engineering work performed by humans and AI agents.

A deterministic control plane owns state and coordination. Models are workers
behind a router. Every unit of work is observable, replayable, and attributable.

![aa social preview](docs/media/social-preview.svg)

## Status

The v1 core is implemented and composed in the `app/` daemon:

- contract schemas, RPC, and Go/TypeScript/Python bindings;
- a control-plane HTTP/JSON API with projects, work packages, approvals, nodes,
  tools, observations, visualizer, registry, and lifecycle reads;
- CLI, TUI, and read-only web console surfaces;
- deterministic scheduler, allocator, gates, runtime sandbox, result intake,
  memory store/repo/retrieval, registry definitions, toolbox, single-node sync,
  observation, and visualizer core.

The forge lifecycle/template libraries, the optional Python inference sidecar,
and the GitHub adapters exist in the repository but are not all wired into the
default v1 daemon. The [current state](docs/current-state.md) page separates
composed behavior from available libraries.

Still in progress or deliberately deferred: real authn/authz, release packaging,
performance/chaos testing, the Wails/React/Three visualizer surface, and the
owner-supplied product media. See [current state](docs/current-state.md) and
[known limitations](docs/current-state.md#known-limitations) for the precise
boundary.

## Quickstart

Requirements: Go 1.26. Python 3.12+ is only needed for the inference service;
Node 20+ is only needed for the TypeScript contract bindings.

```sh
git clone https://github.com/greadee/aa.git
cd aa

# deterministic composition check; no model or network required
go run ./app selfcheck
```

Run the control plane and the console:

```sh
# terminal 1: loopback control plane
AA_CONTROL_TOKEN=dev-token go run ./app serve

# terminal 2: text console
go run ./app console --token dev-token

# or the read-only web surface
go run ./app console --web --token dev-token
```

The console can also run fully offline against the deterministic fake control
plane:

```sh
go run ./app console --fake
```

`aa serve` binds `127.0.0.1:7100` by default. When `--token` (or
`AA_CONTROL_TOKEN`) is set, every path except `/api/v1/health` requires that
bearer token. **An empty token disables the check**, so always set a token
outside local development. Loopback is not authentication. `AA_ROOT` selects the
state directory (default `.aa`).

## Architecture

```mermaid
flowchart LR
    console["ui console<br/>CLI · TUI · web"]
    app["app<br/>composition root + HTTP control plane"]
    kernel["kernel<br/>allocator · scheduler · gates · context"]
    runtime["runtime<br/>worker · sandbox · intake · inference (optional)"]
    memory["memory<br/>records · projection · retrieval"]
    registry["registry<br/>roles · models · teams · routines · policies"]
    toolbox["toolbox<br/>tools · policy · workflows"]
    forge["forge<br/>git/GitHub lifecycle"]
    sync["sync<br/>single-node fabric in v1"]
    obsv["obsv<br/>observation journal"]
    vis["visualizer<br/>replay · layout · browse"]
    contracts["contracts v2<br/>schemas · RPC · bindings"]

    console --> app
    app --> kernel
    app --> memory
    app --> obsv
    app --> vis
    app --> sync
    app --> toolbox
    app --> registry
    kernel --> runtime
    kernel --> registry
    kernel --> memory
    runtime --> contracts
    memory --> contracts
    toolbox --> contracts
    forge --> contracts
    sync --> contracts
    obsv --> contracts
    vis --> contracts
    kernel --> contracts
```

The [current diagrams](docs/architecture/diagrams.md) show the as-built
composition, contracts, work flow, control-plane deployment, and end-to-end
sequence. The [architecture record](docs/architecture/README.md) holds the target
design, decisions, and phase plan. The
[architecture index](docs/index/architecture.md) links the current maps and ADRs.

## Modules

| Module | Responsibility |
|---|---|
| [`app`](app/main.go) | Composition root and control-plane daemon |
| [`contracts`](docs/modules/contracts/README.md) | Single source of truth for schemas, RPC, and generated bindings |
| [`registry`](docs/modules/registry/README.md) | Durable definitions: roles, models, teams, capabilities, routines, policies |
| [`obsv`](docs/modules/obsv/README.md) | Product-neutral work observation: protocol, emitter, transport, journal, reports |
| [`kernel`](docs/modules/kernel/README.md) | Control plane: planning, allocation, scheduling, gates, job learning |
| [`runtime`](docs/modules/runtime/README.md) | Execution mechanics: worker, lifecycle, sandbox, intake, inference |
| [`memory`](docs/modules/memory/README.md) | System of record: work/project history, retrieval, promotion |
| [`sync`](docs/modules/sync/README.md) | Remote sync, transfer, identity, and multi-machine coordination |
| [`forge`](docs/modules/forge/README.md) | Git and GitHub orchestration: issues, PRs, checkpoints, audits, releases |
| [`toolbox`](docs/modules/toolbox/README.md) | Tool, plugin, and MCP registry plus configurable workflow runtime |
| [`visualizer`](docs/modules/visualizer/README.md) | Time-travel replay, deterministic layout, session browsing |
| [`ui`](docs/modules/ui/README.md) | Unified interface: CLI, TUI, and read-only web surfaces |

## Media

![Terminal capture of aa selfcheck](docs/media/terminal-selfcheck.svg)

*`aa selfcheck` builds the composed graph and exercises each seam without a model
or network.*

![Terminal capture of the offline console](docs/media/console-tui.svg)

*`aa console --fake` uses the deterministic fake control plane, so the console can
be explored without a running daemon.*

![Current aa v1 composition](docs/media/architecture.svg)

*Current composition. The [diagram source](docs/architecture/diagrams.md) has the
contracts, data-flow, deployment, and end-to-end sequence maps.*

![Read-only web console content capture](docs/media/console-web.svg)

*Read-only web surface served by `aa console --web` against the local control
plane.*

The [media manifest](docs/media/README.md) records formats, alt text, and the
remaining owner-supplied video/social-preview actions. Assets are small and
reproducible.

## Documentation

- [Documentation entry point](docs/index/README.md)
- [Current state and limitations](docs/current-state.md)
- [Architecture](docs/architecture/README.md)
- [Module index](docs/index/modules.md)
- [Phase and sprint index](docs/index/phases.md)
- [GitHub process](docs/github-os/README.md)

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) and the
[definitions of done](docs/github-os/definitions-of-done.md). Implementation
lands on `dev` before `main`; every issue and PR links its validation evidence.

## License

[MIT](LICENSE.md).
