# aa-contracts

The single source of truth for every cross-module object and wire protocol.

## Contents

| Path | Purpose |
|---|---|
| [POLICY.md](POLICY.md) | Versioning, compatibility, deprecation, and change process |
| [schemas/v2/](schemas/v2/README.md) | JSON Schema (draft 2020-12) definitions — current generation |
| [schemas/v1/](schemas/v1/README.md) | JSON Schema — previous generation (migration source) |
| [rpc/rpc-v1.md](rpc/rpc-v1.md) | Inter-module JSON-RPC specification |
| [openapi/control-plane-v1.yaml](openapi/control-plane-v1.yaml) | External control-plane HTTP API for the ui and visualizer |
| `go/v2/`, `go/v1/` | Go bindings and validators (current / previous) |
| `typescript/v2/`, `typescript/v1/` | TypeScript bindings (current / previous) |
| `python/aa_contracts/v2.py`, `v1.py` | Python bindings (current / previous) |

## Rules

- `contracts` depends on nothing; everything may depend on `contracts`.
- Modules never hand-roll cross-module DTOs; if a shape is needed across modules, it lives here.
- Observation events are owned by `aa-obsv`; these contracts reference them.
- Additive changes bump the minor version; breaking changes require a new major and a migration note in `POLICY.md`.

## Status

Phase 1 complete and merged: the schema spine, RPC and control-plane specs, Go/TypeScript/Python bindings, and the conformance suite. The architecture refactor adds generation **v2** (`schemas/v2`, `go/v2`, `typescript/v2`, `python/aa_contracts/v2.py`): durable-definition and planning objects (`role_spec`, `model_spec`, `team_spec`, `work_plan`, `execution_plan`, `routine`) and the Role × Model ontology (the `trade` field is removed in `v2`). `v1` remains as the previous generation until consumers migrate.

Module update (trace contract): `trace.schema.json` (document contract 1.1) adds bounded, redacted per-step learning evidence. See [plan.md](../docs/modules/contracts/updates/trace-contract/plan.md) and [summary.md](../docs/modules/contracts/updates/trace-contract/summary.md).

## Directive

[aa-contracts.md](aa-contracts.md) · Architecture: [../docs/architecture/README.md](../docs/architecture/README.md)
