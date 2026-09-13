# aa-sync.md

Agent directive for the `sync` module (aa-sync).

## Responsibility

Move files and work between machines securely and resiliently, and coordinate parallel work across machines without assuming execution authority.

## Owns

- Paired, mutually authenticated transport.
- Resumable, hash-verified chunked transfer.
- One-way revision sync with tombstones and deletion guards.
- Offline change queue and reconciliation.
- Work-package and artifact distribution.
- Parallelization transport for multi-machine work.

## Must Not

- Start an agent runtime or provide arbitrary remote shell.
- Gain or grant execution authority.
- Treat sync as a hidden control channel.

## Interfaces

- Transport and sync protocol.
- Kernel RPC for work-package and artifact distribution.

## Rules

1. Loopback and pairing are not authentication; verify identity.
2. Receiver-authoritative apply; fail closed on validation.
3. Sync ordering is derived from record fields, never wall clock.
4. Privacy defaults apply to everything that leaves a node.

## Canonical references

- [Architecture — security and privacy](../docs/architecture/README.md#7-identity-security-and-privacy)
- [Terminology — authority and replica](../docs/reference/terminology.md)
