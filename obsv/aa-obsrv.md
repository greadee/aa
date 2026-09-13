# aa-obsrv.md

Agent directive for the `obsv` module (aa-obsv / work observation).

## Responsibility

Observe work events from agents, tools, files, and git, with bounded, failure-open delivery and deterministic replay.

## Owns

- Observation protocol v1 and its durable projection.
- The emitter (bounded, conformance-checked, panic-safe).
- The local transport (`hello`, `append`, `replay`, `subscribe`) with attach-or-own semantics.
- The journal (cursor replay, live subscription, retention).
- Deterministic work reports.
- The Codex `exec` JSONL translator.

## Must Not

- Call models or make routing decisions.
- Own the 3D graph, session state, or canonical project history.
- Block agent execution on observation (failure-open).

## Interfaces

- `Ensure` / `Runtime`, `Send`, `Replay`, `Subscribe`.
- Transport methods `hello` / `append` / `replay` / `subscribe`.
- Protocol types (shared with `contracts`).

## Rules

1. Observation is metadata-only by default; sanitize to a durable allowlist before journaling.
2. Observation failure must never alter agent behavior.
3. Confidence is semantic and must never be promoted (exact, correlated, observed, inferred).
4. The visualizer adopts this protocol; it does not define its own.

## Canonical references

- [Architecture](../docs/architecture/README.md)
- [Protocol and events](../docs/reference/terminology.md)
