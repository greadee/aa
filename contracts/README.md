# aa-contracts

The single source of truth for every cross-module object and wire protocol.

## Contents

| Path | Purpose |
|---|---|
| [POLICY.md](POLICY.md) | Versioning, compatibility, deprecation, and change process |
| [schemas/v1/](schemas/v1/README.md) | JSON Schema (draft 2020-12) definitions for all cross-module objects |
| [rpc/rpc-v1.md](rpc/rpc-v1.md) | Inter-module JSON-RPC specification |
| [openapi/control-plane-v1.yaml](openapi/control-plane-v1.yaml) | External control-plane HTTP API for console and visualizer |
| `go/v1/` | Go bindings and validators |
| `typescript/v1/` | TypeScript bindings |
| `python/aa_contracts/` | Python bindings |

## Rules

- `contracts` depends on nothing; everything may depend on `contracts`.
- Modules never hand-roll cross-module DTOs; if a shape is needed across modules, it lives here.
- Observation events are owned by `aa-obsv`; these contracts reference them.
- Additive changes bump the minor version; breaking changes require a new major and a migration note in `POLICY.md`.

## Status

Phase 1. Schemas and specs exist. Bindings and the conformance suite are delivered by the remaining phase 1 slices.

## Directive

[aa-contracts.md](aa-contracts.md) · Architecture: [../docs/architecture/README.md](../docs/architecture/README.md)
