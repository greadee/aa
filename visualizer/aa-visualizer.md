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

- Define its own observation protocol (use `obsv`).
- Store canonical history (use `memory`).
- Import `kernel`.
- Persist file contents or raw prompts.

## Interfaces

- `obsv` `Subscribe` / `Replay`.
- `memory` query API for past sessions.
- Control-plane API for the console.

## Rules

1. Live and replay use the same events and the same renderer.
2. The compatibility profile for metadata (secondary paths, access sequence) is explicit and tested.
3. Node identity is re-derived after replay, not persisted as truth.
4. Rendering has a large-graph performance budget.

## Canonical references

- [Architecture](../docs/architecture/README.md)
- [Observation](../obsv/README.md)
