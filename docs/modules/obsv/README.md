# aa-obsv — module documentation

> Project history for the `aa-obsv` module. Part of [docs/modules](../README.md).

**Directive** — [`aa-obsrv.md`](../../../obsv/aa-obsrv.md) · **Source** — [`obsv/`](../../../obsv/) · **Updates** — [`updates/`](./updates/)

## Responsibility

Observe work events from agents, tools, files, and git, with bounded, failure-open delivery and deterministic replay.

## Owns

- Observation protocol v1 and its durable projection.
- The emitter (bounded, conformance-checked, panic-safe).
- The local transport (`hello`, `append`, `replay`, `subscribe`) with attach-or-own semantics.
- The journal (cursor replay, live subscription, retention).
- Deterministic work reports.
- The Codex `exec` JSONL translator.

## Must not

- Call models or make routing decisions.
- Own the 3D graph, session state, or canonical project history.
- Block agent execution on observation (failure-open).

## Interfaces

- `Ensure` / `Runtime`, `Send`, `Replay`, `Subscribe`.
- Transport methods `hello` / `append` / `replay` / `subscribe`.
- Protocol types (shared with `contracts`).

## Submodules

| Submodule | Responsibility |
|---|---|
| [`codex`](./codex.md) | Package codex translates Codex `exec --json` JSONL output into aa-obsv observation events. |
| [`emit`](./emit.md) | Package emit provides aa-obsv's bounded, failure-open observation emitter. |
| [`journal`](./journal.md) | Package journal is aa-obsv's append-only observation journal. |
| [`protocol`](./protocol.md) | Package protocol defines aa-obsv's observation protocol v1. |
| [`report`](./report.md) | Package report derives deterministic work reports from journaled observation. |
| [`transport`](./transport.md) | Package transport is aa-obsv's local observation transport. |

