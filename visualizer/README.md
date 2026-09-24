# aa-visualizer

Time-travel work history, debugging, and 3D graph visualization.

- **Owns:** 3D graph, replay, session trails, live bridge, and memory session browsing.
- **Must not:** own the observation protocol, transport, or event-store; own canonical history.
- **Status:** delivered in `ph8-visualizer`: deterministic projection, layout, replay, browsing, and live `obsv` transport adoption. The shared `obsv` protocol and the `compat` profile replace the protocol, IPC, and event-store duplicated from AAV (see the [module update](module-upd-plan.md), `ISS-OBSV-2`). The Wails/Three UI surface is the remaining follow-up.
- **Directive:** [aa-visualizer.md](aa-visualizer.md)
- **Architecture:** [../docs/architecture/README.md](../docs/architecture/README.md)
- **Module update:** [module-upd-plan.md](module-upd-plan.md)

## Packages

| Package | Responsibility |
|---|---|
| `visualizer` | Re-exported `contracts.Event`, domain types (session, node, edge, graph), sentinel errors, and a deterministic clock |
| `adapter` | The single `obsv/protocol.Event` → `contracts` `v1.Event` mapping; the visualizer's only observation vocabulary |
| `source` | The `EventSource` seam (`Replay`/`Subscribe`) backed by the shared `obsv` transport; `Open` selects the in-process transport now and the kernel socket later |
| `compat` | The explicit, versioned observation-metadata profile (`secondary_paths`, `access_sequence`); unknown metadata is ignored |
| `session` | The deterministic event fold into a graph and the derived node identity |
| `layout` | A pure, order-independent 3D layout with a large-graph budget and benchmark |
| `replay` | The deterministic frame list and the time-travel cursor (snapshot, step, seek) |
| `bridge` | Feeds a live subscription through the same projection and layout path as replay |
| `browse` | Lists and loads past sessions through the aa-memory query seam, then projects them |
| `retention` | Deterministic trimming of the projected session trail |

`obsv` is the only observation protocol, transport, and event-store. A boundary
check (`tools/archtest`) fails if a second one appears under `visualizer/` or if
the `obsv` protocol/transport is consumed outside the single adapter/source.

## Testing

```sh
cd visualizer
go build ./... && go vet ./... && go test ./...
```

The default suites are offline and deterministic: they use the in-memory
`obsv` transport, a fixed clock, and a temporary memory store. No test makes a
network call, and `visualizer` never imports `kernel`. See
[docs/phases/ph8-visualizer](../docs/phases/ph8-visualizer/plan.md) for the phase
plan and test plan.
