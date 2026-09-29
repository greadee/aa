# aa-visualizer.md

Agent directive for the `visualizer` module (aa-visualizer).

## Responsibility

Make work tangible: render the current session live and any past session as a replayable, debuggable 3D graph.

## Owns

- The 3D graph renderer and deterministic layout.
- Time-travel replay and the session trail.
- The live bridge from `obsv` to the renderer.
- Session browsing over `memory`.
- Session analytics, filters, and camera.

## Must Not

- Define its own observation protocol, transport, or event-store (all three are `obsv`'s).
- Store canonical history (use `memory`).
- Import `kernel`.
- Persist file contents or raw prompts.

## Interfaces

- `obsv` `Subscribe` / `Replay`.
- `memory` query API for past sessions.
- Control-plane API for the ui.

## Rules

1. Live and replay use the same events and the same renderer.
2. The compatibility profile for metadata (secondary paths, access sequence) is explicit and tested.
3. Node identity is re-derived after replay, not persisted as truth.
4. Rendering has a large-graph performance budget.
5. The visualizer defines no observation protocol, transport, or event-store: the single `visualizer/adapter` mapping is the only observation vocabulary, enforced by a boundary guard.

## AAV convergence

The shared `obsv` protocol and the `compat` profile replace the duplicated
protocol, IPC, and event-store carried from `agent-action-visualizer` (AAV).
`obsv` is now the only observation vocabulary: `visualizer/source` reads
`obsv` `Replay` and follows `obsv` `Subscribe`, and `visualizer/adapter` is the
one mapping into the `contracts` taxonomy. A boundary check
(`tools/archtest`) fails if a second observation protocol, transport, or
event-store appears under `visualizer/`.

## Canonical references

- [Architecture](../docs/architecture/README.md)
- [Observation](../obsv/README.md)
