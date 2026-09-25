# aa-sync — module documentation

> Project history for the `aa-sync` module. Part of [docs/modules](../README.md).

**Directive** — [`aa-sync.md`](../../../sync/aa-sync.md) · **Source** — [`sync/`](../../../sync/) · **Updates** — [`updates/`](./updates/)

## Responsibility

Move files and work between machines securely and resiliently, and coordinate parallel work across machines without assuming execution authority.

## Owns

- Paired, mutually authenticated transport.
- Resumable, hash-verified chunked transfer.
- One-way revision sync with tombstones and deletion guards.
- Offline change queue and reconciliation.
- Work-package and artifact distribution.
- Parallelization transport for multi-machine work.

## Must not

- Start an agent runtime or provide arbitrary remote shell.
- Gain or grant execution authority.
- Treat sync as a hidden control channel.

## Interfaces

- Transport and sync protocol.
- Kernel RPC for work-package and artifact distribution.

## Submodules

| Submodule | Responsibility |
|---|---|
| [`distribute`](./distribute.md) | Package distribute moves work packages and artifacts between nodes. |
| [`identity`](./identity.md) | Package identity provides aa-sync peer identity, pairing, and mutual authentication. |
| [`parallel`](./parallel.md) | Package parallel coordinates multi-machine work placement. |
| [`queue`](./queue.md) | Package queue implements the offline change queue and reconciliation. |
| [`revision`](./revision.md) | Package revision implements one-way revision sync with tombstones and deletion guards. |
| [`rpc`](./rpc.md) | Package rpc exposes aa-sync over the aa inter-module RPC v1 methods. |
| [`transfer`](./transfer.md) | Package transfer implements resumable, hash-verified chunked transfer. |
| [`transport`](./transport.md) | Package transport defines aa-sync's transport seam and a deterministic in-memory implementation for tests. |

