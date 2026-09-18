# aa-visualizer

Time-travel work history, debugging, and 3D graph visualization.

- **Owns:** 3D graph, replay, session trails, live bridge, and memory session browsing.
- **Must not:** own the observation protocol or canonical history.
- **Status:** delivered in `ph8-visualizer` (deterministic projection, layout, replay, and browsing; the Wails/Three surface and the live `obsv` transport are follow-ups).
- **Directive:** [aa-visualizer.md](aa-visualizer.md)
- **Architecture:** [../docs/architecture/README.md](../docs/architecture/README.md)

## Packages

| Package | Responsibility |
|---|---|
| `visualizer` | Re-exported `contracts.Event`, domain types (session, node, edge, graph), sentinel errors, and a deterministic clock |
| `source` | The `EventSource` seam (`Replay`/`Subscribe`) over the shared taxonomy, plus a deterministic in-memory fake |
| `compat` | The explicit, versioned observation-metadata profile (`secondary_paths`, `access_sequence`); unknown metadata is ignored |
| `session` | The deterministic event fold into a graph and the derived node identity |
| `layout` | A pure, order-independent 3D layout with a large-graph budget and benchmark |
| `replay` | The deterministic frame list and the time-travel cursor (snapshot, step, seek) |
| `bridge` | Feeds a live subscription through the same projection and layout path as replay |
| `browse` | Lists and loads past sessions through the aa-memory query seam, then projects them |
| `retention` | Deterministic trimming of the projected session trail |

## Testing

```sh
cd visualizer
go build ./... && go vet ./... && go test ./...
```

The default suites are offline and deterministic: they use the in-memory
`source.Fake`, a fixed clock, and a temporary memory store. No test makes a
network call, and `visualizer` never imports `kernel`. See
[docs/phases/ph8-visualizer](../docs/phases/ph8-visualizer/plan.md) for the phase
plan and test plan.
