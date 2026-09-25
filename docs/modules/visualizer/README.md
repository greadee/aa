# aa-visualizer — module documentation

> Project history for the `aa-visualizer` module. Part of [docs/modules](../README.md).

**Directive** — [`aa-visualizer.md`](../../../visualizer/aa-visualizer.md) · **Source** — [`visualizer/`](../../../visualizer/) · **Updates** — [`updates/`](./updates/)

## Responsibility

Make work tangible: render the current session live and any past session as a replayable, debuggable 3D graph.

## Owns

- The 3D graph renderer and deterministic layout.
- Time-travel replay and the session trail.
- The live bridge from `obsv` to the renderer.
- Session browsing over `memory`.
- Session analytics, filters, and camera.

## Must not

- Define its own observation protocol, transport, or event-store (all three are `obsv`'s).
- Store canonical history (use `memory`).
- Import `kernel`.
- Persist file contents or raw prompts.

## Interfaces

- `obsv` `Subscribe` / `Replay`.
- `memory` query API for past sessions.
- Control-plane API for the ui.

## Submodules

| Submodule | Responsibility |
|---|---|
| [`adapter`](./adapter.md) | Package adapter is the visualizer's single mapping from the shared aa-obsv observation protocol to the visualizer event type. |
| [`bridge`](./bridge.md) | Package bridge connects a live event subscription to the same projection and layout path replay uses. |
| [`browse`](./browse.md) | Package browse lists and loads past sessions from aa-memory through a query seam. |
| [`compat`](./compat.md) | Package compat is the visualizer's explicit observation-metadata profile. |
| [`layout`](./layout.md) | Package layout places a projected graph deterministically in 3D. |
| [`replay`](./replay.md) | Package replay builds a deterministic frame list from a session's events and provides a time-travel cursor over it. |
| [`retention`](./retention.md) | Package retention trims a projected session trail deterministically. |
| [`session`](./session.md) | Package session folds an event stream into a deterministic graph and re-derives stable node identity. |
| [`source`](./source.md) | Package source is the visualizer's event seam. |

